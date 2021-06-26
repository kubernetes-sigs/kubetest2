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

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/process"
)

func (d *deployer) Down() error {
	args := []string{
		"delete", "cluster",
		"--name", d.ClusterName,
	}

	klog.V(0).Infof("Down(): deleting kind cluster...%s\n", d.ClusterName)
	// we want to see the output so use process.ExecJUnit
	return process.ExecJUnit("kind", args, os.Environ())
}
