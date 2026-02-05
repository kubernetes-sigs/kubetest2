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
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

// setupBastion prepares KUBE_SSH_BASTION env variable with the hostname of some public node of the cluster that could be sshed into. Some Kubernetes e2e tests need it.
// setupBastion supports only one project with only one cluster. It returns error if this condition is not met.
func (d *Deployer) setupBastion() error {
	if d.SshProxyInstanceName == "" {
		return nil
	}

	if len(d.instanceGroups) == 0 {
		return errors.New("instanceGroups map is empty, expected exactly one project")
	}
	if len(d.instanceGroups) > 1 {
		return fmt.Errorf("instanceGroups map contains multiple projects (%d), expected exactly one", len(d.instanceGroups))
	}

	var project string
	var clusterMap map[string][]*ig

	// This loop runs only once due to the length check above
	for projectName, cm := range d.instanceGroups {
		project = projectName
		clusterMap = cm
	}

	if clusterMap == nil {
		return fmt.Errorf("instanceGroups map has nil cluster map for project %s", project)
	}
	if len(clusterMap) == 0 {
		return errors.New("cluster map is empty, expected exactly one cluster")
	}
	if len(clusterMap) > 1 {
		return fmt.Errorf("cluster map contains multiple clusters (%d), expected exactly one", len(clusterMap))
	}

	var instanceGroups []*ig
	// This loop runs only once
	for _, slice := range clusterMap {
		instanceGroups = slice
	}

	if err := setupBastion(project, instanceGroups, d.SshProxyInstanceName); err != nil {
		return err
	}

	return nil
}

func setupBastion(project string, instanceGroups []*ig, sshProxyInstanceName string) error {
	var filtersToTry []string
	// Use exact name first, VM does not have to belong to the cluster
	exactFilter := "name=" + sshProxyInstanceName
	filtersToTry = append(filtersToTry, exactFilter)
	// As a fallback - use proxy instance name as a regex but check only cluster nodes
	var igFilters []string
	// Filter out VMs not belonging to the GKE cluster
	for _, ig := range instanceGroups {
		igFilters = append(igFilters, fmt.Sprintf("(metadata.created-by ~ %s)", ig.path))
	}
	// Match VM name or wildcard passed by kubetest parameters
	fuzzyFilter := fmt.Sprintf("(name ~ %s) AND (%s)",
		sshProxyInstanceName,
		strings.Join(igFilters, " OR "))
	filtersToTry = append(filtersToTry, fuzzyFilter)

	// Find hostname of VM that matches criteria
	var bastion, zone string
	for _, filter := range filtersToTry {
		log.Printf("Checking for proxy instance with filter: %q", filter)
		output, err := exec.Output(exec.Command("gcloud", "compute", "instances", "list",
			"--filter="+filter,
			"--format=value(name,zone)",
			"--project="+project))
		if err != nil {
			return fmt.Errorf("listing instances failed: %s", execError(err))
		}
		if len(output) == 0 {
			continue
		}
		instances := strings.Split(string(output), "\n")
		// Proxy instance found
		fields := strings.Split(strings.TrimSpace(string(instances[0])), "\t")
		if len(fields) != 2 {
			return fmt.Errorf("error parsing instances list output %q", output)
		}
		bastion, zone = fields[0], fields[1]
		break
	}
	if bastion == "" {
		return fmt.Errorf("proxy instance %q not found", sshProxyInstanceName)
	}
	log.Printf("Found proxy instance %q", bastion)

	log.Printf("Adding NAT access config if not present")
	runWithNoOutput(exec.Command("gcloud", "compute", "instances", "add-access-config", bastion,
		"--zone="+zone,
		"--project="+project))

	// Set KUBE_SSH_BASTION env parameter
	err := setKubeShhBastionEnv(project, zone, bastion)
	if err != nil {
		return fmt.Errorf("setting KUBE_SSH_BASTION variable failed: %s", execError(err))
	}
	return nil
}

func setKubeShhBastionEnv(project, zone, sshProxyInstanceName string) error {
	value, err := exec.Output(exec.Command(
		"gcloud", "compute", "instances", "describe",
		sshProxyInstanceName,
		"--project="+project,
		"--zone="+zone,
		"--format=get(networkInterfaces[0].accessConfigs[0].natIP)"))
	if err != nil {
		return fmt.Errorf("failed to get the external IP address of the '%s' instance: %w",
			sshProxyInstanceName, err)
	}
	address := strings.TrimSpace(string(value))
	if address == "" {
		return fmt.Errorf("instance '%s' doesn't have an external IP address", sshProxyInstanceName)
	}
	address += ":22"
	if err := os.Setenv("KUBE_SSH_BASTION", address); err != nil {
		return err
	}
	log.Printf("KUBE_SSH_BASTION set to: %v\n", address)
	return nil
}
