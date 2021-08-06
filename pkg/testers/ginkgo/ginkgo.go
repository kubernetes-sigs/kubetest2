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

	"sigs.k8s.io/kubetest2/pkg/artifacts"
	"sigs.k8s.io/kubetest2/pkg/build"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

var GitTag string

type Tester struct {
	FlakeAttempts      int    `desc:"Make up to this many attempts to run each spec."`
	GinkgoArgs         string `desc:"Additional arguments supported by the ginkgo binary."`
	Parallel           int    `desc:"Run this many tests in parallel at once."`
	SkipRegex          string `desc:"Regular expression of jobs to skip."`
	FocusRegex         string `desc:"Regular expression of jobs to focus on."`
	TestPackageVersion string `desc:"The ginkgo tester uses a test package made during the kubernetes build. The tester downloads this test package from one of the release tars published to GCS. Defaults to latest. Use \"gsutil ls gs://kubernetes-release/release/\" to find release names. Example: v1.20.0-alpha.0"`
	TestPackageBucket  string `desc:"The bucket which release tars will be downloaded from to acquire the test package. Defaults to the main kubernetes project bucket."`
	TestPackageDir     string `desc:"The directory in the bucket which represents the type of release. Default to the release directory."`
	TestPackageMarker  string `desc:"The version marker in the directory containing the package version to download when unspecified. Defaults to latest.txt."`
	TestArgs           string `desc:"Additional arguments supported by the e2e test framework (https://godoc.org/k8s.io/kubernetes/test/e2e/framework#TestContextType)."`
	UseBuiltBinaries   bool   `desc:"determines whether to use binaries built by the deployer instead of extracting the test tars from GCS."`

	kubeconfigPath string
	runDir         string

	// These paths are set up by AcquireTestPackage()
	e2eTestPath string
	ginkgoPath  string
	kubectlPath string
}

// Test runs the test
func (t *Tester) Test() error {
	if err := t.pretestSetup(); err != nil {
		return err
	}

	e2eTestArgs := []string{
		"--kubeconfig=" + t.kubeconfigPath,
		"--kubectl-path=" + t.kubectlPath,
		"--ginkgo.flakeAttempts=" + strconv.Itoa(t.FlakeAttempts),
		"--ginkgo.skip=" + t.SkipRegex,
		"--ginkgo.focus=" + t.FocusRegex,
		"--report-dir=" + artifacts.BaseDir(),
	}
	extraE2EArgs, err := shellquote.Split(t.TestArgs)
	if err != nil {
		return fmt.Errorf("error parsing --test-args: %v", err)
	}
	e2eTestArgs = append(e2eTestArgs, extraE2EArgs...)

	extraGingkoArgs, err := shellquote.Split(t.GinkgoArgs)
	if err != nil {
		return fmt.Errorf("error parsing --gingko-args: %v", err)
	}

	ginkgoArgs := append(extraGingkoArgs,
		"--nodes="+strconv.Itoa(t.Parallel),
		t.e2eTestPath,
		"--")
	ginkgoArgs = append(ginkgoArgs, e2eTestArgs...)

	klog.V(0).Infof("Running ginkgo test as %s %+v", t.ginkgoPath, ginkgoArgs)
	cmd := exec.Command(t.ginkgoPath, ginkgoArgs...)
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

	if t.UseBuiltBinaries {
		if err := t.validateLocalBinaries(); err != nil {
			return err
		}
		return nil
	}

	if err := t.AcquireTestPackage(); err != nil {
		return fmt.Errorf("failed to get ginkgo test package from published releases: %s", err)
	}

	return nil
}

func (t *Tester) validateLocalBinaries() error {
	klog.V(2).Infof("checking existing test binaries ...")
	for _, binary := range build.CommonTestBinaries {
		path := filepath.Join(t.runDir, binary)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("failed to validate %s:%v", binary, err)
		}
		klog.V(2).Infof("found existing %s at %s", binary, path)
	}
	t.e2eTestPath = filepath.Join(t.runDir, "e2e.test")
	t.ginkgoPath = filepath.Join(t.runDir, "ginkgo")
	t.kubectlPath = filepath.Join(t.runDir, "kubectl")
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

	if err := t.initKubetest2Info(); err != nil {
		return err
	}
	return t.Test()
}

// initializes relevant information from the well defined kubetest2 environment variables.
func (t *Tester) initKubetest2Info() error {
	if dir, ok := os.LookupEnv("KUBETEST2_RUN_DIR"); ok {
		t.runDir = dir
		return nil
	}
	// default to current working directory if for some reason the env is not set
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to set run dir: %v", err)
	}
	t.runDir = dir
	return nil
}

func NewDefaultTester() *Tester {
	return &Tester{
		FlakeAttempts:     1,
		Parallel:          1,
		TestPackageBucket: "kubernetes-release",
		TestPackageDir:    "release",
		TestPackageMarker: "latest.txt",
	}
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run ginkgo tester: %v", err)
	}
}
