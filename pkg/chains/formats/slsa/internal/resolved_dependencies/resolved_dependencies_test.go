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

package resolveddependencies

import (
	"encoding/json"
	"reflect"
	"runtime"
	"testing"

	"github.com/google/go-cmp/cmp"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"google.golang.org/protobuf/testing/protocmp"
	logtesting "knative.dev/pkg/logging/testing"
)

const digest = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7"

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name string
		rds  []*intoto.ResourceDescriptor
		want []*intoto.ResourceDescriptor
	}{{
		name: "no duplicate resolvedDependencies",
		rds: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Uri: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
		want: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Uri: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
	}, {
		name: "same uri and digest",
		rds: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "same uri but different digest",
		rds: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
		want: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
	}, {
		name: "same uri but different digest, swap order",
		rds: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "task config must be present",
		rds: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "task",
				Uri:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "task",
				Uri:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "pipeline config must be present",
		rds: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "pipeline",
				Uri:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []*intoto.ResourceDescriptor{
			{
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "pipeline",
				Uri:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rds, err := RemoveDuplicateResolvedDependencies(tc.rds)
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}
			if diff := cmp.Diff(tc.want, rds, cmp.Options{protocmp.Transform()}); diff != "" {
				t.Errorf("resolvedDependencies(): -want +got: %s", diff)
			}
		})
	}
}

