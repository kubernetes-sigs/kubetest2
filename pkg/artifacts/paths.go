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
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"k8s.io/klog"
)

var baseDir string

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

// BindFlags binds the artifact related flags.
func BindFlags(flags *pflag.FlagSet) error {
	defaultArtifacts, err := defaultArtifactsDir()
	if err != nil {
		return err
	}
	flags.StringVar(&baseDir, "artifacts", defaultArtifacts, `top-level directory to put artifacts under for each kubetest2 run, defaulting to "${ARTIFACTS:-./_artifacts}". If using the ginkgo tester, this must be an absolute path.`)
	return nil
}

// MustBindFlags calls BindFlags and panics on an error
func MustBindFlags(flags *pflag.FlagSet) {
	err := BindFlags(flags)
	if err != nil {
		klog.Fatalf("failed to get default artifacts directory: %s", err)
	}
}
