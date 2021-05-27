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
		project := d.Projects[0]
		return ensureFirewallRulesForSingleProject(project, d.Network, d.projectClustersLayout[project], d.instanceGroups)
	}

	return ensureFirewallRulesForMultiProjects(d.Projects, d.Network, d.SubnetworkRanges)
}

// Ensure firewall rules for e2e testing for all clusters in one single project.
func ensureFirewallRulesForSingleProject(project, network string, clusters []cluster, instanceGroups map[string]map[string][]*ig) error {
	for _, cluster := range clusters {
		clusterName := cluster.name
		klog.V(1).Infof("Ensuring firewall rules for cluster %s in %s", clusterName, project)
		firewall := clusterFirewallName(project, clusterName, instanceGroups)
		if runWithNoOutput(exec.Command("gcloud", "compute", "firewall-rules", "describe", firewall,
			"--project="+project,
			"--format=value(name)")) == nil {
			// Assume that if this unique firewall exists, it's good to go.
			continue
		}
		klog.V(1).Infof("Couldn't describe firewall '%s', assuming it doesn't exist and creating it", firewall)

		tagOut, err := exec.Output(exec.Command("gcloud", "compute", "instances", "list",
			"--project="+project,
			"--filter=metadata.created-by:*"+instanceGroups[project][clusterName][0].path,
			"--limit=1",
			"--format=get(tags.items)"))
		if err != nil {
			return fmt.Errorf("instances list failed: %s", execError(err))
		}
		tag := strings.TrimSpace(string(tagOut))
		if tag == "" {
			return fmt.Errorf("instances list returned no instances (or instance has no tags)")
		}

		if err := runWithOutput(exec.Command("gcloud", "compute", "firewall-rules", "create", firewall,
			"--project="+project,
			"--network="+network,
			"--allow="+e2eAllow,
			"--target-tags="+tag)); err != nil {
			return fmt.Errorf("error creating e2e firewall rule: %v", err)
		}
	}
	return nil
}

func clusterFirewallName(project, cluster string, instanceGroups map[string]map[string][]*ig) string {
	// We want to ensure that there's an e2e-ports-* firewall rule
	// that maps to the cluster nodes, but the target tag for the
	// nodes can be slow to get. Use the hash from the lexically first
	// node pool instead.
	return "e2e-ports-" + instanceGroups[project][cluster][0].uniq
}

// Ensure firewall rules for multi-project profile.
// Reference: https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#creating_additional_firewall_rules
// Please note we are not including the firewall rule for SSH connection as it's not needed for testing.
func ensureFirewallRulesForMultiProjects(projects []string, network string, subnetworkRanges []string) error {
	hostProject := projects[0]
	hostProjectNumber, err := getProjectNumber(hostProject)
	if err != nil {
		return fmt.Errorf("error looking up project number for id %q: %w", hostProject, err)
	}
	for i := 1; i < len(projects); i++ {
		curtProject := projects[i]
		curtProjectNumber, err := getProjectNumber(curtProject)
		if err != nil {
			return fmt.Errorf("error looking up project number for id %q: %w", curtProject, err)
		}
		firewall := fmt.Sprintf("rule-%s-%s", hostProjectNumber, curtProjectNumber)
		// sourceRanges need to be separated with ",", while the provided subnetworkRanges are separated with space.
		sourceRanges := strings.ReplaceAll(subnetworkRanges[i-1], " ", ",")
		if err := runWithOutput(exec.Command("gcloud", "compute", "firewall-rules", "create", firewall,
			"--project="+hostProject,
			"--network="+network,
			"--allow=tcp,udp,icmp",
			"--direction=INGRESS",
			"--source-ranges="+sourceRanges)); err != nil {
			return fmt.Errorf("error creating firewall rule for project %q: %v", curtProject, err)
		}
	}
	return nil
}

// Ensure that all firewall-rules are deleted from specific network.
func (d *Deployer) CleanupNetworkFirewalls(hostProject, network string) (int, error) {
	// Do not delete firewall rules for the default network.
	if network == "default" {
		return 0, nil
	}

	klog.V(1).Infof("Cleaning up network firewall rules for network %s in %s", network, hostProject)
	fws, err := exec.Output(exec.Command("gcloud", "compute", "firewall-rules", "list",
		"--format=value(name)",
		"--project="+hostProject,
		"--filter=network:"+network))
	if err != nil {
		return 0, fmt.Errorf("firewall rules list failed: %s", execError(err))
	}
	if len(fws) > 0 {
		fwList := strings.Split(strings.TrimSpace(string(fws)), "\n")
		klog.V(1).Infof("Network %s has %v undeleted firewall rules %v", network, len(fwList), fwList)
		commandArgs := []string{"compute", "firewall-rules", "delete", "-q"}
		commandArgs = append(commandArgs, fwList...)
		commandArgs = append(commandArgs, "--project="+hostProject)
		errFirewall := runWithOutput(exec.Command("gcloud", commandArgs...))
		if errFirewall != nil {
			return 0, fmt.Errorf("error deleting firewall: %v", errFirewall)
		}
		// It looks sometimes gcloud exits before the firewall rules are actually deleted,
		// so sleep 30 seconds to wait for the firewall rules being deleted completely.
		// TODO(chizhg): change to a more reliable way to check if they are deleted or not.
		time.Sleep(30 * time.Second)
	}
	return len(fws), nil
}

func (d *Deployer) GetInstanceGroups() error {
	// If instanceGroups has already been populated, return directly.
	if d.instanceGroups != nil {
		return nil
	}

	// Initialize project instance groups structure
	d.instanceGroups = map[string]map[string][]*ig{}

	location := locationFlag(d.Regions, d.Zones, d.retryCount)

	for _, project := range d.Projects {
		d.instanceGroups[project] = map[string][]*ig{}

		for _, cluster := range d.projectClustersLayout[project] {
			clusterName := cluster.name

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
