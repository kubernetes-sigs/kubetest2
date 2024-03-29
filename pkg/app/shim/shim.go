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

package shim

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"sigs.k8s.io/kubetest2/pkg/process"
)

// GitTag captures the git commit SHA of the build. This gets printed by all the deployers and testers
var GitTag string

// Main implements the kubetest2 root binary entrypoint
func Main() {
	if err := Run(); err != nil {
		os.Exit(1)
	}
}

// Run instantiates and executes the shim cobra command, returning the result
func Run() error {
	return NewCommand().Execute()
}

var usageLong = `kubetest2 is a tool for kubernetes end to end testing.

It orchestrates creating clusters, building kubernetes, deleting clusters, running tests, etc.

kubetest2 should be called with a deployer like: 'kubetest2 kind --help'

For more information see: https://github.com/kubernetes-sigs/kubetest2`

// NewCommand returns a new cobra.Command for building the base image
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           fmt.Sprintf("%s [deployer]", BinaryName),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runE,
	}

	// enable our custom help function, which will list known deployers
	cmd.SetHelpFunc(help)

	// we want all flags to be passed through without parsing
	cmd.DisableFlagParsing = true

	return cmd
}

// runE implements the actual command logic
func runE(cmd *cobra.Command, args []string) error {
	// there should be at least one argument (the deployer) unless the user
	// is asking for help on the shim itself
	if len(args) < 1 {
		return cmd.Help()
	}

	// gracefully handle help or version command if it is the only argument
	if len(args) == 1 {
		// check for -h, --help
		flags := pflag.NewFlagSet(BinaryName, pflag.ContinueOnError)
		help := flags.BoolP("help", "h", false, "")
		// check for -v, --version
		ver := flags.BoolP("version", "v", false, fmt.Sprintf("prints %s version", BinaryName))
		// we don't care about errors, only if -h / --help was set
		_ = flags.Parse(args)
		if *help {
			return cmd.Help()
		}
		if *ver {
			fmt.Printf("%s version %s\n", BinaryName, GitTag)
			return nil
		}
	}

	// otherwise find and execute the deployer with the remaining arguments
	deployerName := args[0]
	deployer, err := FindDeployer(deployerName)
	if err != nil {
		cmd.Printf("Error: could not find kubetest2 deployer %#v\n", deployerName)
		cmd.Println()
		usage(cmd)
		return err
	}

	env := os.Environ()
	version := fmt.Sprintf("kubetest2 version %s", GitTag)
	env = append(env, fmt.Sprintf("KUBETEST2_VERSION=%s", version))
	return process.Exec(deployer, args[1:], env)
}

// custom help info, includes usage()
//
//nolint:revive
func help(cmd *cobra.Command, args []string) {
	cmd.Println(usageLong)
	cmd.Println()
	usage(cmd)
}

// the usage subset of help info, attempts to identify and list known deployers
func usage(cmd *cobra.Command) {
	deployers := FindDeployers()
	cmd.Println("Usage:")
	cmd.Printf("  %s [deployer] [flags]\n", BinaryName)
	cmd.Println()
	cmd.Println("Detected Deployers:")
	for deployer := range deployers {
		cmd.Printf("  %s\n", deployer)
	}
	cmd.Println()
	testers := FindTesters()
	cmd.Println("Detected Testers:")
	for tester := range testers {
		cmd.Printf("  %s\n", tester)
	}
	cmd.Println()
	cmd.Println("For more help, run kubetest2 [deployer] --help")
}
