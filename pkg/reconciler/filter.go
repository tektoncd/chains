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
	"slices"

	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/controller"
)

func isManagedByAllowed(managedBy *string, allowed sets.Set[string]) bool {
	if managedBy == nil || *managedBy == "" || *managedBy == pipeline.ManagedBy {
		return true
	}
	return allowed.Has(*managedBy)
}

// PipelineRunInformerFilterFunc returns a filter function
// for PipelineRuns ensuring PipelineRuns are filtered by spec.managedBy value
// and list of namespaces membership
func PipelineRunInformerFilterFunc(namespaces []string, cfgStore *config.ConfigStore) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		pr, ok := obj.(*v1.PipelineRun)
		if !ok {
			return false
		}
		if !isManagedByAllowed(pr.Spec.ManagedBy, cfgStore.Load().Filter.ManagedByValues) {
			return false
		}
		if len(namespaces) == 0 {
			return true
		}
		return slices.Contains(namespaces, pr.Namespace)
	}
}

// TaskRunInformerFilterFunc returns a filter function
// for TaskRuns ensuring TaskRuns are filtered by spec.managedBy value
// and list of namespaces membership
func TaskRunInformerFilterFunc(namespaces []string, cfgStore *config.ConfigStore) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		tr, ok := obj.(*v1.TaskRun)
		if !ok {
			return false
		}
		if !isManagedByAllowed(tr.Spec.ManagedBy, cfgStore.Load().Filter.ManagedByValues) {
			return false
		}
		if len(namespaces) == 0 {
			return true
		}
		return slices.Contains(namespaces, tr.Namespace)
	}
}

// TaskRunInformerFilterFuncWithOwnership returns a filter function
// for TaskRuns ensuring Ownership by a PipelineRun and filtered by spec.managedBy value
// and list of namespaces membership
func TaskRunInformerFilterFuncWithOwnership(namespaces []string, cfgStore *config.ConfigStore) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		// Ownership filter
		if !controller.FilterController(&v1.PipelineRun{})(obj) {
			return false
		}
		tr, ok := obj.(*v1.TaskRun)
		if !ok {
			return false
		}
		if !isManagedByAllowed(tr.Spec.ManagedBy, cfgStore.Load().Filter.ManagedByValues) {
			return false
		}
		if len(namespaces) == 0 {
			return true
		}
		return slices.Contains(namespaces, tr.Namespace)
	}
}
