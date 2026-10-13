/*
Copyright The Kubernetes Authors.

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

func TestParseInstanceGroupsFromGcloud(t *testing.T) {
	testCases := []struct {
		name        string
		igs         string
		expected    []*ig
		expectError bool
	}{
		{
			name:     "empty output (e.g. Autopilot with 0 initial node pools)",
			igs:      "",
			expected: []*ig{},
		},
		{
			name:     "whitespace-only output",
			igs:      "  \n\t ",
			expected: []*ig{},
		},
		{
			name: "standard GKE instance groups",
			igs:  "https://www.googleapis.com/compute/v1/projects/test-proj/zones/us-central1-b/instanceGroupManagers/gke-cluster1-default-pool-90fcb815-grp;https://www.googleapis.com/compute/v1/projects/test-proj/zones/us-central1-a/instanceGroupManagers/gke-cluster1-default-pool-12345678-grp\n",
			expected: []*ig{
				{
					path: "zones/us-central1-a/instanceGroupManagers/gke-cluster1-default-pool-12345678-grp",
					zone: "us-central1-a",
					name: "gke-cluster1-default-pool-12345678-grp",
					uniq: "12345678",
				},
				{
					path: "zones/us-central1-b/instanceGroupManagers/gke-cluster1-default-pool-90fcb815-grp",
					zone: "us-central1-b",
					name: "gke-cluster1-default-pool-90fcb815-grp",
					uniq: "90fcb815",
				},
			},
		},
		{
			name: "Autopilot gk3 instance group",
			igs:  "https://www.googleapis.com/compute/v1/projects/test-proj/zones/us-central1-a/instanceGroupManagers/gk3-cluster1-pool-1-abcdef01-grp",
			expected: []*ig{
				{
					path: "zones/us-central1-a/instanceGroupManagers/gk3-cluster1-pool-1-abcdef01-grp",
					zone: "us-central1-a",
					name: "gk3-cluster1-pool-1-abcdef01-grp",
					uniq: "abcdef01",
				},
			},
		},
		{
			name:        "invalid instance group URL",
			igs:         "https://www.googleapis.com/compute/v1/projects/test-proj/zones/us-central1-a/instanceGroupManagers/invalid-url",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseInstanceGroupsFromGcloud(tc.igs)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tc.igs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.igs, err)
			}
			if diff := cmp.Diff(tc.expected, got, cmp.AllowUnexported(ig{})); diff != "" {
				t.Errorf("parseInstanceGroupsFromGcloud(%q) mismatch (-want +got):\n%s", tc.igs, diff)
			}
		})
	}
}

func TestClusterFirewallName(t *testing.T) {
	testCases := []struct {
		name           string
		project        string
		cluster        string
		instanceGroups map[string]map[string][]*ig
		expected       string
	}{
		{
			name:    "cluster with instance groups uses first uniq hash",
			project: "test-proj",
			cluster: "cluster1",
			instanceGroups: map[string]map[string][]*ig{
				"test-proj": {
					"cluster1": {
						{uniq: "12345678"},
						{uniq: "90fcb815"},
					},
				},
			},
			expected: "e2e-ports-12345678",
		},
		{
			name:    "cluster with zero instance groups falls back to cluster name",
			project: "test-proj",
			cluster: "autopilot-cluster",
			instanceGroups: map[string]map[string][]*ig{
				"test-proj": {
					"autopilot-cluster": {},
				},
			},
			expected: "e2e-ports-autopilot-cluster",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := clusterFirewallName(tc.project, tc.cluster, tc.instanceGroups)
			if got != tc.expected {
				t.Errorf("clusterFirewallName() = %q, want %q", got, tc.expected)
			}
		})
	}
}
