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
	"os"
	"path/filepath"

	"sigs.k8s.io/kubetest2/pkg/metadata"
)

func WriteVersionToMetadata(version string) error {
	var meta *metadata.CustomJSON
	// check existing metadata and initialize it if it exists
	metadataPath := filepath.Join(os.Getenv("KUBETEST2_RUN_DIR"), "metadata.json")
	if _, err := os.Stat(metadataPath); err == nil {
		metadataJSON, err := os.Open(metadataPath)
		if err != nil {
			return err
		}
		meta, err = metadata.NewCustomJSON(metadataJSON)
		if err != nil {
			return err
		}

		if err := metadataJSON.Sync(); err != nil {
			return err
		}
		if err := metadataJSON.Close(); err != nil {
			return err
		}
	} else {
		meta, err = metadata.NewCustomJSON(nil)
		if err != nil {
			return err
		}
	}

	if err := meta.Add("tester-version", version); err != nil {
		return err
	}

	metadataJSON, err := os.Create(metadataPath)
	if err != nil {
		return err
	}
	if err := meta.Write(metadataJSON); err != nil {
		return err
	}

	if err := metadataJSON.Sync(); err != nil {
		return err
	}
	if err := metadataJSON.Close(); err != nil {
		return err
	}
	return nil
}
