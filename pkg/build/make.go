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

package build

import (
	"fmt"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

type MakeBuilder struct {
	RepoRoot string
}

var _ Builder = &MakeBuilder{}

const (
	target = "quick-release"
)

// Build builds kubernetes with the quick-release make target
func (m *MakeBuilder) Build() (string, error) {
	version, err := sourceVersion(m.RepoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get version: %v", err)
	}
	cmd := exec.Command("make", target)
	cmd.SetDir(m.RepoRoot)
	setSourceDateEpoch(m.RepoRoot, cmd)
	exec.InheritOutput(cmd)
	if err = cmd.Run(); err != nil {
		return "", err
	}
	return version, nil
}
