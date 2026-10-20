/*
Copyright The Kubernetes Authors.

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
	"net"
	"os"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/metadata"
	"sigs.k8s.io/kubetest2/pkg/process"
)

func (d *deployer) Up() error {
	script, err := d.getScript("hack/local-up-cluster.sh")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(d.clusterLogsDir, 0755); err != nil {
		return fmt.Errorf("unable to create log directory %s: %w", d.clusterLogsDir, err)
	}
	// MkdirAll is subject to umask; components started via sudo need to write here
	if err := os.Chmod(d.clusterLogsDir, 0755); err != nil {
		return fmt.Errorf("unable to change log directory permissions: %w", err)
	}

	env := os.Environ()
	env = append(env,
		"ENABLE_DAEMON=true",
		"LOG_DIR="+d.clusterLogsDir,
		"CERT_DIR="+d.CertDir,
		// Needed for at least one conformance e2e test. See kubernetes/kubernetes#59978
		"ALLOW_PRIVILEGED=true",
	)

	// In a DIND scenario use the docker0 address so pod health checks
	// (e.g. for kube-dns) succeed; otherwise use local-up-cluster.sh defaults.
	if dockerIP := docker0IP(); dockerIP != "" {
		klog.V(0).Infof("using %s for API_HOST_IP, HOSTNAME_OVERRIDE, KUBELET_HOST", dockerIP)
		env = append(env,
			"API_HOST_IP="+dockerIP,
			"HOSTNAME_OVERRIDE="+dockerIP,
			"KUBELET_HOST="+dockerIP,
		)
	} else {
		klog.V(0).Info("using local-up-cluster.sh's defaults for API_HOST_IP, HOSTNAME_OVERRIDE, KUBELET_HOST")
	}
	env = append(env, d.Env...)

	klog.V(0).Infof("Up(): starting local cluster with %s", script)
	if err := process.ExecJUnit(script, nil, env); err != nil {
		return err
	}

	kubeconfig, err := d.Kubeconfig()
	if err != nil {
		return err
	}

	// propagated to the tester via the inherited environment
	for k, v := range map[string]string{
		"KUBECONFIG":                  kubeconfig,
		"KUBERNETES_CONFORMANCE_TEST": "yes",
		"KUBERNETES_PROVIDER":         Name,
	} {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}

	return d.waitForReady()
}

func (d *deployer) waitForReady() error {
	stop := time.Now().Add(d.UpTimeout)
	for {
		up, err := d.IsUp()
		if err != nil {
			return err
		}
		if up {
			return nil
		}
		if time.Now().After(stop) {
			return fmt.Errorf("local-up-cluster.sh cluster did not report a Ready node within %s", d.UpTimeout)
		}
		time.Sleep(5 * time.Second)
	}
}

// IsUp returns true if at least one node reports Ready
func (d *deployer) IsUp() (bool, error) {
	script, err := d.getScript("cluster/kubectl.sh")
	if err != nil {
		return false, err
	}
	kubeconfig, err := d.Kubeconfig()
	if err != nil {
		return false, err
	}
	lines, err := exec.OutputLines(exec.Command(script,
		"--kubeconfig="+kubeconfig,
		"get", "nodes",
		`-o=jsonpath={range .items[*]}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}`,
	))
	if err != nil {
		return false, metadata.NewJUnitError(err, strings.Join(lines, "\n"))
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "True" {
			return true, nil
		}
	}
	return false, nil
}

// docker0IP returns the first non-loopback IPv4 address of the docker0
// interface, or "" if there is none.
func docker0IP() string {
	docker0, err := net.InterfaceByName("docker0")
	if err != nil {
		klog.V(1).Infof("unable to find docker0 interface: %v", err)
		return ""
	}
	addresses, err := docker0.Addrs()
	if err != nil {
		klog.V(1).Infof("unable to get addresses from docker0 interface: %v", err)
		return ""
	}
	for _, address := range addresses {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return ""
}
