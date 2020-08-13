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
	"sync"

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *deployer) Down() error {
	if err := d.init(); err != nil {
		return err
	}

	if err := d.prepareGcpIfNeeded(d.projects[0]); err != nil {
		return err
	}

	var wg sync.WaitGroup
	for i := range d.projects {
		project := d.projects[i]
		for j := range d.projectClustersLayout[project] {
			cluster := d.clusters[j]
			firewall, err := d.getClusterFirewall(project, cluster)
			if err != nil {
				// This is expected if the cluster doesn't exist.
				continue
			}
			d.instanceGroups = nil

			loc, err := d.location()
			if err != nil {
				return err
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				// We best-effort try all of these and report errors as appropriate.
				errCluster := runWithOutput(exec.Command(
					"gcloud", d.containerArgs("clusters", "delete", "-q", cluster,
						"--project="+project,
						loc)...))

				// don't delete default network
				if d.network == "default" {
					if errCluster != nil {
						klog.V(1).Infof("Error deleting cluster using default network, allow the error for now %s", errCluster)
					}
					return
				}

				var errFirewall error
				if runWithNoOutput(exec.Command("gcloud", "compute", "firewall-rules", "describe", firewall,
					"--project="+project,
					"--format=value(name)")) == nil {
					klog.V(1).Infof("Found rules for firewall '%s', deleting them", firewall)
					errFirewall = exec.Command("gcloud", "compute", "firewall-rules", "delete", "-q", firewall,
						"--project="+project).Run()
				} else {
					klog.V(1).Infof("Found no rules for firewall '%s', assuming resources are clean", firewall)
				}
				numLeakedFWRules, errCleanFirewalls := d.cleanupNetworkFirewalls(project, d.network)

				if errCluster != nil {
					klog.Errorf("error deleting cluster: %v", errCluster)
				}
				if errFirewall != nil {
					klog.Errorf("error deleting firewall: %v", errFirewall)
				}
				if errCleanFirewalls != nil {
					klog.Errorf("error cleaning-up firewalls: %v", errCleanFirewalls)
				}
				if numLeakedFWRules > 0 {
					klog.Errorf("leaked firewall rules")
				}
			}()
		}
	}
	wg.Wait()

	if err := d.teardownNetwork(); err != nil {
		return err
	}
	if err := d.deleteNetwork(); err != nil {
		return err
	}

	return nil
}
