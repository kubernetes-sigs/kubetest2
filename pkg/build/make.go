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

package build

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"

	"k8s.io/klog"
	"sigs.k8s.io/kubetest2/pkg/exec"
)

type MakeBuilder struct{}

var _ Builder = &MakeBuilder{}

const (
	target = "bazel-release"
)

var (
	// This will need changed to support other platforms.
	tarballsToExtract = []string{
		"kubernetes.tar.gz",
		"kubernetes-test-linux-amd64.tar.gz",
		"kubernetes-test-portable.tar.gz",
		"kubernetes-client-linux-amd64.tar.gz",
	}
)

// Build builds kubernetes with the bazel-release make target
func (m *MakeBuilder) Build() (string, error) {
	// TODO(RonWeber): This needs options
	src, err := K8sDir("kubernetes")
	if err != nil {
		return "", err
	}
	version, err := sourceVersion(src)
	if err != nil {
		return "", fmt.Errorf("failed to get version: %v", err)
	}
	cmd := exec.Command("make", "-C", src, target)
	exec.InheritOutput(cmd)
	if err = cmd.Run(); err != nil {
		return "", err
	}
	return version, extractBuiltTars()
}

// K8sDir returns $GOPATH/src/k8s.io/...
func K8sDir(topdir string, parts ...string) (string, error) {
	gopathList := filepath.SplitList(build.Default.GOPATH)
	for _, gopath := range gopathList {
		kubedir := filepath.Join(gopath, "src", "k8s.io", topdir)
		if _, err := os.Stat(kubedir); !os.IsNotExist(err) {
			p := []string{kubedir}
			p = append(p, parts...)
			return filepath.Join(p...), nil
		}
	}
	return "", fmt.Errorf("could not find directory src/k8s.io/%s in GOPATH", topdir)
}

// TODO(RonWeber): This whole untarring and cd-ing logic is out of
// scope for Build().  It needs a better home.
func extractBuiltTars() error {
	allBuilds, err := K8sDir("kubernetes", "_output", "gcs-stage")
	if err != nil {
		return err
	}

	shouldExtract := make(map[string]bool)
	for _, name := range tarballsToExtract {
		shouldExtract[name] = true
	}

	err = filepath.Walk(allBuilds, func(path string, info os.FileInfo, err error) error { //Untar anything with the filename we want.
		if err != nil {
			return err
		}
		if shouldExtract[info.Name()] {
			klog.V(0).Infof("Extracting %s into current directory", path)
			//Extract it into current directory.
			cmd := exec.Command("tar", "-xzf", path)
			exec.InheritOutput(cmd)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("could not extract built tar archive: %v", err)
			}
			shouldExtract[info.Name()] = false
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("could not untar built archive: %v", err)
	}
	for n, s := range shouldExtract { // Make sure we found all the archives we were expecting.
		if s {
			return fmt.Errorf("expected built tarball was not present: %s", n)
		}
	}
	return nil
}
