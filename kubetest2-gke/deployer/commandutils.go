/*
Copyright 2019 The Kubernetes Authors.

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
	"bytes"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

func (d *deployer) prepareGcpIfNeeded(projectID string) error {
	// TODO(RonWeber): This is an almost direct copy/paste from kubetest's prepareGcp()
	// It badly needs refactored.

	var endpoint string
	switch env := d.environment; {
	case env == "test":
		endpoint = "https://test-container.sandbox.googleapis.com/"
	case env == "staging":
		endpoint = "https://staging-container.sandbox.googleapis.com/"
	case env == "staging2":
		endpoint = "https://staging2-container.sandbox.googleapis.com/"
	case env == "prod":
		endpoint = "https://container.googleapis.com/"
	case urlRe.MatchString(env):
		endpoint = env
	default:
		return fmt.Errorf("--environment must be one of {test,staging,staging2,prod} or match %v, found %q", urlRe, env)
	}

	if err := os.Setenv("CLOUDSDK_CORE_PRINT_UNHANDLED_TRACEBACKS", "1"); err != nil {
		return fmt.Errorf("could not set CLOUDSDK_CORE_PRINT_UNHANDLED_TRACEBACKS=1: %w", err)
	}
	if err := os.Setenv("CLOUDSDK_API_ENDPOINT_OVERRIDES_CONTAINER", endpoint); err != nil {
		return err
	}

	if err := runWithOutput(exec.RawCommand("gcloud config set project " + projectID)); err != nil {
		return fmt.Errorf("failed to set project %s: %w", projectID, err)
	}

	// gcloud creds may have changed
	if err := activateServiceAccount(d.gcpServiceAccount); err != nil {
		return err
	}

	if !d.gcpSSHKeyIgnored {
		// Ensure ssh keys exist
		klog.V(1).Info("Checking existing of GCP ssh keys...")
		k := filepath.Join(home(".ssh"), "google_compute_engine")
		if _, err := os.Stat(k); err != nil {
			return err
		}
		pk := k + ".pub"
		if _, err := os.Stat(pk); err != nil {
			return err
		}
	}

	//TODO(RonWeber): kubemark
	return nil
}

// Activate service account if set or do nothing.
func activateServiceAccount(path string) error {
	if path == "" {
		return nil
	}
	return runWithOutput(exec.RawCommand("gcloud auth activate-service-account --key-file=" + path))
}

// Get the project number for the given project ID.
func getProjectNumber(projectID string) (string, error) {
	// Get the service project number.
	projectNum, err := exec.Output(exec.RawCommand(
		fmt.Sprintf("gcloud projects describe %s --format=value(projectNumber)", projectID)))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(projectNum)), nil
}

// home returns $HOME/part/part/part
func home(parts ...string) string {
	p := append([]string{os.Getenv("HOME")}, parts...)
	return filepath.Join(p...)
}

func getClusterCredentials(project, loc, cluster string) error {
	// Get gcloud to create the file.
	if err := runWithOutput(exec.Command("gcloud",
		containerArgs("clusters", "get-credentials", cluster, "--project="+project, loc)...),
	); err != nil {
		return fmt.Errorf("error executing get-credentials: %w", err)
	}

	return nil
}

// Resolve the current latest version in the given release channel.
func resolveLatestVersionInChannel(loc, channelName string) (string, error) {
	// Get the server config for the current location.
	out, err := exec.Output(exec.RawCommand(
		fmt.Sprintf("gcloud container get-server-config --format=\"yaml(channels:format='yaml(channel,validVersions)')\" %s", loc)))
	if err != nil {
		return "", fmt.Errorf("failed to get the server config: %w", err)
	}

	type Channel struct {
		Name          string   `yaml:"channel"`
		ValidVersions []string `yaml:"validVersions"`
	}
	type Channels struct {
		Channels []Channel `yaml:"channels"`
	}
	var cs Channels
	if err = yaml.Unmarshal(out, &cs); err != nil {
		return "", fmt.Errorf("failed to unmarshal the server config: %w", err)
	}

	for _, channel := range cs.Channels {
		if strings.EqualFold(channel.Name, channelName) {
			if len(channel.ValidVersions) == 0 {
				return "", fmt.Errorf("no valid versions for channel %q", channelName)
			}
			return channel.ValidVersions[0], nil
		}
	}

	return "", fmt.Errorf("channel %q does not exist in the server config", channelName)
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

func runWithOutputAndBuffer(cmd exec.Cmd) (string, error) {
	var buf bytes.Buffer

	exec.SetOutput(cmd, io.MultiWriter(os.Stdout, &buf), io.MultiWriter(os.Stderr, &buf))
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// execError returns a string format of err including stderr if the
// err is an ExitError, useful for errors from e.g. exec.Cmd.Output().
func execError(err error) string {
	if ee, ok := err.(*osexec.ExitError); ok {
		return fmt.Sprintf("%v (output: %q)", err, string(ee.Stderr))
	}
	return err.Error()
}
