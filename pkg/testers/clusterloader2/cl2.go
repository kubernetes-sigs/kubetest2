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

package clusterloader2

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kballard/go-shellquote"
	"github.com/octago/sflags/gen/gpflag"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubetest2/pkg/exec"
	"sigs.k8s.io/kubetest2/pkg/testers"
	suite "sigs.k8s.io/kubetest2/pkg/testers/clusterloader2/suite"
)

var GitTag string

type Tester struct {
	Suites                    []string `desc:"List of standard scale testing suites e.g. load, density"`
	TestOverrides             []string `desc:"List of paths to the config override files. The latter overrides take precedence over changes in former files."`
	TestConfigs               []string `desc:"List of paths to test config files."`
	Provider                  string   `desc:"The type of cluster provider used (e.g gke, gce, skeleton)"`
	KubeConfig                string   `desc:"Path to kubeconfig. If specified will override the path exposed by the kubetest2 deployer."`
	RepoRoot                  string   `desc:"Path to repository root of kubernetes/perf-tests"`
	ReportDir                 string   `desc:"Path to directory, where summaries files should be stored. If not specified, summaries are stored in $ARTIFACTS directory"`
	Nodes                     int      `desc:"Number of nodes in the cluster. 0 will auto-detect schedulable nodes."`
	EnablePrometheusServer    bool     `desc:"Whether to set-up the prometheus server in the cluster."`
	PrometheusPvcStorageClass string   `desc:"Storage class used with prometheus persistent volume claim."`
	ExtraArgs                 string   `flag:"~extra-args" desc:"Additional arguments supported by clusterloader2 (https://github.com/kubernetes/perf-tests/blob/master/clusterloader2/cmd/clusterloader.go)."`
}

func NewDefaultTester() *Tester {
	return &Tester{
		// TODO(amwat): pass kubetest2 deployer info here if possible
		Provider:   "skeleton",
		KubeConfig: os.Getenv("KUBECONFIG"),
		ReportDir:  os.Getenv("ARTIFACTS"),
	}
}

// Test runs the test
func (t *Tester) Test(fs *pflag.FlagSet) error {
	if t.RepoRoot == "" {
		return fmt.Errorf("required path to kubernetes/perf-tests repository")
	}

	var testConfigs, testOverrides []string
	if len(t.TestConfigs) > 0 {
		testConfigs = append(testConfigs, t.TestConfigs...)
	}
	if len(t.TestOverrides) > 0 {
		testOverrides = append(testOverrides, t.TestOverrides...)
	}

	for _, sweet := range t.Suites {
		if s := suite.GetSuite(sweet); s != nil {
			if len(s.TestConfigs) > 0 {
				testConfigs = append(testConfigs, s.TestConfigs...)
			}
			if len(s.TestOverrides) > 0 {
				testOverrides = append(testOverrides, s.TestOverrides...)
			}
		}
	}

	cmdArgs := []string{
		"run",
		"cmd/clusterloader.go",
	}

	args := []string{
		"--provider=" + t.Provider,
		"--kubeconfig=" + t.KubeConfig,
		"--report-dir=" + t.ReportDir,
	}

	if verbosity := fs.Lookup("v"); verbosity != nil {
		args = append(args, "--v="+verbosity.Value.String())
	}

	for _, tc := range testConfigs {
		if tc != "" {
			args = append(args, "--testconfig="+tc)
		}
	}
	for _, to := range testOverrides {
		if to != "" {
			args = append(args, "--testoverrides="+to)
		}
	}

	if t.EnablePrometheusServer {
		args = append(args, "--enable-prometheus-server")
	}
	if t.PrometheusPvcStorageClass != "" {
		args = append(args, "--prometheus-pvc-storage-class="+t.PrometheusPvcStorageClass)
	}
	parsedExtraArgs, err := shellquote.Split(t.ExtraArgs)
	if err != nil {
		return fmt.Errorf("error parsing --extra-args: %v", err)
	}
	args = append(args, parsedExtraArgs...)

	// TODO(amwat): get prebuilt binaries
	cmd := exec.Command("go", append(cmdArgs, args...)...)
	exec.InheritOutput(cmd)
	cmd.SetDir(filepath.Join(t.RepoRoot, "clusterloader2"))
	klog.V(2).Infof("running clusterloader2 %s", args)
	return cmd.Run()
}

func (t *Tester) Execute() error {
	fs, err := gpflag.Parse(t)
	if err != nil {
		return fmt.Errorf("failed to initialize tester: %v", err)
	}

	// initing the klog flags adds them to goflag.CommandLine
	// they can then be added to the built pflag set
	klog.InitFlags(nil)
	fs.AddGoFlagSet(flag.CommandLine)

	help := fs.BoolP("help", "h", false, "")
	if err := fs.Parse(os.Args); err != nil {
		return fmt.Errorf("failed to parse flags: %v", err)
	}

	if *help {
		fs.SetOutput(os.Stdout)
		fs.PrintDefaults()
		return nil
	}
	if err := testers.WriteVersionToMetadata(GitTag, ""); err != nil {
		return err
	}
	return t.Test(fs)
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run clusterloader2 tester: %v", err)
	}
}
