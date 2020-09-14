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

package deployer

import (
	"fmt"
)

func (d *deployer) Build() error {
	if err := d.BuildOptions.Validate(); err != nil {
		return err
	}
	if err := d.BuildOptions.Build(); err != nil {
		return err
	}

	if d.BuildOptions.StageLocation != "" {
		if err := d.BuildOptions.Stage(); err != nil {
			return fmt.Errorf("error staging build: %v", err)
		}
	}
	return nil
}
