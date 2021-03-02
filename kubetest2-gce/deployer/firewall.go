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

	"k8s.io/klog"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

// kube-up.sh builds NODE_TAG based on KUBE_GCE_INSTANCE_PREFIX which the deployer
// sets as d.instacePrefix. This function replicates NODE_TAG string construction
// because it is needed for firewall rules
func (d *deployer) nodeTag() string {
	return fmt.Sprintf("%s-minion", d.instancePrefix)
}

func (d *deployer) nodePortRuleName() string {
	return fmt.Sprintf("%s-nodeports", d.nodeTag())
}

func (d *deployer) createFirewallRuleNodePort() error {
	cmd := exec.Command(
		"gcloud", "compute", "firewall-rules", "create",
		"--project", d.GCPProject,
		"--target-tags", d.nodeTag(),
		"--allow", "tcp:30000-32767,udp:30000-32767",
		"--network", d.network,
		d.nodePortRuleName(),
	)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create nodeports firewall rule: %s", err)
	}

	return nil
}

func (d *deployer) deleteFirewallRuleNodePort() {
	cmd := exec.Command(
		"gcloud", "compute", "firewall-rules", "delete",
		"--project", d.GCPProject,
		d.nodePortRuleName(),
	)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		klog.Warning("failed to delete nodeports firewall rules: might be deleted already?")
	}
}
