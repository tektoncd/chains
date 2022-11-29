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
	"strings"
	"testing"
	"time"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/extract"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	digest1 = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5"
	digest2 = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b6"
	digest3 = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7"
	digest4 = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b8"
	digest5 = "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b9"
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
	expected := &slsa.ProvenanceMetadata{
		BuildStartedOn:  &start,
		BuildFinishedOn: &end,
	}
	got := metadata(objects.NewTaskRunObject(tr))
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v got %v", expected, got)
	}
}

func TestMetadataInTimeZone(t *testing.T) {
	tz := time.FixedZone("Test Time", int((12 * time.Hour).Seconds()))
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
				StartTime:      &v1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, tz)},
				CompletionTime: &v1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, tz)},
			},
		},
	}
	start := time.Date(1995, time.December, 24, 6, 12, 12, 12, tz).UTC()
	end := time.Date(1995, time.December, 24, 6, 12, 12, 24, tz).UTC()
	expected := &slsa.ProvenanceMetadata{
		BuildStartedOn:  &start,
		BuildFinishedOn: &end,
	}
	got := metadata(objects.NewTaskRunObject(tr))
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v got %v", expected, got)
	}
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

	expected := []slsa.ProvenanceMaterial{
		{
			URI: "git+https://github.com/GoogleContainerTools/distroless.git",
			Digest: slsa.DigestSet{
				"sha1": "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe",
			},
		},
	}

	got := materials(objects.NewTaskRunObject(taskRun), logtesting.TestLogger(t))
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v got %v", expected, got)
	}
}

func TestMaterials(t *testing.T) {
	taskrun := &v1beta1.TaskRun{
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
	}

	expected := []slsa.ProvenanceMaterial{
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
	}

	got := materials(objects.NewTaskRunObject(taskrun), logtesting.TestLogger(t))
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v got %v", expected, got)
	}

	// make sure this works with GIT* results as well
	taskRun := &v1beta1.TaskRun{
		Spec: v1beta1.TaskRunSpec{
			Params: []v1beta1.Param{
				{Name: "CHAINS-GIT_COMMIT", Value: v1beta1.ArrayOrString{StringVal: "my-commit"}},
				{Name: "CHAINS-GIT_URL", Value: v1beta1.ArrayOrString{StringVal: "github.com/something"}},
			},
		},
	}
	expected = []slsa.ProvenanceMaterial{
		{
			URI: "git+github.com/something.git",
			Digest: slsa.DigestSet{
				"sha1": "my-commit",
			},
		},
	}

	got = materials(objects.NewTaskRunObject(taskRun), logtesting.TestLogger(t))
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected %v got %v", expected, got)
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
    - "array"
  - name: my-empty-string-param
    value: ""
  - name: my-empty-array-param
    value: []
status:
  taskSpec:
    params:
    - name: my-param
      default: ignored
    - name: my-array-param
      type: array
      default:
      - "also"
      - "ignored"
    - name: my-default-param
      default: string-default-param
    - name: my-default-array-param
      type: array
      default:
      - "array"
      - "default"
      - "param"
    - name: my-empty-string-param
      default: "ignored"
    - name: my-empty-array-param
      type: array
      default:
      - "also"
      - "ignored"
    - name: my-default-empty-string-param
      default: ""
    - name: my-default-empty-array-param
      type: array
      default: []
