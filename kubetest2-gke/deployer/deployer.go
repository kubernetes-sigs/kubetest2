/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package deployer implements the kubetest2 GKE deployer
package deployer

import (
	"flag"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/octago/sflags/gen/gpflag"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"sigs.k8s.io/boskos/client"

	"sigs.k8s.io/kubetest2/kubetest2-gke/deployer/options"
	"sigs.k8s.io/kubetest2/pkg/artifacts"
	"sigs.k8s.io/kubetest2/pkg/build"
	"sigs.k8s.io/kubetest2/pkg/types"
)

// Name is the name of the deployer
const Name = "gke"

var GitTag string

const (
	defaultFirewallRuleAllow = "tcp:22,tcp:80,tcp:8080,tcp:30000-32767,udp:30000-32767"
)

const (
	gceStockoutErrorPattern = ".*does not have enough resources available to fulfill.*"
)

type privateClusterAccessLevel string

const (
	no           privateClusterAccessLevel = "no"
	limited      privateClusterAccessLevel = "limited"
	unrestricted privateClusterAccessLevel = "unrestricted"
)

type subnetMode string

const (
	auto   subnetMode = "auto"
	custom subnetMode = "custom"
)

var (
	// poolRe matches instance group URLs of the form `https://www.googleapis.com/compute/v1/projects/some-project/zones/a-zone/instanceGroupManagers/gke-some-cluster-some-pool-90fcb815-grp`
	// or `https://www.googleapis.com/compute/v1/projects/some-project/zones/a-zone/instanceGroupManagers/gk3-some-cluster-some-pool-90fcb815-grp` for GKE in Autopilot mode.
	// Match meaning:
	// m[0]: path starting with zones/
	// m[1]: zone
	// m[2]: pool name (passed to e2es)
	// m[3]: unique hash (used as nonce for firewall rules)
	poolRe = regexp.MustCompile(`zones/([^/]+)/instanceGroupManagers/(gk[e|3]-.*-([0-9a-f]{8})-grp)$`)

	urlRe = regexp.MustCompile(`https://.*/`)

	defaultNodePool = gkeNodePool{
		Nodes: 3,
	}

	defaultWindowsNodePool = gkeNodePool{
		Nodes: 1,
	}
)

type gkeNodePool struct {
	Nodes       int
	MachineType string
}

type ig struct {
	path string
	zone string
	name string
	uniq string
}

type cluster struct {
	// index is the index of the cluster in the list provided via the --cluster-name flag
	index int
	name  string
}

type extraNodepool struct {
	Index       int
	Name        string
	MachineType string
	ImageType   string
	NumNodes    int
}

type Deployer struct {
	// generic parts
	Kubetest2CommonOptions types.Options

	*options.BuildOptions
	*options.CommonOptions
	*options.ProjectOptions
	*options.NetworkOptions
	*options.ClusterOptions

	// doInit helps to make sure the initialization is performed only once
	doInit sync.Once
	// only used for multi-project multi-cluster profile to save the project-clusters mapping
	projectClustersLayout map[string][]cluster
	// project -> cluster -> instance groups
	instanceGroups map[string]map[string][]*ig

	// extra node pools to create, per cluster.
	extraNodePoolSpecs []*extraNodepool

	kubecfgPath  string
	testPrepared bool

	localLogsDir string
	gcsLogsDir   string

	// gke specific details for retrying
	totalTryCount                        int
	retryCount                           int
	retryableErrorPatternsCompiled       []*regexp.Regexp
	subnetworkRangesInternal             [][]string
	privateClusterMasterIPRangesInternal [][]string

	// the total number of Boskos projects to request
	totalBoskosProjectsRequested int

	// boskos struct field will be non-nil when the deployer is
	// using boskos to acquire a GCP project
	boskos *client.Client

	// this channel serves as a signal channel for the hearbeat goroutine
	// so that it can be explicitly closed
	boskosHeartbeatClose chan struct{}
}

