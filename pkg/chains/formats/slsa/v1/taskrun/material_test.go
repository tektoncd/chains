/*
Copyright 2022 The Tekton Authors

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

package taskrun

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestMaterialsWithTaskRunResults(t *testing.T) {
	// make sure this works with Git resources
	taskrun := `apiVersion: tekton.dev/v1beta1
kind: TaskRun
spec:
  taskSpec:
    resources:
      inputs:
      - name: repo
        type: git
status:
  taskResults:
  - name: CHAINS-GIT_COMMIT
    value: 50c56a48cfb3a5a80fa36ed91c739bdac8381cbe
  - name: CHAINS-GIT_URL
    value: https://github.com/GoogleContainerTools/distroless`

	var taskRun *v1beta1.TaskRun
	if err := yaml.Unmarshal([]byte(taskrun), &taskRun); err != nil {
		t.Fatal(err)
	}

	want := []slsa.ProvenanceMaterial{
		{
			URI: "git+https://github.com/GoogleContainerTools/distroless.git",
			Digest: slsa.DigestSet{
				"sha1": "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe",
			},
		},
	}

	got, err := materials(objects.NewTaskRunObject(taskRun), logtesting.TestLogger(t))
	if err != nil {
		t.Fatalf("Did not expect an error but got %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v got %v", want, got)
	}
}

func TestMaterials(t *testing.T) {
	tests := []struct {
		name    string
		taskRun *v1beta1.TaskRun
		want    []slsa.ProvenanceMaterial
	}{{
		name: "materials from pipeline resources",
		taskRun: &v1beta1.TaskRun{
			Spec: v1beta1.TaskRunSpec{
				Resources: &v1beta1.TaskRunResources{
					Inputs: []v1beta1.TaskResourceBinding{
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "nil-resource-spec",
							},
						}, {
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{
								Name: "repo",
								ResourceSpec: &v1alpha1.PipelineResourceSpec{
									Params: []v1alpha1.ResourceParam{
										{Name: "url", Value: "https://github.com/GoogleContainerTools/distroless"},
									},
									Type: v1alpha1.PipelineResourceTypeGit,
								},
							},
						},
					},
				},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					TaskRunResults: []v1beta1.TaskRunResult{
						{
							Name: "img1_input" + "-" + artifacts.ArtifactsInputsResultName,
							Value: *v1beta1.NewObject(map[string]string{
								"uri":    "gcr.io/foo/bar",
								"digest": digest3,
							}),
						},
					},
					ResourcesResult: []v1beta1.PipelineResourceResult{
						{
							ResourceName: "repo",
							Key:          "commit",
							Value:        "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe",
						}, {
							ResourceName: "repo",
							Key:          "url",
							Value:        "https://github.com/GoogleContainerTools/distroless",
						},
					},
				},
			},
		},
		want: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/foo/bar",
				Digest: slsa.DigestSet{
					"sha256": strings.TrimPrefix(digest3, "sha256:"),
				},
			},
			{
				URI: "git+https://github.com/GoogleContainerTools/distroless.git",
				Digest: slsa.DigestSet{
					"sha1": "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe",
				},
			},
		},
	}, {
		name: "materials from git results",
		taskRun: &v1beta1.TaskRun{
			Spec: v1beta1.TaskRunSpec{
				Params: []v1beta1.Param{{
					Name:  "CHAINS-GIT_COMMIT",
					Value: *v1beta1.NewStructuredValues("my-commit"),
				}, {
					Name:  "CHAINS-GIT_URL",
					Value: *v1beta1.NewStructuredValues("github.com/something"),
				}},
			},
		},
		want: []slsa.ProvenanceMaterial{
			{
				URI: "git+github.com/something.git",
				Digest: slsa.DigestSet{
					"sha1": "my-commit",
				},
			},
		},
	}, {
		name: "materials from step images",
		taskRun: &v1beta1.TaskRun{
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					Steps: []v1beta1.StepState{{
						Name:    "git-source-repo-jwqcl",
						ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
					}, {
						Name:    "git-source-repo-repeat-again-jwqcl",
						ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
					}, {
						Name:    "build",
						ImageID: "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
					}},
				},
			},
		},
		want: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: slsa.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			},
		},
	}, {
		name: "materials from step and sidecar images",
		taskRun: &v1beta1.TaskRun{
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					Steps: []v1beta1.StepState{{
						Name:    "git-source-repo-jwqcl",
						ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
					}, {
						Name:    "git-source-repo-repeat-again-jwqcl",
						ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
					}, {
						Name:    "build",
						ImageID: "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
					}},
					Sidecars: []v1beta1.SidecarState{{
						Name:    "sidecar-jwqcl",
						ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init@sha256:a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
					}},
				},
			},
		},
		want: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: slsa.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: slsa.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
	}}
	for _, tc := range tests {
		mat, err := materials(objects.NewTaskRunObject(tc.taskRun), logtesting.TestLogger(t))
		if err != nil {
			t.Fatalf("Did not expect an error but got %v", err)
		}
		if diff := cmp.Diff(tc.want, mat, test.OptSortMaterial); diff != "" {
			t.Errorf("Materials(): -want +got: %s", diff)
		}
	}
}

func TestAddStepImagesToMaterials(t *testing.T) {
	tests := []struct {
		name      string
		steps     []v1beta1.StepState
		want      []slsa.ProvenanceMaterial
		wantError error
	}{{
		name: "steps with proper imageID",
		steps: []v1beta1.StepState{{
			Name:    "git-source-repo-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}, {
			Name:    "git-source-repo-repeat-again-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}, {
			Name:    "build",
			ImageID: "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
		}},
		want: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: slsa.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			},
		},
	}, {
		name: "step with bad imageId - no uri",
		steps: []v1beta1.StepState{{
			Name:    "git-source-repo-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init-sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}},
		want:      []slsa.ProvenanceMaterial{{}},
		wantError: fmt.Errorf("expected imageID gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init-sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247 to be separable by @"),
	}, {
		name: "step with bad imageId - no digest",
		steps: []v1beta1.StepState{{
			Name:    "git-source-repo-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256-b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}},
		want:      []slsa.ProvenanceMaterial{{}},
		wantError: fmt.Errorf("expected imageID gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256-b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247 to be separable by @ and :"),
	}}
	for _, tc := range tests {
		var mat []slsa.ProvenanceMaterial
		if err := AddStepImagesToMaterials(tc.steps, &mat); err != nil {
			if err.Error() != tc.wantError.Error() {
				t.Fatalf("Expected error %v but got %v", tc.wantError, err)
			}
		}
		if tc.wantError == nil {
			if diff := cmp.Diff(tc.want, mat, test.OptSortMaterial); diff != "" {
				t.Errorf("materials(): -want +got: %s", diff)
			}
		}
	}
}

func TestAddSidecarImagesToMaterials(t *testing.T) {
	tests := []struct {
		name      string
		sidecars  []v1beta1.SidecarState
		want      []slsa.ProvenanceMaterial
		wantError error
	}{{
		name: "sidecars with proper imageID",
		sidecars: []v1beta1.SidecarState{{
			Name:    "git-source-repo-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}, {
			Name:    "git-source-repo-repeat-again-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}, {
			Name:    "build",
			ImageID: "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
		}},
		want: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: slsa.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			},
		},
	}, {
		name: "sidecars with bad imageId - no uri",
		sidecars: []v1beta1.SidecarState{{
			Name:    "git-source-repo-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init-sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}},
		want:      []slsa.ProvenanceMaterial{{}},
		wantError: fmt.Errorf("expected imageID gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init-sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247 to be separable by @"),
	}, {
		name: "sidecars with bad imageId - no digest",
		sidecars: []v1beta1.SidecarState{{
			Name:    "git-source-repo-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256-b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}},
		want:      []slsa.ProvenanceMaterial{{}},
		wantError: fmt.Errorf("expected imageID gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256-b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247 to be separable by @ and :"),
	}}
	for _, tc := range tests {
		var mat []slsa.ProvenanceMaterial
		if err := AddSidecarImagesToMaterials(tc.sidecars, &mat); err != nil {
			if err.Error() != tc.wantError.Error() {
				t.Fatalf("Expected error %v but got %v", tc.wantError, err)
			}
		}
		if tc.wantError == nil {
			if diff := cmp.Diff(tc.want, mat, test.OptSortMaterial); diff != "" {
				t.Errorf("materials(): -want +got: %s", diff)
			}
		}
	}
}

func TestAddImageIDToMaterials(t *testing.T) {
	tests := []struct {
		name      string
		imageID   string
		want      []slsa.ProvenanceMaterial
		wantError error
	}{{
		name:    "proper ImageID",
		imageID: "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
		want: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: slsa.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			},
		},
	}, {
		name:      "bad ImageID",
		imageID:   "badImageId",
		want:      []slsa.ProvenanceMaterial{},
		wantError: fmt.Errorf("expected imageID badImageId to be separable by @"),
	}}
	for _, tc := range tests {
		mat := []slsa.ProvenanceMaterial{}
		if err := AddImageIDToMaterials(tc.imageID, &mat); err != nil {
			if err.Error() != tc.wantError.Error() {
				t.Fatalf("Expected error %v but got %v", tc.wantError, err)
			}
		}
		if tc.wantError == nil {
			if diff := cmp.Diff(tc.want, mat, test.OptSortMaterial); diff != "" {
				t.Errorf("materials(): -want +got: %s", diff)
			}
		}
	}
}
