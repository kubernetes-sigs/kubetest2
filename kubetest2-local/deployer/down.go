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
	"time"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *deployer) Down() error {
	processes := []string{
		"cloud-controller-manager",
		"kube-controller-manager",
		"kube-proxy",
		"kube-scheduler",
		"kube-apiserver",
		"kubelet",
	}

	// Wait for pods stopped with terminationGracePeriod:30, otherwise their
	// containers remain and cannot be deleted before the timeout expires.
	klog.V(0).Info("Down(): waiting 30s for pods to terminate...")
	time.Sleep(30 * time.Second)

	if err := runInheritOutput(exec.Command("sh", "-c", `docker ps -aq | xargs -r docker rm -fv`)); err != nil {
		klog.Warningf("unable to cleanup containers in docker: %v", err)
	}

	for _, p := range processes {
		// -f matches against the full command line; process names longer
		// than 15 characters cannot otherwise be matched.
		if err := runInheritOutput(exec.Command("pkill", "-f", p)); err != nil {
			klog.Warningf("unable to kill kubernetes process %q: %v", p, err)
		}
	}
	if err := runInheritOutput(exec.Command("pkill", "etcd")); err != nil {
		klog.Warningf("unable to kill etcd: %v", err)
	}
	return nil
}

func runInheritOutput(cmd exec.Cmd) error {
	exec.InheritOutput(cmd)
	return cmd.Run()
}
