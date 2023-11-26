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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	"github.com/tektoncd/chains/internal/backport"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/compare"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

const digest = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7"

func createPro(path string) *objects.PipelineRunObjectV1 {
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
	p := objects.NewPipelineRunObjectV1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func TestMaterialsWithResults(t *testing.T) {
	taskRun := &v1.TaskRun{
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				Results: []v1.TaskRunResult{
					{
						Name: "CHAINS-GIT_COMMIT",
						Value: v1.ParamValue{
							StringVal: "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe",
						},
					},
					{
						Name: "CHAINS-GIT_URL",
						Value: v1.ParamValue{
							StringVal: "https://github.com/GoogleContainerTools/distroless",
						},
					},
				},
			},
		},
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
	got, err := TaskMaterials(ctx, objects.NewTaskRunObjectV1(taskRun))
	if err != nil {
		t.Fatalf("Did not expect an error but got %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v got %v", want, got)
	}
}

func TestTaskMaterials(t *testing.T) {
	tests := []struct {
		name string
		obj  objects.TektonObject
		want []common.ProvenanceMaterial
	}{
		{
			name: "materials from pipeline resources",
			obj: objects.NewTaskRunObjectV1Beta1(&v1beta1.TaskRun{ //nolint:staticcheck
				Spec: v1beta1.TaskRunSpec{
					Resources: &v1beta1.TaskRunResources{ //nolint:staticcheck
						Inputs: []v1beta1.TaskResourceBinding{ //nolint:staticcheck
							{
								PipelineResourceBinding: v1beta1.PipelineResourceBinding{ //nolint:staticcheck
									Name: "nil-resource-spec",
								},
							}, {
								PipelineResourceBinding: v1beta1.PipelineResourceBinding{ //nolint:staticcheck
									Name: "repo",
									ResourceSpec: &v1alpha1.PipelineResourceSpec{ //nolint:staticcheck
										Params: []v1alpha1.ResourceParam{ //nolint:staticcheck
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
			}),
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
		},
		{
			name: "materials from git results in task run spec",
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Params: []v1.Param{{
						Name:  "CHAINS-GIT_COMMIT",
						Value: *v1.NewStructuredValues("my-commit"),
					}, {
						Name:  "CHAINS-GIT_URL",
						Value: *v1.NewStructuredValues("github.com/something"),
					}},
				},
			}),
			want: []common.ProvenanceMaterial{
				{
					URI: artifacts.GitSchemePrefix + "github.com/something.git",
					Digest: common.DigestSet{
						"sha1": "my-commit",
					},
				},
			},
		},
		{
			name: "materials from git results in task spec",
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						TaskSpec: &v1.TaskSpec{
							Params: []v1.ParamSpec{{
								Name: "CHAINS-GIT_COMMIT",
								Default: &v1.ParamValue{
									StringVal: "my-commit",
								},
							}, {
								Name: "CHAINS-GIT_URL",
								Default: &v1.ParamValue{
									StringVal: "github.com/something",
								},
							}},
						},
					},
				},
			}),
			want: []common.ProvenanceMaterial{
				{
					URI: artifacts.GitSchemePrefix + "github.com/something.git",
					Digest: common.DigestSet{
						"sha1": "my-commit",
					},
				},
			},
		},
		{
			name: "materials from git results in task spec and taskrun spec",
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				Spec: v1.TaskRunSpec{
					Params: []v1.Param{{
						Name: "CHAINS-GIT_URL",
						Value: v1.ParamValue{
							StringVal: "github.com/something",
						},
					}},
				},
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						TaskSpec: &v1.TaskSpec{
							Params: []v1.ParamSpec{{
								Name: "CHAINS-GIT_URL",
							}, {
								Name: "CHAINS-GIT_COMMIT",
								Default: &v1.ParamValue{
									StringVal: "my-commit",
								},
							}},
						},
					},
				},
			}),
			want: []common.ProvenanceMaterial{{
				URI: "git+github.com/something.git",
				Digest: common.DigestSet{
					"sha1": "my-commit",
				},
			}},
		},
		{
			name: "materials from step images",
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						Steps: []v1.StepState{{
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
			}),
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
		},
		{
			name: "materials from step and sidecar images",
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						Steps: []v1.StepState{{
							Name:    "git-source-repo-jwqcl",
							ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
						}, {
							Name:    "git-source-repo-repeat-again-jwqcl",
							ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
						}, {
							Name:    "build",
							ImageID: "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
						}},
						Sidecars: []v1.SidecarState{{
							Name:    "sidecar-jwqcl",
							ImageID: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init@sha256:a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
						}},
					},
				},
			}),
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
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			// convert tekton object to taskrun objet
			var input *objects.TaskRunObjectV1
			var err error
			if obj, ok := tc.obj.(*objects.TaskRunObjectV1); ok {
				input = obj
			}

			if trV1Beta1, ok := tc.obj.GetObject().(*v1beta1.TaskRun); ok { //nolint:staticcheck
				trV1 := &v1.TaskRun{}
				if err := trV1Beta1.ConvertTo(ctx, trV1); err == nil {
					if trV1Beta1.Spec.Resources != nil { //nolint:staticcheck
						jsonData, err := json.Marshal(trV1Beta1.Spec.Resources) //nolint:staticcheck
						if err != nil {
							t.Errorf("Error serializing to JSON: %v", err)
						}
						trV1.Annotations["tekton.dev/v1beta1-spec-resources"] = string(jsonData)
					}
					input = objects.NewTaskRunObjectV1(trV1)
				}
			}
			mat, err := TaskMaterials(ctx, input)
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
	got, err := PipelineMaterials(ctx, createPro("../../testdata/pipelinerun1.json"), &slsaconfig.SlsaConfig{DeepInspectionEnabled: false})
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
	got, err := PipelineMaterials(ctx, createPro("../../testdata/pipelinerun_structured_results.json"), &slsaconfig.SlsaConfig{DeepInspectionEnabled: false})
	if err != nil {
		t.Errorf("error while extracting materials: %v", err)
	}
	if diff := cmp.Diff(want, got, compare.MaterialsCompareOption()); diff != "" {
		t.Errorf("materials(): -want +got: %s", diff)
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

//nolint:all
func TestFromPipelineParamsAndResults(t *testing.T) {
	tests := []struct {
		name                 string
		pipelineRunObject    *objects.PipelineRunObjectV1
		enableDeepInspection bool
		want                 []common.ProvenanceMaterial
	}{{
		name: "from results",
		pipelineRunObject: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					Results: []v1.PipelineRunResult{{
						Name:  "CHAINS-GIT_COMMIT",
						Value: *v1.NewStructuredValues("my-commit"),
					}, {
						Name:  "CHAINS-GIT_URL",
						Value: *v1.NewStructuredValues("github.com/something"),
					}},
				},
			},
		}),
		want: []common.ProvenanceMaterial{{
			URI: "git+github.com/something.git",
			Digest: common.DigestSet{
				"sha1": "my-commit",
			},
		}},
	}, {
		name: "from pipelinespec",
		pipelineRunObject: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					PipelineSpec: &v1.PipelineSpec{
						Params: []v1.ParamSpec{{
							Name: "CHAINS-GIT_COMMIT",
							Default: &v1.ParamValue{
								StringVal: "my-commit",
							},
						}, {
							Name: "CHAINS-GIT_URL",
							Default: &v1.ParamValue{
								StringVal: "github.com/something",
							},
						}},
					},
				},
			},
		}),
		want: []common.ProvenanceMaterial{{
			URI: "git+github.com/something.git",
			Digest: common.DigestSet{
				"sha1": "my-commit",
			},
		}},
	}, {
		name: "from pipelineRunSpec",
		pipelineRunObject: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
			Spec: v1.PipelineRunSpec{
				Params: []v1.Param{{
					Name: "CHAINS-GIT_COMMIT",
					Value: v1.ParamValue{
						StringVal: "my-commit",
					},
				}, {
					Name: "CHAINS-GIT_URL",
					Value: v1.ParamValue{
						StringVal: "github.com/something",
					},
				}},
			},
		}),
		want: []common.ProvenanceMaterial{{
			URI: "git+github.com/something.git",
			Digest: common.DigestSet{
				"sha1": "my-commit",
			},
		}},
	}, {
		name: "from completeChain",
		pipelineRunObject: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
			Spec: v1.PipelineRunSpec{
				Params: []v1.Param{{
					Name: "CHAINS-GIT_URL",
					Value: v1.ParamValue{
						StringVal: "github.com/something",
					},
				}},
			},
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					PipelineSpec: &v1.PipelineSpec{
						Params: []v1.ParamSpec{{
							Name: "CHAINS-GIT_URL",
						}},
					},
					Results: []v1.PipelineRunResult{{
						Name:  "CHAINS-GIT_COMMIT",
						Value: *v1.NewStructuredValues("my-commit"),
					}},
				},
			},
		}),
		want: []common.ProvenanceMaterial{{
			URI: "git+github.com/something.git",
			Digest: common.DigestSet{
				"sha1": "my-commit",
			},
		}},
	},
		{
			name:                 "deep inspection: pipelinerun param and task result",
			pipelineRunObject:    createProWithPipelineParamAndTaskResult(),
			enableDeepInspection: true,
			want: []common.ProvenanceMaterial{
				{
					URI: "git+github.com/pipelinerun-param.git",
					Digest: common.DigestSet{
						"sha1": "115734d92807a80158b4b7af605d768c647fdb3d",
					},
				}, {
					URI: "github.com/childtask-result",
					Digest: common.DigestSet{
						"sha1": "225734d92807a80158b4b7af605d768c647fdb3d",
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			got := FromPipelineParamsAndResults(ctx, tc.pipelineRunObject, &slsaconfig.SlsaConfig{DeepInspectionEnabled: tc.enableDeepInspection})
			if diff := cmp.Diff(tc.want, got, compare.MaterialsCompareOption()); diff != "" {
				t.Errorf("FromPipelineParamsAndResults(): -want +got: %s", diff)
			}
		})
	}
}

