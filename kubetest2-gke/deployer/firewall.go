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
	"sort"
	"strings"
	"time"

	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *Deployer) EnsureFirewallRules() error {
	// Do not modify the firewall rules for the default network
	if d.Network == "default" {
		return nil
	}

	if len(d.Projects) == 1 {
		return d.ensureFirewallRulesForSingleProject()
	}

	return d.ensureFirewallRulesForMultiProjects()
}

// Ensure firewall rules for e2e testing for all clusters in one single project.
func (d *Deployer) ensureFirewallRulesForSingleProject() error {
	project := d.Projects[0]
	for _, cluster := range d.projectClustersLayout[project] {
		clusterName := cluster.Name
		klog.V(1).Infof("Ensuring firewall rules for cluster %s in %s", clusterName, project)
		firewall := clusterFirewallName(project, clusterName)
		if runWithNoOutput(exec.Command("gcloud", "compute", "firewall-rules", "describe", firewall,
			"--project="+project,
			"--format=value(name)")) == nil {
			// Assume that if this unique firewall exists, it's good to go.
			continue
		}
		klog.V(1).Infof("Couldn't describe firewall '%s', assuming it doesn't exist and creating it", firewall)

		firewallRulesCreateCmd := []string{
			"gcloud", "compute", "firewall-rules", "create", firewall,
			"--project=" + project,
			"--network=" + d.Network,
			"--allow=" + d.FirewallRuleAllow,
		}
		if !d.Autopilot {
			tagOut, err := exec.Output(exec.Command("gcloud", "compute", "instances", "list",
				"--project="+project,
				"--filter=metadata.created-by:"+d.instanceGroups[project][clusterName][0].path,
				"--limit=1",
				"--format=get(tags.items)"))
			if err != nil {
				return fmt.Errorf("instances list failed: %s", execError(err))
			}
			tag := strings.TrimSpace(string(tagOut))
			if tag == "" {
				return fmt.Errorf("instances list returned no instances (or instance has no tags)")
			}
			firewallRulesCreateCmd = append(firewallRulesCreateCmd, "--target-tags="+tag)
		}

		if err := runWithOutput(exec.Command(firewallRulesCreateCmd[0], firewallRulesCreateCmd[1:]...)); err != nil {
			return fmt.Errorf("error creating firewall rule: %v", err)
		}
	}
	return nil
}

func clusterFirewallName(project, cluster string) string {
	projectNumber, _ := getProjectNumber(project)
	return fmt.Sprintf("e2e-ports-%s-%s", projectNumber, cluster)
}

// Ensure firewall rules for multi-project profile.
// Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_additional_firewall_rules
// Please note we are not including the firewall rule for SSH connection as it's not needed for testing.
func (d *Deployer) ensureFirewallRulesForMultiProjects() error {
	hostProject := d.Projects[0]
	for i := 1; i < len(d.Projects); i++ {
		curtProject := d.Projects[i]
		clusters := d.projectClustersLayout[curtProject]
		for _, cluster := range clusters {
			firewall := clusterFirewallName(curtProject, cluster.Name)
			// sourceRanges need to be separated with ",", while the provided subnetworkRanges are separated with space.
			sourceRanges := strings.ReplaceAll(d.SubnetworkRanges[i-1], " ", ",")
			if err := runWithOutput(exec.Command("gcloud", "compute", "firewall-rules", "create", firewall,
				"--project="+hostProject,
				"--network="+d.Network,
				"--allow="+d.FirewallRuleAllow,
				"--direction=INGRESS",
				"--source-ranges="+sourceRanges)); err != nil {
				return fmt.Errorf("error creating firewall rule for project %q cluster %q: %v", curtProject, cluster.Name, err)
			}
		}
	}
	return nil
}

// Ensure that all firewall-rules are deleted from specific network.
func (d *Deployer) CleanupNetworkFirewalls(hostProject, network string) error {
	// Do not delete firewall rules for the default network.
	if network == "default" {
		return nil
	}

	klog.V(1).Infof("Cleaning up network firewall rules for network %s in %s", network, hostProject)
	fws, err := exec.Output(exec.Command("gcloud", "compute", "firewall-rules", "list",
		"--format=value(name)",
		"--project="+hostProject,
		"--filter=network:"+network))
	if err != nil {
		return fmt.Errorf("firewall rules list failed: %s", execError(err))
	}
	if len(fws) > 0 {
		fwList := strings.Split(strings.TrimSpace(string(fws)), "\n")
		klog.V(1).Infof("Network %s has %v undeleted firewall rules %v", network, len(fwList), fwList)
		commandArgs := []string{"compute", "firewall-rules", "delete", "-q"}
		commandArgs = append(commandArgs, fwList...)
		commandArgs = append(commandArgs, "--project="+hostProject)
		errFirewall := runWithOutput(exec.Command("gcloud", commandArgs...))
		if errFirewall != nil {
			return fmt.Errorf("error deleting firewall: %v", errFirewall)
		}
		// It looks sometimes gcloud exits before the firewall rules are actually deleted,
		// so sleep 30 seconds to wait for the firewall rules being deleted completely.
		// TODO(chizhg): change to a more reliable way to check if they are deleted or not.
		time.Sleep(30 * time.Second)
	}
	return nil
}

func (d *Deployer) GetInstanceGroups() error {
	// If instanceGroups has already been populated, return directly.
	if d.instanceGroups != nil {
		return nil
	}

	// Initialize project instance groups structure
	d.instanceGroups = map[string]map[string][]*ig{}

	location := LocationFlag(d.Regions, d.Zones, d.retryCount)

	for _, project := range d.Projects {
		d.instanceGroups[project] = map[string][]*ig{}

		for _, cluster := range d.projectClustersLayout[project] {
			clusterName := cluster.Name

			igs, err := exec.Output(exec.Command("gcloud", containerArgs("clusters", "describe", clusterName,
				"--format=value(instanceGroupUrls)",
				"--project="+project,
				location)...))
			if err != nil {
				return fmt.Errorf("instance group URL fetch failed: %s", execError(err))
			}
			igURLs := strings.Split(strings.TrimSpace(string(igs)), ";")
			if len(igURLs) == 0 {
				return fmt.Errorf("no instance group URLs returned by gcloud, output %q", string(igs))
			}
			sort.Strings(igURLs)

			// Initialize cluster instance groups
			d.instanceGroups[project][clusterName] = make([]*ig, 0)

			for _, igURL := range igURLs {
				m := poolRe.FindStringSubmatch(igURL)
				if len(m) == 0 {
					return fmt.Errorf("instanceGroupUrl %q did not match regex %v", igURL, poolRe)
				}
				d.instanceGroups[project][clusterName] = append(d.instanceGroups[project][clusterName], &ig{path: m[0], zone: m[1], name: m[2], uniq: m[3]})
			}
		}
	}

	return nil
}
