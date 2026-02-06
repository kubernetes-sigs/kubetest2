/*
Copyright 2026 The Kubernetes Authors.

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
	"log"
	"os"
	"strings"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

// setupBastion prepares KUBE_SSH_BASTION env variable with the hostname of some public node of the cluster that could be sshed into. Some Kubernetes e2e tests need it.
func (d *Deployer) setupBastion() error {
	if d.SSHProxyInstanceName == "" {
		return nil
	}

	hostProject := d.Projects[0]
	clusterMap := d.instanceGroups[hostProject]

	if clusterMap == nil {
		return fmt.Errorf("instanceGroups map has nil cluster map for project %s", hostProject)
	}

	for _, instanceGroups := range clusterMap {
		if trySetupBastion(hostProject, instanceGroups, d.SSHProxyInstanceName) {
			return nil
		}
	}

	return fmt.Errorf("Failed to setup bastion. Refer to logs above for more information.")
}

func trySetupBastion(project string, instanceGroups []*ig, SSHProxyInstanceName string) bool {
	var filtersToTry []string
	// Use exact name first, VM does not have to belong to the cluster
	exactFilter := "name=" + SSHProxyInstanceName
	filtersToTry = append(filtersToTry, exactFilter)
	// As a fallback - use proxy instance name as a regex but check only cluster nodes
	var igFilters []string
	// Filter out VMs not belonging to the GKE cluster
	for _, ig := range instanceGroups {
		igFilters = append(igFilters, fmt.Sprintf("(metadata.created-by ~ %s)", ig.path))
	}
	// Match VM name or wildcard passed by kubetest parameters
	fuzzyFilter := fmt.Sprintf("(name ~ %s) AND (%s)",
		SSHProxyInstanceName,
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
			log.Printf("listing instances failed: %s", execError(err))
			return false
		}
		if len(output) == 0 {
			continue
		}
		instances := strings.Split(string(output), "\n")
		// Proxy instance found
		fields := strings.Split(strings.TrimSpace(string(instances[0])), "\t")
		if len(fields) != 2 {
			log.Printf("error parsing instances list output %q", output)
			return false
		}
		bastion, zone = fields[0], fields[1]
		break
	}
	if bastion == "" {
		log.Printf("proxy instance %q not found", SSHProxyInstanceName)
		return false
	}
	log.Printf("Found proxy instance %q", bastion)

	log.Printf("Adding NAT access config if not present")
	if err := runWithNoOutput(exec.Command("gcloud", "compute", "instances", "add-access-config", bastion,
		"--zone="+zone,
		"--project="+project)); err != nil {
		log.Printf("error adding NAT access config: %s", execError(err))
	}

	// Set KUBE_SSH_BASTION env parameter
	err := setKubeShhBastionEnv(project, zone, bastion)
	if err != nil {
		log.Printf("setting KUBE_SSH_BASTION variable failed: %s", execError(err))
		return false
	}
	return true
}

func setKubeShhBastionEnv(project, zone, SSHProxyInstanceName string) error {
	value, err := exec.Output(exec.Command(
		"gcloud", "compute", "instances", "describe",
		SSHProxyInstanceName,
		"--project="+project,
		"--zone="+zone,
		"--format=get(networkInterfaces[0].accessConfigs[0].natIP)"))
	if err != nil {
		return fmt.Errorf("failed to get the external IP address of the '%s' instance: %w",
			SSHProxyInstanceName, err)
	}
	address := strings.TrimSpace(string(value))
	if address == "" {
		return fmt.Errorf("instance '%s' doesn't have an external IP address", SSHProxyInstanceName)
	}
	address += ":22"
	if err := os.Setenv("KUBE_SSH_BASTION", address); err != nil {
		return err
	}
	log.Printf("KUBE_SSH_BASTION set to: %v\n", address)
	return nil
}
