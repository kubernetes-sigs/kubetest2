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
	"path"
	"path/filepath"
	"strings"

	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/fs"
	"sigs.k8s.io/kubetest2/pkg/util"
)

const (
	ciPrivateKeyEnv = "GCE_SSH_PRIVATE_KEY_FILE"
	ciPublicKeyEnv  = "GCE_SSH_PUBLIC_KEY_FILE"
)

func (d *deployer) IsUp() (up bool, err error) {
	klog.V(1).Info("GCE deployer starting IsUp()")

	if err := d.init(); err != nil {
		return false, fmt.Errorf("isUp failed to init: %s", err)
	}

	if d.GCPProject == "" {
		return false, fmt.Errorf("isup requires a GCP project")
	}

	env := d.buildEnv()
	// naive assumption: nodes reported = cluster up
	// similar to other deployers' implementations
	args := []string{
		d.kubectlPath,
		"get",
		"nodes",
		"-o=name",
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.SetEnv(env...)
	cmd.SetStderr(os.Stderr)
	lines, err := exec.OutputLines(cmd)
	if err != nil {
		return false, fmt.Errorf("is up failed to get nodes: %s", err)
	}

	return len(lines) > 0, nil
}

func (d *deployer) Up() error {
	klog.V(1).Info("GCE deployer starting Up()")

	if err := d.init(); err != nil {
		return fmt.Errorf("up failed to init: %s", err)
	}

	env := d.buildEnv()
	// if --build isn't passed, fetch the kubernetes binaries
	if !d.commonOptions.ShouldBuild() {
		script := filepath.Join(d.RepoRoot, "cluster", "get-kube.sh")
		klog.V(2).Infof("About to run script at: %s", script)
		kubeURL, err := util.ParseKubernetesMarker(d.KubernetesVersion)
		if err != nil {
			return fmt.Errorf("failed to resolve kubernetes version marker: %s", err)
		}
		version := path.Base(kubeURL)
		releaseURL, found := strings.CutSuffix(kubeURL, "/"+version)
		if !found {
			releaseURL = "https://dl.k8s.io/release"
		}

		cmd := exec.Command(script)
		env = append(env,
			fmt.Sprintf("KUBERNETES_RELEASE_URL=%s", releaseURL),
			fmt.Sprintf("KUBERNETES_RELEASE=%s", version),
			fmt.Sprintf("KUBE_ROOT=%s", d.RepoRoot+"/kubernetes"),
			"KUBERNETES_SKIP_CONFIRM=y",
			"KUBERNETES_SKIP_CREATE_CLUSTER=y",
			"KUBERNETES_DOWNLOAD_TESTS=n",
		)
		cmd.SetEnv(env...)
		exec.InheritOutput(cmd)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error encountered during %s: %s", script, err)
		}
	}
	path, err := d.verifyKubectl()
	if err != nil {
		return err
	}
	d.kubectlPath = path

	if d.EnableComputeAPI {
		klog.V(2).Info("enabling compute API for project")
		if err := enableComputeAPI(d.GCPProject); err != nil {
			return fmt.Errorf("up couldn't enable compute API: %s", err)
		}
	}

	maybeSetupSSHKeys()

	script := filepath.Join(d.RepoRoot, "cluster", "kube-up.sh")
	klog.V(2).Infof("About to run script at: %s", script)

	cmd := exec.Command(script)
	cmd.SetEnv(env...)
	exec.InheritOutput(cmd)

	if err := cmd.Run(); err != nil {
		if err := d.DumpClusterLogs(); err != nil {
			klog.Warningf("Dumping cluster logs at the end of Up() failed: %s", err)
		}
		return fmt.Errorf("error encountered during %s: %s", script, err)
	}

	if isUp, err := d.IsUp(); err != nil {
		if err := d.DumpClusterLogs(); err != nil {
			klog.Warningf("Dumping cluster logs at the end of Up() failed: %s", err)
		}
		klog.Warningf("failed to check if cluster is up: %s", err)
	} else if isUp {
		klog.V(1).Infof("cluster reported as up")
	} else {
		if err := d.DumpClusterLogs(); err != nil {
			klog.Warningf("Dumping cluster logs at the end of Up() failed: %s", err)
		}
		klog.Errorf("cluster reported as down")
	}

	klog.V(2).Info("about to create nodeport firewall rule")
	if err := d.createFirewallRuleNodePort(); err != nil {
		if err := d.DumpClusterLogs(); err != nil {
			klog.Warningf("Dumping cluster logs at the end of Up() failed: %s", err)
		}
		return fmt.Errorf("failed to create firewall rule: %s", err)
	}

	return nil
}

func enableComputeAPI(project string) error {
	// In freshly created GCP projects, the compute API is
	// not enabled. We need it. Enabling it after it has
	// already been enabled is a relatively fast no-op,
	// so this can be called without consequence.

	env := os.Environ()
	cmd := exec.Command(
		"gcloud",
		"services",
		"enable",
		"compute.googleapis.com",
		"--project="+project,
	)
	cmd.SetEnv(env...)
	exec.InheritOutput(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to enable compute API: %s", err)
	}

	return nil
}

func (d *deployer) verifyUpFlags() error {
	if d.NumNodes < 1 {
		return fmt.Errorf("number of nodes must be at least 1")
	}

	if err := d.setRepoPathIfNotSet(); err != nil {
		return err
	}

	// verifyUpFlags does not check for a gcp project because it is
	// assumed that one will be acquired from boskos if it is not set

	return nil
}

// maybeSetupSSHKeys will best-effort try to setup ssh keys for gcloud to reuse
// from existing files pointed to by "well-known" environment variables used in CI
func maybeSetupSSHKeys() {
	home, err := os.UserHomeDir()
	if err != nil {
		klog.Warningf("failed to get user's home directory")
		return
	}
	// check if there are existing ssh keys, if either exist don't do anything
	klog.V(2).Info("checking for existing gcloud ssh keys...")
	privateKey := filepath.Join(home, ".ssh", "google_compute_engine")
	if _, err := os.Stat(privateKey); err == nil {
		klog.V(2).Infof("found existing private key at %s", privateKey)
		return
	}
	publicKey := privateKey + ".pub"
	if _, err := os.Stat(publicKey); err == nil {
		klog.V(2).Infof("found existing public key at %s", publicKey)
		return
	}

	// no existing keys check for CI variables, create gcloud key files if both exist
	// note only checks if relevant envs are non-empty, no actual key verification checks
	maybePrivateKey, privateKeyEnvSet := os.LookupEnv(ciPrivateKeyEnv)
	if !privateKeyEnvSet {
		klog.V(2).Infof("%s is not set", ciPrivateKeyEnv)
		return
	}
	maybePublicKey, publicKeyEnvSet := os.LookupEnv(ciPublicKeyEnv)
	if !publicKeyEnvSet {
		klog.V(2).Infof("%s is not set", ciPublicKeyEnv)
		return
	}

	if err := fs.CopyFile(maybePrivateKey, privateKey); err != nil {
		klog.Warningf("failed to copy %s to %s: %v", maybePrivateKey, privateKey, err)
		return
	}

	if err := fs.CopyFile(maybePublicKey, publicKey); err != nil {
		klog.Warningf("failed to copy %s to %s: %v", maybePublicKey, publicKey, err)
	}
}
