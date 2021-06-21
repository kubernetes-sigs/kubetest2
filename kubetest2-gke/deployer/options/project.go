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

type ProjectOptions struct {
	Projects []string `flag:"~project" desc:"Comma separated list of GCP Project(s) to use for creating the cluster."`

	BoskosLocation                 string `flag:"~boskos-location" desc:"If set, manually specifies the location of the Boskos server."`
	BoskosResourceType             string `flag:"~boskos-resource-type" desc:"If set, manually specifies the resource type of GCP projects to acquire from Boskos."`
	BoskosAcquireTimeoutSeconds    int    `flag:"~boskos-acquire-timeout-seconds" desc:"How long (in seconds) to hang on a request to Boskos to acquire a resource before erroring."`
	BoskosHeartbeatIntervalSeconds int    `flag:"~boskos-heartbeat-interval-seconds" desc:"How often (in seconds) to send a heartbeat to Boskos to hold the acquired resource. 0 means no heartbeat."`
	BoskosProjectsRequested        int    `flag:"~projects-requested" desc:"Number of projects to request from Boskos. It is only respected if projects is empty, and must be larger than zero."`
}
