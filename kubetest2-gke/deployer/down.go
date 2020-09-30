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
	"sync"

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *deployer) Down() error {
	if err := d.init(); err != nil {
		return err
	}
	if len(d.projects) > 0 {
		if err := d.prepareGcpIfNeeded(d.projects[0]); err != nil {
			return err
		}

		var wg sync.WaitGroup
		for i := range d.projects {
			project := d.projects[i]
			for j := range d.projectClustersLayout[project] {
				cluster := d.projectClustersLayout[project][j]
				loc := location(d.region, d.zone)

				wg.Add(1)
				go func() {
					defer wg.Done()
					// We best-effort try all of these and report errors as appropriate.
					if err := runWithOutput(exec.Command(
						"gcloud", containerArgs("clusters", "delete", "-q", cluster,
							"--project="+project,
							loc)...)); err != nil {
						klog.Errorf("Error deleting cluster: %v", err)
					}
				}()
			}
		}
		wg.Wait()

		numDeletedFWRules, errCleanFirewalls := d.cleanupNetworkFirewalls(d.projects[0], d.network)
		if errCleanFirewalls != nil {
			klog.Errorf("Error cleaning-up firewall rules: %v", errCleanFirewalls)
		} else {
			klog.V(1).Infof("Deleted %d network firewall rules", numDeletedFWRules)
		}

		if err := d.teardownNetwork(); err != nil {
			return err
		}
		if err := d.deleteNetwork(); err != nil {
			return err
		}
	}
	return nil
}

// verifyDownFlags validates flags for down phase.
func (d *deployer) verifyDownFlags() error {
	if len(d.clusters) == 0 {
		return fmt.Errorf("--cluster-name must be set for GKE deployment")
	}
	if len(d.projects) == 0 {
		return fmt.Errorf("--project must be set for GKE deployment")
	}
	if err := d.verifyLocationFlags(); err != nil {
		return err
	}
	return nil
}
