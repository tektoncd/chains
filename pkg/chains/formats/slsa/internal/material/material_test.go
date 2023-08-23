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

package material

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	"github.com/tektoncd/chains/internal/backport"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/compare"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	logtesting "knative.dev/pkg/logging/testing"
	"sigs.k8s.io/yaml"
)

const digest = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7"

func createPro(path string) *objects.PipelineRunObject {
	var err error
	pr, err := objectloader.PipelineRunFromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunFromFile("../../testdata/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunFromFile("../../testdata/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObject(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

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

	want := []common.ProvenanceMaterial{
		{
			URI: artifacts.GitSchemePrefix + "https://github.com/GoogleContainerTools/distroless.git",
			Digest: common.DigestSet{
				"sha1": "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe",
			},
		},
	}

	ctx := logtesting.TestContextWithLogger(t)
	got, err := TaskMaterials(ctx, objects.NewTaskRunObject(taskRun))
	if err != nil {
		t.Fatalf("Did not expect an error but got %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v got %v", want, got)
	}
}

func TestTaskMaterials(t *testing.T) {
	tests := []struct {
		name    string
		taskRun *v1beta1.TaskRun
		want    []common.ProvenanceMaterial
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
										{Name: "revision", Value: "my-revision"},
									},
									Type: backport.PipelineResourceTypeGit,
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
								"digest": digest,
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
		want: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/foo/bar",
				Digest: common.DigestSet{
					"sha256": strings.TrimPrefix(digest, "sha256:"),
				},
			},
			{
				URI: artifacts.GitSchemePrefix + "https://github.com/GoogleContainerTools/distroless.git@my-revision",
				Digest: common.DigestSet{
					"sha1": "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe",
				},
			},
		},
	}, {
		name: "materials from git results in task run spec",
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
		want: []common.ProvenanceMaterial{
			{
				URI: artifacts.GitSchemePrefix + "github.com/something.git",
				Digest: common.DigestSet{
					"sha1": "my-commit",
				},
			},
		},
	}, {
		name: "materials from git results in task spec",
		taskRun: &v1beta1.TaskRun{
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					TaskSpec: &v1beta1.TaskSpec{
						Params: []v1beta1.ParamSpec{{
							Name: "CHAINS-GIT_COMMIT",
							Default: &v1beta1.ParamValue{
								StringVal: "my-commit",
							},
						}, {
							Name: "CHAINS-GIT_URL",
							Default: &v1beta1.ParamValue{
								StringVal: "github.com/something",
							},
						}},
					},
				},
			},
		},
		want: []common.ProvenanceMaterial{
			{
				URI: artifacts.GitSchemePrefix + "github.com/something.git",
				Digest: common.DigestSet{
					"sha1": "my-commit",
				},
			},
		},
	}, {
		name: "materials from git results in task spec and taskrun spec",
		taskRun: &v1beta1.TaskRun{
			Spec: v1beta1.TaskRunSpec{
				Params: []v1beta1.Param{{
					Name: "CHAINS-GIT_URL",
					Value: v1beta1.ParamValue{
						StringVal: "github.com/something",
					},
				}},
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					TaskSpec: &v1beta1.TaskSpec{
						Params: []v1beta1.ParamSpec{{
							Name: "CHAINS-GIT_URL",
						}, {
							Name: "CHAINS-GIT_COMMIT",
							Default: &v1beta1.ParamValue{
								StringVal: "my-commit",
							},
						}},
					},
				},
			},
		},
		want: []common.ProvenanceMaterial{{
			URI: "git+github.com/something.git",
			Digest: common.DigestSet{
				"sha1": "my-commit",
			},
		}},
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
		want: []common.ProvenanceMaterial{
			{
				URI: artifacts.OCIScheme + "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: artifacts.OCIScheme + "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
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
		want: []common.ProvenanceMaterial{
			{
				URI: artifacts.OCIScheme + "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: artifacts.OCIScheme + "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: artifacts.OCIScheme + "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			mat, err := TaskMaterials(ctx, objects.NewTaskRunObject(tc.taskRun))
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}
			if diff := cmp.Diff(tc.want, mat); diff != "" {
				t.Errorf("Materials(): -want +got: %s", diff)
			}
		})
	}
}

