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
	e2eAllow      = "tcp:22,tcp:80,tcp:8080,tcp:30000-32767,udp:30000-32767"
	defaultCreate = "container clusters create --quiet"
	image         = "cos"
)

type privateClusterAccessLevel string

const (
	no           privateClusterAccessLevel = "no"
	limited      privateClusterAccessLevel = "limited"
	unrestricted privateClusterAccessLevel = "unrestricted"
)

var (
	// poolRe matches instance group URLs of the form `https://www.googleapis.com/compute/v1/projects/some-project/zones/a-zone/instanceGroupManagers/gke-some-cluster-some-pool-90fcb815-grp`. Match meaning:
	// m[0]: path starting with zones/
	// m[1]: zone
	// m[2]: pool name (passed to e2es)
	// m[3]: unique hash (used as nonce for firewall rules)
	poolRe = regexp.MustCompile(`zones/([^/]+)/instanceGroupManagers/(gke-.*-([0-9a-f]{8})-grp)$`)

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

type deployer struct {
	// generic parts
	commonOptions types.Options

	BuildOptions *options.BuildOptions

	Version string `desc:"Use a specific GKE version e.g. 1.16.13.gke-400 or 'latest'. If --build is specified it will default to building kubernetes from source."`

	// doInit helps to make sure the initialization is performed only once
	doInit sync.Once
	// gke specific details
	projects []string
	zone     string
	region   string
	clusters []string
	// only used for multi-project multi-cluster profile to save the project-clusters mapping
	projectClustersLayout map[string][]string
	nodes                 int
	machineType           string
	network               string
	subnetworkRanges      []string
	environment           string
	createCommandFlag     string
	gcpServiceAccount     string

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
	privateClusterAccessLevel   string
	privateClusterMasterIPRange string

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
			Builder:  &build.NoopBuilder{},
			Stager:   &build.NoopStager{},
			Strategy: "bazel",
		},
		localLogsDir: filepath.Join(opts.ArtifactsDir(), "logs"),
		Version:      "latest",
	}

	// register flags
	fs := bindFlags(d)

	// register flags for klog
	klog.InitFlags(nil)
	fs.AddGoFlagSet(flag.CommandLine)
	return d, fs
}

// verifyCommonFlags validates flags for up phase.
func (d *deployer) verifyUpFlags() error {
	if len(d.projects) == 0 && d.boskosProjectsRequested <= 0 {
		return fmt.Errorf("either --project or --projects-requested with a value larger than 0 must be set for GKE deployment")
	}
	if err := d.verifyNetworkFlags(); err != nil {
		return err
	}
	if len(d.clusters) == 0 {
		return fmt.Errorf("--cluster-name must be set for GKE deployment")
	}
	if _, err := d.location(); err != nil {
		return err
	}
	if d.nodes <= 0 {
		return fmt.Errorf("--num-nodes must be larger than 0")
	}
	if err := validateVersion(d.Version); err != nil {
		return err
	}
	return nil
}

func validateVersion(version string) error {
	switch version {
	case "latest":
		return nil
	default:
		re, err := regexp.Compile(`(\d)\.(\d)+\.(\d)*(.*)`)
		if err != nil {
			return err
		}
		if !re.MatchString(version) {
			return fmt.Errorf("unknown version %q", version)
		}
	}
	return nil
}

// verifyDownFlags validates flags for down phase.
func (d *deployer) verifyDownFlags() error {
	if len(d.clusters) == 0 {
		return fmt.Errorf("--cluster-name must be set for GKE deployment")
	}
	if len(d.projects) == 0 {
		return fmt.Errorf("--project must be set for GKE deployment")
	}
	if _, err := d.location(); err != nil {
		return err
	}
	return nil
}

func (d *deployer) location() (string, error) {
	if d.zone == "" && d.region == "" {
		return "", fmt.Errorf("--zone or --region must be set for GKE deployment")
	} else if d.zone != "" && d.region != "" {
		return "", fmt.Errorf("--zone and --region cannot both be set")
	}

	if d.zone != "" {
		return "--zone=" + d.zone, nil
	}
	return "--region=" + d.region, nil
}

