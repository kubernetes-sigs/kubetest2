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

package deployer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/kubetest2/kubetest2-gke/deployer/options"
)

func TestMergeMetadata(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dumplogs-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("ARTIFACTS", tmpDir)
	defer os.Unsetenv("ARTIFACTS")

	metadataPath := filepath.Join(tmpDir, "metadata.json")
	initialMetadata := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	initialData, err := json.Marshal(initialMetadata)
	if err != nil {
		t.Fatalf("failed to marshal initial metadata: %v", err)
	}
	if err := os.WriteFile(metadataPath, initialData, 0644); err != nil {
		t.Fatalf("failed to write metadata.json: %v", err)
	}

	source1Path := filepath.Join(tmpDir, "source1.json")
	source1Metadata := map[string]string{
		"key2": "new-value2",
		"key3": "value3",
	}
	source1Data, err := json.Marshal(source1Metadata)
	if err != nil {
		t.Fatalf("failed to marshal source1 metadata: %v", err)
	}
	if err := os.WriteFile(source1Path, source1Data, 0644); err != nil {
		t.Fatalf("failed to write source1.json: %v", err)
	}

	source2Path := filepath.Join(tmpDir, "source2.json")
	source2Metadata := map[string]string{
		"key4": "value4",
	}
	source2Data, err := json.Marshal(source2Metadata)
	if err != nil {
		t.Fatalf("failed to marshal source2 metadata: %v", err)
	}
	if err := os.WriteFile(source2Path, source2Data, 0644); err != nil {
		t.Fatalf("failed to write source2.json: %v", err)
	}

	d := &Deployer{
		CommonOptions: &options.CommonOptions{
			MetadataSources: source1Path + "," + source2Path,
		},
	}

	if err := d.mergeMetadata(); err != nil {
		t.Fatalf("mergeMetadata failed: %v", err)
	}

	resultData, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("failed to read result metadata.json: %v", err)
	}

	var resultMetadata map[string]string
	if err := json.Unmarshal(resultData, &resultMetadata); err != nil {
		t.Fatalf("failed to unmarshal result metadata.json: %v", err)
	}

	expectedMetadata := map[string]string{
		"key1": "value1",
		"key2": "new-value2",
		"key3": "value3",
		"key4": "value4",
	}

	if len(resultMetadata) != len(expectedMetadata) {
		t.Errorf("expected %d keys, got %d", len(expectedMetadata), len(resultMetadata))
	}

	for k, v := range expectedMetadata {
		if resultMetadata[k] != v {
			t.Errorf("expected key %s to be %s, got %s", k, v, resultMetadata[k])
		}
	}
}
