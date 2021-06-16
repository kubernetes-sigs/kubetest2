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

import "fmt"

type UpOptions struct {
	GcloudCommandGroup string `flag:"~gcloud-command-group" desc:"gcloud command group, can be one of empty, alpha, beta."`
	Autopilot          bool   `flag:"~autopilot" desc:"Whether to create GKE Autopilot clusters or not."`
	GcloudExtraFlags   string `flag:"~gcloud-extra-flags" desc:"Extra gcloud flags to pass when creating the clusters."`
	CreateCommandFlag  string `flag:"~create-command" desc:"gcloud subcommand and additional flags used to create a cluster, such as container clusters create --quiet. If it's specified, --gcloud-command-group, --autopilot, --gcloud-extra-flags will be ignored."`

	NumClusters             int    `flag:"~num-clusters" desc:"Number of clusters to create, will auto-generate names as (kt2-<run-id>-<index>)."`
	MachineType             string `flag:"~machine-type" desc:"For use with gcloud commands to specify the machine type for the cluster."`
	NumNodes                int    `flag:"~num-nodes" desc:"For use with gcloud commands to specify the number of nodes for the cluster."`
	ImageType               string `flag:"~image-type" desc:"The image type to use for the cluster."`
	ReleaseChannel          string `desc:"Use a GKE release channel, could be one of empty, rapid, regular and stable - https://cloud.google.com/kubernetes-engine/docs/concepts/release-channels"`
	Version                 string `desc:"Use a specific GKE version e.g. 1.16.13.gke-400, 'latest' or ''. If --build is specified it will default to building kubernetes from source."`
	WorkloadIdentityEnabled bool   `flag:"~enable-workload-identity" desc:"Whether enable workload identity for the cluster or not. See the details in https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity."`
	GCPSSHKeyIgnored        bool   `flag:"~ignore-gcp-ssh-key" desc:"Whether the GCP SSH key should be ignored or not for bringing up the cluster."`
	WindowsEnabled          bool   `flag:"~enable-windows" desc:"Whether enable Windows node pool in the cluster or not."`
	WindowsNumNodes         int    `flag:"~windows-num-nodes" desc:"For use with gcloud commands to specify the number of nodes for Windows node pools in the cluster."`
	WindowsMachineType      string `flag:"~windows-machine-type" desc:"For use with gcloud commands to specify the machine type for Windows node in the cluster."`
	WindowsImageType        string `flag:"~windows-image-type" desc:"The Windows image type to use for the cluster."`

	BoskosLocation                 string `flag:"~boskos-location" desc:"If set, manually specifies the location of the Boskos server."`
	BoskosResourceType             string `flag:"~boskos-resource-type" desc:"If set, manually specifies the resource type of GCP projects to acquire from Boskos."`
	BoskosAcquireTimeoutSeconds    int    `flag:"~boskos-acquire-timeout-seconds" desc:"How long (in seconds) to hang on a request to Boskos to acquire a resource before erroring."`
	BoskosHeartbeatIntervalSeconds int    `flag:"~boskos-heartbeat-interval-seconds" desc:"How often (in seconds) to send a heartbeat to Boskos to hold the acquired resource. 0 means no heartbeat."`
	BoskosProjectsRequested        int    `flag:"~projects-requested" desc:"Number of projects to request from Boskos. It is only respected if projects is empty, and must be larger than zero."`

	PrivateClusterAccessLevel    string   `flag:"~private-cluster-access-level" desc:"Private cluster access level, if not empty, must be one of 'no', 'limited' or 'unrestricted'. See the details in https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters."`
	PrivateClusterMasterIPRanges []string `flag:"~private-cluster-master-ip-range" desc:"Private cluster master IP ranges. It should be IPv4 CIDR(s), and its length must be the same as the number of clusters if private cluster is requested."`
	SubnetworkRanges             []string `flag:"~subnetwork-ranges" desc:"Subnetwork ranges as required for shared VPC setup as described in https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_network_and_two_subnets. For multi-project profile, it is required and should be in the format of 10.0.4.0/22 10.0.32.0/20 10.4.0.0/14,172.16.4.0/22 172.16.16.0/20 172.16.4.0/22, where the subnetworks configuration for different project are separated by comma, and the ranges of each subnetwork configuration is separated by space."`

	RetryableErrorPatterns []string `flag:"~retryable-error-patterns" desc:"Comma separated list of regex match patterns for retryable errors during cluster creation."`
}

func (uo *UpOptions) Validate() error {
	// allow max 99 clusters (should be sufficient for most use cases)
	if uo.NumClusters < 1 || uo.NumClusters > 99 {
		return fmt.Errorf("need to specify between 1 and 99 clusters got %q: ", uo.NumClusters)
	}
	return nil
}
