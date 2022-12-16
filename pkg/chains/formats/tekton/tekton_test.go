/*
Copyright 2020 The Tekton Authors
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

package tekton

import (
	"context"
	"reflect"
	"testing"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestTekton_CreatePayload(t *testing.T) {
	tests := []struct {
		name string
		tr   *v1beta1.TaskRun
	}{
		{
			name: "tr",
			tr: &v1beta1.TaskRun{
				Status: v1beta1.TaskRunStatus{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Tekton{}
			got, err := i.CreatePayload(context.Background(), objects.NewTaskRunObject(tt.tr))
			if err != nil {
				t.Errorf("Tekton.CreatePayload() error = %v", err)
				return
			}
			// This payloader just returns the taskrun unmodified.
			if !reflect.DeepEqual(got, tt.tr.Status) {
				t.Errorf("Tekton.CreatePayload() = %v, want %v", got, tt.tr)
			}
		})
	}
}
