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

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

type Bazel struct {
	RepoRoot      string
	StageLocation string
	ImageLocation string
}

var _ Builder = &Bazel{}
var _ Stager = &Bazel{}

func (b *Bazel) Stage(version string) error {
	location := b.StageLocation + "/v" + version
	klog.V(0).Infof("Staging builds to %s ...", location)
	cmd := exec.Command("bazel", "run", "//:push-build", "--", location)
	cmd.SetDir(b.RepoRoot)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func (b *Bazel) Build() (string, error) {
	klog.V(0).Infof("Building kubernetes from %s ...", b.RepoRoot)
	version, err := sourceVersion(b.RepoRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get version: %v", err)
	}
	cmd := exec.Command("bazel", "build", "//build/release-tars")
	cmd = cmd.SetDir(b.RepoRoot)
	setSourceDateEpoch(b.RepoRoot, cmd)
	exec.InheritOutput(cmd)
	return version, cmd.Run()
}
