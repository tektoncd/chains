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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	v1slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/internal/backport"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/compare"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	logtesting "knative.dev/pkg/logging/testing"
)

const digest = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7"

var pro *objects.PipelineRunObjectV1Beta1
var proStructuredResults *objects.PipelineRunObjectV1Beta1

func init() {
	pro = createPro("../../../testdata/v2alpha2/pipelinerun1.json")
	proStructuredResults = createPro("../../../testdata/v2alpha2/pipelinerun_structured_results.json")
}

func createPro(path string) *objects.PipelineRunObjectV1Beta1 {
	var err error
	pr, err := objectloader.PipelineRunV1Beta1FromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunV1Beta1FromFile("../../../testdata/v2alpha2/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunV1Beta1FromFile("../../../testdata/v2alpha2/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObjectV1Beta1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func tektonTaskRuns() map[string][]byte {
	trs := make(map[string][]byte)
	tr1, err := objectloader.TaskRunV1Beta1FromFile("../../../testdata/v2alpha2/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunV1Beta1FromFile("../../../testdata/v2alpha2/taskrun2.json")
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

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name string
		rds  []v1.ResourceDescriptor
		want []v1.ResourceDescriptor
	}{{
		name: "no duplicate resolvedDependencies",
		rds: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
		want: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
	}, {
		name: "same uri and digest",
		rds: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "same uri but different digest",
		rds: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
		want: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
	}, {
		name: "same uri but different digest, swap order",
		rds: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "task config must be present",
		rds: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "task",
				URI:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "task",
				URI:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "pipeline config must be present",
		rds: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "pipeline",
				URI:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "pipeline",
				URI:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rds, err := removeDuplicateResolvedDependencies(tc.rds)
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}
			if diff := cmp.Diff(tc.want, rds); diff != "" {
				t.Errorf("resolvedDependencies(): -want +got: %s", diff)
			}
		})
	}
}

