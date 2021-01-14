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
	"sigs.k8s.io/kubetest2/pkg/build"
)

type BuildOptions struct {
	CommonBuildOptions *build.Options
}

var _ build.Builder = &BuildOptions{}
var _ build.Stager = &BuildOptions{}

func (bo *BuildOptions) Validate() error {
	// force extra GCP files to be staged
	bo.CommonBuildOptions.StageExtraGCPFiles = true
	return bo.CommonBuildOptions.Validate()
}

func (bo *BuildOptions) Build() (string, error) {
	return bo.CommonBuildOptions.Build()
}

func (bo *BuildOptions) Stage(version string) error {
	return bo.CommonBuildOptions.Stage(version)
}
