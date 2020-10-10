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

	"sigs.k8s.io/kubetest2/pkg/exec"
)

type ReleasePushBuild struct {
	Location string
}

var _ Stager = &ReleasePushBuild{}

// Stage stages the build to GCS using
// essentially release/push-build.sh --bucket=B --ci --gcs-suffix=S --noupdatelatest
func (rpb *ReleasePushBuild) Stage(version string) error {
	re := regexp.MustCompile(`^gs://([\w-]+)/(devel|ci)(/.*)?`)
	mat := re.FindStringSubmatch(rpb.Location)
	if mat == nil {
		return fmt.Errorf("invalid stage location: %v. Use gs://bucket/ci/optional-suffix", rpb.Location)
	}
	bucket := mat[1]
	ci := mat[2] == "ci"
	gcsSuffix := mat[3]

	args := []string{
		"--nomock",
		"--verbose",
		"--noupdatelatest",
		fmt.Sprintf("--bucket=%v", bucket),
	}
	if len(gcsSuffix) > 0 {
		args = append(args, fmt.Sprintf("--gcs-suffix=%v", gcsSuffix))
	}
	if ci {
		args = append(args, "--ci")
	}

	name, err := K8sDir("release", "push-build.sh")
	if err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	exec.InheritOutput(cmd)
	cmdDir, err := K8sDir("kubernetes")
	if err != nil {
		return err
	}
	cmd.SetDir(cmdDir)
	return cmd.Run()
}
