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
	"strings"
	"testing"

	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	logtesting "knative.dev/pkg/logging/testing"
)

func newTestConfigStore(t *testing.T, managedByValues sets.Set[string]) *config.ConfigStore {
	t.Helper()
	logger := logtesting.TestLogger(t)
	store := config.NewConfigStore(logger)

	data := map[string]string{}
	if managedByValues.Len() > 0 {
		data["filter.managed-by"] = strings.Join(sets.List(managedByValues), ",")
	}

	store.OnConfigChanged(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: config.ChainsConfig},
		Data:       data,
	})
	return store
}

func defaultTestConfigStore(t *testing.T) *config.ConfigStore {
	t.Helper()
	return newTestConfigStore(t, sets.New[string]("tekton.dev/pipeline"))
}

func productionDefaultConfigStore(t *testing.T) *config.ConfigStore {
	t.Helper()
	return newTestConfigStore(t, nil)
}

// TestIsManagedByAllowed tests the isManagedByAllowed helper
func TestIsManagedByAllowed(t *testing.T) {
	allowed := sets.New[string]("tekton.dev/pipeline")

	tests := []struct {
		name      string
		managedBy *string
		allowed   sets.Set[string]
		expected  bool
	}{
		{
			name:      "nil managedBy is allowed",
			managedBy: nil,
			allowed:   allowed,
			expected:  true,
		},
		{
			name:      "empty string managedBy is allowed",
			managedBy: ptr.To(""),
			allowed:   allowed,
			expected:  true,
		},
		{
			name:      "tekton.dev/pipeline is allowed",
			managedBy: ptr.To("tekton.dev/pipeline"),
			allowed:   allowed,
			expected:  true,
		},
		{
			name:      "custom controller is rejected",
			managedBy: ptr.To("custom-controller"),
			allowed:   allowed,
			expected:  false,
		},
		{
			name:      "custom controller is allowed when configured",
			managedBy: ptr.To("custom-controller"),
			allowed:   sets.New[string]("tekton.dev/pipeline", "custom-controller"),
			expected:  true,
		},
		{
			name:      "nil allowlist rejects custom controller",
			managedBy: ptr.To("custom-controller"),
			allowed:   nil,
			expected:  false,
		},
		{
			name:      "nil allowlist accepts tekton.dev/pipeline",
			managedBy: ptr.To("tekton.dev/pipeline"),
			allowed:   nil,
			expected:  true,
		},
		{
			name:      "nil allowlist accepts nil managedBy",
			managedBy: nil,
			allowed:   nil,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isManagedByAllowed(tt.managedBy, tt.allowed)
			if result != tt.expected {
				t.Errorf("isManagedByAllowed() = %v, wanted %v", result, tt.expected)
			}
		})
	}
}

