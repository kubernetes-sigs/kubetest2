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

// Package deployer implements the kubetest2 kind deployer
package deployer

import (
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubetest2/pkg/types"
)

// Name is the name of the deployer
const Name = "kind"

// New implements deployer.New for kind
func New(opts types.Options) (types.Deployer, *pflag.FlagSet) {
	// create a deployer object and set fields that are not flag controlled
	d := &deployer{
		commonOptions: opts,
		logsDir:       filepath.Join(opts.RunDir(), "logs"),
	}
	// register flags and return
	return d, bindFlags(d)
}

// assert that New implements types.NewDeployer
var _ types.NewDeployer = New

type deployer struct {
	// generic parts
	commonOptions types.Options
	// kind specific details
	nodeImage      string // name of the node image built / deployed
	clusterName    string // --name flag value for kind
	logLevel       string // log level for kind commands
	logsDir        string // dir to export logs to
	buildType      string // --type flag to kind build node-image
	configPath     string // --config flag for kind create cluster
	kubeconfigPath string // --kubeconfig flag for kind create cluster
	kubeRoot       string // --kube-root for kind build node-image
	verbosity      int    // --verbosity for kind
}

func (d *deployer) Kubeconfig() (string, error) {
	if d.kubeconfigPath != "" {
		return d.kubeconfigPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kube", "config"), nil
}

// helper used to create & bind a flagset to the deployer
func bindFlags(d *deployer) *pflag.FlagSet {
	flags := pflag.NewFlagSet(Name, pflag.ContinueOnError)
	flags.StringVar(
		&d.clusterName, "cluster-name", "kind-kubetest2", "the kind cluster --name",
	)
	flags.StringVar(
		&d.logLevel, "loglevel", "", "--loglevel for kind commands",
	)
	flags.StringVar(
		&d.nodeImage, "image-name", "", "the image name to use for build and up",
	)
	flags.StringVar(
		&d.buildType, "build-type", "", "--type for kind build node-image",
	)
	flags.StringVar(
		&d.configPath, "config", "", "--config for kind create cluster",
	)
	flags.StringVar(
		&d.kubeconfigPath, "kubeconfig", "", "--kubeconfig flag for kind create cluster",
	)
	flags.StringVar(
		&d.kubeRoot, "kube-root", "", "--kube-root flag for kind build node-image",
	)
	flags.IntVar(
		&d.verbosity, "verbosity", 0, "--verbosity flag for kind",
	)
	return flags
}

// assert that deployer implements types.DeployerWithKubeconfig
var _ types.DeployerWithKubeconfig = &deployer{}

// well-known kind related constants
const kindDefaultBuiltImageName = "kindest/node:latest"
