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

package deployer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPrivateClusterArgs(t *testing.T) {
	testCases := []struct {
		desc           string
		network        string
		accessLevel    string
		masterIPRanges []string
		clusterInfo    cluster
		expected       []string
	}{
		{
			desc:           "no private cluster args are needed for non-private clusters",
			network:        "whatever-network",
			accessLevel:    "",
			masterIPRanges: []string{"whatever-master-IP-range"},
			clusterInfo:    cluster{index: 0, name: "whatever-cluster-name"},
			expected:       []string{},
		},
		{
			desc:           "--private-cluster-master-ip-range can be empty for non-private clusters",
			network:        "whatever-network",
			accessLevel:    "",
			masterIPRanges: []string{},
			clusterInfo:    cluster{index: 0, name: "whatever-cluster-name"},
			expected:       []string{},
		},
		{
			desc:           "test private cluster args for private cluster with no limit",
			network:        "test-network1",
			accessLevel:    string(no),
			masterIPRanges: []string{"172.16.0.32/28"},
			clusterInfo:    cluster{index: 0, name: "cluster1"},
			expected: []string{
				"--create-subnetwork=name=test-network1-cluster1",
				"--enable-ip-alias",
				"--enable-private-nodes",
				"--no-enable-basic-auth",
				"--master-ipv4-cidr=172.16.0.32/28",
				"--no-issue-client-certificate",
				"--enable-master-authorized-networks",
				"--enable-private-endpoint",
			},
		},
		{
			desc:           "test private cluster args for private cluster with limited network access",
			network:        "test-network2",
			accessLevel:    string(limited),
			masterIPRanges: []string{"173.16.0.32/28"},
			clusterInfo:    cluster{index: 0, name: "cluster2"},
			expected: []string{
				"--create-subnetwork=name=test-network2-cluster2",
				"--enable-ip-alias",
				"--enable-private-nodes",
				"--no-enable-basic-auth",
				"--master-ipv4-cidr=173.16.0.32/28",
				"--no-issue-client-certificate",
				"--enable-master-authorized-networks",
			},
		},
		{
			desc:           "test private cluster args for private cluster with unrestricted network access",
			network:        "test-network3",
			accessLevel:    string(unrestricted),
			masterIPRanges: []string{"173.16.0.32/28", "175.16.0.32/22"},
			clusterInfo:    cluster{index: 1, name: "cluster3"},
			expected: []string{
				"--create-subnetwork=name=test-network3-cluster3",
				"--enable-ip-alias",
				"--enable-private-nodes",
				"--no-enable-basic-auth",
				"--master-ipv4-cidr=175.16.0.32/22",
				"--no-issue-client-certificate",
				"--no-enable-master-authorized-networks",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(st *testing.T) {
			st.Parallel()
			actual := privateClusterArgs(tc.network, tc.accessLevel, tc.masterIPRanges, tc.clusterInfo)
			if diff := cmp.Diff(actual, tc.expected); diff != "" {
				st.Error("Got private cluster args (-want, +got) =", diff)
			}
		})
	}
}
