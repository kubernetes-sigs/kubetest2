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
	"path/filepath"

	"k8s.io/klog/v2"
	"sigs.k8s.io/kubetest2/pkg/boskos"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *deployer) Down() error {
	klog.V(1).Info("GCE deployer starting Down()")

	if err := d.init(); err != nil {
		return fmt.Errorf("down failed to init: %s", err)
	}

	if err := d.DumpClusterLogs(); err != nil {
		klog.Warningf("Dumping cluster logs at the beginning of Down() failed: %s", err)
	}

	path, err := d.verifyKubectl()
	if err != nil {
		return err
	}
	d.kubectlPath = path

	env := d.buildEnv()
	script := filepath.Join(d.RepoRoot, "cluster", "kube-down.sh")
	klog.V(2).Infof("About to run script at: %s", script)

	cmd := exec.Command(script)
	cmd.SetEnv(env...)
	exec.InheritOutput(cmd)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error encountered during %s: %s", script, err)
	}

	klog.V(2).Info("about to delete nodeport firewall rule")
	// best-effort try to delete the explicitly created firewall rules
	// ideally these should already be deleted by kube-down
	d.deleteFirewallRuleNodePort()

	if d.boskos != nil {
		klog.V(2).Info("releasing boskos project")
		err := boskos.Release(
			d.boskos,
			[]string{d.GCPProject},
			d.boskosHeartbeatClose,
		)
		if err != nil {
			return fmt.Errorf("down failed to release boskos project: %s", err)
		}
	}

	return nil
}

func (d *deployer) verifyDownFlags() error {
	if err := d.setRepoPathIfNotSet(); err != nil {
		return err
	}

	if d.GCPProject == "" {
		return fmt.Errorf("gcp project must be set")
	}

	return nil
}
