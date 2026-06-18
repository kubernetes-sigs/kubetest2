/*
Copyright 2026 The Kubernetes Authors.

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

package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type fakeDeployer struct{}

func (fakeDeployer) Up() error              { return nil }
func (fakeDeployer) Down() error            { return nil }
func (fakeDeployer) IsUp() (bool, error)    { return false, nil }
func (fakeDeployer) DumpClusterLogs() error { return nil }
func (fakeDeployer) Build() error           { return nil }

func TestWriteVersionToMetadataJSONExtra(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ARTIFACTS", dir)

	if err := writeVersionToMetadataJSON(fakeDeployer{}, map[string]string{"variant": "restart"}); err != nil {
		t.Fatalf("writeVersionToMetadataJSON: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	if got["variant"] != "restart" {
		t.Errorf("variant not recorded, got %v", got)
	}
	if _, ok := got["kubetest-version"]; !ok {
		t.Errorf("kubetest-version not written, got %v", got)
	}
}
