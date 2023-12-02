/*
Copyright 2021 The Tekton Authors

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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/attest"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/yaml"
)

func TestBuildConfig(t *testing.T) {
	taskrun := `apiVersion: tekton.dev/v1
kind: TaskRun
status:
  taskSpec:
    steps:
    - image: gcr.io/cloud-marketplace-containers/google/bazel:3.4.1
      name: build
      resources: {}
      script: |
        myscript
    - args:
      - arg1
      - arg2
      command:
      - sh
      image: gcr.io/go-containerregistry/crane:debug
      name: crane
      resources: {}
  steps:
  - container: step-git-source-repo-jwqcl
    imageID: gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247
    name: git-source-repo-jwqcl
    terminated:
      containerID: containerd://3e1aee4d75207745803b904a79dfdec3b9d5e30652b1af9336d0259f86c43873
  - container: step-build
    imageID: gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964
    name: build
    terminated:
      containerID: containerd://e2fadd134495619cccd1c48d8a9df2aed2afd64e6c62ea55135f90796102231e`

	var taskRun *v1.TaskRun
	if err := yaml.Unmarshal([]byte(taskrun), &taskRun); err != nil {
		t.Fatal(err)
	}

	expected := BuildConfig{
		Steps: []attest.StepAttestation{
			{
				EntryPoint: "",
				Environment: map[string]interface{}{
					"image":     artifacts.OCIScheme + "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
					"container": "git-source-repo-jwqcl",
				},
				Arguments: interface{}([]string(nil)),
			}, {
				Environment: map[string]interface{}{
					"image":     artifacts.OCIScheme + "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
					"container": "build",
				},
				EntryPoint: "myscript\n",
				Arguments:  interface{}([]string(nil)),
			},
		},
	}

	got := buildConfig(objects.NewTaskRunObjectV1(taskRun))
	if !reflect.DeepEqual(expected, got) {
		if d := cmp.Diff(expected, got); d != "" {
			t.Log(d)
		}
		t.Fatalf("expected \n%v\n got \n%v\n", expected, got)
	}
}