func bindFlags(d *deployer) *pflag.FlagSet {
	flags, err := gpflag.Parse(d)
	if err != nil {
		klog.Fatalf("unable to generate flags from deployer")
		return nil
	}
	flags.StringSliceVar(&d.clusters, "cluster-name", []string{}, "Cluster names separated by comma. Must be set. "+
		"For multi-project profile, it should be in the format of clusterA:0,clusterB:1,clusterC:2, where the index means the index of the project.")
	flags.StringVar(&d.createCommandFlag, "create-command", defaultCreate, "gcloud subcommand used to create a cluster. Modify if you need to pass arbitrary arguments to create.")
	flags.StringVar(&d.gcpServiceAccount, "gcp-service-account", "", "Service account to activate before using gcloud")
	flags.StringVar(&d.network, "network", "default", "Cluster network. Defaults to the default network if not provided. For multi-project use cases, this will be the Shared VPC network name.")
	flags.StringSliceVar(&d.subnetworkRanges, "subnetwork-ranges", []string{}, "Subnetwork ranges as required for shared VPC setup as described in https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_network_and_two_subnets."+
		"For multi-project profile, it is required and should be in the format of `10.0.4.0/22 10.0.32.0/20 10.4.0.0/14,172.16.4.0/22 172.16.16.0/20 172.16.4.0/22`, where the subnetworks configuration for different project"+
		"are separated by comma, and the ranges of each subnetwork configuration is separated by space.")
	flags.StringVar(&d.environment, "environment", "prod", "Container API endpoint to use, one of 'test', 'staging', 'prod', or a custom https:// URL. Defaults to prod if not provided")
	flags.StringSliceVar(&d.projects, "project", []string{}, "Project to deploy to separated by comma.")
	flags.StringVar(&d.region, "region", "", "For use with gcloud commands to specify the cluster region.")
	flags.StringVar(&d.zone, "zone", "", "For use with gcloud commands to specify the cluster zone.")
	flags.IntVar(&d.nodes, "num-nodes", defaultNodePool.Nodes, "For use with gcloud commands to specify the number of nodes for the cluster.")
	flags.StringVar(&d.machineType, "machine-type", defaultNodePool.MachineType, "For use with gcloud commands to specify the machine type for the cluster.")
	flags.BoolVar(&d.gcpSSHKeyIgnored, "ignore-gcp-ssh-key", false, "Whether the GCP SSH key should be ignored or not for bringing up the cluster.")
	flags.BoolVar(&d.workloadIdentityEnabled, "enable-workload-identity", false, "Whether enable workload identity for the cluster or not.")
	flags.StringVar(&d.privateClusterAccessLevel, "private-cluster-access-level", "", "Private cluster access level, if not empty, must be one of 'no', 'limited' or 'unrestricted'")
	flags.StringVar(&d.privateClusterMasterIPRange, "private-cluster-master-ip-range", "172.16.0.32/28", "Private cluster master IP range. It should be an IPv4 CIDR, and must not be empty if private cluster is requested.")
	flags.StringVar(&d.boskosLocation, "boskos-location", defaultBoskosLocation, "If set, manually specifies the location of the Boskos server")
	flags.StringVar(&d.boskosResourceType, "boskos-resource-type", defaultGKEProjectResourceType, "If set, manually specifies the resource type of GCP projects to acquire from Boskos")
	flags.IntVar(&d.boskosAcquireTimeoutSeconds, "boskos-acquire-timeout-seconds", 300, "How long (in seconds) to hang on a request to Boskos to acquire a resource before erroring")
	flags.IntVar(&d.boskosProjectsRequested, "projects-requested", 1, "Number of projects to request from Boskos. It is only respected if projects is empty, and must be larger than zero ")

	return flags
}
