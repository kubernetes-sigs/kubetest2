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
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/octago/sflags/gen/gpflag"
	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

const (
	binary = "ginkgo" // TODO(RonWeber): Actually find these binaries.
)

type Tester struct {
	FlakeAttempts int    `desc:"Make up to this many attempts to run each spec."`
	Parallel      int    `desc:"Run this many tests in parallel at once."`
	SkipRegex     string `desc:"Regular expression of jobs to skip."`
	FocusRegex    string `desc:"Regular expression of jobs to focus on."`

	kubeconfigPath string
}

// Test runs the test
func (t *Tester) Test() error {
	if err := t.pretestSetup(); err != nil {
		return err
	}
	// Fixing this path temporarily for local testing
	// TODO(amwat): implement actual logic
	e2eTestPath := filepath.Join(os.Getenv("ARTIFACTS"), "kubernetes", "test", "bin", "e2e.test")

	e2eTestArgs := []string{
		"--kubeconfig=" + t.kubeconfigPath,
		"--ginkgo.flakeAttempts=" + strconv.Itoa(t.FlakeAttempts),
		"--ginkgo.skip=" + t.SkipRegex,
		"--ginkgo.focus=" + t.FocusRegex,
	}
	ginkgoArgs := append([]string{
		"--nodes=" + strconv.Itoa(t.Parallel),
		e2eTestPath,
		"--"}, e2eTestArgs...)

	log.Printf("Running ginkgo test as %s %+v", binary, ginkgoArgs)
	cmd := exec.Command(binary, ginkgoArgs...)
	exec.InheritOutput(cmd)
	return cmd.Run()
}

func (t *Tester) pretestSetup() error {
	if config := os.Getenv("KUBECONFIG"); config != "" {
		t.kubeconfigPath = config
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to find home directory: %v", err)
		}
		t.kubeconfigPath = filepath.Join(home, ".kube", "config")
	}
	log.Printf("Using kubeconfig at %s", t.kubeconfigPath)

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
		FlakeAttempts: 1,
		Parallel:      1,
	}
}

func Main() {
	t := NewDefaultTester()
	if err := t.Execute(); err != nil {
		klog.Fatalf("failed to run ginkgo tester: %v", err)
	}
}
