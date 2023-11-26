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

package objects

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getPullSecretTemplate(pullSecret string) *pod.PodTemplate {
	return &pod.PodTemplate{
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: pullSecret,
			},
		},
	}
}

func getEmptyTemplate() *pod.PodTemplate {
	return &pod.PodTemplate{}
}

func getTaskRun() *v1.TaskRun {
	return &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "objects-test",
			Labels: map[string]string{
				PipelineTaskLabel: "foo-task",
			},
		},
		Spec: v1.TaskRunSpec{
			ServiceAccountName: "taskrun-sa",
			Params: []v1.Param{
				{
					Name:  "runtime-param",
					Value: *v1.NewStructuredValues("runtime-value"),
				},
			},
		},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				Provenance: &v1.Provenance{
					RefSource: &v1.RefSource{
						URI:        "https://github.com/tektoncd/chains",
						Digest:     map[string]string{"sha1": "abcdef"},
						EntryPoint: "pkg/chains/objects.go",
					},
				},
				TaskSpec: &v1.TaskSpec{
					Params: []v1.ParamSpec{
						{
							Name:    "param1",
							Default: v1.NewStructuredValues("default-value"),
						},
					},
				},
				Results: []v1.TaskRunResult{
					{
						Name: "img1_input_ARTIFACT_INPUTS",
						Value: *v1.NewObject(map[string]string{
							"uri":    "gcr.io/foo/bar",
							"digest": "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7",
						}),
					},
					{Name: "mvn1_ARTIFACT_URI", Value: *v1.NewStructuredValues("projects/test-project/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre")},
					{Name: "mvn1_ARTIFACT_DIGEST", Value: *v1.NewStructuredValues("sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
				},
				Steps: []v1.StepState{{
					ImageID: "step-image",
				}},
				Sidecars: []v1.SidecarState{{
					ImageID: "sidecar-image",
				}},
			},
		},
	}
}

func getPipelineRun() *v1.PipelineRun {
	return &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "objects-test",
		},
		Spec: v1.PipelineRunSpec{
			TaskRunTemplate: v1.PipelineTaskRunTemplate{
				ServiceAccountName: "pipelinerun-sa",
			},
			Params: []v1.Param{
				{
					Name:  "runtime-param",
					Value: *v1.NewStructuredValues("runtime-value"),
				},
			},
		},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				Provenance: &v1.Provenance{
					RefSource: &v1.RefSource{
						URI:        "https://github.com/tektoncd/chains",
						Digest:     map[string]string{"sha1": "abcdef"},
						EntryPoint: "pkg/chains/objects.go",
					},
				},
				PipelineSpec: &v1.PipelineSpec{
					Params: []v1.ParamSpec{
						{
							Name:    "param1",
							Default: v1.NewStructuredValues("default-value"),
						},
					},
				},
				Results: []v1.PipelineRunResult{
					{
						Name: "img1_input_ARTIFACT_INPUTS",
						Value: *v1.NewObject(map[string]string{
							"uri":    "gcr.io/foo/bar",
							"digest": "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7",
						}),
					},
					{Name: "mvn1_ARTIFACT_URI", Value: *v1.NewStructuredValues("projects/test-project/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre")},
					{Name: "mvn1_ARTIFACT_DIGEST", Value: *v1.NewStructuredValues("sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
				},
			},
		},
	}
}

func TestTaskRun_ImagePullSecrets(t *testing.T) {
	pullSecret := "pull-secret"

	tests := []struct {
		name     string
		template *pod.PodTemplate
		want     []string
	}{
		{
			name:     "Test pull secret found",
			template: getPullSecretTemplate(pullSecret),
			want:     []string{pullSecret},
		},
		{
			name:     "Test pull secret missing",
			template: nil,
			want:     []string{},
		},
		{
			name:     "Test podTemplate missing",
			template: getEmptyTemplate(),
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTaskRunObjectV1(getTaskRun())
			tr.Spec.PodTemplate = tt.template
			secret := tr.GetPullSecrets()
			assert.ElementsMatch(t, secret, tt.want)
		})
	}

}

