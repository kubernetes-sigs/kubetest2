/*
Copyright 2021 The Kubernetes Authors.

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

	"sigs.k8s.io/kubetest2/pkg/types"
)

// Name is the name of the deployer
const Name = "noop"

var GitTag string

// New implements deployer.New for kind
func New(opts types.Options) (types.Deployer, *pflag.FlagSet) {
	// create a deployer object and set fields that are not flag controlled
	d := &Deployer{
		commonOptions: opts,
	}
	// register flags and return
	return d, bindFlags(d)
}

// assert that New implements types.NewDeployer
var _ types.NewDeployer = New

type Deployer struct {
	// generic parts
	commonOptions types.Options

	KubeconfigPath string `flag:"kubeconfig" desc:"Absolute path to existing kubeconfig for cluster"`
}

func (d *Deployer) Up() error {
	return nil
}

func (d *Deployer) Down() error {
	return nil
}

func (d *Deployer) IsUp() (up bool, err error) {
	return false, nil
}

func (d *Deployer) DumpClusterLogs() error {
	return nil
}

func (d *Deployer) Build() error {
	// TODO: build should probably still exist with common options
	return nil
}

func (d *Deployer) Kubeconfig() (string, error) {
	// noop deployer is specifically used with an existing cluster and KUBECONFIG
	if d.KubeconfigPath != "" {
		return d.KubeconfigPath, nil
	}
	if kconfig, ok := os.LookupEnv("KUBECONFIG"); ok {
		return kconfig, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kube", "config"), nil
}

func (d *Deployer) Version() string {
	return GitTag
}

// helper used to create & bind a flagset to the deployer
func bindFlags(d *Deployer) *pflag.FlagSet {
	flags, err := gpflag.Parse(d)
	if err != nil {
		klog.Fatalf("unable to generate flags from deployer")
		return nil
	}

	flags.AddGoFlagSet(flag.CommandLine)

	return flags
}

// assert that deployer implements types.DeployerWithKubeconfig
var _ types.DeployerWithKubeconfig = &Deployer{}
