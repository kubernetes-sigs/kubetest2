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

package testers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteVersionToMetadata(t *testing.T) {
	for _, test := range []struct {
		name     string
		existing map[string]string
		extra    map[string]string
		want     map[string]string
		wantErr  string
	}{
		{
			name: "nil extras",
			want: map[string]string{"tester-version": "tester", "job-version": "job"},
		},
		{
			name:     "preserve existing metadata",
			existing: map[string]string{"deployer-version": "deployer", "custom": "old"},
			extra:    map[string]string{"additional": "value"},
			want: map[string]string{
				"tester-version": "tester", "job-version": "job",
				"deployer-version": "deployer", "custom": "old", "additional": "value",
			},
		},
		{
			name:    "extras cannot replace version fields",
			extra:   map[string]string{"tester-version": "custom-tester"},
			wantErr: "key tester-version already exists in the metadata",
		},
		{
			name:     "extras cannot replace existing metadata",
			existing: map[string]string{"custom": "old"},
			extra:    map[string]string{"custom": "new"},
			wantErr:  "key custom already exists in the metadata",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			t.Setenv("ARTIFACTS", directory)
			metadataPath := filepath.Join(directory, "metadata.json")
			if test.existing != nil {
				data, err := json.Marshal(test.existing)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(metadataPath, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			err := WriteVersionToMetadata("tester", "job", test.extra)
			if test.wantErr != "" {
				if err == nil || err.Error() != test.wantErr {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(metadataPath)
			if err != nil {
				t.Fatal(err)
			}
			var actual map[string]string
			if err := json.Unmarshal(data, &actual); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, test.want) {
				t.Errorf("metadata = %v, want %v", actual, test.want)
			}
		})
	}
}
