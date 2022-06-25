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

package artifacts

import (
	"fmt"
	"github.com/spf13/pflag"
	"k8s.io/klog"
	"os"
	"path/filepath"
)

var baseDir string
var RunDirFlag string

// BaseDir returns the path to the directory where artifacts should be written
// (including metadata files like junit_runner.xml)
func BaseDir() string {
	d := baseDir
	if d == "" {
		def, err := defaultArtifactsDir()
		if err != nil {
			klog.Fatalf("failed to get default artifacts directory: %s", err)
		}
		d = def
	}
	return d
}

// RunDir returns the path to the directory used for storing all files in general
// specific to a single run of kubetest2
func RunDir() string {
	d := RunDirFlag
	if d == "" {
		def, err := defaultRunDir()
		if err != nil {
			klog.Fatalf("failed to get default Run directory: %s", err)
		}
		d = def
	}
	return d
}

// the default is $ARTIFACTS if set, otherwise ./_artifacts
// constructed as an absolute path to help the ginkgo tester because
// for some reason it needs an absolute path to the kubeconfig
func defaultArtifactsDir() (string, error) {
	if path, set := os.LookupEnv("ARTIFACTS"); set {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to convert filepath from $ARTIFACTS (%s) to absolute path: %s", path, err)
		}
		return absPath, nil
	}

	absPath, err := filepath.Abs("_artifacts")
	if err != nil {
		return "", fmt.Errorf("when constructing default artifacts dir, failed to get absolute path: %s", err)
	}
	return absPath, nil
}

// the default is $KUBETEST2_RUN_DIR if set, otherwise ./_rundir
// constructed as an absolute path to help the ginkgo tester because
// for some reason it needs an absolute path to the kubeconfig
func defaultRunDir() (string, error) {
	if path, set := os.LookupEnv("KUBETEST2_RUN_DIR"); set {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to convert filepath from $KUBETEST2_RUN_DIR (%s) to absolute path: %s", path, err)
		}
		return absPath, nil
	}
	absPath, err := filepath.Abs("_rundir")
	if err != nil {
		return "", fmt.Errorf("when constructing default rundir, failed to get absolute path: %s", err)
	}
	return absPath, nil
}

// BindFlags binds the artifact and rundir related flags.
func BindFlags(flags *pflag.FlagSet) error {
	defaultArtifacts, err := defaultArtifactsDir()
	if err != nil {
		return err
	}
	flags.StringVar(&baseDir, "artifacts", defaultArtifacts, `top-level directory to put artifacts under for each kubetest2 run, defaulting to "${ARTIFACTS:-./_artifacts}". If using the ginkgo tester, this must be an absolute path.`)
	flags.StringVar(&RunDirFlag, "rundir", "", `directory to put run related test binaries like e2e.test, ginkgo, kubectl for each kubetest2 run, defaulting to "${KUBETEST2_RUN_DIR:-./_rundir}". If using the ginkgo tester, this must be an absolute path.`)
	return nil
}

// MustBindFlags calls BindFlags and panics on an error
func MustBindFlags(flags *pflag.FlagSet) {
	err := BindFlags(flags)
	if err != nil {
		klog.Fatalf("failed to get default artifacts || rundir directory: %s", err)
	}
}