// assert that New implements types.NewDeployer
var _ types.NewDeployer = New

// assert that deployer implements types.Deployer
var _ types.Deployer = &Deployer{}

func (d *Deployer) Provider() string {
	return Name
}

func (d *Deployer) Version() string {
	return GitTag
}

// New implements deployer.New for gke
func New(opts types.Options) (types.Deployer, *pflag.FlagSet) {
	d := NewDeployer(opts)

	// register flags
	fs := bindFlags(d)

	// initing the klog flags adds them to goflag.CommandLine
	// they can then be added to the built pflag set
	klog.InitFlags(nil)
	fs.AddGoFlagSet(flag.CommandLine)
	return d, fs
}

// NewDeployer returns a deployer object with fields that are not flag controlled
func NewDeployer(opts types.Options) *Deployer {
	d := &Deployer{
		Kubetest2CommonOptions: opts,
		BuildOptions: &options.BuildOptions{
			CommonBuildOptions: &build.Options{
				Builder:  &build.NoopBuilder{},
				Stager:   &build.NoopStager{},
				Strategy: "make",
			},
		},
		CommonOptions: &options.CommonOptions{
			GCPSSHKeyIgnored: true,
		},
		ProjectOptions: &options.ProjectOptions{
			BoskosLocation:                 defaultBoskosLocation,
			BoskosResourceType:             []string{defaultGKEProjectResourceType},
			BoskosAcquireTimeoutSeconds:    defaultBoskosAcquireTimeoutSeconds,
			BoskosHeartbeatIntervalSeconds: defaultBoskosHeartbeatIntervalSeconds,
			BoskosProjectsRequested:        []int{1},
		},
		NetworkOptions: &options.NetworkOptions{
			Network: "default",
		},
		ClusterOptions: &options.ClusterOptions{
			Environment: "prod",
			NumClusters: 1,
			NumNodes:    defaultNodePool.Nodes,
			MachineType: defaultNodePool.MachineType,
			// Leave ClusterVersion as empty to use the default cluster version.
			ClusterVersion:    "",
			FirewallRuleAllow: defaultFirewallRuleAllow,

			WindowsNumNodes:    defaultWindowsNodePool.Nodes,
			WindowsMachineType: defaultWindowsNodePool.MachineType,

			RetryableErrorPatterns: []string{gceStockoutErrorPattern},
		},
		localLogsDir: filepath.Join(artifacts.BaseDir(), "logs"),
	}

	return d
}

func (d *Deployer) VerifyLocationFlags() error {
	if len(d.Zones) == 0 && len(d.Regions) == 0 {
		return fmt.Errorf("--zone or --region must be set for GKE deployment")
	} else if len(d.Zones) != 0 && len(d.Regions) != 0 {
		return fmt.Errorf("--zone and --region cannot both be set")
	}
	return nil
}

// locationFlag builds the zone/region flag from the provided zone/region
// used by gcloud commands.
func locationFlag(regions, zones []string, retryCount int) string {
	if len(zones) != 0 {
		return "--zone=" + zones[retryCount]
	}
	return "--region=" + regions[retryCount]
}

// location builds a location from the provided zone/region.
func location(regions, zones []string, retryCount int) string {
	if len(zones) != 0 {
		return zones[retryCount]
	}
	return regions[retryCount]
}

// regionFromLocation computes the region from the specified zone/region
// used by some commands (such as subnets), which do not support zones.
func regionFromLocation(regions, zones []string, retryCount int) string {
	if len(zones) != 0 {
		zone := zones[retryCount]
		return zone[0:strings.LastIndex(zone, "-")]
	}
	return regions[retryCount]
}

func bindFlags(d *Deployer) *pflag.FlagSet {
	flags, err := gpflag.Parse(d)
	if err != nil {
		klog.Fatalf("unable to generate flags from deployer")
		return nil
	}

	return flags
}
