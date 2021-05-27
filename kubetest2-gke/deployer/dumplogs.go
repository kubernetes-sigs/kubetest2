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
	"os"
	"strings"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

// DumpClusterLogs for GKE generates a small script that wraps
// log-dump.sh with the appropriate shell-fu to get the cluster
// dumped.
//
// TODO(RonWeber): This whole path is really gross, but this seemed
// the least gross hack to get this done.
//
// TODO(RonWeber): Make this work with multizonal and regional clusters.
func (d *Deployer) DumpClusterLogs() error {
	if len(d.Zones) <= 0 {
		return fmt.Errorf("DumpClusterLogs is currently only supported for zonal clusters")
	}
	// gkeLogDumpTemplate is a template of a shell script where
	// - %[1]s is the project
	// - %[2]s is the zone
	// - %[3]s is the KUBE_NODE_OS_DISTRIBUTION
	// - %[4]s is a filter composed of the instance groups
	// - %[5]s is the log-dump.sh command line
	const gkeLogDumpTemplate = `
function log_dump_custom_get_instances() {
  if [[ $1 == "master" ]]; then
    return 0
  fi

  gcloud compute instances list '--project=%[1]s' '--filter=%[4]s' '--format=get(name)'
}
export -f log_dump_custom_get_instances
# Set below vars that log-dump.sh expects in order to use scp with gcloud.
export PROJECT=%[1]s
export ZONE='%[2]s'
export KUBERNETES_PROVIDER=gke
export KUBE_NODE_OS_DISTRIBUTION='%[3]s'
%[5]s
`
	for _, project := range d.Projects {
		// Prevent an obvious injection.
		if strings.Contains(d.localLogsDir, "'") || strings.Contains(d.gcsLogsDir, "'") {
			return fmt.Errorf("%q or %q contain single quotes - nice try", d.localLogsDir, d.gcsLogsDir)
		}

		// Generate a slice of filters to be OR'd together below
		var filters []string
		for _, cluster := range d.projectClustersLayout[project] {
			if err := d.GetInstanceGroups(); err != nil {
				return err
			}
			for _, ig := range d.instanceGroups[project][cluster.name] {
				filters = append(filters, fmt.Sprintf("(metadata.created-by:*%s)", ig.path))
			}
		}

		// Generate the log-dump.sh command-line
		dumpCmd := fmt.Sprintf("./cluster/log-dump/log-dump.sh '%s'", d.localLogsDir)
		if d.gcsLogsDir != "" {
			dumpCmd += " " + d.gcsLogsDir
		}
		cmd := exec.Command("bash", "-c", fmt.Sprintf(gkeLogDumpTemplate,
			project,
			d.Zones[d.retryCount],
			os.Getenv("NODE_OS_DISTRIBUTION"),
			strings.Join(filters, " OR "),
			dumpCmd))
		cmd.SetDir(d.RepoRoot)
		if err := runWithOutput(cmd); err != nil {
			return err
		}
	}

	return nil
}
