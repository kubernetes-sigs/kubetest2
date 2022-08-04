/*
Copyright 2022 The Kubernetes Authors.

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

package util

import (
	"testing"
)

func TestFileName(t *testing.T) {
	testCases := []struct {
		desc             string
		version          string
		markerPrefix     string
		expectedFileName string
	}{
		{
			desc:             "latest-green, 1.23",
			version:          "1.23",
			markerPrefix:     "latest-green",
			expectedFileName: "latest-green-1.23.txt",
		},
		{
			desc:             "latest, v1.22.0-alpha.3",
			version:          "v1.22.0-alpha.3",
			markerPrefix:     "latest",
			expectedFileName: "latest-1.22.txt",
		},
		{
			desc:             "latest, invalid tag",
			version:          "beta",
			markerPrefix:     "latest",
			expectedFileName: "latest.txt",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.desc, func(st *testing.T) {
			st.Parallel()
			fileName := fileName(tc.version, tc.markerPrefix)
			if fileName != tc.expectedFileName {
				t.Errorf("FileName mismatch: got %s, want %s.", fileName, tc.expectedFileName)
			}
		})
	}
}
