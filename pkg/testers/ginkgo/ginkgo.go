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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/octago/sflags/gen/gpflag"
	"k8s.io/klog"

	"sigs.k8s.io/kubetest2/pkg/exec"
)

const (
	binary = "ginkgo" // TODO(RonWeber): Actually find these binaries.
)

var (
	// This path is set up by AcquireTestPackage()
	e2eTestPath = filepath.Join(os.Getenv("ARTIFACTS"), "e2e.test")
)

type Tester struct {
	FlakeAttempts      int    `desc:"Make up to this many attempts to run each spec."`
	Parallel           int    `desc:"Run this many tests in parallel at once."`
	SkipRegex          string `desc:"Regular expression of jobs to skip."`
	FocusRegex         string `desc:"Regular expression of jobs to focus on."`
	TestPackageVersion string `desc:"The ginkgo tester uses a test package made during the kubernetes build. The tester downloads this test package from one of the release tars published to GCS. Defaults to latest. Use \"gsutil ls gs://kubernetes-release/release/\" to find release names. Example: v1.20.0-alpha.0"`
	TestPackageBucket  string `desc:"The bucket which release tars will be downloaded from to acquire the test package. Defaults to the main kubernetes project bucket."`

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
			log.Printf("Ginkgo tester received a non-absolute path for KUBECONFIG. Updating to: %s", newKubeconfig)
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
	log.Printf("Using kubeconfig at %s", t.kubeconfigPath)

	if err := t.AcquireTestPackage(); err != nil {
		return fmt.Errorf("failed to get ginkgo test package from published releases: %s", err)
	}

	return nil
}

// TODO(michaelmdresser): change behavior if a local built e2e.test package
// is available
func (t *Tester) AcquireTestPackage() error {
	releaseTar := fmt.Sprintf("kubernetes-test-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)

	downloadDir, err := ioutil.TempDir("", "kubetest2-ginkgo-download")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory for download: %s", err)
	}

	defer func(dir string) {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("failed to remove temp dir %s used for release tar download: %s", dir, err)
		}
	}(downloadDir)

	downloadPath := filepath.Join(downloadDir, releaseTar)

	// first, get the name of the latest release (e.g. v1.20.0-alpha.0)
	if t.TestPackageVersion == "" {
		cmd := exec.Command(
			"gsutil",
			"cat",
			fmt.Sprintf("gs://%s/release/latest.txt", t.TestPackageBucket),
		)
		lines, err := exec.OutputLines(cmd)
		if err != nil {
			return fmt.Errorf("failed to get latest release name: %s", err)
		}
		if len(lines) == 0 {
			return fmt.Errorf("getting latest release name had no output")
		}
		t.TestPackageVersion = lines[0]

		log.Printf("Test package version was not specified. Defaulting to latest: %s", t.TestPackageVersion)
	}

	// next, download the matching release tar

	cmd := exec.Command("gsutil", "cp",
		fmt.Sprintf(
			"gs://%s/release/%s/%s",
			t.TestPackageBucket,
			t.TestPackageVersion,
			releaseTar,
		),
		downloadPath,
	)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download release tar %s for release %s: %s", releaseTar, t.TestPackageVersion, err)
	}

	// finally, search for the test package and extract it

	f, err := os.Open(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded tar at %s: %s", downloadPath, err)
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %s", err)
	}

	tarReader := tar.NewReader(gzf)

	// this is the expected path of the package inside the tar
	// it will be extracted to e2eTestPath in the loop
	testPackagePath := "kubernetes/test/bin/e2e.test"

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error during tar read: %s", err)
		}

		if header.Name == testPackagePath {
			outFile, err := os.Create(e2eTestPath)
			if err != nil {
				return fmt.Errorf("error creating file at %s: %s", e2eTestPath, err)
			}
			defer outFile.Close()

			if err := outFile.Chmod(0700); err != nil {
				return fmt.Errorf("failed to make %s executable: %s", e2eTestPath, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("error reading data from tar with header name %s: %s", header.Name, err)
			}

			return nil
		}
	}

	return fmt.Errorf("failed to find %s in %s", testPackagePath, downloadPath)
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
