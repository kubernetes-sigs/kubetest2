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
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCustomJSON_AddWrite(t *testing.T) {
	meta := CustomJSON{}
	if err := meta.Add("foo", "bar"); err != nil {
		t.Errorf("did not expect an error, but got: %v", err)
	}
	if err := meta.Add("baz", "qwe"); err != nil {
		t.Errorf("did not expect an error, but got: %v", err)
	}

	var buff bytes.Buffer
	if err := meta.Write(&buff); err != nil {
		t.Errorf("did not expect an error, but got: %v", err)
	}

	actualBytes := buff.Bytes()

	expectedBytes, err := json.Marshal(meta.data)
	if err != nil {
		t.Errorf("did not expect an error, but got: %v", err)
	}
	if !bytes.Equal(buff.Bytes(), expectedBytes) {
		t.Errorf("mismatched metadata bytes, got: %v, want: %v", actualBytes, expectedBytes)
	}
}

func TestNewCustomJSON(t *testing.T) {
	json := `{"baz":"qwe","foo":"bar"}`
	meta, err := NewCustomJSON(strings.NewReader(json))
	if err != nil {
		t.Errorf("did not expect an error, but got: %v", err)
	}
	expectedData := map[string]string{
		"foo": "bar",
		"baz": "qwe",
	}
	if !reflect.DeepEqual(meta.data, expectedData) {
		t.Errorf("mismatched metadata bytes, got: %v, want: %v", meta.data, expectedData)
	}
}
