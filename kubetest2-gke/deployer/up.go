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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/metadata"
)

// Deployer implementation methods below
func (d *deployer) Up() error {
	if err := d.init(); err != nil {
		return err
	}

	// Only run prepare once for the first GCP project.
	if err := d.prepareGcpIfNeeded(d.projects[0]); err != nil {
		return err
	}
	if err := d.createNetwork(); err != nil {
		return err
	}
	if err := d.setupNetwork(); err != nil {
		return err
	}

	klog.V(2).Infof("Environment: %v", os.Environ())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eg, ctx := errgroup.WithContext(ctx)
	loc := location(d.region, d.zone)
	for i := range d.projects {
		project := d.projects[i]
		subNetworkArgs := subNetworkArgs(d.projects, d.region, d.network, i)
		for j := range d.projectClustersLayout[project] {
			cluster := d.projectClustersLayout[project][j]
			privateClusterArgs := privateClusterArgs(d.network, cluster, d.privateClusterAccessLevel, d.privateClusterMasterIPRange)
			eg.Go(func() error {
				// Create the cluster
				args := make([]string, len(d.createCommand()))
				copy(args, d.createCommand())
				args = append(args,
					"--project="+project,
					loc,
					"--machine-type="+d.machineType,
					"--image-type="+image,
					"--num-nodes="+strconv.Itoa(d.nodes),
					"--network="+transformNetworkName(d.projects, d.network),
					"--cluster-version="+d.Version,
				)
				if d.workloadIdentityEnabled {
					args = append(args, fmt.Sprintf("--workload-pool=%s.svc.id.goog", project))
				}
				args = append(args, subNetworkArgs...)
				args = append(args, privateClusterArgs...)
				args = append(args, cluster)
				klog.V(1).Infof("Gcloud command: gcloud %+v\n", args)
				if err := runWithOutput(exec.CommandContext(ctx, "gcloud", args...)); err != nil {
					// Cancel the context to kill other cluster creation processes if any error happens.
					cancel()
					return fmt.Errorf("error creating cluster: %v", err)
				}
				return nil
			})
		}
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("error creating clusters: %v", err)
	}

	if err := d.testSetup(); err != nil {
		return fmt.Errorf("error running setup for the tests: %v", err)
	}

	return nil
}

func (d *deployer) createCommand() []string {
	return strings.Fields(d.createCommandFlag)
}

func (d *deployer) IsUp() (up bool, err error) {
	if err := d.prepareGcpIfNeeded(d.projects[0]); err != nil {
		return false, err
	}

	for _, project := range d.projects {
		for _, cluster := range d.projectClustersLayout[project] {
			if err := getClusterCredentials(project, location(d.region, d.zone), cluster); err != nil {
				return false, err
			}

			// naively assume that if the api server reports nodes, the cluster is up
			lines, err := exec.CombinedOutputLines(
				exec.RawCommand("kubectl get nodes -o=name"),
			)
			if err != nil {
				return false, metadata.NewJUnitError(err, strings.Join(lines, "\n"))
			}
			if len(lines) == 0 {
				return false, fmt.Errorf("project had no nodes active: %s", project)
			}
		}
	}

	return true, nil
}

func (d *deployer) testSetup() error {
	if d.testPrepared {
		// Ensure setup is a singleton.
		return nil
	}

	// Only run prepare once for the first GCP project.
	if err := d.prepareGcpIfNeeded(d.projects[0]); err != nil {
		return err
	}
	if _, err := d.Kubeconfig(); err != nil {
		return err
	}
	if err := d.getInstanceGroups(); err != nil {
		return err
	}
	if err := d.ensureFirewallRules(); err != nil {
		return err
	}
	d.testPrepared = true
	return nil
}

// Kubeconfig returns a path to a kubeconfig file for the cluster in
// a temp directory, creating one if one does not exist.
// It also sets the KUBECONFIG environment variable appropriately.
func (d *deployer) Kubeconfig() (string, error) {
	if d.kubecfgPath != "" {
		return d.kubecfgPath, nil
	}

	tmpdir, err := ioutil.TempDir("", "kubetest2-gke")
	if err != nil {
		return "", err
	}

	kubecfgFiles := make([]string, 0)
	for _, project := range d.projects {
		for _, cluster := range d.projectClustersLayout[project] {
			filename := filepath.Join(tmpdir, fmt.Sprintf("kubecfg-%s-%s", project, cluster))
			if err := os.Setenv("KUBECONFIG", filename); err != nil {
				return "", err
			}
			if err := getClusterCredentials(project, location(d.region, d.zone), cluster); err != nil {
				return "", err
			}
			kubecfgFiles = append(kubecfgFiles, filename)
		}
	}

	d.kubecfgPath = strings.Join(kubecfgFiles, string(os.PathListSeparator))
	return d.kubecfgPath, nil
}
