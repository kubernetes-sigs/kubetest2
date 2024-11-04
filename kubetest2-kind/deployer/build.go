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
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/build"
	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/fs"
	"sigs.k8s.io/kubetest2/pkg/process"
)

const (
	target = "all"
)

func (d *deployer) Build() error {
	args := []string{
		"build", "node-image",
	}
	if d.KubeRoot != "" {
		args = append(args, d.KubeRoot)
	}
	if d.BuildType != "" {
		args = append(args, "--type", d.BuildType)
	}
	if d.BaseImage != "" {
		args = append(args, "--base-image", d.BaseImage)
	}
	// set the explicitly specified image name if set
	if d.NodeImage != "" {
		args = append(args, "--image", d.NodeImage)
	} else if d.commonOptions.ShouldBuild() {
		// otherwise if we just built an image, use that
		args = append(args, "--image", kindDefaultBuiltImageName)
	}

	klog.V(0).Infof("Build(): building kind node image...\n")
	// we want to see the output so use process.ExecJUnit
	if err := process.ExecJUnit("kind", args, os.Environ()); err != nil {
		return err
	}
	klog.V(0).Infof("Build(): build e2e requirements...\n")
	e2ePath := "test/e2e/e2e.test"
	kubectlPath := "cmd/kubectl"
	ginkgoPath := "vendor/github.com/onsi/ginkgo/v2/ginkgo"

	// Ginkgo v1 is used by Kubernetes 1.24 and earlier, fallback if v2 is not available.
	_, err := os.Stat(ginkgoPath)
	if err != nil {
		ginkgoPath = "vendor/github.com/onsi/ginkgo/ginkgo"
	}

	// make sure we have e2e requirements
	cmd := exec.Command("make", target,
		fmt.Sprintf("WHAT=%s %s %s", kubectlPath, e2ePath, ginkgoPath))
	cmd.SetDir(d.KubeRoot)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return err
	}

	//move files
	const dockerizedOutput = "_output/dockerized"
	for _, binary := range build.CommonTestBinaries {
		source := filepath.Join(d.KubeRoot, "_output/bin", binary)
		dest := filepath.Join(d.KubeRoot, dockerizedOutput, "bin", runtime.GOOS, runtime.GOARCH, binary)
		if err := fs.CopyFile(source, dest); err != nil {
			klog.Warningf("failed to copy %s to %s: %v", source, dest, err)
		}
	}

	build.StoreCommonBinaries(d.KubeRoot, d.commonOptions.RunDir())
	return nil
}