func TestPipelineMaterials(t *testing.T) {
	expected := []common.ProvenanceMaterial{
		{URI: "github.com/test", Digest: common.DigestSet{"sha1": "28b123"}},
		{
			URI:    artifacts.OCIScheme + "gcr.io/test1/test1",
			Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
		},
		{URI: "github.com/catalog", Digest: common.DigestSet{"sha1": "x123"}},
		{
			URI:    artifacts.OCIScheme + "gcr.io/test2/test2",
			Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
		},
		{
			URI:    artifacts.OCIScheme + "gcr.io/test3/test3",
			Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
		},
		{URI: "github.com/test", Digest: common.DigestSet{"sha1": "ab123"}},
		{URI: "abc", Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}},
		{URI: artifacts.GitSchemePrefix + "https://git.test.com.git", Digest: common.DigestSet{"sha1": "abcd"}},
	}
	ctx := logtesting.TestContextWithLogger(t)
	got, err := PipelineMaterials(ctx, createPro("../../testdata/pipelinerun1.json"))
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(expected, got, compare.MaterialsCompareOption()); diff != "" {
		t.Errorf("Materials(): -want +got: %s", diff)
	}
}

func TestStructuredResultPipelineMaterials(t *testing.T) {
	want := []common.ProvenanceMaterial{
		{URI: "github.com/test", Digest: common.DigestSet{"sha1": "28b123"}},
		{
			URI:    artifacts.OCIScheme + "gcr.io/test1/test1",
			Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
		},
		{URI: "github.com/catalog", Digest: common.DigestSet{"sha1": "x123"}},
		{
			URI:    artifacts.OCIScheme + "gcr.io/test2/test2",
			Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
		},
		{
			URI:    artifacts.OCIScheme + "gcr.io/test3/test3",
			Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
		},
		{URI: "github.com/test", Digest: common.DigestSet{"sha1": "ab123"}},
		{
			URI: "abcd",
			Digest: common.DigestSet{
				"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
			},
		},
	}
	ctx := logtesting.TestContextWithLogger(t)
	got, err := PipelineMaterials(ctx, createPro("../../testdata/pipelinerun_structured_results.json"))
	if err != nil {
		t.Errorf("error while extracting materials: %v", err)
	}
	if diff := cmp.Diff(want, got, compare.MaterialsCompareOption()); diff != "" {
		t.Errorf("materials(): -want +got: %s", diff)
	}
}

