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
	realexec "os/exec"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/boskos"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

const (
	defaultBoskosLocation         = "http://boskos.test-pods.svc.cluster.local."
	defaultGKEProjectResourceType = "gke-project"
)

func (d *deployer) init() error {
	var err error
	d.doInit.Do(func() { err = d.initialize() })
	return err
}

// initialize should only be called by init(), behind a sync.Once
func (d *deployer) initialize() error {
	if d.commonOptions.ShouldUp() {
		if err := d.verifyUpFlags(); err != nil {
			return fmt.Errorf("init failed to verify flags for up: %w", err)
		}

		if len(d.projects) == 0 {
			klog.V(1).Infof("No GCP projects provided, acquiring from Boskos %d project/s", d.boskosProjectsRequested)

			boskosClient, err := boskos.NewClient(d.boskosLocation)
			if err != nil {
				return fmt.Errorf("failed to make boskos client: %w", err)
			}
			d.boskos = boskosClient

			for i := 0; i < d.boskosProjectsRequested; i++ {
				resource, err := boskos.Acquire(
					d.boskos,
					d.boskosResourceType,
					time.Duration(d.boskosAcquireTimeoutSeconds)*time.Second,
					d.boskosHeartbeatClose,
				)

				if err != nil {
					return fmt.Errorf("init failed to get project from boskos: %w", err)
				}
				d.projects = append(d.projects, resource.Name)
				klog.V(1).Infof("Got project %s from boskos", resource.Name)
			}
		}

		// Multi-cluster name adjustment
		numProjects := len(d.projects)
		d.projectClustersLayout = make(map[string][]cluster, numProjects)
		if numProjects > 1 {
			if err := buildProjectClustersLayout(d.projects, d.clusters, d.projectClustersLayout); err != nil {
				return fmt.Errorf("failed to build the project clusters layout: %v", err)
			}
		} else {
			// Backwards compatible construction
			clusters := make([]cluster, len(d.clusters))
			for i, clusterName := range d.clusters {
				clusters[i] = cluster{i, clusterName}
			}
			d.projectClustersLayout[d.projects[0]] = clusters
		}
	}

	if d.commonOptions.ShouldDown() {
		if err := d.verifyDownFlags(); err != nil {
			return fmt.Errorf("init failed to verify flags for down: %w", err)
		}
	}

	return nil
}

// buildProjectClustersLayout builds the projects and real cluster names mapping based on the provided --cluster-name flag.
func buildProjectClustersLayout(projects, clusters []string, projectClustersLayout map[string][]cluster) error {
	for i, clusterName := range clusters {
		parts := strings.Split(clusterName, ":")
		if len(parts) != 2 {
			return fmt.Errorf("cluster name does not follow expected format (name:projectIndex): %s", clusterName)
		}
		projectIndex, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("cluster name does not follow contain a valid project index (name:projectIndex. E.g: cluster:0): %v", err)
		}
		if projectIndex >= len(projects) {
			return fmt.Errorf("project index %d specified in the cluster name should be smaller than the number of projects %d", projectIndex, len(projects))
		}
		projectClustersLayout[projects[projectIndex]] = append(projectClustersLayout[projects[projectIndex]], cluster{i, parts[0]})
	}
	return nil
}

func containerArgs(args ...string) []string {
	return append(append([]string{}, "container"), args...)
}

func runWithNoOutput(cmd exec.Cmd) error {
	exec.NoOutput(cmd)
	return cmd.Run()
}

func runWithOutput(cmd exec.Cmd) error {
	exec.InheritOutput(cmd)
	return cmd.Run()
}

// execError returns a string format of err including stderr if the
// err is an ExitError, useful for errors from e.g. exec.Cmd.Output().
func execError(err error) string {
	if ee, ok := err.(*realexec.ExitError); ok {
		return fmt.Sprintf("%v (output: %q)", err, string(ee.Stderr))
	}
	return err.Error()
}