func TestTaskRun(t *testing.T) {
	tests := []struct {
		name    string
		taskRun *v1beta1.TaskRun //nolint:staticcheck
		want    []v1.ResourceDescriptor
	}{{
		name: "resolvedDependencies from pipeline resources",
		taskRun: &v1beta1.TaskRun{ //nolint:staticcheck
			Spec: v1beta1.TaskRunSpec{
				Resources: &v1beta1.TaskRunResources{ //nolint:all //incompatible with pipelines v0.45
					Inputs: []v1beta1.TaskResourceBinding{ //nolint:all //incompatible with pipelines v0.45
						{
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{ //nolint:all //incompatible with pipelines v0.45
								Name: "nil-resource-spec",
							},
						}, {
							PipelineResourceBinding: v1beta1.PipelineResourceBinding{ //nolint:all //incompatible with pipelines v0.45
								Name: "repo",
								ResourceSpec: &v1alpha1.PipelineResourceSpec{ //nolint:all //incompatible with pipelines v0.45
									Params: []v1alpha1.ResourceParam{ //nolint:all //incompatible with pipelines v0.45
										{Name: "url", Value: "https://github.com/GoogleContainerTools/distroless"},
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
		want: []v1.ResourceDescriptor{
			{
				Name: "inputs/result",
				URI:  "gcr.io/foo/bar",
				Digest: common.DigestSet{
					"sha256": strings.TrimPrefix(digest, "sha256:"),
				},
			},
			{
				Name: "pipelineResource",
				URI:  "git+https://github.com/GoogleContainerTools/distroless.git",
				Digest: common.DigestSet{
					"sha1": "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe",
				},
			},
		},
	}, {
		name: "resolvedDependencies from remote task",
		taskRun: &v1beta1.TaskRun{ //nolint:staticcheck
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					Provenance: &v1beta1.Provenance{
						RefSource: &v1beta1.RefSource{
							URI: "git+github.com/something.git",
							Digest: map[string]string{
								"sha1": "abcd1234",
							},
						},
					},
				},
			},
		},
		want: []v1.ResourceDescriptor{
			{
				Name: "task",
				URI:  "git+github.com/something.git",
				Digest: common.DigestSet{
					"sha1": "abcd1234",
				},
			},
		},
	}, {
		name: "git resolvedDependencies from taskrun params",
		taskRun: &v1beta1.TaskRun{ //nolint:staticcheck
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
		want: []v1.ResourceDescriptor{
			{
				Name: "inputs/result",
				URI:  "git+github.com/something.git",
				Digest: common.DigestSet{
					"sha1": "my-commit",
				},
			},
		},
	}, {
		name: "resolvedDependencies from step images",
		taskRun: &v1beta1.TaskRun{ //nolint:staticcheck
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
		want: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
			{
				URI: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			},
		},
	}, {
		name: "resolvedDependencies from step and sidecar images",
		taskRun: &v1beta1.TaskRun{ //nolint:staticcheck
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
		want: []v1.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			rd, err := TaskRun(ctx, objects.NewTaskRunObjectV1Beta1(tc.taskRun))
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}
			if diff := cmp.Diff(tc.want, rd); diff != "" {
				t.Errorf("ResolvedDependencies(): -want +got: %s", diff)
			}
		})
	}
}

func TestPipelineRun(t *testing.T) {
	taskRuns := tektonTaskRuns()
	tests := []struct {
		name           string
		taskDescriptor addTaskDescriptorContent
		want           []v1slsa.ResourceDescriptor
	}{
		{
			name:           "test slsa build type",
			taskDescriptor: AddSLSATaskDescriptor,
			want: []v1slsa.ResourceDescriptor{
				{Name: "pipeline", URI: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "28b123"}},
				{Name: "pipelineTask", URI: "git+https://github.com/catalog", Digest: common.DigestSet{"sha1": "x123"}},
				{
					URI:    "oci://gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
				{Name: "pipelineTask", URI: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "ab123"}},
				{
					URI:    "oci://gcr.io/test2/test2",
					Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
				},
				{
					URI:    "oci://gcr.io/test3/test3",
					Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
				},
				{Name: "inputs/result", URI: "abc", Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}},
				{Name: "inputs/result", URI: "git+https://git.test.com.git", Digest: common.DigestSet{"sha1": "abcd"}},
			},
		},
		{
			name:           "test tekton build type",
			taskDescriptor: AddTektonTaskDescriptor,
			want: []v1slsa.ResourceDescriptor{
				{Name: "pipeline", URI: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "28b123"}},
				{Name: "pipelineTask", URI: "git+https://github.com/catalog", Digest: common.DigestSet{"sha1": "x123"}, Content: taskRuns["git-clone"]},
				{
					URI:    "oci://gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
				{Name: "pipelineTask", URI: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "ab123"}, Content: taskRuns["taskrun-build"]},
				{
					URI:    "oci://gcr.io/test2/test2",
					Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
				},
				{
					URI:    "oci://gcr.io/test3/test3",
					Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
				},
				{Name: "inputs/result", URI: "abc", Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}},
				{Name: "inputs/result", URI: "git+https://git.test.com.git", Digest: common.DigestSet{"sha1": "abcd"}},
			},
		},
	}

	ctx := logtesting.TestContextWithLogger(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PipelineRun(ctx, pro, &slsaconfig.SlsaConfig{DeepInspectionEnabled: false}, tc.taskDescriptor)
			if err != nil {
				t.Error(err)
			}
			if d := cmp.Diff(tc.want, got); d != "" {
				t.Errorf("PipelineRunResolvedDependencies(): -want +got: %s", got)
			}
		})
	}
}

func TestPipelineRunStructuredResult(t *testing.T) {
	want := []v1slsa.ResourceDescriptor{
		{Name: "pipeline", URI: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "28b123"}},
		{Name: "pipelineTask", URI: "git+https://github.com/catalog", Digest: common.DigestSet{"sha1": "x123"}},
		{
			URI:    "oci://gcr.io/test1/test1",
			Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
		},
		{Name: "pipelineTask", URI: "git+https://github.com/test", Digest: common.DigestSet{"sha1": "ab123"}},
		{
			URI:    "oci://gcr.io/test2/test2",
			Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
		},
		{
			URI:    "oci://gcr.io/test3/test3",
			Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
		},
		{
			Name: "inputs/result",
			URI:  "abc",
			Digest: common.DigestSet{
				"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
			},
		},
		{
			URI:    "git+https://git.test.com.git",
			Digest: common.DigestSet{"sha1": "abcd"},
			Name:   "inputs/result",
		},
	}
	ctx := logtesting.TestContextWithLogger(t)
	got, err := PipelineRun(ctx, pro, &slsaconfig.SlsaConfig{DeepInspectionEnabled: false}, AddSLSATaskDescriptor)
	if err != nil {
		t.Errorf("error while extracting resolvedDependencies: %v", err)
	}
	if diff := cmp.Diff(want, got, compare.SLSAV1CompareOptions()...); diff != "" {
		t.Errorf("resolvedDependencies(): -want +got: %s", diff)
	}
}