func TestPipelineRun_ImagePullSecrets(t *testing.T) {
	pullSecret := "pull-secret"

	tests := []struct {
		name     string
		template *pod.PodTemplate
		want     []string
	}{
		{
			name:     "Test pull secret found",
			template: getPullSecretTemplate(pullSecret),
			want:     []string{pullSecret},
		},
		{
			name:     "Test pull secret missing",
			template: nil,
			want:     []string{},
		},
		{
			name:     "Test podTemplate missing",
			template: getEmptyTemplate(),
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := NewPipelineRunObjectV1(getPipelineRun())
			pr.Spec.TaskRunTemplate.PodTemplate = tt.template
			secret := pr.GetPullSecrets()
			assert.ElementsMatch(t, secret, tt.want)
		})
	}
}

func TestPipelineRun_GetProvenance(t *testing.T) {

	t.Run("TestPipelineRun_GetProvenance", func(t *testing.T) {
		pr := NewPipelineRunObjectV1(getPipelineRun())
		got := pr.GetProvenance()
		want := &ProvenanceV1{&v1.Provenance{
			RefSource: &v1.RefSource{
				URI:        "https://github.com/tektoncd/chains",
				Digest:     map[string]string{"sha1": "abcdef"},
				EntryPoint: "pkg/chains/objects.go",
			},
		}}
		if d := cmp.Diff(want, got); d != "" {
			t.Fatalf("metadata (-want, +got):\n%s", d)
		}
	})

}

func TestTaskRun_GetProvenance(t *testing.T) {

	t.Run("TestTaskRun_GetProvenance", func(t *testing.T) {
		tr := NewTaskRunObjectV1(getTaskRun())
		got := tr.GetProvenance()
		want := &ProvenanceV1{&v1.Provenance{
			RefSource: &v1.RefSource{
				URI:        "https://github.com/tektoncd/chains",
				Digest:     map[string]string{"sha1": "abcdef"},
				EntryPoint: "pkg/chains/objects.go",
			},
		}}
		if d := cmp.Diff(want, got); d != "" {
			t.Fatalf("metadata (-want, +got):\n%s", d)
		}
	})

}

