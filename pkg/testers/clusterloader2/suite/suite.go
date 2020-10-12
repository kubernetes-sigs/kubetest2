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

package suite

type Suite struct {
	TestConfigs   []string
	TestOverrides []string
}

// GetSuite returns the default configurations for well-known testing setups.
func GetSuite(suite string) *Suite {
	const (
		load           = "load"
		density        = "density"
		nodeThroughput = "node-throughput"
	)

	var supportedSuites = map[string]*Suite{
		load: {
			TestConfigs: []string{
				"testing/load/config.yaml",
			},
		},

		density: {
			TestConfigs: []string{
				"testing/density/config.yaml",
			},
		},

		nodeThroughput: {
			TestConfigs: []string{
				"testing/node-throughput/config.yaml",
			},
		},
	}
	return supportedSuites[suite]
}
