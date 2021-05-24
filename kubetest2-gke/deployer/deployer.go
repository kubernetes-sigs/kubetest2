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
	"k8s.io/klog"
	"sigs.k8s.io/boskos/client"

	"sigs.k8s.io/kubetest2/kubetest2-gke/deployer/options"
	"sigs.k8s.io/kubetest2/pkg/build"
	"sigs.k8s.io/kubetest2/pkg/types"
)

// Name is the name of the deployer
const Name = "gke"

const (
	e2eAllow     = "tcp:22,tcp:80,tcp:8080,tcp:30000-32767,udp:30000-32767"
	defaultImage = "cos"
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

var (
	// poolRe matches instance group URLs of the form `https://www.googleapis.com/compute/v1/projects/some-project/zones/a-zone/instanceGroupManagers/gke-some-cluster-some-pool-90fcb815-grp`
	// or `https://www.googleapis.com/compute/v1/projects/some-project/zones/a-zone/instanceGroupManagers/gk3-some-cluster-some-pool-90fcb815-grp` for GKE in Autopilot mode.
	// Match meaning:
	// m[0]: path starting with zones/
	// m[1]: zone
	// m[2]: pool name (passed to e2es)
	// m[3]: unique hash (used as nonce for firewall rules)
	poolRe = regexp.MustCompile(`zones/([^/]+)/instanceGroupManagers/(gk[e|3]-.*-([0-9a-f]{8})-grp)$`)

	urlRe           = regexp.MustCompile(`https://.*/`)
	defaultNodePool = gkeNodePool{
		Nodes:       3,
		MachineType: "n1-standard-2",
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

type deployer struct {
	// generic parts
	commonOptions types.Options

	BuildOptions *options.BuildOptions
	UpOptions    *options.UpOptions

	RepoRoot       string `desc:"Path to root of the kubernetes repo. Used with --build and for dumping cluster logs."`
	ReleaseChannel string `desc:"Use a GKE release channel, could be one of empty, rapid, regular and stable - https://cloud.google.com/kubernetes-engine/docs/concepts/release-channels"`
	Version        string `desc:"Use a specific GKE version e.g. 1.16.13.gke-400, 'latest' or ''. If --build is specified it will default to building kubernetes from source."`

	// doInit helps to make sure the initialization is performed only once
	doInit sync.Once
	// gke specific details
	projects                       []string
	zones                          []string
	regions                        []string
	retryCount                     int
	retryableErrorPatterns         []string
	retryableErrorPatternsCompiled []*regexp.Regexp
	clusters                       []string
	// only used for multi-project multi-cluster profile to save the project-clusters mapping
	projectClustersLayout map[string][]cluster
	nodes                 int
	machineType           string
	imageType             string
	network               string
	subnetworkRanges      []string
	environment           string
	gcpServiceAccount     string

	gcloudCommandGroup string
	autopilot          bool
	gcloudExtraFlags   string
	createCommandFlag  string

	kubecfgPath  string
	testPrepared bool
	// project -> cluster -> instance groups
	instanceGroups map[string]map[string][]*ig

	localLogsDir string
	gcsLogsDir   string

	// whether the GCP SSH key is required or not
	gcpSSHKeyIgnored bool

	// Enable workload identity or not.
	// See the details in https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity
	workloadIdentityEnabled bool

	// Private cluster access level, must be one of "no", "limited" and "unrestricted".
	// See the details in https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters
	privateClusterAccessLevel    string
	privateClusterMasterIPRanges []string

	boskosLocation              string
	boskosResourceType          string
	boskosAcquireTimeoutSeconds int
	// number of boskos projects to request if `projects` is empty
	boskosProjectsRequested int

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
var _ types.Deployer = &deployer{}

func (d *deployer) Provider() string {
	return Name
}

// New implements deployer.New for gke
func New(opts types.Options) (types.Deployer, *pflag.FlagSet) {
	// create a deployer object and set fields that are not flag controlled
	d := &deployer{
		commonOptions: opts,
		BuildOptions: &options.BuildOptions{
			CommonBuildOptions: &build.Options{
				Builder:  &build.NoopBuilder{},
				Stager:   &build.NoopStager{},
				Strategy: "make",
			},
		},
		UpOptions: &options.UpOptions{
			NumClusters: 1,
		},
		localLogsDir: filepath.Join(opts.RunDir(), "logs"),
		// Leave Version as empty to use the default cluster version.
		Version: "",
	}

	// register flags
	fs := bindFlags(d)

	// register flags for klog
	klog.InitFlags(nil)
	fs.AddGoFlagSet(flag.CommandLine)
	return d, fs
}

func (d *deployer) verifyLocationFlags() error {
	if len(d.zones) == 0 && len(d.regions) == 0 {
		return fmt.Errorf("--zone or --region must be set for GKE deployment")
	} else if len(d.zones) != 0 && len(d.regions) != 0 {
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

// regionFromLocation computes the region from the specified zone/region
// used by some commands (such as subnets), which do not support zones.
func regionFromLocation(regions, zones []string, retryCount int) string {
	if len(zones) != 0 {
		zone := zones[retryCount]
		return zone[0:strings.LastIndex(zone, "-")]
	}
	return regions[retryCount]
}

func bindFlags(d *deployer) *pflag.FlagSet {
	flags, err := gpflag.Parse(d)
	if err != nil {
		klog.Fatalf("unable to generate flags from deployer")
		return nil
	}

	flags.StringVar(&d.gcloudCommandGroup, "gcloud-command-group", "", "gcloud command group, can be one of empty, alpha, beta")
	flags.BoolVar(&d.autopilot, "autopilot", false, "Whether to create GKE Autopilot clusters or not")
	flags.StringVar(&d.gcloudExtraFlags, "gcloud-extra-flags", "", "Extra gcloud flags to pass when creating the clusters")
	flags.StringVar(&d.createCommandFlag, "create-command", "", "gcloud subcommand and additional flags used to create a cluster, such as `container clusters create --quiet`."+
		"If it's specified, --gcloud-command-group, --autopilot, --gcloud-extra-flags will be ignored.")
	flags.StringSliceVar(&d.clusters, "cluster-name", []string{}, "Cluster names separated by comma. Must be set. "+
		"For multi-project profile, it should be in the format of clusterA:0,clusterB:1,clusterC:2, where the index means the index of the project.")
	flags.StringVar(&d.gcpServiceAccount, "gcp-service-account", "", "Service account to activate before using gcloud")
	flags.StringVar(&d.network, "network", "default", "Cluster network. Defaults to the default network if not provided. For multi-project use cases, this will be the Shared VPC network name.")
	flags.StringSliceVar(&d.subnetworkRanges, "subnetwork-ranges", []string{}, "Subnetwork ranges as required for shared VPC setup as described in https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_network_and_two_subnets."+
		"For multi-project profile, it is required and should be in the format of `10.0.4.0/22 10.0.32.0/20 10.4.0.0/14,172.16.4.0/22 172.16.16.0/20 172.16.4.0/22`, where the subnetworks configuration for different project"+
		"are separated by comma, and the ranges of each subnetwork configuration is separated by space.")
	flags.StringVar(&d.environment, "environment", "prod", "Container API endpoint to use, one of 'test', 'staging', 'prod', or a custom https:// URL. Defaults to prod if not provided")
	flags.StringSliceVar(&d.projects, "project", []string{}, "Comma separated list of GCP Project(s) to use for creating the cluster.")
	flags.StringSliceVar(&d.regions, "region", []string{}, "Comma separated list for use with gcloud commands to specify the cluster region(s). The first region will be considered the primary region, and the rest will be considered the backup regions.")
	flags.StringSliceVar(&d.zones, "zone", []string{}, "Comma separated list for use with gcloud commands to specify the cluster zone(s). The first zone will be considered the primary zone, and the rest will be considered the backup zones.")
	flags.StringSliceVar(&d.retryableErrorPatterns, "retryable-error-patterns", []string{gceStockoutErrorPattern}, "Comma separated list of regex match patterns for retryable errors during cluster creation.")
	flags.IntVar(&d.nodes, "num-nodes", defaultNodePool.Nodes, "For use with gcloud commands to specify the number of nodes for the cluster.")
	flags.StringVar(&d.machineType, "machine-type", defaultNodePool.MachineType, "For use with gcloud commands to specify the machine type for the cluster.")
	flags.StringVar(&d.imageType, "image-type", defaultImage, "The image type to use for the cluster.")
	flags.BoolVar(&d.gcpSSHKeyIgnored, "ignore-gcp-ssh-key", true, "Whether the GCP SSH key should be ignored or not for bringing up the cluster.")
	flags.BoolVar(&d.workloadIdentityEnabled, "enable-workload-identity", false, "Whether enable workload identity for the cluster or not.")
	flags.StringVar(&d.privateClusterAccessLevel, "private-cluster-access-level", "", "Private cluster access level, if not empty, must be one of 'no', 'limited' or 'unrestricted'")
	flags.StringSliceVar(&d.privateClusterMasterIPRanges, "private-cluster-master-ip-range", []string{"172.16.0.32/28"}, "Private cluster master IP ranges. It should be IPv4 CIDR(s), and its length must be the same as the number of clusters if private cluster is requested.")
	flags.StringVar(&d.boskosLocation, "boskos-location", defaultBoskosLocation, "If set, manually specifies the location of the Boskos server")
	flags.StringVar(&d.boskosResourceType, "boskos-resource-type", defaultGKEProjectResourceType, "If set, manually specifies the resource type of GCP projects to acquire from Boskos")
	flags.IntVar(&d.boskosAcquireTimeoutSeconds, "boskos-acquire-timeout-seconds", 300, "How long (in seconds) to hang on a request to Boskos to acquire a resource before erroring")
	flags.IntVar(&d.boskosProjectsRequested, "projects-requested", 1, "Number of projects to request from Boskos. It is only respected if projects is empty, and must be larger than zero ")

	return flags
}
