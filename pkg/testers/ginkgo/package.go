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
	"strings"

	"k8s.io/klog"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

// AcquireTestPackage obtains two test binaries and places them in $ARTIFACTS.
// The first is "ginkgo", the actual ginkgo executable.
// The second is "e2e.test", which contains kubernetes e2e test cases.
func (t *Tester) AcquireTestPackage() error {
	// first, get the name of the latest release (e.g. v1.20.0-alpha.0)
	if t.TestPackageVersion == "" {
		cmd := exec.Command(
			"gsutil",
			"cat",
			fmt.Sprintf("gs://%s/%s/latest.txt", t.TestPackageBucket, t.TestPackageDir),
		)
		lines, err := exec.OutputLines(cmd)
		if err != nil {
			return fmt.Errorf("failed to get latest release name: %s", err)
		}
		if len(lines) == 0 {
			return fmt.Errorf("getting latest release name had no output")
		}
		t.TestPackageVersion = lines[0]

		klog.V(1).Infof("Test package version was not specified. Defaulting to latest: %s", t.TestPackageVersion)
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

	return t.extractBinaries(downloadPath)
}

func (t *Tester) extractBinaries(downloadPath string) error {
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
	foundTestPackage := false
	// likewise for the actual ginkgo binary
	binaryPath := "kubernetes/test/bin/ginkgo"
	foundBinary := false
	for {
		// Put this check before any break condition so we don't
		// accidentally incorrectly error
		if foundTestPackage && foundBinary {
			return nil
		}

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

			foundTestPackage = true
		} else if header.Name == binaryPath {
			outFile, err := os.Create(binary)
			if err != nil {
				return fmt.Errorf("error creating file at %s: %s", binary, err)
			}
			defer outFile.Close()

			if err := outFile.Chmod(0700); err != nil {
				return fmt.Errorf("failed to make %s executable: %s", binary, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("error reading data from tar with header name %s: %s", header.Name, err)
			}

			foundBinary = true
		}
	}
	if !foundBinary && !foundTestPackage {
		return fmt.Errorf("failed to find %s or %s in %s", binaryPath, testPackagePath, downloadPath)
	}
	if !foundBinary {
		return fmt.Errorf("failed to find %s in %s", binaryPath, downloadPath)
	}
	return fmt.Errorf("failed to find %s in %s", testPackagePath, downloadPath)
}

// ensureReleaseTar checks if the kubernetes test tarball already exists
// and verifies the hashes
// else downloads it from GCS
func (t *Tester) ensureReleaseTar(downloadPath, releaseTar string) error {
	if _, err := os.Stat(downloadPath); err == nil {
		klog.V(0).Infof("Found existing tar at %v", downloadPath)
		if err := t.compareSHA(downloadPath, releaseTar); err == nil {
			klog.V(0).Infof("Validated hash for existing tar at %v", downloadPath)
			return nil
		}
		klog.Warning(err)
	}

	cmd := exec.Command("gsutil", "cp",
		fmt.Sprintf(
			"gs://%s/%s/%s/%s",
			t.TestPackageBucket,
			t.TestPackageDir,
			t.TestPackageVersion,
			releaseTar,
		),
		downloadPath,
	)
	exec.InheritOutput(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download release tar %s for release %s: %s", releaseTar, t.TestPackageVersion, err)
	}
	return nil
}

func (t *Tester) compareSHA(downloadPath string, releaseTar string) error {
	cmd := exec.Command("gsutil", "cat",
		fmt.Sprintf(
			"gs://%s/%s/%s/%s",
			t.TestPackageBucket,
			t.TestPackageDir,
			t.TestPackageVersion,
			releaseTar+".sha256",
		),
	)
	expectedSHABytes, err := exec.Output(cmd)
	if err != nil {
		return fmt.Errorf("failed to get sha256 for release tar %s for release %s: %s", releaseTar, t.TestPackageVersion, err)
	}
	expectedSHA := strings.TrimSuffix(string(expectedSHABytes), "\n")
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
