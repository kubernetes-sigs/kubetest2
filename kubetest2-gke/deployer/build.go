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
	"regexp"
	"strings"
)

func (d *deployer) Build() error {
	if err := d.verifyBuildFlags(); err != nil {
		return err
	}
	version, err := d.BuildOptions.Build()
	if err != nil {
		return err
	}
	version = strings.TrimPrefix(version, "v")
	if version, err = normalizeVersion(version); err != nil {
		return err
	}
	// append the kubetest2 run id
	version += "+" + d.commonOptions.RunID()
	if d.BuildOptions.StageLocation != "" {
		if err := d.BuildOptions.Stage(version); err != nil {
			return fmt.Errorf("error staging build: %v", err)
		}
	}
	d.Version = version
	return nil
}

func (d *deployer) verifyBuildFlags() error {
	if d.RepoRoot == "" {
		return fmt.Errorf("required repo-root when building from source")
	}
	d.BuildOptions.RepoRoot = d.RepoRoot
	return d.BuildOptions.Validate()
}

// ensure that the version is a valid gke version
func normalizeVersion(version string) (string, error) {
	// (1.18.10)(-gke)(.601.123)(+existingSuffix)
	// capture repeated occurrences of group 3 and don't capture just last
	re, err := regexp.Compile(`(\d\.\d+\.\d*)(-\w+)?((?:\.\d+)*)(\+.*)*`)
	if err != nil {
		return "", err
	}
	matches := re.FindStringSubmatch(version)
	if matches == nil || len(matches) != 5 {
		return "", fmt.Errorf("%q is not a valid gke version", version)
	}
	fmt.Println(matches)
	// get the core version as is
	finalVersion := matches[1]
	// force append -gke, ignore alpha,beta
	finalVersion += "-gke"
	// add the optional patch number or .0
	if matches[3] == "" {
		finalVersion += ".0"
	} else {
		finalVersion += matches[3]
	}
	// append the optional suffix as is
	finalVersion += matches[4]
	return finalVersion, nil
}
