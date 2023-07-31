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
	"strings"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/metadata"
	"sigs.k8s.io/kubetest2/pkg/process"
)

func (d *deployer) IsUp() (up bool, err error) {
	// naively assume that if the api server reports nodes, the cluster is up
	lines, err := exec.CombinedOutputLines(
		exec.Command("kubectl", "get", "nodes", "-o=name"),
	)
	if err != nil {
		return false, metadata.NewJUnitError(err, strings.Join(lines, "\n"))
	}
	return len(lines) > 0, nil
}

func (d *deployer) Up() error {
	args := []string{
		"create", "cluster",
		"--name", d.ClusterName,
	}

	// set the explicitly specified image name if set
	if d.NodeImage != "" {
		args = append(args, "--image", d.NodeImage)
	} else if d.commonOptions.ShouldBuild() {
		// otherwise if we just built an image, use that
		// NOTE: this is safe in the face of upstream changes, because
		// we use the same logic / constant for Build()
		args = append(args, "--image", kindDefaultBuiltImageName)
	}
	if d.ConfigPath != "" {
		args = append(args, "--config", d.ConfigPath)
	}
	if d.KubeconfigPath != "" {
		args = append(args, "--kubeconfig", d.KubeconfigPath)
	}

	klog.V(0).Infof("Up(): creating kind cluster...\n")
	// we want to see the output so use process.ExecJUnit
	return process.ExecJUnit("kind", args, os.Environ())
}
