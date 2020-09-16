/*
Copyright 2019 The Kubernetes Authors.

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

// Package build implements a common system for building kubernetes for deployers to use.
package build

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

type Builder interface {
	// Build determines how kubernetes artifacts are built from sources or existing artifacts
	// and returns the version being built
	Build() (string, error)
}

type NoopBuilder struct{}

var _ Builder = &NoopBuilder{}

func (n *NoopBuilder) Build() (string, error) {
	return "", nil
}

// sourceVersion the kubernetes git version based on hack/print-workspace-status.sh
// the raw version is also returned
func sourceVersion(kubeRoot string) (string, error) {
	// get the version output
	cmd := exec.Command("sh", "-c", "hack/print-workspace-status.sh")
	cmd.SetDir(kubeRoot)
	output, err := exec.CombinedOutputLines(cmd)
	if err != nil {
		return "", err
	}

	// parse it, and populate it into _output/git_version
	version := ""
	for _, line := range output {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("could not parse kubernetes version: %q", strings.Join(output, "\n"))
		}
		if parts[0] == "gitVersion" {
			version = parts[1]
			return version, nil
		}
	}
	if version == "" {
		return "", fmt.Errorf("could not obtain kubernetes version: %q", strings.Join(output, "\n"))

	}
	return "", fmt.Errorf("could not find kubernetes version in output: %q", strings.Join(output, "\n"))
}
