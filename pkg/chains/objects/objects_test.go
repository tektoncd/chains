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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
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

func getTaskRun() *v1beta1.TaskRun {
	return &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "objects-test",
			Labels: map[string]string{
				PipelineTaskLabel: "foo-task",
			},
		},
		Spec: v1beta1.TaskRunSpec{
			ServiceAccountName: "taskrun-sa",
			Params: []v1beta1.Param{
				{
					Name:  "runtime-param",
					Value: *v1beta1.NewStructuredValues("runtime-value"),
				},
			},
		},
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				Provenance: &v1beta1.Provenance{
					RefSource: &v1beta1.RefSource{
						URI:        "https://github.com/tektoncd/chains",
						Digest:     map[string]string{"sha1": "abcdef"},
						EntryPoint: "pkg/chains/objects.go",
					},
				},
				TaskSpec: &v1beta1.TaskSpec{
					Params: []v1beta1.ParamSpec{
						{
							Name:    "param1",
							Default: v1beta1.NewStructuredValues("default-value"),
						},
					},
				},
				TaskRunResults: []v1beta1.TaskRunResult{
					{
						Name: "img1_input_ARTIFACT_INPUTS",
						Value: *v1beta1.NewObject(map[string]string{
							"uri":    "gcr.io/foo/bar",
							"digest": "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7",
						}),
					},
					{Name: "mvn1_ARTIFACT_URI", Value: *v1beta1.NewStructuredValues("projects/test-project/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre")},
					{Name: "mvn1_ARTIFACT_DIGEST", Value: *v1beta1.NewStructuredValues("sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
				},
				Steps: []v1beta1.StepState{{
					ImageID: "step-image",
				}},
				Sidecars: []v1beta1.SidecarState{{
					ImageID: "sidecar-image",
				}},
			},
		},
	}
}

func getPipelineRun() *v1beta1.PipelineRun {
	return &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "objects-test",
		},
		Spec: v1beta1.PipelineRunSpec{
			ServiceAccountName: "pipelinerun-sa",
			Params: []v1beta1.Param{
				{
					Name:  "runtime-param",
					Value: *v1beta1.NewStructuredValues("runtime-value"),
				},
			},
		},
		Status: v1beta1.PipelineRunStatus{
			PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
				Provenance: &v1beta1.Provenance{
					RefSource: &v1beta1.RefSource{
						URI:        "https://github.com/tektoncd/chains",
						Digest:     map[string]string{"sha1": "abcdef"},
						EntryPoint: "pkg/chains/objects.go",
					},
				},
				PipelineSpec: &v1beta1.PipelineSpec{
					Params: []v1beta1.ParamSpec{
						{
							Name:    "param1",
							Default: v1beta1.NewStructuredValues("default-value"),
						},
					},
				},
				PipelineResults: []v1beta1.PipelineRunResult{
					{
						Name: "img1_input_ARTIFACT_INPUTS",
						Value: *v1beta1.NewObject(map[string]string{
							"uri":    "gcr.io/foo/bar",
							"digest": "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7",
						}),
					},
					{Name: "mvn1_ARTIFACT_URI", Value: *v1beta1.NewStructuredValues("projects/test-project/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre")},
					{Name: "mvn1_ARTIFACT_DIGEST", Value: *v1beta1.NewStructuredValues("sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
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
			tr := NewTaskRunObject(getTaskRun())
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
			pr := NewPipelineRunObject(getPipelineRun())
			pr.Spec.PodTemplate = tt.template
			secret := pr.GetPullSecrets()
			assert.ElementsMatch(t, secret, tt.want)
		})
	}
}

func TestPipelineRun_GetProvenance(t *testing.T) {

	t.Run("TestPipelineRun_GetProvenance", func(t *testing.T) {
		pr := NewPipelineRunObject(getPipelineRun())
		got := pr.GetProvenance()
		want := &v1beta1.Provenance{
			RefSource: &v1beta1.RefSource{
				URI:        "https://github.com/tektoncd/chains",
				Digest:     map[string]string{"sha1": "abcdef"},
				EntryPoint: "pkg/chains/objects.go",
			},
		}
		if d := cmp.Diff(want, got); d != "" {
			t.Fatalf("metadata (-want, +got):\n%s", d)
		}
	})

}

