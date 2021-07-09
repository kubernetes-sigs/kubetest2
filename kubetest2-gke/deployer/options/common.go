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

package options

type CommonOptions struct {
	RepoRoot          string `desc:"Path to root of the kubernetes repo. Used with --build and for dumping cluster logs."`
	GCPServiceAccount string `flag:"~gcp-service-account" desc:"Service account to activate before using gcloud."`
	GCPSSHKeyIgnored  bool   `flag:"~ignore-gcp-ssh-key" desc:"Whether the GCP SSH key should be ignored or not for bringing up the cluster."`
}