func TestFromStepImages(t *testing.T) {
	tests := []struct {
		name      string
		steps     []v1beta1.StepState
		want      []common.ProvenanceMaterial
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
		want: []common.ProvenanceMaterial{
			{
				URI: artifacts.OCIScheme + "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: artifacts.OCIScheme + "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: artifacts.OCIScheme + "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
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
		want:      []common.ProvenanceMaterial{{}},
		wantError: fmt.Errorf("expected imageID gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init-sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247 to be separable by @"),
	}, {
		name: "step with bad imageId - no digest",
		steps: []v1beta1.StepState{{
			Name:    "git-source-repo-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256-b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}},
		want:      []common.ProvenanceMaterial{{}},
		wantError: fmt.Errorf("expected imageID gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256-b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247 to be separable by @ and :"),
	}}
	for _, tc := range tests {
		mat, err := FromStepImages(tc.steps)
		if err != nil {
			if err.Error() != tc.wantError.Error() {
				t.Fatalf("Expected error %v but got %v", tc.wantError, err)
			}
		}
		if tc.wantError == nil {
			if diff := cmp.Diff(tc.want, mat); diff != "" {
				t.Errorf("materials(): -want +got: %s", diff)
			}
		}
	}
}

func TestFromSidecarImages(t *testing.T) {
	tests := []struct {
		name      string
		sidecars  []v1beta1.SidecarState
		want      []common.ProvenanceMaterial
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
		want: []common.ProvenanceMaterial{
			{
				URI: artifacts.OCIScheme + "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: artifacts.OCIScheme + "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: artifacts.OCIScheme + "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
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
		want:      []common.ProvenanceMaterial{{}},
		wantError: fmt.Errorf("expected imageID gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init-sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247 to be separable by @"),
	}, {
		name: "sidecars with bad imageId - no digest",
		sidecars: []v1beta1.SidecarState{{
			Name:    "git-source-repo-jwqcl",
			ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256-b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
		}},
		want:      []common.ProvenanceMaterial{{}},
		wantError: fmt.Errorf("expected imageID gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256-b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247 to be separable by @ and :"),
	}}
	for _, tc := range tests {
		mat, err := FromSidecarImages(tc.sidecars)
		if err != nil {
			if err.Error() != tc.wantError.Error() {
				t.Fatalf("Expected error %v but got %v", tc.wantError, err)
			}
		}
		if tc.wantError == nil {
			if diff := cmp.Diff(tc.want, mat); diff != "" {
				t.Errorf("materials(): -want +got: %s", diff)
			}
		}
	}
}

func TestFromImageID(t *testing.T) {
	tests := []struct {
		name      string
		imageID   string
		want      common.ProvenanceMaterial
		wantError error
	}{{
		name:    "proper ImageID",
		imageID: "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
		want: common.ProvenanceMaterial{
			URI: artifacts.OCIScheme + "gcr.io/cloud-marketplace-containers/google/bazel",
			Digest: common.DigestSet{
				"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
			},
		},
	}, {
		name:      "bad ImageID",
		imageID:   "badImageId",
		want:      common.ProvenanceMaterial{},
		wantError: fmt.Errorf("expected imageID badImageId to be separable by @"),
	}}
	for _, tc := range tests {
		mat, err := fromImageID(tc.imageID)
		if err != nil {
			if err.Error() != tc.wantError.Error() {
				t.Fatalf("Expected error %v but got %v", tc.wantError, err)
			}
		}
		if tc.wantError == nil {
			if diff := cmp.Diff(tc.want, mat); diff != "" {
				t.Errorf("materials(): -want +got: %s", diff)
			}
		}
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name string
		mats []common.ProvenanceMaterial
		want []common.ProvenanceMaterial
	}{{
		name: "no duplicate materials",
		mats: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
		want: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
	}, {
		name: "same uri and digest",
		mats: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "same uri but different digest",
		mats: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
		want: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
	}, {
		name: "same uri but different digest, swap order",
		mats: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mat, err := removeDuplicateMaterials(tc.mats)
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}
			if diff := cmp.Diff(tc.want, mat); diff != "" {
				t.Errorf("materials(): -want +got: %s", diff)
			}
		})
	}
}

func TestFromPipelineParamsAndResults(t *testing.T) {
	tests := []struct {
		name        string
		pipelineRun *v1beta1.PipelineRun
		want        []common.ProvenanceMaterial
	}{{
		name: "from results",
		pipelineRun: &v1beta1.PipelineRun{
			Status: v1beta1.PipelineRunStatus{
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					PipelineResults: []v1beta1.PipelineRunResult{{
						Name:  "CHAINS-GIT_COMMIT",
						Value: *v1beta1.NewStructuredValues("my-commit"),
					}, {
						Name:  "CHAINS-GIT_URL",
						Value: *v1beta1.NewStructuredValues("github.com/something"),
					}},
				},
			},
		},
		want: []common.ProvenanceMaterial{{
			URI: "git+github.com/something.git",
			Digest: common.DigestSet{
				"sha1": "my-commit",
			},
		}},
	}, {
		name: "from pipelinespec",
		pipelineRun: &v1beta1.PipelineRun{
			Status: v1beta1.PipelineRunStatus{
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					PipelineSpec: &v1beta1.PipelineSpec{
						Params: []v1beta1.ParamSpec{{
							Name: "CHAINS-GIT_COMMIT",
							Default: &v1beta1.ParamValue{
								StringVal: "my-commit",
							},
						}, {
							Name: "CHAINS-GIT_URL",
							Default: &v1beta1.ParamValue{
								StringVal: "github.com/something",
							},
						}},
					},
				},
			},
		},
		want: []common.ProvenanceMaterial{{
			URI: "git+github.com/something.git",
			Digest: common.DigestSet{
				"sha1": "my-commit",
			},
		}},
	}, {
		name: "from pipelineRunSpec",
		pipelineRun: &v1beta1.PipelineRun{
			Spec: v1beta1.PipelineRunSpec{
				Params: []v1beta1.Param{{
					Name: "CHAINS-GIT_COMMIT",
					Value: v1beta1.ParamValue{
						StringVal: "my-commit",
					},
				}, {
					Name: "CHAINS-GIT_URL",
					Value: v1beta1.ParamValue{
						StringVal: "github.com/something",
					},
				}},
			},
		},
		want: []common.ProvenanceMaterial{{
			URI: "git+github.com/something.git",
			Digest: common.DigestSet{
				"sha1": "my-commit",
			},
		}},
	}, {
		name: "from completeChain",
		pipelineRun: &v1beta1.PipelineRun{
			Spec: v1beta1.PipelineRunSpec{
				Params: []v1beta1.Param{{
					Name: "CHAINS-GIT_URL",
					Value: v1beta1.ParamValue{
						StringVal: "github.com/something",
					},
				}},
			},
			Status: v1beta1.PipelineRunStatus{
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					PipelineSpec: &v1beta1.PipelineSpec{
						Params: []v1beta1.ParamSpec{{
							Name: "CHAINS-GIT_URL",
						}},
					},
					PipelineResults: []v1beta1.PipelineRunResult{{
						Name:  "CHAINS-GIT_COMMIT",
						Value: *v1beta1.NewStructuredValues("my-commit"),
					}},
				},
			},
		},
		want: []common.ProvenanceMaterial{{
			URI: "git+github.com/something.git",
			Digest: common.DigestSet{
				"sha1": "my-commit",
			},
		}},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			got := FromPipelineParamsAndResults(ctx, objects.NewPipelineRunObject(tc.pipelineRun))
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("FromPipelineParamsAndResults(): -want +got: %s", diff)
			}
		})
	}
}
