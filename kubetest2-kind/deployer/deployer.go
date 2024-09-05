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
	"flag"
	"os"
	"path/filepath"

	"github.com/octago/sflags/gen/gpflag"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/artifacts"
	"sigs.k8s.io/kubetest2/pkg/types"
)

// Name is the name of the deployer
const Name = "kind"

var GitTag string

// New implements deployer.New for kind
func New(opts types.Options) (types.Deployer, *pflag.FlagSet) {
	// create a deployer object and set fields that are not flag controlled
	d := &deployer{
		commonOptions: opts,
		logsDir:       filepath.Join(artifacts.BaseDir(), "logs"),
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
	NodeImage      string `flag:"image-name" desc:"the image name to use for build and up"`
	BaseImage      string `flag:"base-image" desc:"the base image name to use for the build"`
	ClusterName    string `flag:"cluster-name" desc:"the kind cluster --name"`
	BuildType      string `desc:"--type for kind build node-image"`
	ConfigPath     string `flag:"config" desc:"--config for kind create cluster"`
	KubeconfigPath string `flag:"kubeconfig" desc:"--kubeconfig flag for kind create cluster"`
	KubeRoot       string `desc:"the Kubernetes source for kind build node-image"`

	logsDir string
}

func (d *deployer) Kubeconfig() (string, error) {
	if d.KubeconfigPath != "" {
		return d.KubeconfigPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kube", "config"), nil
}

func (d *deployer) Version() string {
	return GitTag
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

// assert that deployer implements types.DeployerWithKubeconfig
var _ types.DeployerWithKubeconfig = &deployer{}

// well-known kind related constants
const kindDefaultBuiltImageName = "kindest/node:latest"
