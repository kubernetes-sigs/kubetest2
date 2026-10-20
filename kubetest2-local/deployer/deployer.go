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

// Package deployer implements the kubetest2 local deployer, which brings up
// a single node cluster on the current host using hack/local-up-cluster.sh
// from a kubernetes/kubernetes checkout.
package deployer

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/octago/sflags/gen/gpflag"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/artifacts"
	"sigs.k8s.io/kubetest2/pkg/types"
)

// Name is the name of the deployer
const Name = "local"

var GitTag string

// New implements deployer.New for local
func New(opts types.Options) (types.Deployer, *pflag.FlagSet) {
	d := &deployer{
		commonOptions:  opts,
		clusterLogsDir: filepath.Join(opts.RunDir(), "local-up-cluster-logs"),
		logsDir:        filepath.Join(artifacts.BaseDir(), "cluster-logs"),
		CertDir:        "/var/run/kubernetes",
		UpTimeout:      2 * time.Minute,
	}
	return d, bindFlags(d)
}

// assert that New implements types.NewDeployer
var _ types.NewDeployer = New

type deployer struct {
	// generic parts
	commonOptions types.Options

	RepoRoot  string        `desc:"The path to the root of the local kubernetes/kubernetes repo. Defaults to the current directory."`
	CertDir   string        `desc:"Sets CERT_DIR for local-up-cluster.sh. The admin kubeconfig is read from <cert-dir>/admin.kubeconfig."`
	UpTimeout time.Duration `desc:"Time limit between local-up-cluster.sh returning and the cluster reporting a Ready node."`
	Env       []string      `desc:"A list of env variables (KEY=VALUE) to pass to local-up-cluster.sh. These take precedence over deployer defaults."`

	// LOG_DIR passed to local-up-cluster.sh
	clusterLogsDir string
	// where DumpClusterLogs copies clusterLogsDir to
	logsDir string
}

func (d *deployer) Kubeconfig() (string, error) {
	path := filepath.Join(d.CertDir, "admin.kubeconfig")
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("failed to find kubeconfig at %s: %w", path, err)
	}
	return path, nil
}

func (d *deployer) Provider() string {
	return Name
}

func (d *deployer) Version() string {
	return GitTag
}

func (d *deployer) setRepoRootIfNotSet() error {
	if d.RepoRoot != "" {
		return nil
	}
	path, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory for setting Kubernetes root path: %w", err)
	}
	klog.V(1).Infof("defaulting repo root to the current directory: %s", path)
	d.RepoRoot = path
	return nil
}

// getScript returns the absolute path of a script relative to the repo root
func (d *deployer) getScript(scriptPath string) (string, error) {
	if err := d.setRepoRootIfNotSet(); err != nil {
		return "", err
	}
	path := filepath.Join(d.RepoRoot, scriptPath)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("unable to find script %s in directory %s: %w", scriptPath, d.RepoRoot, err)
	}
	return path, nil
}

// helper used to create & bind a flagset to the deployer
func bindFlags(d *deployer) *pflag.FlagSet {
	flags, err := gpflag.Parse(d)
	if err != nil {
		klog.Fatalf("unable to generate flags from deployer")
		return nil
	}

	// initing the klog flags adds them to goflag.CommandLine
	// they can then be added to the built pflag set
	klog.InitFlags(nil)
	flags.AddGoFlagSet(flag.CommandLine)

	return flags
}

// assert that deployer implements optional interfaces
var _ types.DeployerWithKubeconfig = &deployer{}
var _ types.DeployerWithProvider = &deployer{}
var _ types.DeployerWithVersion = &deployer{}