// TestPipelineRunInformerFilterFunc tests the PipelineRunInformerFilterFunc
func TestPipelineRunInformerFilterFunc(t *testing.T) {
	tests := []struct {
		name       string
		namespaces []string
		obj        interface{}
		cfgStore   *config.ConfigStore
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
		{
			name:       "managedBy nil, should match",
			namespaces: []string{},
			obj:        &v1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   true,
		},
		{
			name:       "managedBy empty string, should match",
			namespaces: []string{},
			obj: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.PipelineRunSpec{ManagedBy: ptr.To("")},
			},
			expected: true,
		},
		{
			name:       "managedBy tekton.dev/pipeline, should match",
			namespaces: []string{},
			obj: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.PipelineRunSpec{ManagedBy: ptr.To("tekton.dev/pipeline")},
			},
			expected: true,
		},
		{
			name:       "managedBy custom controller, should not match",
			namespaces: []string{},
			obj: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.PipelineRunSpec{ManagedBy: ptr.To("custom-controller")},
			},
			expected: false,
		},
		{
			name:       "managedBy custom controller with custom config, should match",
			namespaces: []string{},
			obj: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.PipelineRunSpec{ManagedBy: ptr.To("custom-controller")},
			},
			cfgStore: newTestConfigStore(t, sets.New[string]("tekton.dev/pipeline", "custom-controller")),
			expected: true,
		},
		{
			name:       "matching namespace but custom managedBy, should not match",
			namespaces: []string{"default"},
			obj: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.PipelineRunSpec{ManagedBy: ptr.To("custom-controller")},
			},
			expected: false,
		},
		{
			name:       "non-matching namespace but tekton managedBy, should not match",
			namespaces: []string{"test"},
			obj: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.PipelineRunSpec{ManagedBy: ptr.To("tekton.dev/pipeline")},
			},
			expected: false,
		},
		{
			name:       "production default config rejects custom managedBy",
			namespaces: []string{},
			obj: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.PipelineRunSpec{ManagedBy: ptr.To("custom-controller")},
			},
			cfgStore: productionDefaultConfigStore(t),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgStore := tt.cfgStore
			if cfgStore == nil {
				cfgStore = defaultTestConfigStore(t)
			}
			filterFunc := PipelineRunInformerFilterFunc(tt.namespaces, cfgStore)
			result := filterFunc(tt.obj)
			if result != tt.expected {
				t.Errorf("PipelineRunInformerFilterFunc() result = %v, wanted %v", result, tt.expected)
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
		cfgStore   *config.ConfigStore
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
		{
			name:       "managedBy nil, should match",
			namespaces: []string{},
			obj:        &v1.TaskRun{ObjectMeta: metav1.ObjectMeta{Namespace: "default"}},
			expected:   true,
		},
		{
			name:       "managedBy empty string, should match",
			namespaces: []string{},
			obj: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.TaskRunSpec{ManagedBy: ptr.To("")},
			},
			expected: true,
		},
		{
			name:       "managedBy tekton.dev/pipeline, should match",
			namespaces: []string{},
			obj: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.TaskRunSpec{ManagedBy: ptr.To("tekton.dev/pipeline")},
			},
			expected: true,
		},
		{
			name:       "managedBy custom controller, should not match",
			namespaces: []string{},
			obj: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.TaskRunSpec{ManagedBy: ptr.To("custom-controller")},
			},
			expected: false,
		},
		{
			name:       "managedBy custom controller with custom config, should match",
			namespaces: []string{},
			obj: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.TaskRunSpec{ManagedBy: ptr.To("custom-controller")},
			},
			cfgStore: newTestConfigStore(t, sets.New[string]("tekton.dev/pipeline", "custom-controller")),
			expected: true,
		},
		{
			name:       "matching namespace but custom managedBy, should not match",
			namespaces: []string{"default"},
			obj: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.TaskRunSpec{ManagedBy: ptr.To("custom-controller")},
			},
			expected: false,
		},
		{
			name:       "non-matching namespace but tekton managedBy, should not match",
			namespaces: []string{"test"},
			obj: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.TaskRunSpec{ManagedBy: ptr.To("tekton.dev/pipeline")},
			},
			expected: false,
		},
		{
			name:       "production default config rejects custom managedBy",
			namespaces: []string{},
			obj: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec:       v1.TaskRunSpec{ManagedBy: ptr.To("custom-controller")},
			},
			cfgStore: productionDefaultConfigStore(t),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgStore := tt.cfgStore
			if cfgStore == nil {
				cfgStore = defaultTestConfigStore(t)
			}
			filterFunc := TaskRunInformerFilterFunc(tt.namespaces, cfgStore)
			result := filterFunc(tt.obj)
			if result != tt.expected {
				t.Errorf("TaskRunInformerFilterFunc() result = %v, wanted %v", result, tt.expected)
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
		cfgStore   *config.ConfigStore
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
		{
			name:       "managedBy custom controller with ownership, should not match",
			namespaces: []string{},
			obj: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{APIVersion: "tekton.dev/v1", Kind: "PipelineRun", Controller: &boolValue},
					},
				},
				Spec: v1.TaskRunSpec{ManagedBy: ptr.To("custom-controller")},
			},
			expected: false,
		},
		{
			name:       "managedBy tekton.dev/pipeline with ownership, should match",
			namespaces: []string{},
			obj: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					OwnerReferences: []metav1.OwnerReference{
						{APIVersion: "tekton.dev/v1", Kind: "PipelineRun", Controller: &boolValue},
					},
				},
				Spec: v1.TaskRunSpec{ManagedBy: ptr.To("tekton.dev/pipeline")},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgStore := tt.cfgStore
			if cfgStore == nil {
				cfgStore = defaultTestConfigStore(t)
			}
			filterFunc := TaskRunInformerFilterFuncWithOwnership(tt.namespaces, cfgStore)
			result := filterFunc(tt.obj)
			if result != tt.expected {
				t.Errorf("TaskRunInformerFilterFuncWithOwnership() result = %v, wanted %v", result, tt.expected)
			}
		})
	}
}
