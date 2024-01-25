/*
Copyright 2023 The Tekton Authors

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

package externalparameters

import (
	externalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/external_parameters"
	"github.com/tektoncd/chains/pkg/chains/objects"
)

// PipelineRun adds the pipeline run spec and provenance if available
func PipelineRun(pro *objects.PipelineRunObjectV1) map[string]any {
	externalParams := make(map[string]any)

	if provenance := pro.GetRemoteProvenance(); provenance != nil {
		externalParams["buildConfigSource"] = externalparameters.BuildConfigSource(provenance)
	}
	externalParams["runSpec"] = pro.Spec
	return externalParams
}

// TaskRun adds the task run spec and provenance if available
func TaskRun(tro *objects.TaskRunObjectV1) map[string]any {
	externalParams := make(map[string]any)

	if provenance := tro.GetRemoteProvenance(); provenance != nil {
		externalParams["buildConfigSource"] = externalparameters.BuildConfigSource(provenance)
	}
	externalParams["runSpec"] = tro.Spec
	return externalParams
}
