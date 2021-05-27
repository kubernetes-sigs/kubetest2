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
	RepoRoot string `desc:"Path to root of the kubernetes repo. Used with --build and for dumping cluster logs."`

	Environment       string   `flag:"~environment" desc:"Container API endpoint to use, one of 'test', 'staging', 'prod', or a custom https:// URL. Defaults to prod if not provided"`
	Projects          []string `flag:"~project" desc:"Comma separated list of GCP Project(s) to use for creating the cluster."`
	Clusters          []string `flag:"~cluster-name" desc:"Cluster names separated by comma. Must be set. For multi-project profile, it should be in the format of clusterA:0,clusterB:1,clusterC:2, where the index means the index of the project."`
	Regions           []string `flag:"~region" desc:"Comma separated list for use with gcloud commands to specify the cluster region(s). The first region will be considered the primary region, and the rest will be considered the backup regions."`
	Zones             []string `flag:"~zone" desc:"Comma separated list for use with gcloud commands to specify the cluster zone(s). The first zone will be considered the primary zone, and the rest will be considered the backup zones."`
	Network           string   `flag:"~network" desc:"Cluster network. Defaults to the default network if not provided. For multi-project use cases, this will be the Shared VPC network name."`
	GCPServiceAccount string   `flag:"~gcp-service-account" desc:"Service account to activate before using gcloud."`
}
