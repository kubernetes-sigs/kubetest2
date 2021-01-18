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

import "fmt"

type UpOptions struct {
	NumClusters int `flag:"~num-clusters" desc:"Number of clusters to create, will auto-generate names as (kt2-<run-id>-<index>)"`
}

func (uo *UpOptions) Validate() error {
	// allow max 99 clusters (should be sufficient for most use cases)
	if uo.NumClusters < 1 || uo.NumClusters > 99 {
		return fmt.Errorf("need to specify between 1 and 99 clusters got %q", uo.NumClusters)
	}
	return nil
}