func createPro(path string) *objects.PipelineRunObjectV1 {
	var err error
	pr, err := objectloader.PipelineRunV1FromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunV1FromFile("../../testdata/slsa-v2alpha3/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunV1FromFile("../../testdata/slsa-v2alpha3/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObjectV1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func tektonTaskRuns() map[string][]byte {
	trs := make(map[string][]byte)
	tr1, err := objectloader.TaskRunV1FromFile("../../testdata/slsa-v2alpha3/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunV1FromFile("../../testdata/slsa-v2alpha3/taskrun2.json")
	if err != nil {
		panic(err)
	}

	tr1Desc, err := json.Marshal(tr1)
	if err != nil {
		panic(err)
	}
	trs[tr1.Name] = tr1Desc

	tr2Desc, err := json.Marshal(tr2)
	if err != nil {
		panic(err)
	}
	trs[tr2.Name] = tr2Desc

	return trs
}

func TestTaskRun(t *testing.T) {
	tests := []struct {
		name        string
		obj         objects.TektonObject
		resolveOpts ResolveOptions
		want        []*intoto.ResourceDescriptor
	}{
		{
			name: "resolvedDependencies from remote task",
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						Provenance: &v1.Provenance{
							RefSource: &v1.RefSource{
								URI: "git+github.com/something.git",
								Digest: map[string]string{
									"sha1": "abcd1234",
								},
							},
						},
					},
				},
			}),
			want: []*intoto.ResourceDescriptor{
				{
					Name: "task",
					Uri:  "git+github.com/something.git",
					Digest: common.DigestSet{
						"sha1": "abcd1234",
					},
				},
			},
		},
		{
			name: "git resolvedDependencies from taskrun params",
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
			want: []*intoto.ResourceDescriptor{
				{
					Name: "inputs/result",
					Uri:  "git+github.com/something.git",
					Digest: common.DigestSet{
						"sha1": "my-commit",
					},
				},
			},
		},
		{
			name: "resolvedDependencies from step images",
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
			want: []*intoto.ResourceDescriptor{
				{
					Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
					Digest: common.DigestSet{
						"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
					},
				},
				{
					Uri: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
					Digest: common.DigestSet{
						"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
					},
				},
			},
		},
		{
			name: "resolvedDependencies from step and sidecar images",
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						Steps: []v1.StepState{{
							Results: []v1.TaskRunStepResult{
								{Name: "res1_ARTIFACT_INPUTS", Value: *v1.NewObject(map[string]string{
									"uri":    "https://github.com/tektoncd/pipeline",
									"digest": "sha1:7f2f46e1b97df36b2b82d1b1d87c81b8b3d21601",
								})},
							},
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
			want: []*intoto.ResourceDescriptor{
				{
					Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
					Digest: common.DigestSet{
						"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
					},
				}, {
					Uri: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
					Digest: common.DigestSet{
						"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
					},
				}, {
					Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
					Digest: common.DigestSet{
						"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
					},
				},
			},
		},
		{
			name: "resolvedDependencies with nested results",
			resolveOpts: ResolveOptions{
				WithStepActionsResults: true,
			},
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						Steps: []v1.StepState{{
							Results: []v1.TaskRunStepResult{
								{Name: "res1_ARTIFACT_INPUTS", Value: *v1.NewObject(map[string]string{
									"uri":    "https://github.com/tektoncd/pipeline",
									"digest": "sha1:7f2f46e1b97df36b2b82d1b1d87c81b8b3d21601",
								})},
							},
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
			want: []*intoto.ResourceDescriptor{
				{
					Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
					Digest: common.DigestSet{
						"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
					},
				}, {
					Uri: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
					Digest: common.DigestSet{
						"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
					},
				}, {
					Uri: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
					Digest: common.DigestSet{
						"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
					},
				}, {
					Name: "inputs/result",
					Uri:  "https://github.com/tektoncd/pipeline",
					Digest: common.DigestSet{
						"sha1": "7f2f46e1b97df36b2b82d1b1d87c81b8b3d21601",
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			var input *objects.TaskRunObjectV1
			var err error
			if obj, ok := tc.obj.(*objects.TaskRunObjectV1); !ok {
				t.Fatalf("Unexpected tekton object type: %T", tc.obj)
			} else {
				input = obj
			}

			rd, err := TaskRun(ctx, tc.resolveOpts, input)
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}
			if diff := cmp.Diff(tc.want, rd, cmp.Options{protocmp.Transform()}); diff != "" {
				t.Errorf("ResolvedDependencies(): -want +got: %s", diff)
			}
		})
	}
}

func TestPipelineRun(t *testing.T) {
	taskRuns := tektonTaskRuns()
	tests := []struct {
		name           string
		taskDescriptor AddTaskDescriptorContent
		want           []*intoto.ResourceDescriptor
	}{
		{
			name:           "test slsa build type",
			taskDescriptor: AddSLSATaskDescriptor,
			want: []*intoto.ResourceDescriptor{
				{Name: "pipeline", Uri: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "28b123"}},
				{Name: "pipelineTask", Uri: "git+https://github.com/catalog", Digest: common.DigestSet{"sha1": "x123"}},
				{
					Uri:    "oci://gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
				{Name: "pipelineTask", Uri: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "ab123"}},
				{
					Uri:    "oci://gcr.io/test2/test2",
					Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
				},
				{
					Uri:    "oci://gcr.io/test3/test3",
					Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
				},
				{Name: "inputs/result", Uri: "abc", Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}},
				{Name: "inputs/result", Uri: "git+https://git.test.com.git", Digest: common.DigestSet{"sha1": "abcd"}},
			},
		},
		{
			name:           "test tekton build type",
			taskDescriptor: AddTektonTaskDescriptor,
			want: []*intoto.ResourceDescriptor{
				{Name: "pipeline", Uri: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "28b123"}},
				{Name: "pipelineTask", Uri: "git+https://github.com/catalog", Digest: common.DigestSet{"sha1": "x123"}, Content: taskRuns["git-clone"]},
				{
					Uri:    "oci://gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
				{Name: "pipelineTask", Uri: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "ab123"}, Content: taskRuns["taskrun-build"]},
				{
					Uri:    "oci://gcr.io/test2/test2",
					Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
				},
				{
					Uri:    "oci://gcr.io/test3/test3",
					Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
				},
				{Name: "inputs/result", Uri: "abc", Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}},
				{Name: "inputs/result", Uri: "git+https://git.test.com.git", Digest: common.DigestSet{"sha1": "abcd"}},
			},
		},
	}

	ctx := logtesting.TestContextWithLogger(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pro := createPro("../../testdata/slsa-v2alpha3/pipelinerun1.json")
			got, err := PipelineRun(ctx, pro, &slsaconfig.SlsaConfig{DeepInspectionEnabled: false}, ResolveOptions{}, tc.taskDescriptor)
			if err != nil {
				t.Error(err)
			}
			if d := cmp.Diff(tc.want, got, protocmp.Transform()); d != "" {
				t.Errorf("PipelineRunResolvedDependencies(): -want +got: %s", d)
			}
		})
	}
}

func TestGetTaskDescriptor(t *testing.T) {
	tests := []struct {
		name                string
		buildDefinitionType string
		expected            AddTaskDescriptorContent
		shouldErr           bool
	}{
		{
			name:                "slsa task descriptor",
			buildDefinitionType: "https://tekton.dev/chains/v2/slsa",
			expected:            AddSLSATaskDescriptor,
		},
		{
			name:                "tekton task descriptor",
			buildDefinitionType: "https://tekton.dev/chains/v2/slsa-tekton",
			expected:            AddTektonTaskDescriptor,
		},
		{
			name:                "bad descriptor",
			buildDefinitionType: "https://foo.io/fake",
			shouldErr:           true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			td, err := GetTaskDescriptor(test.buildDefinitionType)
			didErr := err != nil

			if test.shouldErr != didErr {
				t.Fatalf("Unexpected behavior in error, shouldErr: %v, didErr: %v, err: %v", test.shouldErr, didErr, err)
			}

			got := runtime.FuncForPC(reflect.ValueOf(td).Pointer()).Name()
			expected := runtime.FuncForPC(reflect.ValueOf(test.expected).Pointer()).Name()

			if d := cmp.Diff(expected, got); d != "" {
				t.Errorf("GetTaskDescriptor(): -want +got: %v", d)
			}
		})
	}
}