`

	var taskRun *v1beta1.TaskRun
	if err := yaml.Unmarshal([]byte(taskrun), &taskRun); err != nil {
		t.Fatal(err)
	}

	expected := slsa.ProvenanceInvocation{
		Parameters: map[string]v1beta1.ArrayOrString{
			"my-param":                      {Type: "string", StringVal: "string-param"},
			"my-array-param":                {Type: "array", ArrayVal: []string{"my", "array"}},
			"my-default-param":              {Type: "string", StringVal: "string-default-param"},
			"my-default-array-param":        {Type: "array", ArrayVal: []string{"array", "default", "param"}},
			"my-empty-string-param":         {Type: "string", StringVal: ""},
			"my-empty-array-param":          {Type: "array", ArrayVal: []string{}},
			"my-default-empty-string-param": {Type: "string", StringVal: ""},
			"my-default-empty-array-param":  {Type: "array", ArrayVal: []string{}},
		},
	}

	got := invocation(objects.NewTaskRunObject(taskRun))
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
							Name: "nil-check",
						},
					}, {
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
						Value: *v1beta1.NewArrayOrString("registry/myimage"),
					},
					{
						Name:  "IMAGE_DIGEST",
						Value: *v1beta1.NewArrayOrString(digest1),
					},
					{
						Name:  "mvn1_ARTIFACT_URI",
						Value: *v1beta1.NewArrayOrString("maven-test-0.1.1.jar"),
					},
					{
						Name:  "mvn1_ARTIFACT_DIGEST",
						Value: *v1beta1.NewArrayOrString(digest3),
					},
					{
						Name:  "mvn1_pom_ARTIFACT_URI",
						Value: *v1beta1.NewArrayOrString("maven-test-0.1.1.pom"),
					},
					{
						Name:  "mvn1_pom_ARTIFACT_DIGEST",
						Value: *v1beta1.NewArrayOrString(digest4),
					},
					{
						Name:  "mvn1_src_ARTIFACT_URI",
						Value: *v1beta1.NewArrayOrString("maven-test-0.1.1-sources.jar"),
					},
					{
						Name:  "mvn1_src_ARTIFACT_DIGEST",
						Value: *v1beta1.NewArrayOrString(digest5),
					},
					{
						Name:  "invalid_ARTIFACT_DIGEST",
						Value: *v1beta1.NewArrayOrString(digest5),
					},
					{
						Name: "mvn1_pkg" + "-" + artifacts.ArtifactsOutputsResultName,
						Value: *v1beta1.NewObject(map[string]string{
							"uri":    "projects/test-project-1/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre",
							"digest": digest1,
						}),
					},
					{
						Name: "mvn1_pom_sha512" + "-" + artifacts.ArtifactsOutputsResultName,
						Value: *v1beta1.NewObject(map[string]string{
							"uri":    "com.google.guava:guava:1.0-jre.pom",
							"digest": digest2,
						}),
					},
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
			Name: "com.google.guava:guava:1.0-jre.pom",
			Digest: slsa.DigestSet{
				"sha256": strings.TrimPrefix(digest2, "sha256:"),
			},
		}, {
			Name: "index.docker.io/registry/myimage",
			Digest: slsa.DigestSet{
				"sha256": strings.TrimPrefix(digest1, "sha256:"),
			},
		}, {
			Name: "maven-test-0.1.1-sources.jar",
			Digest: slsa.DigestSet{
				"sha256": strings.TrimPrefix(digest5, "sha256:"),
			},
		}, {
			Name: "maven-test-0.1.1.jar",
			Digest: slsa.DigestSet{
				"sha256": strings.TrimPrefix(digest3, "sha256:"),
			},
		}, {
			Name: "maven-test-0.1.1.pom",
			Digest: slsa.DigestSet{
				"sha256": strings.TrimPrefix(digest4, "sha256:"),
			},
		}, {
			Name: "projects/test-project-1/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre",
			Digest: slsa.DigestSet{
				"sha256": strings.TrimPrefix(digest1, "sha256:"),
			},
		}, {
			Name: "registry/resource-image",
			Digest: slsa.DigestSet{
				"sha256": strings.TrimPrefix(digest2, "sha256:"),
			},
		},
	}
	tro := objects.NewTaskRunObject(tr)
	got := extract.SubjectDigests(tro, logtesting.TestLogger(t))
	if !reflect.DeepEqual(expected, got) {
		if d := cmp.Diff(expected, got); d != "" {
			t.Log(d)
		}
		t.Fatalf("expected \n%v\n got \n%v\n", expected, got)
	}
}
