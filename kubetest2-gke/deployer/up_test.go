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
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestClusterVersion(t *testing.T) {
	testCases := []struct {
		version string
		valid   bool
	}{
		{"1.18", true},
		{"1.18.16", true},
		{"1.18.16-gke.502", true},
		{"1", false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.version, func(t *testing.T) {
			err := validateVersion(tc.version)
			if tc.valid && err != nil {
				t.Error("unexpected error", err)
			} else if !tc.valid && err == nil {
				t.Error("expected error for case", tc.version)
			}
		})
	}
}

func TestGenerateClusterNames(t *testing.T) {
	testCases := []struct {
		name                 string
		numClusters          int
		uid                  string
		expectedClusterNames []string
	}{
		{
			name:                 "zero clusters",
			uid:                  "foobar",
			expectedClusterNames: []string{},
		},
		{
			name:        "empty uid",
			numClusters: 3,
			expectedClusterNames: []string{
				"kt2-1",
				"kt2-2",
				"kt2-3",
			},
		},
		{
			name:        "3 clusters, 6 character uid",
			numClusters: 3,
			uid:         "foobar",
			expectedClusterNames: []string{
				"kt2-foobar-1",
				"kt2-foobar-2",
				"kt2-foobar-3",
			},
		},
		{
			name:        "20 clusters, 36 character uid",
			numClusters: 20,
			uid:         "abcdefghijklmnopqrstuvwxyz0123456789",
			expectedClusterNames: []string{
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-1",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-2",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-3",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-4",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-5",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-6",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-7",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-8",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-9",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-10",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-11",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-12",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-13",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-14",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-15",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-16",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-17",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-18",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-19",
				"kt2-abcdefghijklmnopqrstuvwxyz0123456-20",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actualClusterNames := generateClusterNames(tc.numClusters, tc.uid)
			if !reflect.DeepEqual(actualClusterNames, tc.expectedClusterNames) {
				t.Errorf("expected cluster names to be: %v\nbut got %v", tc.expectedClusterNames, actualClusterNames)
			}
		})
	}
}

func TestBuildExtraNodePoolOptions(t *testing.T) {
	for _, c := range []struct {
		name             string
		np               string
		expectedNodepool extraNodepool
		expectedError    string
	}{
		{
			name: "valid nodepool",
			np:   "name=extra-nodepool&machine-type=test-machine-type&image-type=test-image-type&num-nodes=2",
			expectedNodepool: extraNodepool{
				Name:        "extra-nodepool",
				MachineType: "test-machine-type",
				ImageType:   "test-image-type",
				NumNodes:    2,
			},
			expectedError: "%!s(<nil>)",
		},
		{
			name:          "num-nodes not set",
			np:            "name=extra-nodepool&machine-type=test-machine-type&image-type=test-image-type",
			expectedError: "num-nodes must be > 0",
		},
		{
			name:          "undefined name",
			np:            "machine-type=test-machine-type&image-type=test-image-type&num-nodes=1",
			expectedError: "name required",
		},

		{
			name:          "undefined machine-type",
			np:            "name=extra-nodepool&image-type=test-image-type&num-nodes=1",
			expectedError: "machine-type required",
		},

		{
			name:          "undefined image-type",
			np:            "name=extra-nodepool&machine-type=test-machine-type&num-nodes=1",
			expectedError: "image-type required",
		},
	} {
		tc := c
		t.Run(tc.name, func(t *testing.T) {
			enp := extraNodepool{}
			err := buildExtraNodePoolOptions(tc.np, &enp)
			if fmt.Sprintf("%s", err) != tc.expectedError {
				t.Logf("unexpected error: want %q, got %q", tc.expectedError, fmt.Errorf("%s", err))
				t.Fail()
			}
			if err != nil {
				return
			}

			if !cmp.Equal(enp, tc.expectedNodepool) {
				t.Logf("unexpected extra nodepool, got(+), want(-): %s",
					cmp.Diff(tc.expectedNodepool, enp))
				t.Fail()
			}
		})

	}
}
