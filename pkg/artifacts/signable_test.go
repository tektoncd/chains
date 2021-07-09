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
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	digest1 = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5"
	digest2 = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b6"
)

var ignore = []cmp.Option{cmpopts.IgnoreUnexported(name.Registry{}, name.Repository{}, name.Digest{})}

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
								Value:        digest1,
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
								Value:        digest1,
							},
							{
								ResourceName: "my-image2",
								Key:          "url",
								Value:        "gcr.io/foo/baz",
							},
							{
								ResourceName: "my-image2",
								Key:          "digest",
								Value:        digest2,
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
			name: "resource and result",
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
								Value:        digest1,
							},
						},
						TaskRunResults: []v1beta1.TaskRunResult{
							{
								Name:  "IMAGE_URL",
								Value: "gcr.io/foo/bat",
							},
							{
								Name:  "IMAGE_DIGEST",
								Value: "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b4",
							},
						},
						TaskSpec: &v1beta1.TaskSpec{
							Results: []v1beta1.TaskResult{
								{
									Name: "IMAGE_URL",
								},
								{
									Name: "IMAGE_DIGEST",
								},
							},
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
			want: []interface{}{
				digest(t, "gcr.io/foo/bat@sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b4"),
				digest(t, "gcr.io/foo/bar@sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
		},
		{
			name: "extra",
			tr: &v1beta1.TaskRun{
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskRunResults: []v1beta1.TaskRunResult{
							{
								Name:  "IMAGE_URL",
								Value: "foo",
							},
							{
								Name:  "gibberish",
								Value: "baz",
							},
						},
						ResourcesResult: []v1beta1.PipelineResourceResult{
							{
								ResourceName: "my-image",
								Key:          "url",
								Value:        "gcr.io/foo/bar",
							},
							{
								ResourceName: "my-image",
								Key:          "digest",
								Value:        digest1,
							},
							{
								ResourceName: "gibberish",
								Key:          "url",
								Value:        "gcr.io/foo/bar",
							},
							{
								ResourceName: "gobble-dygook",
								Key:          "digest",
								Value:        digest1,
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
			got := oa.ExtractObjects(tt.tr)
			sort.Slice(got, func(i, j int) bool {
				a := got[i].(name.Digest)
				b := got[j].(name.Digest)
				return a.DigestStr() < b.DigestStr()
			})
			if !cmp.Equal(got, tt.want, ignore...) {
				t.Errorf("OCIArtifact.ExtractObjects() = %s", cmp.Diff(got, tt.want, ignore...))
			}
		})
	}
}

func TestExtractOCIImagesFromResults(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				TaskRunResults: []v1beta1.TaskRunResult{
					{Name: "img1_IMAGE_URL", Value: "img1"},
					{Name: "img1_IMAGE_DIGEST", Value: digest1},
					{Name: "img2_IMAGE_URL", Value: "img2"},
					{Name: "img2_IMAGE_DIGEST", Value: digest2},
					{Name: "IMAGE_URL", Value: "img3"},
					{Name: "IMAGE_DIGEST", Value: digest1},
				},
			},
		},
	}
	want := []interface{}{
		digest(t, fmt.Sprintf("img1@%s", digest1)),
		digest(t, fmt.Sprintf("img2@%s", digest2)),
		digest(t, fmt.Sprintf("img3@%s", digest1)),
	}
	got := ExtractOCIImagesFromResults(tr, logtesting.TestLogger(t))
	sort.Slice(got, func(i, j int) bool {
		a := got[i].(name.Digest)
		b := got[j].(name.Digest)
		return a.String() < b.String()
	})
	if !cmp.Equal(got, want, ignore...) {
		t.Fatalf("not the same %s", cmp.Diff(got, want, ignore...))
	}
}

func digest(t *testing.T, dgst string) name.Digest {
	result, err := name.NewDigest(dgst)
	if err != nil {
		t.Fatal(err)
	}
	return result

}
