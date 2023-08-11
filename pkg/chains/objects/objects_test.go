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

	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"
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
		},
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
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
		},
		Status: v1beta1.PipelineRunStatus{
			PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
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

func TestSBOMObject(t *testing.T) {
	namespace := "test-namespace"
	serviceaccount := "test-serviceaccount"

	taskRun := getTaskRun()
	taskRun.ObjectMeta.Namespace = namespace
	taskRun.Spec.ServiceAccountName = serviceaccount
	taskRunObject := NewTaskRunObject(taskRun)

	sbomObject := NewSBOMObject("sbomURL", "sbomFormat", "imageURL", "imageDigest", taskRunObject)

	assert.Equal(t, "sbomURL", sbomObject.GetSBOMURL())
	assert.Equal(t, "sbomFormat", sbomObject.GetSBOMFormat())
	assert.Equal(t, "imageURL", sbomObject.GetImageURL())
	assert.Equal(t, "imageDigest", sbomObject.GetImageDigest())

	ctx, _ := rtesting.SetupFakeContext(t)
	kc := fakekubeclient.Get(ctx)
	if _, err := kc.CoreV1().ServiceAccounts(namespace).Create(ctx, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: serviceaccount, Namespace: namespace},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	got, err := sbomObject.OCIRemoteOption(ctx, kc)
	assert.NoError(t, err)
	// TODO: Not sure how to compare the returned remote.Option
	assert.NotNil(t, got)
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
