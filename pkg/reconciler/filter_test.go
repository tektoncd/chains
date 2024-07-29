/*
Copyright 2024 The Tekton Authors
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
package reconciler

import (
	"testing"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestPipelineRunInformerFilterFunc tests the PipelineRunInformerFilterFunc
func TestPipelineRunInformerFilterFunc(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []string
		obj        interface{}
		expected   bool
	}{
		{
			name:       "Empty namespaces, should match",
			namespaces: []string{},
			obj:        &v1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   true,
		},
		{
			name:       "Matching namespace",
			namespaces: []string{"default", "test"},
			obj:        &v1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   true,
		},
		{
			name:       "Non-matching namespace",
			namespaces: []string{"test"},
			obj:        &v1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   false,
		},
		{
			name:       "Non PipelineRun object",
			namespaces: []string{"default"},
			obj:        &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterFunc := PipelineRunInformerFilterFunc(tt.namespaces)
			result := filterFunc(tt.obj)
			if result != tt.expected {
				t.Errorf("Reconciler.PipelineRunInformerFilterFunc() result = %v, wanted %v", result, tt.expected)
			}
		})
	}
}

// TestTaskRunInformerFilterFunc tests the TaskRunInformerFilterFunc
func TestTaskRunInformerFilterFunc(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []string
		obj        interface{}
		expected   bool
	}{
		{
			name:       "Matching namespace",
			namespaces: []string{"default", "test"},
			obj:        &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   true,
		},
		{
			name:       "Empty namespaces, should match",
			namespaces: []string{},
			obj:        &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   true,
		},
		{
			name:       "Non-matching namespace",
			namespaces: []string{"test"},
			obj:        &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   false,
		},
		{
			name:       "Non TaskRun object",
			namespaces: []string{"default"},
			obj:        &v1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterFunc := TaskRunInformerFilterFunc(tt.namespaces)
			result := filterFunc(tt.obj)
			if result != tt.expected {
				t.Errorf("Reconciler.TaskRunInformerFilterFunc() result = %v, wanted %v", result, tt.expected)
			}
		})
	}
}

// TestTaskRunInformerFilterFuncWithOwnership tests the TaskRunInformerFilterFuncWithOwnership
func TestTaskRunInformerFilterFuncWithOwnership(t *testing.T) {
	boolValue := true
	tests := []struct {
		name       string
		namespaces []string
		obj        interface{}
		expected   bool
	}{
		{
			name:       "Empty namespaces and ownership, should match",
			namespaces: []string{},
			obj: &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: "tekton.dev/v1", Kind: "PipelineRun", Controller: &boolValue},
				},
			}},
			expected: true,
		},
		{
			name:       "Matching namespace and ownership",
			namespaces: []string{"default", "test"},
			obj: &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: "tekton.dev/v1", Kind: "PipelineRun", Controller: &boolValue},
				},
			}},
			expected: true,
		},
		{
			name:       "Non-matching namespace and ownership",
			namespaces: []string{"test"},
			obj: &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{APIVersion: "tekton.dev/v1", Kind: "PipelineRun", Controller: &boolValue},
				},
			}},
			expected: false,
		},
		{
			name:       "No ownership",
			namespaces: []string{"default"},
			obj:        &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterFunc := TaskRunInformerFilterFuncWithOwnership(tt.namespaces)
			result := filterFunc(tt.obj)
			if result != tt.expected {
				t.Errorf("Reconciler.TaskRunInformerFilterFuncWithOwnership() result = %v, wanted %v", result, tt.expected)
			}
		})
	}
}
