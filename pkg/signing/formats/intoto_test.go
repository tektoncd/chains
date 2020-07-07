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

package formats

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestInToto_CreatePayload(t *testing.T) {
	tests := []struct {
		name string
		tr   *v1beta1.TaskRun
		want interface{}
	}{
		{
			name: "single input",
			tr:   newBuilder().addInput("my-input").addResult("commit", "foo", "my-input").TaskRun,
			want: in_toto.Link{
				Type: "_link",
				Materials: map[string]interface{}{
					"my-input": map[string]string{
						"commit": "foo",
					},
				},
				Products: map[string]interface{}{},
			},
		},
		{
			name: "multi input",
			tr:   newBuilder().addInput("my-input").addResult("commit", "foo", "my-input").addInput("my-other-input").addResult("digest", "baz", "my-other-input").TaskRun,
			want: in_toto.Link{
				Type: "_link",
				Materials: map[string]interface{}{
					"my-input": map[string]string{
						"commit": "foo",
					},
					"my-other-input": map[string]string{
						"digest": "baz",
					},
				},
				Products: map[string]interface{}{},
			},
		},
		{
			name: "extra results",
			tr:   newBuilder().addInput("my-input").addResult("commit", "foo", "my-input").addResult("digest", "baz", "my-other-input").TaskRun,
			want: in_toto.Link{
				Type: "_link",
				Materials: map[string]interface{}{
					"my-input": map[string]string{
						"commit": "foo",
					},
				},
				Products: map[string]interface{}{},
			},
		},
		{
			name: "inputs and outputs",
			tr:   newBuilder().addInput("my-input").addResult("commit", "foo", "my-input").addOutput("my-output").addResult("digest", "baz", "my-output").TaskRun,
			want: in_toto.Link{
				Type: "_link",
				Materials: map[string]interface{}{
					"my-input": map[string]string{
						"commit": "foo",
					},
				},
				Products: map[string]interface{}{
					"my-output": map[string]string{
						"digest": "baz",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &InToto{}
			got, err := i.CreatePayload(tt.tr)

			// Every test case should have "tekton" field of the link file set.
			link, ok := tt.want.(in_toto.Link)
			if !ok {
				t.Error("error casting interface to link file")
			}
			if link.Environment == nil {
				link.Environment = map[string]interface{}{}
			}

			link.Environment["tekton"] = tt.tr.Status
			if err != nil {
				t.Errorf("InToto.CreatePayload() error = %v", err)
				return
			}
			if diff := cmp.Diff(got, link); diff != "" {
				t.Errorf("InToto.CreatePayload(): -want +got: %s", diff)
			}
		})
	}
}

type trBuilder struct {
	*v1beta1.TaskRun
}

func newBuilder() *trBuilder {
	return &trBuilder{
		TaskRun: &v1beta1.TaskRun{},
	}
}

func (trb *trBuilder) addInput(name string) *trBuilder {
	if trb.Spec.Resources == nil {
		trb.Spec.Resources = &v1beta1.TaskRunResources{}
	}
	trb.Spec.Resources.Inputs = append(trb.Spec.Resources.Inputs, v1beta1.TaskResourceBinding{
		PipelineResourceBinding: v1beta1.PipelineResourceBinding{
			Name: name,
		},
	})
	return trb
}

func (trb *trBuilder) addOutput(name string) *trBuilder {
	if trb.Spec.Resources == nil {
		trb.Spec.Resources = &v1beta1.TaskRunResources{}
	}
	trb.Spec.Resources.Outputs = append(trb.Spec.Resources.Outputs, v1beta1.TaskResourceBinding{
		PipelineResourceBinding: v1beta1.PipelineResourceBinding{
			Name: name,
		},
	})
	return trb
}

func (trb *trBuilder) addResult(k, v, n string) *trBuilder {
	trb.Status.ResourcesResult = append(trb.Status.ResourcesResult, v1beta1.PipelineResourceResult{
		Key:          k,
		Value:        v,
		ResourceName: n,
	})
	return trb
}
