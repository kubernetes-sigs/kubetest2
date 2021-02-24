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

package deployer

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"k8s.io/klog"
	"sigs.k8s.io/kubetest2/pkg/build"
)

var (
	// (1.18.10-gke.12.34)(...)
	// $1 == prefix
	// $2 == suffix
	gkeCIBuildPrefix = regexp.MustCompile(`^(\d\.\d+\.\d+-gke\.\d+\.\d+)([+-].*)?$`)

	// (1.18.10-gke.1234)(...)
	// $1 == prefix
	// $2 == suffix
	gkeBuildPrefix = regexp.MustCompile(`^(\d\.\d+\.\d+-gke\.\d+)([+-].*)?$`)

	// (1.18.10)(...)
	// $1 == prefix
	// $2 == suffix
	buildPrefix = regexp.MustCompile(`^(\d\.\d+\.\d+)([+-].*)?$`)

	defaultImageTag = "gke.gcr.io"
)

func (d *deployer) Build() error {
	imageTag := defaultImageTag
	if d.BuildOptions.CommonBuildOptions.ImageLocation != "" {
		imageTag = d.BuildOptions.CommonBuildOptions.ImageLocation
	}

	klog.V(2).Infof("setting KUBE_DOCKER_REGISTRY to %s for tagging images", imageTag)
	if err := os.Setenv("KUBE_DOCKER_REGISTRY", imageTag); err != nil {
		return err
	}
	if err := d.verifyBuildFlags(); err != nil {
		return err
	}
	version, err := d.BuildOptions.Build()
	if err != nil {
		return err
	}
	version = strings.TrimPrefix(version, "v")
	if version, err = normalizeVersion(version); err != nil {
		return err
	}

	// append the kubetest2 run id
	// avoid double + in the version
	// so they are valid docker tags
	if strings.Contains(version, "+") {
		version += "-" + d.commonOptions.RunID()
	} else {
		version += "+" + d.commonOptions.RunID()
	}

	// stage build if requested
	if d.BuildOptions.CommonBuildOptions.StageLocation != "" {
		if err := d.BuildOptions.Stage(version); err != nil {
			return fmt.Errorf("error staging build: %v", err)
		}
	}
	d.Version = version
	build.StoreCommonBinaries(d.RepoRoot, d.commonOptions.RunDir())
	return nil
}

func (d *deployer) verifyBuildFlags() error {
	if d.RepoRoot == "" {
		return fmt.Errorf("required repo-root when building from source")
	}
	d.BuildOptions.CommonBuildOptions.RepoRoot = d.RepoRoot
	if d.commonOptions.ShouldBuild() && d.commonOptions.ShouldUp() && d.BuildOptions.CommonBuildOptions.StageLocation == "" {
		return fmt.Errorf("creating a gke cluster from built sources requires staging them to a specific GCS bucket, use --stage=gs://<bucket>")
	}
	// force extra GCP files to be staged
	d.BuildOptions.CommonBuildOptions.StageExtraGCPFiles = true
	return d.BuildOptions.Validate()
}

// ensure that the version is a valid gke version
func normalizeVersion(version string) (string, error) {

	finalVersion := ""
	if matches := gkeCIBuildPrefix.FindStringSubmatch(version); matches != nil {
		// prefix is usable as-is
		finalVersion = matches[1]
		// preserve suffix if present
		if suffix := strings.TrimLeft(matches[2], "+-"); len(suffix) > 0 {
			finalVersion += "+" + suffix
		}
	} else if matches := gkeBuildPrefix.FindStringSubmatch(version); matches != nil {
		// prefix needs .0 appended
		finalVersion = matches[1] + ".0"
		// preserve suffix if present
		if suffix := strings.TrimLeft(matches[2], "+-"); len(suffix) > 0 {
			finalVersion += "+" + suffix
		}
	} else if matches := buildPrefix.FindStringSubmatch(version); matches != nil {
		// prefix needs -gke.99.0 appended
		finalVersion = matches[1] + "-gke.99.0"
		// preserve suffix if present
		if suffix := strings.TrimLeft(matches[2], "+-"); len(suffix) > 0 {
			finalVersion += "+" + suffix
		}
	} else {
		return "", fmt.Errorf("could not construct version from %s", version)
	}

	if finalVersion != version {
		klog.V(2).Infof("modified version %q to %q", version, finalVersion)
	}
	return finalVersion, nil
}
