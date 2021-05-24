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

func (d *Deployer) Down() error {
	if err := d.Init(); err != nil {
		return err
	}

	if len(d.Projects) > 0 {
		if err := d.PrepareGcpIfNeeded(d.Projects[0]); err != nil {
			return err
		}

		d.DeleteClusters(d.retryCount)

		numDeletedFWRules, errCleanFirewalls := d.CleanupNetworkFirewalls(d.Projects[0], d.Network)
		if errCleanFirewalls != nil {
			klog.Errorf("Error cleaning-up firewall rules: %v", errCleanFirewalls)
		} else {
			klog.V(1).Infof("Deleted %d network firewall rules", numDeletedFWRules)
		}

		if err := d.TeardownNetwork(); err != nil {
			return err
		}
		if err := d.DeleteSubnets(d.retryCount); err != nil {
			return err
		}
		if err := d.DeleteNetwork(); err != nil {
			return err
		}
	}

	return nil
}

func (d *Deployer) DeleteClusters(retryCount int) {
	// We best-effort try all of these and report errors as appropriate.
	var wg sync.WaitGroup
	for i := range d.Projects {
		project := d.Projects[i]
		for j := range d.projectClustersLayout[project] {
			cluster := d.projectClustersLayout[project][j]
			loc := locationFlag(d.Regions, d.Zones, retryCount)

			wg.Add(1)
			go func() {
				defer wg.Done()
				d.DeleteCluster(project, loc, cluster)
			}()
		}
	}
	wg.Wait()
}

func (d *Deployer) DeleteCluster(project, loc string, cluster cluster) {
	if err := runWithOutput(exec.Command(
		"gcloud", containerArgs("clusters", "delete", "-q", cluster.name,
			"--project="+project,
			loc)...)); err != nil {
		klog.Errorf("Error deleting cluster: %v", err)
	}
}

// VerifyDownFlags validates flags for down phase.
func (d *Deployer) VerifyDownFlags() error {
	if len(d.Clusters) == 0 {
		return fmt.Errorf("--cluster-name must be set for GKE deployment")
	}
	if len(d.Projects) == 0 {
		return fmt.Errorf("--project must be set for GKE deployment")
	}
	if err := d.VerifyLocationFlags(); err != nil {
		return err
	}
	return nil
}
