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

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/boskos"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *Deployer) Down() error {
	if err := d.Init(); err != nil {
		return err
	}
	// Nothing to clean if there is no GCP project.
	// This edge case happens e.g. when Up fails to acquire the Boskos project.
	if len(d.Projects) == 0 {
		return nil
	}

	if err := d.DumpClusterLogs(); err != nil {
		klog.Warningf("Dumping cluster logs at the end of Up() failed: %v", err)
	}

	// If the GCP projects are acquired from Boskos, release the projects and
	// rely on boskos-janitor to do clean-ups for them.
	if d.totalBoskosProjectsRequested > 0 {
		return boskos.Release(d.boskos, d.Projects, d.boskosHeartbeatClose)
	}

	d.DeleteClusters(d.retryCount)

	numDeletedFWRules, errCleanFirewalls := d.CleanupNetworkFirewalls(d.Projects[0], d.Network)
	if errCleanFirewalls != nil {
		klog.Errorf("Error cleaning-up firewall rules: %v", errCleanFirewalls)
	} else {
		klog.V(1).Infof("Deleted %d network firewall rules", numDeletedFWRules)
	}

	errNat := d.CleanupNat()
	errNetwork := d.TeardownNetwork()
	errSubnets := d.DeleteSubnets(d.retryCount)

	if errNat != nil {
		return errNat
	}
	if errNetwork != nil {
		return errNetwork
	}
	if errSubnets != nil {
		return errSubnets
	}
	return d.DeleteNetwork()
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
	args := []string{"clusters", "delete", "-q", cluster.name,
		"--project=" + project, loc}
	if d.DownTimeout > 0 {
		args = append(args, fmt.Sprintf("--timeout=%d", int(d.DownTimeout.Seconds())))
	}
	if err := runWithOutput(exec.Command(
		"gcloud", containerArgs(args...)...)); err != nil {
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
	return d.VerifyLocationFlags()
}
