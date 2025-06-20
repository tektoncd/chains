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
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/controller"
	"slices"
)

// PipelineRunInformerFilterFunc returns a filter function
// for PipelineRuns ensuring PipelineRuns are filtered by list of namespaces membership
func PipelineRunInformerFilterFunc(namespaces []string) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		// Namespace filter
		if len(namespaces) == 0 {
			return true
		}
		if pr, ok := obj.(*v1.PipelineRun); ok {
			if slices.Contains(namespaces, pr.Namespace) {
				return true
			}
		}
		return false
	}
}

// TaskRunInformerFilterFunc returns a filter function
// for TaskRuns ensuring TaskRuns are filtered by list of namespaces membership
func TaskRunInformerFilterFunc(namespaces []string) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		// Namespace filter
		if len(namespaces) == 0 {
			return true
		}
		if tr, ok := obj.(*v1.TaskRun); ok {
			if slices.Contains(namespaces, tr.Namespace) {
				return true
			}
		}
		return false
	}
}

// TaskRunInformerFilterFuncWithOwnership returns a filter function
// for TaskRuns ensuring Ownership by a PipelineRun and filtered by list of namespaces membership and
func TaskRunInformerFilterFuncWithOwnership(namespaces []string) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		// Ownership filter
		if !controller.FilterController(&v1.PipelineRun{})(obj) {
			return false
		}
		// Namespace filter
		if len(namespaces) == 0 {
			return true
		}
		if tr, ok := obj.(*v1.TaskRun); ok {
			if slices.Contains(namespaces, tr.Namespace) {
				return true
			}
		}
		return false
	}
}
