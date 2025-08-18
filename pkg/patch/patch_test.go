// Copyright 2021 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package patch

import (
	"reflect"
	"testing"
)

// mockTektonObject implements TektonObject interface for testing
type mockTektonObject struct {
	name      string
	namespace string
	gvk       string
}

func (m *mockTektonObject) GetName() string      { return m.name }
func (m *mockTektonObject) GetNamespace() string { return m.namespace }
func (m *mockTektonObject) GetGVK() string       { return m.gvk }

func TestGetAnnotationsPatch(t *testing.T) {
	mockObj := &mockTektonObject{
		name:      "test-taskrun",
		namespace: "test-namespace",
		gvk:       "tekton.dev/v1/TaskRun",
	}

	tests := []struct {
		name           string
		newAnnotations map[string]string
		want           string
		wantErr        bool
	}{
		{
			name:           "empty",
			newAnnotations: map[string]string{},
			want:           `{"apiVersion":"tekton.dev/v1","kind":"TaskRun","metadata":{"name":"test-taskrun","namespace":"test-namespace"}}`,
		},
		{
			name: "one",
			newAnnotations: map[string]string{
				"foo": "bar",
			},
			want: `{"apiVersion":"tekton.dev/v1","kind":"TaskRun","metadata":{"name":"test-taskrun","namespace":"test-namespace","annotations":{"foo":"bar"}}}`,
		},
		{
			name: "many",
			newAnnotations: map[string]string{
				"foo": "bar",
				"baz": "bat",
			},
			want: `{"apiVersion":"tekton.dev/v1","kind":"TaskRun","metadata":{"name":"test-taskrun","namespace":"test-namespace","annotations":{"baz":"bat","foo":"bar"}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAnnotationsPatch(tt.newAnnotations, mockObj)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAnnotationsPatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotStr := string(got)
			if !reflect.DeepEqual(gotStr, tt.want) {
				t.Errorf("GetAnnotationsPatch() = %v, want %v", gotStr, tt.want)
			}
		})
	}
}
