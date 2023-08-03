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

package metadata

import (
	"encoding/json"
	"fmt"
	"io"
)

type CustomJSON struct {
	data map[string]string
}

func NewCustomJSON(from io.Reader) (*CustomJSON, error) {
	meta := &CustomJSON{}
	if from != nil {
		dataBytes, err := io.ReadAll(from)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(dataBytes, &meta.data); err != nil {
			return nil, err
		}
	}
	return meta, nil
}

func (m *CustomJSON) Add(key, value string) error {
	if m.data == nil {
		m.data = map[string]string{}
	}
	if _, exists := m.data[key]; exists {
		return fmt.Errorf("key %s already exists in the metadata", key)
	}
	m.data[key] = value
	return nil
}

func (m *CustomJSON) Write(writer io.Writer) error {
	data, err := json.Marshal(m.data)
	if err == nil {
		_, err = writer.Write(data)
	}
	return err
}
