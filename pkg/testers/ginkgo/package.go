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

package ginkgo

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-resty/resty/v2"
	"k8s.io/klog/v2"
	"sigs.k8s.io/kubetest2/pkg/artifacts"
)

// AcquireTestPackage obtains three test binaries and places them in $KUBETEST2_RUN_DIR.
// The first is "ginkgo", the actual ginkgo executable.
// The second is "e2e.test", which contains kubernetes e2e test cases.
// The third is "kubectl".
func (t *Tester) AcquireTestPackage() error {
	// first, get the name of the latest release (e.g. v1.20.0-alpha.0)
	if t.TestPackageVersion == "" {
		resp, err := resty.New().R().Get(fmt.Sprintf("%s/%s/%s", t.TestPackageURL, t.TestPackageDir, t.TestPackageMarker))
		if err != nil {
			return fmt.Errorf("failed to get latest release name: %s", err)
		}
		if resp.String() == "" {
			return fmt.Errorf("getting latest release name had no output")
		}
		t.TestPackageVersion = resp.String()

		klog.V(1).Infof("Test package version was not specified. Defaulting to version from %s: %s", t.TestPackageMarker, t.TestPackageVersion)
	}

	releaseTar := fmt.Sprintf("kubernetes-test-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)

	downloadDir, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get user cache directory: %v", err)
	}

	downloadPath := filepath.Join(downloadDir, releaseTar)

	if err := t.ensureReleaseTar(downloadPath, releaseTar); err != nil {
		return err
	}
	if err := t.extractBinaries(downloadPath); err != nil {
		return err
	}

	t.kubectlPath = filepath.Join(artifacts.RunDir(), "kubectl")
	return t.ensureKubectl(t.kubectlPath)
}

func (t *Tester) extractBinaries(downloadPath string) error {
	// ensure the artifacts dir
	if err := os.MkdirAll(artifacts.BaseDir(), os.ModePerm); err != nil {
		return err
	}
	// ensure the rundir
	if err := os.MkdirAll(artifacts.RunDir(), os.ModePerm); err != nil {
		return err
	}

	// Extract files from the test package
	f, err := os.Open(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to open downloaded tar at %s: %s", downloadPath, err)
	}
	defer f.Close()
	gzf, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %s", err)
	}
	defer gzf.Close()

	tarReader := tar.NewReader(gzf)

	// Map of paths in archive to destination paths
	t.e2eTestPath = filepath.Join(artifacts.RunDir(), "e2e.test")
	t.ginkgoPath = filepath.Join(artifacts.RunDir(), "ginkgo")
	extract := map[string]string{
		"kubernetes/test/bin/e2e.test": t.e2eTestPath,
		"kubernetes/test/bin/ginkgo":   t.ginkgoPath,
	}
	extracted := map[string]bool{}

	for {
		if len(extracted) == len(extract) {
			break
		}

		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error during tar read: %s", err)
		}

		if dest := extract[header.Name]; dest != "" {
			outFile, err := os.Create(dest)
			if err != nil {
				return fmt.Errorf("error creating file at %s: %s", dest, err)
			}
			defer outFile.Close()

			if err := outFile.Chmod(0700); err != nil {
				return fmt.Errorf("failed to make %s executable: %s", dest, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("error reading data from tar with header name %s: %s", header.Name, err)
			}

			extracted[header.Name] = true
		}
	}
	for k := range extract {
		if !extracted[k] {
			return fmt.Errorf("failed to find %s in %s", k, downloadPath)
		}
	}
	return nil
}

// ensureKubectl checks if the kubectl exists and verifies the hashes
// else downloads it from GCS
func (t *Tester) ensureKubectl(downloadPath string) error {

	kubectlPathInURL := fmt.Sprintf(
		"%s/%s/%s/bin/%s/%s/kubectl",
		t.TestPackageURL,
		t.TestPackageDir,
		t.TestPackageVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
	if _, err := os.Stat(downloadPath); err == nil {
		klog.V(0).Infof("Found existing kubectl at %v", downloadPath)
		err := t.compareSHA(downloadPath, kubectlPathInURL)
		if err == nil {
			klog.V(0).Infof("Validated hash for existing kubectl at %v", downloadPath)
			return nil
		}
		klog.Warning(err)
	}

	_, err := resty.New().R().SetOutput(downloadPath).Get(kubectlPathInURL)
	if err != nil {
		return fmt.Errorf("failed to download kubectl for release %s: %s", t.TestPackageVersion, err)
	}
	if err := os.Chmod(downloadPath, 0700); err != nil {
		return fmt.Errorf("failed to make %s executable: %s", downloadPath, err)
	}
	return nil
}

// ensureReleaseTar checks if the kubernetes test tarball already exists
// and verifies the hashes
// else downloads it from GCS
func (t *Tester) ensureReleaseTar(downloadPath, releaseTar string) error {

	releaseTarPathInURL := fmt.Sprintf(
		"%s/%s/%s/%s",
		t.TestPackageURL,
		t.TestPackageDir,
		t.TestPackageVersion,
		releaseTar,
	)

	if _, err := os.Stat(downloadPath); err == nil {
		klog.V(0).Infof("Found existing tar at %v", downloadPath)
		err := t.compareSHA(downloadPath, releaseTarPathInURL)
		if err == nil {
			klog.V(0).Infof("Validated hash for existing tar at %v", downloadPath)
			return nil
		}
		klog.Warning(err)
	}

	klog.V(0).Infof("Downloading test tar ball from: %s", releaseTarPathInURL)
	_, err := resty.New().R().SetOutput(downloadPath).Get(releaseTarPathInURL)
	if err != nil {
		return fmt.Errorf("failed to download release tar %s for release %s: %s", releaseTar, t.TestPackageVersion, err)
	}
	return nil
}

func (t *Tester) compareSHA(downloadPath string, gcsFilePath string) error {
	resp, err := resty.New().R().Get(fmt.Sprintf("%s.sha256", gcsFilePath))
	if err != nil {
		return fmt.Errorf("failed to get sha256 for file %s for release %s: %s", gcsFilePath, t.TestPackageVersion, err)
	}
	expectedSHA := resp.String()
	actualSHA, err := sha256sum(downloadPath)
	if err != nil {
		return fmt.Errorf("failed to compute sha256 for %q: %v", downloadPath, err)
	}
	if actualSHA != expectedSHA {
		return fmt.Errorf("sha256 does not match")
	}
	return nil
}

func sha256sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
