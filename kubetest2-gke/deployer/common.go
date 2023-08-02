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
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/math"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/boskos"
)

const (
	defaultBoskosLocation                 = "http://boskos.test-pods.svc.cluster.local."
	defaultGKEProjectResourceType         = "gke-project"
	defaultBoskosAcquireTimeoutSeconds    = 300
	defaultBoskosHeartbeatIntervalSeconds = 300
)

func (d *Deployer) Init() error {
	var err error
	d.doInit.Do(func() { err = d.Initialize() })
	return err
}

// Initialize should only be called by init(), behind a sync.Once
func (d *Deployer) Initialize() error {
	if d.ClusterVersion == "" && d.LegacyClusterVersion != "" {
		klog.Warningf("--version is deprecated please use --cluster-version")
		d.ClusterVersion = d.LegacyClusterVersion
	}
	if d.Kubetest2CommonOptions.ShouldUp() {
		d.totalTryCount = math.Max(len(d.Regions), len(d.Zones))

		if err := d.VerifyUpFlags(); err != nil {
			return fmt.Errorf("init failed to verify flags for up: %w", err)
		}

		// Compile the retryable error patterns as regex objects.
		d.retryableErrorPatternsCompiled = make([]*regexp.Regexp, len(d.RetryableErrorPatterns))
		for i, regxString := range d.RetryableErrorPatterns {
			var err error
			d.retryableErrorPatternsCompiled[i], err = regexp.Compile(regxString)
			if err != nil {
				return fmt.Errorf("error compiling regex: %w", err)
			}
		}

		if len(d.Projects) == 0 {
			klog.V(1).Infof("No GCP projects provided, acquiring from Boskos %d project/s", d.BoskosProjectsRequested)

			boskosClient, err := boskos.NewClient(d.BoskosLocation)
			if err != nil {
				return fmt.Errorf("failed to make boskos client: %w", err)
			}
			d.boskos = boskosClient
			d.boskosHeartbeatClose = make(chan struct{})

			for i := 0; i < len(d.BoskosProjectsRequested); i++ {
				for j := 0; j < d.BoskosProjectsRequested[i]; j++ {
					resource, err := boskos.Acquire(
						d.boskos,
						d.BoskosResourceType[i],
						time.Duration(d.BoskosAcquireTimeoutSeconds)*time.Second,
						time.Duration(d.BoskosHeartbeatIntervalSeconds)*time.Second,
						d.boskosHeartbeatClose,
					)

					if err != nil {
						return fmt.Errorf("init failed to get project from boskos: %w", err)
					}
					d.Projects = append(d.Projects, resource.Name)
					klog.V(1).Infof("Got project %s from boskos", resource.Name)
				}
			}
		}
	}

	if d.Kubetest2CommonOptions.ShouldDown() {
		if err := d.VerifyDownFlags(); err != nil {
			return fmt.Errorf("init failed to verify flags for down: %w", err)
		}
	}

	// Multi-cluster name adjustment
	numProjects := len(d.Projects)
	d.projectClustersLayout = make(map[string][]cluster, numProjects)
	if numProjects > 1 {
		if err := buildProjectClustersLayout(d.Projects, d.Clusters, d.projectClustersLayout); err != nil {
			return fmt.Errorf("failed to build the project clusters layout: %v", err)
		}
	} else {
		// Backwards compatible construction
		clusters := make([]cluster, len(d.Clusters))
		for i, clusterName := range d.Clusters {
			clusters[i] = cluster{i, clusterName}
		}
		d.projectClustersLayout[d.Projects[0]] = clusters
	}

	// build extra node pool specs.
	for i, np := range d.ExtraNodePool {
		// defaults
		enp := &extraNodepool{
			Index:       i,
			Name:        fmt.Sprintf("extra-node-pool-%d", i),
			MachineType: d.MachineType,
			ImageType:   d.ImageType,
			NumNodes:    1, // default nodepool size.
		}

		if err := buildExtraNodePoolOptions(np, enp); err != nil {
			return fmt.Errorf("invalid extra nodepool spec %q", np)
		}

		d.extraNodePoolSpecs = append(d.extraNodePoolSpecs, enp)

		klog.V(1).Infof("parsed extra nodepool spec %q: %v", np, enp)
	}

	// Prepare the GCP environment for the following operations.
	return d.PrepareGcpIfNeeded(d.Projects[0])
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

func buildExtraNodePoolOptions(np string, enp *extraNodepool) error {
	values, err := url.ParseQuery(np)
	if err != nil {
		return err
	}
	for k := range values {

		switch k {
		case "name":
			enp.Name = values.Get("name")
		case "machine-type":
			enp.MachineType = values.Get("machine-type")
		case "image-type":
			enp.ImageType = values.Get("image-type")
		case "num-nodes":
			n, err := strconv.Atoi(values.Get("num-nodes"))
			if err != nil {
				return err
			}
			if n < 0 {
				return fmt.Errorf("num-nodes must be a positive integer, got %d", n)
			}
			enp.NumNodes = n
		default:
			return fmt.Errorf("unknown parameter: %q", k)
		}
	}
	return validateExtraNodepoolOptions(enp)
}

func validateExtraNodepoolOptions(enp *extraNodepool) error {
	if enp.Name == "" {
		return fmt.Errorf("name required")
	}

	if enp.MachineType == "" {
		return fmt.Errorf("machine-type required")
	}

	if enp.ImageType == "" {
		return fmt.Errorf("image-type required")
	}

	if enp.NumNodes <= 0 {
		return fmt.Errorf("num-nodes must be > 0")
	}
	return nil
}
