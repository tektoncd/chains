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
		},
		Spec: v1beta1.TaskRunSpec{},
	}
}

func getPipelineRun() *v1beta1.PipelineRun {
	return &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "objects-test",
		},
		Spec: v1beta1.PipelineRunSpec{},
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
