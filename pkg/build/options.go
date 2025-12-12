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

package build

import (
	"fmt"
)

// ignore package name stutter
type BuildAndStageStrategy string //nolint:revive

const (
	// BazelStrategy builds and (optionally) stages using bazel
	BazelStrategy BuildAndStageStrategy = "bazel"
	// MakeStrategy builds using make and (optionally) stages using krel
	MakeStrategy BuildAndStageStrategy = "make"
)

type Options struct {
	Strategy           string `flag:"~strategy" desc:"Determines the build strategy to use either make or bazel."`
	StageLocation      string `flag:"~stage" desc:"Upload binaries to gs://bucket/ci/job-suffix if set"`
	RepoRoot           string `flag:"-"`
	ImageLocation      string `flag:"~image-location" desc:"Image registry where built images are stored."`
	StageExtraGCPFiles bool   `flag:"-"`
	VersionSuffix      string `flag:"-"`
	UpdateLatest       bool   `flag:"~update-latest" desc:"Whether should upload the build number to the GCS"`
	TargetBuildArch    string `flag:"~target-build-arch" desc:"Target architecture for the test artifacts for dockerized build"`
	Builder
	Stager
}

func (o *Options) Validate() error {
	return o.implementationFromStrategy()
}

func (o *Options) implementationFromStrategy() error {
	switch BuildAndStageStrategy(o.Strategy) {
	case BazelStrategy:
		bazel := &Bazel{
			RepoRoot:      o.RepoRoot,
			StageLocation: o.StageLocation,
			ImageLocation: o.ImageLocation,
		}
		o.Builder = bazel
		o.Stager = bazel
	case MakeStrategy:
		o.Builder = &MakeBuilder{
			RepoRoot:        o.RepoRoot,
			TargetBuildArch: o.TargetBuildArch,
		}
		o.Stager = &Krel{
			RepoRoot:        o.RepoRoot,
			StageLocation:   o.StageLocation,
			ImageLocation:   o.ImageLocation,
			StageExtraFiles: o.StageExtraGCPFiles,
			UpdateLatest:    o.UpdateLatest,
		}
	default:
		return fmt.Errorf("unknown build strategy: %v", o.Strategy)
	}
	return nil
}
