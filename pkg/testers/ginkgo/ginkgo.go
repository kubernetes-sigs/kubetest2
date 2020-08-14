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

package ginkgo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kballard/go-shellquote"
	"github.com/octago/sflags/gen/gpflag"
	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

var (
	// These paths are set up by AcquireTestPackage()
	e2eTestPath = filepath.Join(os.Getenv("ARTIFACTS"), "e2e.test")
	binary      = filepath.Join(os.Getenv("ARTIFACTS"), "ginkgo")
)

type Tester struct {
	FlakeAttempts      int    `desc:"Make up to this many attempts to run each spec."`
	Parallel           int    `desc:"Run this many tests in parallel at once."`
	SkipRegex          string `desc:"Regular expression of jobs to skip."`
	FocusRegex         string `desc:"Regular expression of jobs to focus on."`
	TestPackageVersion string `desc:"The ginkgo tester uses a test package made during the kubernetes build. The tester downloads this test package from one of the release tars published to GCS. Defaults to latest. Use \"gsutil ls gs://kubernetes-release/release/\" to find release names. Example: v1.20.0-alpha.0"`
	TestPackageBucket  string `desc:"The bucket which release tars will be downloaded from to acquire the test package. Defaults to the main kubernetes project bucket."`
	TestArgs           string `desc:"Additional arguments supported by the e2e test framework (https://godoc.org/k8s.io/kubernetes/test/e2e/framework#TestContextType)."`

	kubeconfigPath string
}

// Test runs the test
func (t *Tester) Test() error {
	if err := t.pretestSetup(); err != nil {
		return err
	}

	e2eTestArgs := []string{
		"--kubeconfig=" + t.kubeconfigPath,
		"--ginkgo.flakeAttempts=" + strconv.Itoa(t.FlakeAttempts),
		"--ginkgo.skip=" + t.SkipRegex,
		"--ginkgo.focus=" + t.FocusRegex,
		"--report-dir=" + os.Getenv("ARTIFACTS"),
	}
	extraArgs, err := shellquote.Split(t.TestArgs)
	if err != nil {
		return fmt.Errorf("error parsing --test-args: %v", err)
	}
	e2eTestArgs = append(e2eTestArgs, extraArgs...)
	ginkgoArgs := append([]string{
		"--nodes=" + strconv.Itoa(t.Parallel),
		e2eTestPath,
		"--"}, e2eTestArgs...)

	klog.V(0).Infof("Running ginkgo test as %s %+v", binary, ginkgoArgs)
	cmd := exec.Command(binary, ginkgoArgs...)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func (t *Tester) pretestSetup() error {
	if config := os.Getenv("KUBECONFIG"); config != "" {
		// The ginkgo tester errors out if the kubeconfig provided
		// is not an absolute path, likely because ginkgo changes its
		// working directory while executing. To get around this problem
		// we can manually edit the provided KUBECONFIG to ensure a
		// successful run.
		if !filepath.IsAbs(config) {
			newKubeconfig, err := filepath.Abs(config)
			if err != nil {
				return fmt.Errorf("failed to convert kubeconfig to absolute path: %s", err)
			}
			klog.V(0).Infof("Ginkgo tester received a non-absolute path for KUBECONFIG. Updating to: %s", newKubeconfig)
			config = newKubeconfig
		}

		t.kubeconfigPath = config
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to find home directory: %v", err)
		}
		t.kubeconfigPath = filepath.Join(home, ".kube", "config")
	}
	klog.V(0).Infof("Using kubeconfig at %s", t.kubeconfigPath)

	if err := t.AcquireTestPackage(); err != nil {
		return fmt.Errorf("failed to get ginkgo test package from published releases: %s", err)
	}

	return nil
}

func (t *Tester) Execute() error {
	fs, err := gpflag.Parse(t)
	if err != nil {
		return fmt.Errorf("failed to initialize tester: %v", err)
	}

	help := fs.BoolP("help", "h", false, "")
	if err := fs.Parse(os.Args); err != nil {
		return fmt.Errorf("failed to parse flags: %v", err)
	}

	if *help {
		fs.SetOutput(os.Stdout)
		fs.PrintDefaults()
		return nil
	}

	return t.Test()
}

func NewDefaultTester() *Tester {
	return &Tester{
		FlakeAttempts:     1,
		Parallel:          1,
		TestPackageBucket: "kubernetes-release",
	}
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run ginkgo tester: %v", err)
	}
}
