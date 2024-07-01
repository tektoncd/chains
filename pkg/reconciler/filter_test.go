package reconciler

import (
	"testing"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestPipelineRunInformerFilterFunc tests the PipelineRunInformerFilterFunc
//
//nolint:all
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
//
//nolint:all
func TestTaskRunInformerFilterFunc(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []string
		obj        interface{}
		expected   bool
	}{
		{
			name:       "Empty namespaces, should match",
			namespaces: []string{},
			obj:        &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   true,
		},
		{
			name:       "Matching namespace",
			namespaces: []string{"default", "test"},
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