func TestTaskRun_GetProvenance(t *testing.T) {

	t.Run("TestTaskRun_GetProvenance", func(t *testing.T) {
		tr := NewTaskRunObject(getTaskRun())
		got := tr.GetProvenance()
		want := &v1beta1.Provenance{
			RefSource: &v1beta1.RefSource{
				URI:        "https://github.com/tektoncd/chains",
				Digest:     map[string]string{"sha1": "abcdef"},
				EntryPoint: "pkg/chains/objects.go",
			},
		}
		if d := cmp.Diff(want, got); d != "" {
			t.Fatalf("metadata (-want, +got):\n%s", d)
		}
	})

}

func TestPipelineRun_GetResults(t *testing.T) {

	t.Run("TestPipelineRun_GetResults", func(t *testing.T) {
		pr := NewPipelineRunObject(getPipelineRun())
		got := pr.GetResults()
		assert.ElementsMatch(t, got, []Result{
			{
				Name: "img1_input_ARTIFACT_INPUTS",
				Value: *v1beta1.NewObject(map[string]string{
					"uri":    "gcr.io/foo/bar",
					"digest": "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7",
				}),
			},
			{Name: "mvn1_ARTIFACT_URI", Value: *v1beta1.NewStructuredValues("projects/test-project/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre")},
			{Name: "mvn1_ARTIFACT_DIGEST", Value: *v1beta1.NewStructuredValues("sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
		})
	})

}

func TestTaskRun_GetStepImages(t *testing.T) {

	t.Run("TestTaskRun_GetStepImages", func(t *testing.T) {
		tr := NewTaskRunObject(getTaskRun())
		got := tr.GetStepImages()
		want := []string{"step-image"}
		if d := cmp.Diff(want, got); d != "" {
			t.Fatalf("metadata (-want, +got):\n%s", d)
		}
	})

}

func TestTaskRun_GetSidecarImages(t *testing.T) {

	t.Run("TestTaskRun_GetSidecarImages", func(t *testing.T) {
		tr := NewTaskRunObject(getTaskRun())
		got := tr.GetSidecarImages()
		want := []string{"sidecar-image"}
		if d := cmp.Diff(want, got); d != "" {
			t.Fatalf("metadata (-want, +got):\n%s", d)
		}
	})

}

func TestTaskRun_GetResults(t *testing.T) {

	t.Run("TestTaskRun_GetResults", func(t *testing.T) {
		pr := NewTaskRunObject(getTaskRun())
		got := pr.GetResults()
		assert.ElementsMatch(t, got, []Result{
			{
				Name: "img1_input_ARTIFACT_INPUTS",
				Value: *v1beta1.NewObject(map[string]string{
					"uri":    "gcr.io/foo/bar",
					"digest": "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b7",
				}),
			},
			{Name: "mvn1_ARTIFACT_URI", Value: *v1beta1.NewStructuredValues("projects/test-project/locations/us-west4/repositories/test-repo/mavenArtifacts/com.google.guava:guava:31.0-jre")},
			{Name: "mvn1_ARTIFACT_DIGEST", Value: *v1beta1.NewStructuredValues("sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5")},
		})
	})

}

func TestPipelineRun_GetGVK(t *testing.T) {
	assert.Equal(t, "tekton.dev/v1beta1/PipelineRun", NewPipelineRunObject(getPipelineRun()).GetGVK())
}

func TestTaskRun_GetGVK(t *testing.T) {
	assert.Equal(t, "tekton.dev/v1beta1/TaskRun", NewTaskRunObject(getTaskRun()).GetGVK())
}

func TestPipelineRun_GetKindName(t *testing.T) {
	assert.Equal(t, "pipelinerun", NewPipelineRunObject(getPipelineRun()).GetKindName())
}

func TestTaskRun_GetKindName(t *testing.T) {
	assert.Equal(t, "taskrun", NewTaskRunObject(getTaskRun()).GetKindName())
}

func TestPipelineRun_GetServiceAccountName(t *testing.T) {
	assert.Equal(t, "pipelinerun-sa", NewPipelineRunObject(getPipelineRun()).GetServiceAccountName())
}

func TestTaskRun_GetServiceAccountName(t *testing.T) {
	assert.Equal(t, "taskrun-sa", NewTaskRunObject(getTaskRun()).GetServiceAccountName())
}

func TestNewTektonObject(t *testing.T) {
	tro, err := NewTektonObject(getTaskRun())
	assert.NoError(t, err)
	assert.IsType(t, &TaskRunObject{}, tro)

	pro, err := NewTektonObject(getPipelineRun())
	assert.NoError(t, err)
	assert.IsType(t, &PipelineRunObject{}, pro)

	unknown, err := NewTektonObject("someting-else")
	assert.Nil(t, unknown)
	assert.ErrorContains(t, err, "unrecognized type")
}

func TestPipelineRun_GetTaskRunFromTask(t *testing.T) {
	pro := NewPipelineRunObject(getPipelineRun())

	assert.Nil(t, pro.GetTaskRunFromTask("missing"))
	assert.Nil(t, pro.GetTaskRunFromTask("foo-task"))

	pro.AppendTaskRun(getTaskRun())
	assert.Nil(t, pro.GetTaskRunFromTask("missing"))
	tr := pro.GetTaskRunFromTask("foo-task")
	assert.Equal(t, "foo", tr.Name)
}

func TestProvenanceExists(t *testing.T) {
	pro := NewPipelineRunObject(getPipelineRun())
	provenance := &v1beta1.Provenance{
		RefSource: &v1beta1.RefSource{
			URI: "tekton.com",
		},
	}
	pro.Status.Provenance = &v1beta1.Provenance{
		RefSource: &v1beta1.RefSource{
			URI: "tekton.com",
		},
	}
	assert.Equal(t, provenance, pro.GetProvenance())
}

func TestPipelineRunRemoteProvenance(t *testing.T) {
	pro := NewPipelineRunObject(getPipelineRun())
	provenance := &v1beta1.Provenance{
		RefSource: &v1beta1.RefSource{
			URI: "tekton.com",
		},
	}
	pro.Status.Provenance = &v1beta1.Provenance{
		RefSource: &v1beta1.RefSource{
			URI: "tekton.com",
		},
	}
	assert.Equal(t, provenance, pro.GetProvenance())
}

func TestTaskRunRemoteProvenance(t *testing.T) {
	tro := NewTaskRunObject(getTaskRun())
	provenance := &v1beta1.Provenance{
		RefSource: &v1beta1.RefSource{
			URI: "tekton.com",
		},
	}
	tro.Status.Provenance = &v1beta1.Provenance{
		RefSource: &v1beta1.RefSource{
			URI: "tekton.com",
		},
	}
	assert.Equal(t, provenance, tro.GetProvenance())
}

func TestPipelineRunIsRemote(t *testing.T) {
	pro := NewPipelineRunObject(getPipelineRun())
	pro.Spec.PipelineRef = &v1beta1.PipelineRef{
		ResolverRef: v1beta1.ResolverRef{
			Resolver: "Bundle",
		},
	}
	assert.Equal(t, true, pro.IsRemote())
}

func TestTaskRunIsRemote(t *testing.T) {
	tro := NewTaskRunObject(getTaskRun())
	tro.Spec.TaskRef = &v1beta1.TaskRef{
		ResolverRef: v1beta1.ResolverRef{
			Resolver: "Bundle",
		},
	}
	assert.Equal(t, true, tro.IsRemote())
}
