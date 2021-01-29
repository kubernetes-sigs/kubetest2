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

type Options struct {
	Strategy           string `flag:"~strategy" desc:"Determines the build strategy to use either make or bazel."`
	StageLocation      string `flag:"~stage" desc:"Upload binaries to gs://bucket/ci/job-suffix if set"`
	RepoRoot           string `flag:"-"`
	ImageLocation      string `flag:"~image-location" desc:"Image registry where built images are stored."`
	StageExtraGCPFiles bool   `flag:"-"`
	Builder
	Stager
}

func (o *Options) Validate() error {
	return o.implementationFromStrategy()
}

func (o *Options) implementationFromStrategy() error {
	switch o.Strategy {
	case "bazel":
		bazel := &Bazel{
			RepoRoot:      o.RepoRoot,
			StageLocation: o.StageLocation,
			ImageLocation: o.ImageLocation,
		}
		o.Builder = bazel
		o.Stager = bazel
	case "make":
		o.Builder = &MakeBuilder{
			RepoRoot: o.RepoRoot,
		}
		o.Stager = &ReleasePushBuild{
			RepoRoot:        o.RepoRoot,
			StageLocation:   o.StageLocation,
			ImageLocation:   o.ImageLocation,
			StageExtraFiles: o.StageExtraGCPFiles,
		}
	default:
		return fmt.Errorf("unknown build strategy: %v", o.Strategy)
	}
	return nil
}
