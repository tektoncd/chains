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

const (
	namespace  = "objects-test"
	pullSecret = "pull-secret"
)

var (
	tr = &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: namespace,
		},
		Spec: v1beta1.TaskRunSpec{},
	}
	pr = &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: namespace,
		},
		Spec: v1beta1.PipelineRunSpec{},
	}
	templatePullSecret = &pod.PodTemplate{
		ImagePullSecrets: []corev1.LocalObjectReference{
			{
				Name: pullSecret,
			},
		},
	}
	emptyTemplate = &pod.PodTemplate{}
)

func TestTaskRun_ImagePullSecrets(t *testing.T) {

	tests := []struct {
		name     string
		taskRun  *TaskRunObject
		template *pod.PodTemplate
		want     []string
	}{
		{
			name:     "Test pull secret found",
			taskRun:  NewTaskRunObject(tr),
			template: templatePullSecret,
			want:     []string{pullSecret},
		},
		{
			name:     "Test pull secret missing",
			taskRun:  NewTaskRunObject(tr),
			template: nil,
			want:     []string{},
		},
		{
			name:     "Test podTemplate missing",
			taskRun:  NewTaskRunObject(tr),
			template: emptyTemplate,
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.taskRun.Spec.PodTemplate = tt.template
			secret := tt.taskRun.GetPullSecrets()
			assert.ElementsMatch(t, secret, tt.want)
		})
	}

}

func TestPipelineRun_ImagePullSecrets(t *testing.T) {
	tests := []struct {
		name        string
		pipelineRun *PipelineRunObject
		template    *pod.PodTemplate
		want        []string
	}{
		{
			name:        "Test pull secret found",
			pipelineRun: NewPipelineRunObject(pr),
			template:    templatePullSecret,
			want:        []string{pullSecret},
		},
		{
			name:        "Test pull secret missing",
			pipelineRun: NewPipelineRunObject(pr),
			template:    nil,
			want:        []string{},
		},
		{
			name:        "Test podTemplate missing",
			pipelineRun: NewPipelineRunObject(pr),
			template:    emptyTemplate,
			want:        []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.pipelineRun.Spec.PodTemplate = tt.template
			secret := tt.pipelineRun.GetPullSecrets()
			assert.ElementsMatch(t, secret, tt.want)
		})
	}

}
