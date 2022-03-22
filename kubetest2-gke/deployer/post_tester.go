/*
Copyright 2022 The Kubernetes Authors.

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

	util "sigs.k8s.io/kubetest2/kubetest2-gke/deployer/utils"
)

var (
	greenMarkerPrefix = "latest-green"
)

// PostTest will check if there's any error in the test. If there's no
// error in the test, and the --update-latest-green-marker is set to true,
// this method will stage the build marker to the GCS bucket.
func (d *Deployer) PostTest(testErr error) error {
	if testErr != nil || !d.BuildOptions.UpdateLatestGreenMarker {
		return nil
	}
	// If we are here, that means the new build passes the smoke test.
	if err := util.StageGKEBuildMarker(d.ClusterVersion, d.CommonBuildOptions.StageLocation, greenMarkerPrefix); err != nil {
		return fmt.Errorf("error during green build marker staging: %s", err)
	}
	return nil
}
