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

	"k8s.io/klog/v2"

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

func (d *Deployer) Build() error {
	imageTag := defaultImageTag
	if d.BuildOptions.CommonBuildOptions.ImageLocation != "" {
		imageTag = d.BuildOptions.CommonBuildOptions.ImageLocation
	}

	klog.V(2).Infof("setting KUBE_DOCKER_REGISTRY to %s for tagging images", imageTag)
	if err := os.Setenv("KUBE_DOCKER_REGISTRY", imageTag); err != nil {
		return err
	}
	if err := d.VerifyBuildFlags(); err != nil {
		return err
	}
	version, err := d.BuildOptions.Build()
	if err != nil {
		return err
	}
	klog.V(2).Infof("got build version: %s", version)
	version = strings.TrimPrefix(version, "v")
	if version, err = normalizeVersion(version); err != nil {
		return err
	}

	// append the kubetest2 run id
	// avoid double + in the version
	// so they are valid docker tags
	if !strings.HasSuffix(version, d.Kubetest2CommonOptions.RunID()) {
		if strings.Contains(version, "+") {
			version += "-" + d.Kubetest2CommonOptions.RunID()
		} else {
			version += "+" + d.Kubetest2CommonOptions.RunID()
		}
	}

	// stage build if requested
	if d.BuildOptions.CommonBuildOptions.StageLocation != "" {
		if err := d.BuildOptions.Stage(version); err != nil {
			return fmt.Errorf("error staging build: %v", err)
		}
	}
	d.ClusterVersion = version
	build.StoreCommonBinaries(d.RepoRoot, d.Kubetest2CommonOptions.RunDir())
	return nil
}

func (d *Deployer) VerifyBuildFlags() error {
	if d.RepoRoot == "" {
		return fmt.Errorf("required repo-root when building from source")
	}
	d.BuildOptions.CommonBuildOptions.RepoRoot = d.RepoRoot
	if d.Kubetest2CommonOptions.ShouldBuild() && d.Kubetest2CommonOptions.ShouldUp() && d.BuildOptions.CommonBuildOptions.StageLocation == "" {
		return fmt.Errorf("creating a gke cluster from built sources requires staging them to a specific GCS bucket, use --stage=gs://<bucket>")
	}
	// force extra GCP files to be staged
	d.BuildOptions.CommonBuildOptions.StageExtraGCPFiles = true
	// add kubetest2 runid as the version suffix
	d.BuildOptions.CommonBuildOptions.VersionSuffix = d.Kubetest2CommonOptions.RunID()
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
