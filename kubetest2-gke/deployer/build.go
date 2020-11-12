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

	"k8s.io/klog"
)

var (
	// (1.18.10)(-gke)(.601.123)(+existingSuffix)
	// capture repeated occurrences of group 3 and don't capture just last
	normalizeVersionRegex = regexp.MustCompile(`(?P<core>\d\.\d+\.\d*)(?P<release>-\w+)?(?P<patch>(?:\.\d+)*)(?P<suffix>\+.*)*`)
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
	matches := normalizeVersionRegex.FindStringSubmatch(version)
	if matches == nil || len(matches) != 5 {
		return "", fmt.Errorf("%q is not a valid gke version: %#v", version, matches)
	}
	namedMatch := make(map[string]string)
	for i, name := range normalizeVersionRegex.SubexpNames() {
		if i != 0 && name != "" {
			namedMatch[name] = matches[i]
		}
	}
	// get the core version as is
	finalVersion := namedMatch["core"]

	// force append -gke, append alpha,beta as a suffix later
	var suffix string
	if namedMatch["release"] != "-gke" && namedMatch["release"] != "" {
		suffix += "+" + strings.TrimPrefix(namedMatch["release"], "-")
	}
	finalVersion += "-gke"

	// add the patch number or .99.0
	if namedMatch["patch"] == "" {
		finalVersion += ".99.0"
	} else {
		finalVersion += namedMatch["patch"]
	}

	// append the optional suffixes
	finalVersion += suffix + namedMatch["suffix"]

	if finalVersion != version {
		klog.V(2).Infof("modified version %q to %q", version, finalVersion)
	}
	return finalVersion, nil
}