func TestPipelineRun_GetResults(t *testing.T) {

	t.Run("TestPipelineRun_GetResults", func(t *testing.T) {
		pr := NewPipelineRunObjectV1(getPipelineRun())
		got := pr.GetResults()
		assert.ElementsMatch(t, got, []GenericResult{
			ResultV1{
				Name: "img1_input_ARTIFACT_INPUTS",
				Value: *v1.NewObject(map[string]string{
					"uri":    "gcr.io/foo/bar",
					"digest": "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7",
				}),
			},
			ResultV1{Name: "mvn1_ARTIFACT_URI", Value: *v1.NewStructuredValues("projects/test-project/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre")},
			ResultV1{Name: "mvn1_ARTIFACT_DIGEST", Value: *v1.NewStructuredValues("sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
		})
	})

}

func TestTaskRun_GetStepImages(t *testing.T) {

	t.Run("TestTaskRun_GetStepImages", func(t *testing.T) {
		tr := NewTaskRunObjectV1(getTaskRun())
		got := tr.GetStepImages()
		want := []string{"step-image"}
		if d := cmp.Diff(want, got); d != "" {
			t.Fatalf("metadata (-want, +got):\n%s", d)
		}
	})

}

func TestTaskRun_GetSidecarImages(t *testing.T) {

	t.Run("TestTaskRun_GetSidecarImages", func(t *testing.T) {
		tr := NewTaskRunObjectV1(getTaskRun())
		got := tr.GetSidecarImages()
		want := []string{"sidecar-image"}
		if d := cmp.Diff(want, got); d != "" {
			t.Fatalf("metadata (-want, +got):\n%s", d)
		}
	})

}

func TestTaskRun_GetResults(t *testing.T) {

	t.Run("TestTaskRun_GetResults", func(t *testing.T) {
		pr := NewTaskRunObjectV1(getTaskRun())
		got := pr.GetResults()
		assert.ElementsMatch(t, got, []GenericResult{
			ResultV1{
				Name: "img1_input_ARTIFACT_INPUTS",
				Value: *v1.NewObject(map[string]string{
					"uri":    "gcr.io/foo/bar",
					"digest": "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7",
				}),
			},
			ResultV1{Name: "mvn1_ARTIFACT_URI", Value: *v1.NewStructuredValues("projects/test-project/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre")},
			ResultV1{Name: "mvn1_ARTIFACT_DIGEST", Value: *v1.NewStructuredValues("sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
		})
	})

}

func TestPipelineRun_GetGVK(t *testing.T) {
	assert.Equal(t, "tekton.dev/v1/PipelineRun", NewPipelineRunObjectV1(getPipelineRun()).GetGVK())
}

func TestTaskRun_GetGVK(t *testing.T) {
	assert.Equal(t, "tekton.dev/v1/TaskRun", NewTaskRunObjectV1(getTaskRun()).GetGVK())
}

func TestPipelineRun_GetKindName(t *testing.T) {
	assert.Equal(t, "pipelinerun", NewPipelineRunObjectV1(getPipelineRun()).GetKindName())
}

func TestTaskRun_GetKindName(t *testing.T) {
	assert.Equal(t, "taskrun", NewTaskRunObjectV1(getTaskRun()).GetKindName())
}

func TestPipelineRun_GetServiceAccountName(t *testing.T) {
	assert.Equal(t, "pipelinerun-sa", NewPipelineRunObjectV1(getPipelineRun()).GetServiceAccountName())
}

func TestTaskRun_GetServiceAccountName(t *testing.T) {
	assert.Equal(t, "taskrun-sa", NewTaskRunObjectV1(getTaskRun()).GetServiceAccountName())
}

func TestNewTektonObject(t *testing.T) {
	tro, err := NewTektonObject(getTaskRun())
	assert.NoError(t, err)
	assert.IsType(t, &TaskRunObjectV1{}, tro)

	pro, err := NewTektonObject(getPipelineRun())
	assert.NoError(t, err)
	assert.IsType(t, &PipelineRunObjectV1{}, pro)

	unknown, err := NewTektonObject("someting-else")
	assert.Nil(t, unknown)
	assert.ErrorContains(t, err, "unrecognized type")
}

func TestPipelineRun_GetTaskRunFromTask(t *testing.T) {
	pro := NewPipelineRunObjectV1(getPipelineRun())

	assert.Nil(t, pro.GetTaskRunFromTask("missing"))
	assert.Nil(t, pro.GetTaskRunFromTask("foo-task"))

	pro.AppendTaskRun(getTaskRun())
	assert.Nil(t, pro.GetTaskRunFromTask("missing"))
	tr := pro.GetTaskRunFromTask("foo-task")
	assert.Equal(t, "foo", tr.Name)
}

func TestProvenanceExists(t *testing.T) {
	pro := NewPipelineRunObjectV1(getPipelineRun())
	provenance := &ProvenanceV1{&v1.Provenance{
		RefSource: &v1.RefSource{
			URI: "tekton.com",
		},
	}}
	pro.Status.Provenance = &v1.Provenance{
		RefSource: &v1.RefSource{
			URI: "tekton.com",
		},
	}
	assert.Equal(t, provenance, pro.GetProvenance())
}

func TestPipelineRunRemoteProvenance(t *testing.T) {
	pro := NewPipelineRunObjectV1(getPipelineRun())
	provenance := &ProvenanceV1{&v1.Provenance{
		RefSource: &v1.RefSource{
			URI: "tekton.com",
		},
	}}
	pro.Status.Provenance = &v1.Provenance{
		RefSource: &v1.RefSource{
			URI: "tekton.com",
		},
	}
	assert.Equal(t, provenance, pro.GetProvenance())
}

func TestTaskRunRemoteProvenance(t *testing.T) {
	tro := NewTaskRunObjectV1(getTaskRun())
	provenance := &ProvenanceV1{&v1.Provenance{
		RefSource: &v1.RefSource{
			URI: "tekton.com",
		},
	}}
	tro.Status.Provenance = &v1.Provenance{
		RefSource: &v1.RefSource{
			URI: "tekton.com",
		},
	}
	assert.Equal(t, provenance, tro.GetProvenance())
}

func TestPipelineRunIsRemote(t *testing.T) {
	pro := NewPipelineRunObjectV1(getPipelineRun())
	pro.Spec.PipelineRef = &v1.PipelineRef{
		ResolverRef: v1.ResolverRef{
			Resolver: "Bundle",
		},
	}
	assert.Equal(t, true, pro.IsRemote())
}

func TestTaskRunIsRemote(t *testing.T) {
	tro := NewTaskRunObjectV1(getTaskRun())
	tro.Spec.TaskRef = &v1.TaskRef{
		ResolverRef: v1.ResolverRef{
			Resolver: "Bundle",
		},
	}
	assert.Equal(t, true, tro.IsRemote())
}
