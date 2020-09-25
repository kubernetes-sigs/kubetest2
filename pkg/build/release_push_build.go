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

	"github.com/pkg/errors"
	rbuild "k8s.io/release/pkg/build"
)

type ReleasePushBuild struct {
	Location      string
	ImageLocation string
}

var _ Stager = &ReleasePushBuild{}

// Stage stages the build to GCS using
// essentially release/push-build.sh --bucket=B --ci --gcs-suffix=S --noupdatelatest
func (rpb *ReleasePushBuild) Stage(version string) error {
	re := regexp.MustCompile(`^gs://([\w-]+)/(devel|ci)(/.*)?`)
	mat := re.FindStringSubmatch(rpb.Location)
	if mat == nil || len(mat) < 4 {
		return fmt.Errorf("invalid stage location: %v. Use gs://bucket/ci/optional-suffix", rpb.Location)
	}

	return errors.Wrap(
		rbuild.NewInstance(&rbuild.Options{
			Bucket:         mat[1],
			GCSSuffix:      mat[3],
			AllowDup:       true,
			CI:             mat[2] == "ci",
			NoUpdateLatest: true,
			Registry:       rpb.ImageLocation,
			Version:        version,
		}).Push(),
		"stage via krel push",
	)
}
