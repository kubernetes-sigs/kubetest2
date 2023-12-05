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
	"os"
	"path"
	"strings"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/build"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *Deployer) Build() error {
	klog.V(1).Info("GCE deployer starting Build()")

	if err := d.init(); err != nil {
		return fmt.Errorf("build failed to init: %s", err)
	}

	if d.LegacyMode {
		// this supports the kubernetes/kubernetes build
		klog.V(2).Info("starting the legacy kubernetes/kubernetes build")
		version, err := d.BuildOptions.Build()
		if err != nil {
			return err
		}

		// append the kubetest2 run id
		// avoid double + in the version
		// so they are valid docker tags
		if strings.Contains(version, "+") {
			version += "-" + d.commonOptions.RunID()
		} else {
			version += "+" + d.commonOptions.RunID()
		}

		// stage build if requested
		if d.BuildOptions.CommonBuildOptions.StageLocation != "" {
			if err := d.BuildOptions.Stage(version); err != nil {
				return fmt.Errorf("error staging build: %v", err)
			}
		}
		build.StoreCommonBinaries(d.RepoRoot, d.commonOptions.RunDir())
	} else {
		// this code path supports the kubernetes/cloud-provider-gcp build
		klog.V(2).Info("starting the build")

		var cmd exec.Cmd
		// determine the build system for kubernetes/cloud-provider-gcp
		if _, err := os.Stat(path.Join(d.RepoRoot, "Makefile")); err == nil {
			// For releases that uses Makefile
			cmd = exec.Command("make", "release-tars")
		} else if _, err := os.Stat(path.Join(d.RepoRoot, "BUILD")); err == nil {
			// For releases that uses Bazel
			cmd = exec.Command("bazel", "build", "//release:release-tars")
		} else {
			return fmt.Errorf("cannot determine build system")
		}

		exec.InheritOutput(cmd)
		cmd.SetDir(d.RepoRoot)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error during make step of build: %s", err)
		}
	}

	// no untarring, uploading, etc is necessary because
	// kube-up/down use find-release-tars and upload-tars
	// which know how to find the tars, assuming KUBE_ROOT
	// is set

	return nil
}

func (d *Deployer) setRepoPathIfNotSet() error {
	if d.RepoRoot != "" {
		return nil
	}

	path, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory for setting Kubernetes root path: %s", err)
	}
	klog.V(1).Infof("defaulting repo root to the current directory: %s", path)
	d.RepoRoot = path

	return nil
}

// verifyBuildFlags only checks flags that are needed for Build
func (d *Deployer) verifyBuildFlags() error {
	if err := d.setRepoPathIfNotSet(); err != nil {
		return err
	}
	d.BuildOptions.CommonBuildOptions.RepoRoot = d.RepoRoot
	return d.BuildOptions.Validate()
}
