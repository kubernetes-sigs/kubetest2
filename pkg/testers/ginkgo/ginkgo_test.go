/*
Copyright The Kubernetes Authors.

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

package ginkgo

import "testing"

func TestAddMetadata(t *testing.T) {
	tester := &Tester{}
	tester.AddMetadata("kops-version", "1.36.0")
	tester.AddMetadata("variant", "base")
	tester.AddMetadata("kops-version", "1.36.1")

	if got := tester.extraMetadataValues["kops-version"]; got != "1.36.1" {
		t.Errorf("kops-version = %q, want %q", got, "1.36.1")
	}
	if got := tester.extraMetadataValues["variant"]; got != "base" {
		t.Errorf("variant = %q, want %q", got, "base")
	}
	if got := len(tester.extraMetadataValues); got != 2 {
		t.Errorf("metadata entries = %d, want 2", got)
	}
}