//nolint:all
func createProWithPipelineParamAndTaskResult() *objects.PipelineRunObjectV1 {
	pro := objects.NewPipelineRunObjectV1(&v1.PipelineRun{
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				PipelineSpec: &v1.PipelineSpec{
					Params: []v1.ParamSpec{{
						Name: "CHAINS-GIT_COMMIT",
						Default: &v1.ParamValue{
							StringVal: "115734d92807a80158b4b7af605d768c647fdb3d",
						},
					}, {
						Name: "CHAINS-GIT_URL",
						Default: &v1.ParamValue{
							StringVal: "github.com/pipelinerun-param",
						},
					}},
				},
			},
		},
	})

	pipelineTaskName := "my-clone-task"
	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{objects.PipelineTaskLabel: pipelineTaskName}},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
				Results: []v1.TaskRunResult{
					{
						Name: "ARTIFACT_INPUTS",
						Value: *v1.NewObject(map[string]string{
							"uri":    "github.com/childtask-result",
							"digest": "sha1:225734d92807a80158b4b7af605d768c647fdb3d",
						})},
				},
			},
		},
	}

	pro.AppendTaskRun(tr)
	pro.Status.PipelineSpec.Tasks = []v1.PipelineTask{{Name: pipelineTaskName}}
	return pro
}
