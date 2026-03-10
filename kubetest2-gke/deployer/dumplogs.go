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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

type configMap struct {
	Name      string
	Namespace string
	DataKey   string
}

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

		logDumpScript := "./cluster/log-dump/log-dump.sh"
		if envOverride := os.Getenv("LOG_DUMP_SCRIPT_PATH"); envOverride != "" {
			logDumpScript = envOverride
		}

		// Generate the log-dump.sh command-line
		dumpCmd := fmt.Sprintf("%s '%s'", logDumpScript, d.localLogsDir)
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

	if err := dumpConfigMaps(d.DumpConfigMaps); err != nil {
		log.Printf("Failed to dump configmaps: %v\n", err)
	}

	return nil
}

func dumpConfigMaps(dumpConfigMapsJSONString string) error {
	var configMaps []configMap
	if err := json.Unmarshal([]byte(dumpConfigMapsJSONString), &configMaps); err != nil {
		return err
	}

	var errorMessages []string
	dumpValues := make(map[string]string)
	ctx := context.Background()

	clientset, err := createClientset()
	if err != nil {
		return err
	}

	for _, cm := range configMaps {
		kubeCm, err := clientset.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("failed to get %s/%s: %v", cm.Namespace, cm.Name, err))
			continue
		}

		val, ok := kubeCm.Data[cm.DataKey]
		if !ok {
			errorMessages = append(errorMessages, fmt.Sprintf("key %s not found in %s/%s", cm.DataKey, cm.Namespace, cm.Name))
			continue
		}

		jsonKey := strings.Join([]string{cm.Namespace, cm.Name, cm.DataKey}, ".")
		dumpValues[jsonKey] = val
	}

	if len(errorMessages) > 0 {
		return fmt.Errorf("errors while dumping ConfigMaps: %s", strings.Join(errorMessages, "; "))
	}

	jsonDump, err := json.Marshal(dumpValues)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(os.Getenv("ARTIFACTS"), "gke-configmap.json")
	return os.WriteFile(outputPath, jsonDump, 0644)
}

// createClientset initializes a Kubernetes clientset.
// It tries to use the default kubeconfig file.
func createClientset() (kubernetes.Interface, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}
	return clientset, nil
}
