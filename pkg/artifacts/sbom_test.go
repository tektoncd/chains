/*
Copyright 2023 The Tekton Authors
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
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestSBOMArtifactExtractObjects(t *testing.T) {
	tests := []struct {
		name string
		obj  objects.TektonObject
		want []any
	}{
		{
			name: "one SBOM image",
			obj: objects.NewTaskRunObject(&v1beta1.TaskRun{ //nolint:staticcheck
				TypeMeta: metav1.TypeMeta{
					Kind: "TaskRun",
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskRunResults: []v1beta1.TaskRunResult{
							{
								Name:  "IMAGE_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:latest"),
							},
							{
								Name:  "IMAGE_DIGEST",
								Value: *v1beta1.NewArrayOrString(digest1),
							},
							{
								Name:  "IMAGE_SBOM_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:sbom@" + digest2),
							},
							{
								Name:  "IMAGE_SBOM_FORMAT",
								Value: *v1beta1.NewArrayOrString("https://cyclonedx.org/schema"),
							},
						},
					},
				},
			}),
			want: []any{
				objects.NewSBOMObject(
					"gcr.io/foo/bat:sbom@"+digest2,
					"https://cyclonedx.org/schema",
					"gcr.io/foo/bat:latest",
					digest1,
					nil,
				),
			},
		},
		{
			name: "multiple SBOM images",
			obj: objects.NewTaskRunObject(&v1beta1.TaskRun{ //nolint:staticcheck
				TypeMeta: metav1.TypeMeta{
					Kind: "TaskRun",
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskRunResults: []v1beta1.TaskRunResult{
							{
								Name:  "BAT_IMAGE_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:latest"),
							},
							{
								Name:  "BAT_IMAGE_DIGEST",
								Value: *v1beta1.NewArrayOrString(digest1),
							},
							{
								Name:  "BAT_IMAGE_SBOM_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:sbom@" + digest2),
							},
							{
								Name:  "BAT_IMAGE_SBOM_FORMAT",
								Value: *v1beta1.NewArrayOrString("https://cyclonedx.org/schema"),
							},
							{
								Name:  "BAZ_IMAGE_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/baz:latest"),
							},
							{
								Name:  "BAZ_IMAGE_DIGEST",
								Value: *v1beta1.NewArrayOrString(digest3),
							},
							{
								Name:  "BAZ_IMAGE_SBOM_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/baz:sbom@" + digest4),
							},
							{
								Name:  "BAZ_IMAGE_SBOM_FORMAT",
								Value: *v1beta1.NewArrayOrString("https://spdx.dev/Document"),
							},
						},
					},
				},
			}),
			want: []any{
				objects.NewSBOMObject(
					"gcr.io/foo/bat:sbom@"+digest2,
					"https://cyclonedx.org/schema",
					"gcr.io/foo/bat:latest",
					digest1,
					nil,
				),
				objects.NewSBOMObject(
					"gcr.io/foo/baz:sbom@"+digest4,
					"https://spdx.dev/Document",
					"gcr.io/foo/baz:latest",
					digest3,
					nil,
				),
			},
		},
		{
			name: "missing IMAGE_SBOM_FORMAT",
			obj: objects.NewTaskRunObject(&v1beta1.TaskRun{ //nolint:staticcheck
				TypeMeta: metav1.TypeMeta{
					Kind: "TaskRun",
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskRunResults: []v1beta1.TaskRunResult{{
							Name:  "IMAGE_URL",
							Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:latest"),
						},
							{
								Name:  "IMAGE_DIGEST",
								Value: *v1beta1.NewArrayOrString(digest1),
							},
							{
								Name:  "IMAGE_SBOM_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:sbom@" + digest2),
							},
						},
					},
				},
			}),
			want: nil,
		},
		{
			name: "missing IMAGE_URL",
			obj: objects.NewTaskRunObject(&v1beta1.TaskRun{ //nolint:staticcheck
				TypeMeta: metav1.TypeMeta{
					Kind: "TaskRun",
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskRunResults: []v1beta1.TaskRunResult{
							{
								Name:  "IMAGE_DIGEST",
								Value: *v1beta1.NewArrayOrString(digest1),
							},
							{
								Name:  "IMAGE_SBOM_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:sbom@" + digest2),
							},
							{
								Name:  "IMAGE_SBOM_FORMAT",
								Value: *v1beta1.NewArrayOrString("https://cyclonedx.org/schema"),
							},
						},
					},
				},
			}),
			want: nil,
		},
		{
			name: "missing IMAGE_DIGEST",
			obj: objects.NewTaskRunObject(&v1beta1.TaskRun{ //nolint:staticcheck
				TypeMeta: metav1.TypeMeta{
					Kind: "TaskRun",
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskRunResults: []v1beta1.TaskRunResult{
							{
								Name:  "IMAGE_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:latest"),
							},
							{
								Name:  "IMAGE_SBOM_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:sbom@" + digest2),
							},
							{
								Name:  "IMAGE_SBOM_FORMAT",
								Value: *v1beta1.NewArrayOrString("https://cyclonedx.org/schema"),
							},
						},
					},
				},
			}),
			want: nil,
		},
		{
			name: "missing IMAGE_SBOM_URL",
			obj: objects.NewTaskRunObject(&v1beta1.TaskRun{ //nolint:staticcheck
				TypeMeta: metav1.TypeMeta{
					Kind: "TaskRun",
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskRunResults: []v1beta1.TaskRunResult{
							{
								Name:  "IMAGE_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:latest"),
							},
							{
								Name:  "IMAGE_DIGEST",
								Value: *v1beta1.NewArrayOrString(digest1),
							},
							{
								Name:  "IMAGE_SBOM_FORMAT",
								Value: *v1beta1.NewArrayOrString("https://cyclonedx.org/schema"),
							},
						},
					},
				},
			}),
			want: nil,
		},
		{
			name: "non-pinned IMAGE_SBOM_URL ignored",
			obj: objects.NewTaskRunObject(&v1beta1.TaskRun{ //nolint:staticcheck
				TypeMeta: metav1.TypeMeta{
					Kind: "TaskRun",
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskRunResults: []v1beta1.TaskRunResult{
							{
								Name:  "IMAGE_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:latest"),
							},
							{
								Name:  "IMAGE_DIGEST",
								Value: *v1beta1.NewArrayOrString(digest1),
							},
							{
								Name:  "IMAGE_SBOM_URL",
								Value: *v1beta1.NewArrayOrString("gcr.io/foo/bat:sbom"),
							},
							{
								Name:  "IMAGE_SBOM_FORMAT",
								Value: *v1beta1.NewArrayOrString("https://cyclonedx.org/schema"),
							},
						},
					},
				},
			}),
			want: nil,
		},
	}

	transformer := cmp.Transformer("sort_sbom", func(in []any) []string {
		items := make([]string, 0, len(in))
		for _, item := range in {
			var itemString string
			sbomObject, ok := item.(*objects.SBOMObject)
			if ok {
				itemString = fmt.Sprintf(
					"ImageDigest=%q\nImageURL=%q\nSBOMFormat=%q\nSBOMURL=%q",
					sbomObject.GetImageDigest(), sbomObject.GetImageURL(), sbomObject.GetSBOMFormat(),
					sbomObject.GetSBOMURL())
			} else {
				// This shouldn't happen, but in case there's another []any value, perform a
				// generic conversion.
				itemString = fmt.Sprintf("%#v", item)
			}

			items = append(items, itemString)
		}
		sort.Strings(items)
		return items
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			sbomArtifact := &SBOMArtifact{}
			got := sbomArtifact.ExtractObjects(ctx, tt.obj)
			if !cmp.Equal(got, tt.want, transformer) {
				t.Errorf("SBOMArtifact.ExtractObjects() = %s", cmp.Diff(got, tt.want, transformer))
			}
		})
	}
}

func createDigest(t *testing.T, dgst string) name.Digest {
	result, err := name.NewDigest(dgst)
	if err != nil {
		t.Fatal(err)
	}
	return result

}
