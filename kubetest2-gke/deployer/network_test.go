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
		projects       []string
		network        string
		accessLevel    string
		masterIPRanges []string
		clusterInfo    Cluster
		autopilot      bool
		expected       []string
	}{
		{
			desc:           "test private cluster args for private cluster with no limit",
			projects:       []string{"project1"},
			network:        "test-network1",
			accessLevel:    string(no),
			masterIPRanges: []string{"172.16.0.32/28"},
			clusterInfo:    Cluster{Index: 0, Name: "cluster1"},
			expected: []string{
				"--enable-private-nodes",
				"--enable-ip-alias",
				"--no-enable-basic-auth",
				"--master-ipv4-cidr=172.16.0.32/28",
				"--no-issue-client-certificate",
				"--create-subnetwork=name=test-network1-cluster1",
				"--enable-master-authorized-networks",
				"--enable-private-endpoint",
			},
		},
		{
			desc:           "test private cluster args for private cluster with limited network access",
			projects:       []string{"project1"},
			network:        "test-network2",
			accessLevel:    string(limited),
			masterIPRanges: []string{"173.16.0.32/28"},
			clusterInfo:    Cluster{Index: 0, Name: "cluster2"},
			expected: []string{
				"--enable-private-nodes",
				"--enable-ip-alias",
				"--no-enable-basic-auth",
				"--master-ipv4-cidr=173.16.0.32/28",
				"--no-issue-client-certificate",
				"--create-subnetwork=name=test-network2-cluster2",
				"--enable-master-authorized-networks",
			},
		},
		{
			desc:           "test private cluster args for private cluster with unrestricted network access",
			projects:       []string{"project1"},
			network:        "test-network3",
			accessLevel:    string(unrestricted),
			masterIPRanges: []string{"173.16.0.32/28", "175.16.0.32/22"},
			clusterInfo:    Cluster{Index: 1, Name: "cluster3"},
			expected: []string{
				"--enable-private-nodes",
				"--enable-ip-alias",
				"--no-enable-basic-auth",
				"--master-ipv4-cidr=175.16.0.32/22",
				"--no-issue-client-certificate",
				"--create-subnetwork=name=test-network3-cluster3",
				"--no-enable-master-authorized-networks",
			},
		},
		{
			desc:           "--create-submnetwork is not needed for private clusters with multi-project profile",
			projects:       []string{"project1", "project2"},
			network:        "test-network4",
			accessLevel:    string(unrestricted),
			masterIPRanges: []string{"173.16.0.32/28", "175.16.0.32/22"},
			clusterInfo:    Cluster{Index: 1, Name: "cluster3"},
			expected: []string{
				"--enable-private-nodes",
				"--enable-ip-alias",
				"--no-enable-basic-auth",
				"--master-ipv4-cidr=175.16.0.32/22",
				"--no-issue-client-certificate",
				"--no-enable-master-authorized-networks",
			},
		},
		{
			desc:           "multiple flags are not needed for GKE Autopilot private clusters",
			projects:       []string{"project1"},
			network:        "test-network5",
			accessLevel:    string(unrestricted),
			masterIPRanges: []string{"173.16.0.32/28", "175.16.0.32/22"},
			clusterInfo:    Cluster{Index: 0, Name: "cluster1"},
			autopilot:      true,
			expected: []string{
				"--enable-private-nodes",
				"--create-subnetwork=name=test-network5-cluster1",
				"--no-enable-master-authorized-networks",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(st *testing.T) {
			st.Parallel()
			actual := getPrivateClusterArgs(tc.projects, tc.network, tc.accessLevel, tc.masterIPRanges, tc.clusterInfo, tc.autopilot)
			if diff := cmp.Diff(actual, tc.expected); diff != "" {
				st.Error("Got private cluster args (-want, +got) =", diff)
			}
		})
	}
}

func TestAssertNoOverlaps(t *testing.T) {
	testCases := []struct {
		ranges     []string
		shouldPass bool
	}{
		{
			ranges:     []string{"10.0.0.0/24", "11.0.0.0/24"},
			shouldPass: true,
		},
		{
			ranges:     []string{"10.0.0.0/24", "10.0.0.0/25"},
			shouldPass: false,
		},
		{
			ranges:     []string{"10.0.0.0/24", "11.0.0.0/24", "10.0.0.0/25"},
			shouldPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := assertNoOverlaps(tc.ranges)
		if (err == nil) != tc.shouldPass {
			if tc.shouldPass {
				t.Errorf("test case should have passed, but failed: %q (error: %v)", tc.ranges, err)
			} else {
				t.Errorf("test case should have failed, but passed: %q", tc.ranges)
			}
		}
	}
}
