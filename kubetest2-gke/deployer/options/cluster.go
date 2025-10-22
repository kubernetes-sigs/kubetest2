/*
Copyright 2020 The Kubernetes Authors.

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

package options

import (
	"fmt"
	"time"
)

type ExtraNodePoolOptions struct {
	Name        string
	MachineType string
	ImageType   string
	NumNodes    int
}

type ClusterOptions struct {
	Environment string `flag:"~environment" desc:"Container API endpoint to use, one of 'test', 'staging', 'prod', or a custom https:// URL. Defaults to prod if not provided"`

	GcloudCommand      string `flag:"~gcloud-command" desc:"gcloud command used to create a cluster. Modify if you need to pass custom gcloud to create cluster. Defaults to gcloud if not provided"`
	GcloudCommandGroup string `flag:"~gcloud-command-group" desc:"gcloud command group, can be one of empty, alpha, beta."`
	Autopilot          bool   `flag:"~autopilot" desc:"Whether to create GKE Autopilot clusters or not."`
	GcloudExtraFlags   string `flag:"~gcloud-extra-flags" desc:"Extra gcloud flags to pass when creating the clusters."`
	CreateCommandFlag  string `flag:"~create-command" desc:"gcloud subcommand and additional flags used to create a cluster, such as container clusters create --quiet. If it's specified, --gcloud-command-group, --autopilot, --gcloud-extra-flags will be ignored."`

	Regions []string `flag:"~region" desc:"Comma separated list for use with gcloud commands to specify the cluster region(s). The first region will be considered the primary region, and the rest will be considered the backup regions."`
	Zones   []string `flag:"~zone" desc:"Comma separated list for use with gcloud commands to specify the cluster zone(s). The first zone will be considered the primary zone, and the rest will be considered the backup zones."`

	NumClusters             int      `flag:"~num-clusters" desc:"Number of clusters to create, will auto-generate names as (kt2-<run-id>-<index>)."`
	Clusters                []string `flag:"~cluster-name" desc:"Cluster names separated by comma. Must be set. For multi-project profile, it should be in the format of clusterA:0,clusterB:1,clusterC:2, where the index means the index of the project."`
	MachineType             string   `flag:"~machine-type" desc:"For use with gcloud commands to specify the machine type for the cluster."`
	NumNodes                int      `flag:"~num-nodes" desc:"For use with gcloud commands to specify the number of nodes for each of the cluster's zones."`
	ImageType               string   `flag:"~image-type" desc:"The image type to use for the cluster."`
	ReleaseChannel          string   `desc:"Use a GKE release channel, could be one of empty, rapid, regular, stable and extended - https://cloud.google.com/kubernetes-engine/docs/concepts/release-channels"`
	LegacyClusterVersion    string   `flag:"~version,deprecated" desc:"Use --cluster-version instead"`
	ClusterVersion          string   `desc:"Use a specific GKE version e.g. 1.16.13.gke-400, 'latest' or ''. If --build is specified it will default to building kubernetes from source."`
	WorkloadIdentityEnabled bool     `flag:"~enable-workload-identity" desc:"Whether enable workload identity for the cluster or not. See the details in https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity."`
	FirewallRuleAllow       string   `desc:"A list of protocols and ports whose traffic will be allowed for the firewall rules created for the cluster."`

	WindowsEnabled     bool   `flag:"~enable-windows" desc:"Whether enable Windows node pool in the cluster or not."`
	WindowsNumNodes    int    `flag:"~windows-num-nodes" desc:"For use with gcloud commands to specify the number of nodes for Windows node pools in the cluster."`
	WindowsMachineType string `flag:"~windows-machine-type" desc:"For use with gcloud commands to specify the machine type for Windows node in the cluster."`
	WindowsImageType   string `flag:"~windows-image-type" desc:"The Windows image type to use for the cluster."`

	NodePoolCreateConcurrency int      `flag:"~nodepool-create-concurrency" desc:"Number of nodepools to create concurrently, default is 1"`
	ExtraNodePool             []string `flag:"~extra-nodepool" desc:"create an extra nodepool. repeat the flag for another nodepool. options as key=value&key=value... supported options are name,machine-type,image-type,num-nodes. "`

	RetryableErrorPatterns []string `flag:"~retryable-error-patterns" desc:"Comma separated list of regex match patterns for retryable errors during cluster creation."`

	DownTimeout time.Duration `flag:"~down-timeout" desc:"Timeout for gcloud container clusters delete call."`
}

func (uo *ClusterOptions) Validate() error {
	// allow max 99 clusters (should be sufficient for most use cases)
	if uo.NumClusters < 1 || uo.NumClusters > 99 {
		return fmt.Errorf("need to specify between 1 and 99 clusters got %q: ", uo.NumClusters)
	}

	return nil
}
