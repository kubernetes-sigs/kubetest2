/*
Copyright 2021 The Kubernetes Authors.

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

type NetworkOptions struct {
	Network string `flag:"~network" desc:"Cluster network. Defaults to the default network if not provided. For multi-project use cases, this will be the Shared VPC network name."`

	PrivateClusterAccessLevel    string   `flag:"~private-cluster-access-level" desc:"Private cluster access level, if not empty, must be one of 'no', 'limited' or 'unrestricted'. See the details in https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters."`
	PrivateClusterMasterIPRanges []string `flag:"~private-cluster-master-ip-range" desc:"Private cluster master IP ranges. It should be IPv4 CIDR(s), and its length must be the same as the number of clusters if private cluster is requested."`
	SubnetworkRanges             []string `flag:"~subnetwork-ranges" desc:"Subnetwork ranges as required for shared VPC setup as described in https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_a_network_and_two_subnets. For multi-project profile, it is required and should be in the format of 10.0.4.0/22 10.0.32.0/20 10.4.0.0/14,172.16.4.0/22 172.16.16.0/20 172.16.4.0/22, where the subnetworks configuration for different project are separated by comma, and the ranges of each subnetwork configuration is separated by space."`
	CreateNat                    bool     `flag:"~create-nat" desc:"Configure Cloud NAT allowing outbound connections in cluster with private nodes."`
}
