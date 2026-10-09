/*
Copyright The Kubernetes Authors.

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
	"path/filepath"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/build"
	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/fs"
)

// Build builds the test binaries and stores them in the run dir.
// Cluster components are built by local-up-cluster.sh during Up.
func (d *deployer) Build() error {
	if err := d.setRepoRootIfNotSet(); err != nil {
		return err
	}

	klog.V(0).Infof("Build(): building e2e requirements...\n")
	cmd := exec.Command("make", "all",
		"WHAT=cmd/kubectl test/e2e/e2e.test vendor/github.com/onsi/ginkgo/v2/ginkgo")
	cmd.SetDir(d.RepoRoot)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build e2e requirements: %w", err)
	}

	for _, binary := range build.CommonTestBinaries {
		source := filepath.Join(d.RepoRoot, "_output", "bin", binary)
		dest := filepath.Join(d.commonOptions.RunDir(), binary)
		if err := fs.CopyFile(source, dest); err != nil {
			klog.Warningf("failed to copy %s to %s: %v", source, dest, err)
		}
	}
	return nil
}
