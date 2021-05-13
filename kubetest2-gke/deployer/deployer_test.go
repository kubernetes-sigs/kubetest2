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

package deployer

import "testing"

func TestLocationFlag(t *testing.T) {
	testCases := []struct {
		regions    []string
		zones      []string
		retryCount int
		expected   string
	}{
		{
			regions:    []string{"us-central1"},
			zones:      []string{},
			retryCount: 0,
			expected:   "--region=us-central1",
		},
		{
			regions:    []string{},
			zones:      []string{"us-central1-c"},
			retryCount: 0,
			expected:   "--zone=us-central1-c",
		},
		{
			regions:    []string{"us-central1", "us-east"},
			zones:      []string{},
			retryCount: 1,
			expected:   "--region=us-east",
		},
	}

	for _, tc := range testCases {
		got := locationFlag(tc.regions, tc.zones, tc.retryCount)
		if got != tc.expected {
			t.Errorf("expected %q but got %q", tc.expected, got)
		}
	}
}

func TestRegionFromLocation(t *testing.T) {
	testCases := []struct {
		regions    []string
		zones      []string
		retryCount int
		expected   string
	}{
		{
			regions:    []string{"us-central1"},
			zones:      []string{},
			retryCount: 0,
			expected:   "us-central1",
		},
		{
			regions:    []string{},
			zones:      []string{"us-central1-c"},
			retryCount: 0,
			expected:   "us-central1",
		},
		{
			regions:    []string{"us-central1", "us-east"},
			zones:      []string{},
			retryCount: 1,
			expected:   "us-east",
		},
	}

	for _, tc := range testCases {
		got := regionFromLocation(tc.regions, tc.zones, tc.retryCount)
		if got != tc.expected {
			t.Errorf("expected %q but got %q", tc.expected, got)
		}
	}
}
