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

package clusterloader2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestWriteVersionToMetadata(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
	}{
		{name: "server version", status: http.StatusOK},
		{name: "API server error", status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			const gitVersion = "v1.32.2-gke.100"
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != "/version" {
					t.Errorf("unexpected request path: %s", request.URL.Path)
					http.NotFound(writer, request)
					return
				}
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(test.status)
				if err := json.NewEncoder(writer).Encode(version.Info{GitVersion: gitVersion}); err != nil {
					t.Errorf("failed to write server version: %v", err)
				}
			}))
			defer server.Close()

			directory := t.TempDir()
			t.Setenv("ARTIFACTS", directory)
			t.Setenv("KUBECONFIG", filepath.Join(directory, "unused-kubeconfig"))
			kubeconfig := filepath.Join(directory, "kubeconfig")
			config := clientcmdapi.Config{
				Clusters:       map[string]*clientcmdapi.Cluster{"test": {Server: server.URL}},
				Contexts:       map[string]*clientcmdapi.Context{"test": {Cluster: "test"}},
				CurrentContext: "test",
			}
			if err := clientcmd.WriteToFile(config, kubeconfig); err != nil {
				t.Fatal(err)
			}

			tester := &Tester{KubeConfig: kubeconfig}
			err := tester.writeVersionToMetadata()
			metadataPath := filepath.Join(directory, "metadata.json")
			if test.status != http.StatusOK {
				if err == nil || !strings.Contains(err.Error(), "failed to get Kubernetes API server version") {
					t.Fatalf("expected server version error, got %v", err)
				}
				if _, err := os.Stat(metadataPath); !os.IsNotExist(err) {
					t.Fatalf("expected no metadata file, got %v", err)
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
			var metadata map[string]string
			if err := json.Unmarshal(data, &metadata); err != nil {
				t.Fatal(err)
			}
			if metadata["job-version"] != gitVersion {
				t.Errorf("job-version = %q, want %q", metadata["job-version"], gitVersion)
			}
			if actual, exists := metadata["tester-version"]; !exists || actual != GitTag {
				t.Errorf("tester-version = %q (present: %t), want %q", actual, exists, GitTag)
			}
		})
	}
}

func TestWriteVersionToMetadataInvalidKubeconfig(t *testing.T) {
	tester := &Tester{KubeConfig: filepath.Join(t.TempDir(), "missing")}
	if err := tester.writeVersionToMetadata(); err == nil || !strings.Contains(err.Error(), "failed to load kubeconfig") {
		t.Fatalf("expected kubeconfig error, got %v", err)
	}
}
