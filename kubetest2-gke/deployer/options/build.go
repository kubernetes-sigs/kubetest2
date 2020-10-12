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

package options

import (
	"fmt"

	"sigs.k8s.io/kubetest2/pkg/build"
)

type BuildOptions struct {
	Strategy      string `flag:"~strategy" desc:"Determines the build strategy to use either make or bazel."`
	StageLocation string `flag:"~stage" desc:"Upload binaries to gs://bucket/ci/job-suffix if set"`
	RepoRoot      string `flag:"-"`
	build.Builder
	build.Stager
}

func (bo *BuildOptions) Validate() error {
	return bo.implementationFromStrategy()
}

func (bo *BuildOptions) implementationFromStrategy() error {
	switch bo.Strategy {
	case "bazel":
		bazel := &build.Bazel{
			RepoRoot:      bo.RepoRoot,
			StageLocation: bo.StageLocation,
			ImageLocation: "gke.gcr.io",
		}
		bo.Builder = bazel
		bo.Stager = bazel
	case "make":
		bo.Builder = &build.MakeBuilder{}
		bo.Stager = &build.ReleasePushBuild{
			Location: bo.StageLocation,
		}
	default:
		return fmt.Errorf("unknown build strategy: %v", bo.Strategy)
	}
	return nil
}
