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

	"sigs.k8s.io/kubetest2/kubetest2-gke/deployer/options"
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

func TestGetInstanceGroupsCaching(t *testing.T) {
	origDescribe := describeClusterInstanceGroups
	defer func() {
		describeClusterInstanceGroups = origDescribe
	}()

	outputs := map[string]string{
		"cluster-empty":     "",
		"cluster-populated": "https://www.googleapis.com/compute/v1/projects/test-proj/zones/us-central1-a/instanceGroupManagers/gke-cluster-populated-pool-12345678-grp",
	}
	callCounts := map[string]int{}

	describeClusterInstanceGroups = func(project, clusterName, location string) ([]byte, error) {
		callCounts[clusterName]++
		return []byte(outputs[clusterName]), nil
	}

	d := &Deployer{
		ProjectOptions: &options.ProjectOptions{
			Projects: []string{"test-proj"},
		},
		ClusterOptions: &options.ClusterOptions{
			Regions: []string{"us-central1"},
		},
		projectClustersLayout: map[string][]cluster{
			"test-proj": {
				{index: 0, name: "cluster-empty"},
				{index: 1, name: "cluster-populated"},
			},
		},
	}

	// 1. First call: both clusters are queried; cluster-empty has 0 IGs, cluster-populated has 1 IG.
	if err := d.GetInstanceGroups(); err != nil {
		t.Fatalf("first GetInstanceGroups() failed: %v", err)
	}
	if callCounts["cluster-empty"] != 1 || callCounts["cluster-populated"] != 1 {
		t.Fatalf("unexpected call counts after first call: %v", callCounts)
	}
	if len(d.instanceGroups["test-proj"]["cluster-empty"]) != 0 {
		t.Errorf("expected 0 IGs for cluster-empty, got %d", len(d.instanceGroups["test-proj"]["cluster-empty"]))
	}
	if len(d.instanceGroups["test-proj"]["cluster-populated"]) != 1 {
		t.Errorf("expected 1 IG for cluster-populated, got %d", len(d.instanceGroups["test-proj"]["cluster-populated"]))
	}

	// 2. Simulate node pool creation on cluster-empty and call GetInstanceGroups() again:
	// cluster-populated should be served from cache, while cluster-empty is re-queried.
	outputs["cluster-empty"] = "https://www.googleapis.com/compute/v1/projects/test-proj/zones/us-central1-a/instanceGroupManagers/gk3-cluster-empty-pool-1-abcdef01-grp"
	if err := d.GetInstanceGroups(); err != nil {
		t.Fatalf("second GetInstanceGroups() failed: %v", err)
	}
	if callCounts["cluster-empty"] != 2 {
		t.Errorf("expected cluster-empty to be queried a 2nd time, got %d", callCounts["cluster-empty"])
	}
	if callCounts["cluster-populated"] != 1 {
		t.Errorf("expected cluster-populated to remain at 1 query (cached), got %d", callCounts["cluster-populated"])
	}
	if len(d.instanceGroups["test-proj"]["cluster-empty"]) != 1 {
		t.Errorf("expected 1 IG for cluster-empty after scale-up, got %d", len(d.instanceGroups["test-proj"]["cluster-empty"]))
	}

	// 3. Third call: both clusters now have non-empty cached IGs, so neither should be re-queried.
	if err := d.GetInstanceGroups(); err != nil {
		t.Fatalf("third GetInstanceGroups() failed: %v", err)
	}
	if callCounts["cluster-empty"] != 2 || callCounts["cluster-populated"] != 1 {
		t.Errorf("expected no additional queries on third call, got %v", callCounts)
	}
}
