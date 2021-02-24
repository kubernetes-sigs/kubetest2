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
	"os"

	"sigs.k8s.io/kubetest2/pkg/process"
)

func (d *deployer) Build() error {
	// TODO(bentheelder): build type should be configurable
	args := []string{
		"build", "node-image",
	}
	if d.logLevel != "" {
		args = append(args, "--loglevel", d.logLevel)
	}
	if d.buildType != "" {
		args = append(args, "--type", d.buildType)
	}
	if d.kubeRoot != "" {
		args = append(args, "--kube-root", d.kubeRoot)
	}
	// set the explicitly specified image name if set
	if d.nodeImage != "" {
		args = append(args, "--image", d.nodeImage)
	} else if d.commonOptions.ShouldBuild() {
		// otherwise if we just built an image, use that
		args = append(args, "--image", kindDefaultBuiltImageName)
	}

	println("Build(): building kind node image...\n")
	// we want to see the output so use process.ExecJUnit
	return process.ExecJUnit("kind", args, os.Environ())
}
