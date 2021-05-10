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
		region   string
		zone     string
		expected string
	}{
		{
			region:   "us-central1",
			zone:     "",
			expected: "--region=us-central1",
		},
		{
			region:   "",
			zone:     "us-central1-c",
			expected: "--zone=us-central1-c",
		},
	}

	for _, tc := range testCases {
		got := locationFlag(tc.region, tc.zone)
		if got != tc.expected {
			t.Errorf("expected %q but got %q", tc.expected, got)
		}
	}
}

func TestRegionFromLocation(t *testing.T) {
	testCases := []struct {
		region   string
		zone     string
		expected string
	}{
		{
			region:   "us-central1",
			zone:     "",
			expected: "us-central1",
		},
		{
			region:   "",
			zone:     "us-central1-c",
			expected: "us-central1",
		},
	}

	for _, tc := range testCases {
		got := regionFromLocation(tc.region, tc.zone)
		if got != tc.expected {
			t.Errorf("expected %q but got %q", tc.expected, got)
		}
	}
}
