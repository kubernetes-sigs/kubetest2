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

import "testing"

func TestNormalizeVersion(t *testing.T) {
	testCases := []struct {
		name            string
		version         string
		expectedVersion string
		expectError     bool
	}{
		{
			name:            "empty input",
			version:         "",
			expectedVersion: "",
			expectError:     true,
		},
		{
			name:            "only core version",
			version:         "1.19.4",
			expectedVersion: "1.19.4-gke.99.0",
			expectError:     false,
		},
		{
			name:            "full version",
			version:         "1.18.10-gke.601",
			expectedVersion: "1.18.10-gke.601",
			expectError:     false,
		},
		{
			name:            "core version, pre-existing suffix",
			version:         "1.17.12+foobar",
			expectedVersion: "1.17.12-gke.99.0+foobar",
			expectError:     false,
		},
		{
			name:            "full version, pre-existing suffix",
			version:         "1.16.13-gke.401+qwerty",
			expectedVersion: "1.16.13-gke.401+qwerty",
			expectError:     false,
		},
		{
			name:            "full version, multiple pre-existing suffix",
			version:         "1.16.13-gke.401+qwerty+foobar",
			expectedVersion: "1.16.13-gke.401+qwerty+foobar",
			expectError:     false,
		},
		{
			name:            "full version, multiple patch, multiple pre-existing suffix",
			version:         "1.16.13-gke.401.123+qwerty+foobar",
			expectedVersion: "1.16.13-gke.401.123+qwerty+foobar",
			expectError:     false,
		},
		{
			name:            "alpha version with patch",
			version:         "1.16.13-alpha.123",
			expectedVersion: "1.16.13-gke.123+alpha",
			expectError:     false,
		},
		{
			name:            "beta version, no patch, pre-existing suffix",
			version:         "1.20.0-beta+qwe123",
			expectedVersion: "1.20.0-gke.99.0+beta+qwe123",
			expectError:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actualVersion, err := normalizeVersion(tc.version)
			if err != nil && !tc.expectError {
				t.Errorf("did not expect an error but got: %v", err)
			}
			if err == nil && tc.expectError {
				t.Errorf("expected an error but got none")
			}
			if actualVersion != tc.expectedVersion {
				t.Errorf("expected version: %q, but got: %q", tc.expectedVersion, actualVersion)
			}
		})
	}
}
