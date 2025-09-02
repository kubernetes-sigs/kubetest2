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
	"regexp"
	"strings"

	rbuild "k8s.io/release/pkg/build"
)

type Krel struct {
	StageLocation   string
	ImageLocation   string
	RepoRoot        string
	StageExtraFiles bool
	UpdateLatest    bool
}

var _ Stager = &Krel{}

// Stage stages the build to GCS using
// essentially release/push-build.sh --bucket=B --ci --gcs-suffix=S --noupdatelatest
func (rpb *Krel) Stage(version string) error {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	re := regexp.MustCompile(`^gs://([\w-]+)/(devel|ci)(/.*)?`)
	mat := re.FindStringSubmatch(rpb.StageLocation)
	if len(mat) < 4 {
		return fmt.Errorf("invalid stage location: %v. Use gs://<bucket>/<ci|devel>/<optional-suffix>", rpb.StageLocation)
	}

	if err := rbuild.NewInstance(&rbuild.Options{
		Bucket:          mat[1],
		GCSRoot:         mat[3],
		AllowDup:        true,
		CI:              mat[2] == "ci",
		NoUpdateLatest:  !rpb.UpdateLatest,
		Registry:        rpb.ImageLocation,
		Version:         version,
		StageExtraFiles: rpb.StageExtraFiles,
		RepoRoot:        rpb.RepoRoot,
	}).Push(); err != nil {

		return fmt.Errorf("stage via krel push: %w", err)
	}
	return nil
}
