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

package provenance

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/chains/provenance"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	digest1 = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5"
	digest2 = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b6"
)

func TestMetadata(t *testing.T) {
	tr := &v1beta1.TaskRun{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "my-namespace",
			Annotations: map[string]string{
				"chains.tekton.dev/reproducible": "true",
			},
		},
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				StartTime:      &v1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)},
				CompletionTime: &v1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
			},
		},
	}
	start := time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)
	end := time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)
	expected := provenance.ProvenanceMetadata{
		BuildStartedOn:  &start,
		BuildFinishedOn: &end,
	}
	got := metadata(tr)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v got %v", expected, got)
	}
}

func TestMaterials(t *testing.T) {
	// make sure this works with Git resources
	taskrun := `apiVersion: tekton.dev/v1beta1
kind: TaskRun
spec:
  resources:
    inputs:
    - name: repo
      resourceSpec:
        params:
        - name: url
          value: https://github.com/GoogleContainerTools/distroless
        type: git
  taskSpec:
    resources:
      inputs:
      - name: repo
        type: git
status:
  resourcesResult:
  - key: commit
    resourceName: repo
    resourceRef:
      name: repo
    value: 50c56a48cfb3a5a80fa36ed91c739bdac8381cbe
  - key: url
    resourceName: repo
    resourceRef:
      name: repo
    value: https://github.com/GoogleContainerTools/distroless`

	var taskRun *v1beta1.TaskRun
	if err := yaml.Unmarshal([]byte(taskrun), &taskRun); err != nil {
		t.Fatal(err)
	}

	expected := []provenance.ProvenanceMaterial{
		{
			URI: "https://github.com/GoogleContainerTools/distroless",
			Digest: provenance.DigestSet{
				"revision": "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe",
			},
		},
	}

	got := materials(taskRun)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v got %v", expected, got)
	}

	// make sure this works with GIT* results as well
	taskrun = `apiVersion: tekton.dev/v1beta1
kind: TaskRun
spec:
  params:
  - name: CHAINS-GIT_COMMIT
    value: my-commit
  - name: CHAINS-GIT_URL
    value: github.com/something`
	taskRun = &v1beta1.TaskRun{}
	if err := yaml.Unmarshal([]byte(taskrun), &taskRun); err != nil {
		t.Fatal(err)
	}

	expected = []provenance.ProvenanceMaterial{
		{
			URI: "github.com/something",
			Digest: provenance.DigestSet{
				"revision": "my-commit",
			},
		},
	}

	got = materials(taskRun)
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v got %v", expected, got)
	}
}

func TestRecipe(t *testing.T) {
	taskrun := `apiVersion: tekton.dev/v1beta1
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

	var taskRun *v1beta1.TaskRun
	if err := yaml.Unmarshal([]byte(taskrun), &taskRun); err != nil {
		t.Fatal(err)
	}

	expected := provenance.ProvenanceRecipe{
		Steps: []provenance.RecipeStep{
			{
				EntryPoint: "",
				Environment: map[string]interface{}{
					"image":     "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
					"container": "git-source-repo-jwqcl",
				},
				Arguments: interface{}([]string(nil)),
			}, {
				Environment: map[string]interface{}{
					"image":     "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
					"container": "build",
				},
				EntryPoint: "myscript\n",
				Arguments:  interface{}([]string(nil)),
			},
		},
	}

	got := recipe(taskRun)
	if !reflect.DeepEqual(expected, got) {
		if d := cmp.Diff(expected, got); d != "" {
			t.Log(d)
		}
		t.Fatalf("expected \n%v\n got \n%v\n", expected, got)
	}
}

func TestInvocation(t *testing.T) {
	taskrun := `apiVersion: tekton.dev/v1beta1
kind: TaskRun
metadata:
  uid: my-uid
spec:
  params:
  - name: my-param
    value: string-param
  - name: my-array-param
    value:
    - "my"
    - "array"`

	var taskRun *v1beta1.TaskRun
	if err := yaml.Unmarshal([]byte(taskrun), &taskRun); err != nil {
		t.Fatal(err)
	}

	expected := provenance.Invocation{
		EventID: "my-uid",
		Parameters: []string{
			"my-param={string string-param []}",
			"my-array-param={array  [my array]}",
		},
		ID: "tekton-chains",
	}

	got := invocation("tekton-chains", taskRun)
	if !reflect.DeepEqual(expected, got) {
		if d := cmp.Diff(expected, got); d != "" {
			t.Log(d)
		}
		t.Fatalf("expected \n%v\n got \n%v\n", expected, got)
	}
}

func TestGetSubjectDigests(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Spec: v1beta1.TaskRunSpec{
			Resources: &v1beta1.TaskRunResources{
				Outputs: []v1beta1.TaskResourceBinding{
					{
						PipelineResourceBinding: v1beta1.PipelineResourceBinding{
							Name: "built-image",
							ResourceSpec: &v1alpha1.PipelineResourceSpec{
								Type: v1alpha1.PipelineResourceTypeImage,
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
						Name:  "IMAGE_URL",
						Value: "registry/myimage",
					},
					{
						Name:  "IMAGE_DIGEST",
						Value: digest1,
					},
				},
				ResourcesResult: []v1beta1.PipelineResourceResult{
					{
						ResourceName: "built-image",
						Key:          "url",
						Value:        "registry/resource-image",
					}, {
						ResourceName: "built-image",
						Key:          "digest",
						Value:        digest2,
					},
				},
			},
		},
	}

	expected := []in_toto.Subject{
		{
			Name: "index.docker.io/registry/myimage",
			Digest: in_toto.DigestSet{
				"sha256": strings.TrimPrefix(digest1, "sha256:"),
			},
		}, {
			Name: "registry/resource-image",
			Digest: in_toto.DigestSet{
				"sha256": strings.TrimPrefix(digest2, "sha256:"),
			},
		},
	}
	got := getSubjectDigests(tr, logtesting.TestLogger(t))
	if !reflect.DeepEqual(expected, got) {
		if d := cmp.Diff(expected, got); d != "" {
			t.Log(d)
		}
		t.Fatalf("expected \n%v\n got \n%v\n", expected, got)
	}
}
