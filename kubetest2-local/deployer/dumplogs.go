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
	"os"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *deployer) DumpClusterLogs() error {
	klog.V(0).Infof("DumpClusterLogs(): copying %s to %s", d.clusterLogsDir, d.logsDir)
	if err := os.MkdirAll(d.logsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create logs directory %s: %w", d.logsDir, err)
	}
	// trailing "/." copies the directory contents rather than the directory itself
	return runInheritOutput(exec.Command("cp", "-r", d.clusterLogsDir+"/.", d.logsDir))
}
