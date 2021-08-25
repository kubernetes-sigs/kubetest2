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

import (
	"testing"
)

func TestValidateVersion(t *testing.T) {
	testCases := []struct {
		desc    string
		version string
		valid   bool
	}{
		{
			desc:    "empty version string is valid",
			version: "",
			valid:   true,
		},
		{
			desc:    "latest version string is valid",
			version: "latest",
			valid:   true,
		},
		{
			desc:    "major.minor version string is valid",
			version: "1.16",
			valid:   true,
		},
		{
			desc:    "major.minor.patch version string is valid",
			version: "1.16.8",
			valid:   true,
		},
		{
			desc:    "full version string is valid",
			version: "1.16.13-gke.400",
			valid:   true,
		},
		{
			desc:    "arbitrary version string is invalid",
			version: "abc.123",
			valid:   false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(st *testing.T) {
			st.Parallel()
			err := validateVersion(tc.version)
			if tc.valid && err != nil {
				t.Errorf("unexpected error %v for %s", err, tc.version)
			} else if !tc.valid && err == nil {
				t.Error("expected error for case but got nil", tc.version)
			}
		})
	}
}

func TestValidateReleaseChannel(t *testing.T) {
	testCases := []struct {
		desc           string
		releaseChannel string
		valid          bool
	}{
		{
			desc:           "empty release channel is valid",
			releaseChannel: "",
			valid:          true,
		},
		{
			desc:           "None release channel is valid",
			releaseChannel: "None",
			valid:          true,
		},
		{
			desc:           "rapid release channel is valid",
			releaseChannel: "rapid",
			valid:          true,
		},
		{
			desc:           "regular release channel is valid",
			releaseChannel: "regular",
			valid:          true,
		},
		{
			desc:           "stable release channel is valid",
			releaseChannel: "stable",
			valid:          true,
		},
		{
			desc:           "latest release channel is invalid",
			releaseChannel: "latest",
			valid:          false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(st *testing.T) {
			st.Parallel()
			err := validateReleaseChannel(tc.releaseChannel)
			if tc.valid && err != nil {
				t.Errorf("unexpected error %v for %s", err, tc.releaseChannel)
			} else if !tc.valid && err == nil {
				t.Error("expected error for case but got nil", tc.releaseChannel)
			}
		})
	}
}

func TestIsClusterVersionMatch(t *testing.T) {
	testCases := []struct {
		desc          string
		version       string
		target        string
		expectMatched bool
	}{
		{
			desc:          "the same cluster version is a match",
			version:       "1.19.10-gke.1000",
			target:        "1.19.10-gke.1000",
			expectMatched: true,
		},
		{
			desc:          "the same major.minor version is a match",
			version:       "1.19",
			target:        "1.19.10-gke.1000",
			expectMatched: true,
		},
		{
			desc:          "the same major.minor.patch version is a match",
			version:       "1.19.10",
			target:        "1.19.10-gke.1000",
			expectMatched: true,
		},
		{
			desc:          "different major.minor.patch version is not a match",
			version:       "1.20.1",
			target:        "1.19.10-gke.1000",
			expectMatched: false,
		},
		{
			desc:          "the same prefix but different numbers is not a match",
			version:       "1.19.1",
			target:        "1.19.10-gke.1000",
			expectMatched: false,
		},
		{
			desc:          "version having more parts than the target cannot be matched",
			version:       "1.19.10.1.2",
			target:        "1.19.10-gke.1000",
			expectMatched: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(st *testing.T) {
			st.Parallel()
			matched := isClusterVersionMatch(tc.version, tc.target)
			if tc.expectMatched && !matched {
				t.Error("expected the versions are matched but not")
			} else if !tc.expectMatched && matched {
				t.Error("expected the versions are not matched but got matched")
			}
		})
	}
}
