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

package artifacts

import (
	"reflect"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestOCIArtifact_ExtractObjects(t *testing.T) {

	tests := []struct {
		name string
		tr   *v1beta1.TaskRun
		want []interface{}
	}{
		{
			name: "one image",
			tr: &v1beta1.TaskRun{
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						ResourcesResult: []v1beta1.PipelineResourceResult{
							{
								ResourceName: "my-image",
								Key:          "url",
								Value:        "gcr.io/foo/bar",
							},
							{
								ResourceName: "my-image",
								Key:          "digest",
								Value:        "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5",
							},
						},
						TaskSpec: &v1beta1.TaskSpec{
							Resources: &v1beta1.TaskResources{
								Outputs: []v1beta1.TaskResource{
									{
										ResourceDeclaration: v1beta1.ResourceDeclaration{
											Name: "my-image",
											Type: "image",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []interface{}{digest(t, "gcr.io/foo/bar@sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
		},
		{
			name: "two images",
			tr: &v1beta1.TaskRun{
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						ResourcesResult: []v1beta1.PipelineResourceResult{
							{
								ResourceName: "my-image1",
								Key:          "url",
								Value:        "gcr.io/foo/bar",
							},
							{
								ResourceName: "my-image1",
								Key:          "digest",
								Value:        "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5",
							},
							{
								ResourceName: "my-image2",
								Key:          "url",
								Value:        "gcr.io/foo/baz",
							},
							{
								ResourceName: "my-image2",
								Key:          "digest",
								Value:        "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b6",
							},
						},
						TaskSpec: &v1beta1.TaskSpec{
							Resources: &v1beta1.TaskResources{
								Outputs: []v1beta1.TaskResource{
									{
										ResourceDeclaration: v1beta1.ResourceDeclaration{
											Name: "my-image1",
											Type: "image",
										},
									},
									{
										ResourceDeclaration: v1beta1.ResourceDeclaration{
											Name: "my-image2",
											Type: "image",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []interface{}{
				digest(t, "gcr.io/foo/bar@sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5"),
				digest(t, "gcr.io/foo/baz@sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b6"),
			},
		},
		{
			name: "extra",
			tr: &v1beta1.TaskRun{
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						ResourcesResult: []v1beta1.PipelineResourceResult{
							{
								ResourceName: "my-image",
								Key:          "url",
								Value:        "gcr.io/foo/bar",
							},
							{
								ResourceName: "my-image",
								Key:          "digest",
								Value:        "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5",
							},
							{
								ResourceName: "gibberish",
								Key:          "url",
								Value:        "gcr.io/foo/bar",
							},
							{
								ResourceName: "gobble-dygook",
								Key:          "digest",
								Value:        "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5",
							},
						},
						TaskSpec: &v1beta1.TaskSpec{
							Resources: &v1beta1.TaskResources{
								Outputs: []v1beta1.TaskResource{
									{
										ResourceDeclaration: v1beta1.ResourceDeclaration{
											Name: "my-image",
											Type: "image",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []interface{}{digest(t, "gcr.io/foo/bar@sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logtesting.TestLogger(t)
			oa := &OCIArtifact{
				Logger: logger,
			}
			if got := oa.ExtractObjects(tt.tr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OCIArtifact.ExtractObjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func digest(t *testing.T, dgst string) name.Digest {
	result, err := name.NewDigest(dgst)
	if err != nil {
		t.Fatal(err)
	}
	return result

}
