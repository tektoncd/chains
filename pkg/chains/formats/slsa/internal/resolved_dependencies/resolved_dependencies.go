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

package resolveddependencies

import (
	"encoding/json"

	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
)

const (
	// PipelineConfigName is the name of the resolved dependency of the pipelineRef.
	PipelineConfigName = "pipeline"
	// TaskConfigName is the name of the resolved dependency of the top level taskRef.
	TaskConfigName = "task"
	// PipelineTaskConfigName is the name of the resolved dependency of the pipeline task.
	PipelineTaskConfigName = "pipelineTask"
	// InputResultName is the name of the resolved dependency generated from Type hinted parameters or results.
	InputResultName = "inputs/result"
	// PipelineResourceName is the name of the resolved dependency of pipeline resource.
	PipelineResourceName = "pipelineResource"
)

// ConvertMaterialToResolvedDependency converts a SLSAv0.2 Material to a resolved dependency
func ConvertMaterialsToResolvedDependencies(mats []common.ProvenanceMaterial, name string) []v1.ResourceDescriptor {
	rds := []v1.ResourceDescriptor{}
	for _, mat := range mats {
		rd := v1.ResourceDescriptor{}
		rd.URI = mat.URI
		rd.Digest = mat.Digest
		if len(name) > 0 {
			rd.Name = name
		}
		rds = append(rds, rd)
	}
	return rds
}

// RemoveDuplicateResolvedDependencies removes duplicate resolved dependencies from the slice of resolved dependencies.
// Original order of resolved dependencies is retained.
func RemoveDuplicateResolvedDependencies(resolvedDependencies []v1.ResourceDescriptor) ([]v1.ResourceDescriptor, error) {
	out := make([]v1.ResourceDescriptor, 0, len(resolvedDependencies))

	// make map to store seen resolved dependencies
	seen := map[string]bool{}
	for _, resolvedDependency := range resolvedDependencies {
		// Since resolvedDependencies contain names, we want to ignore those while checking for duplicates.
		// Therefore, make a copy of the resolved dependency that only contains the uri and digest fields.
		rDep := v1.ResourceDescriptor{}
		rDep.URI = resolvedDependency.URI
		rDep.Digest = resolvedDependency.Digest
		// pipelinTasks store content with the slsa-tekton buildType
		rDep.Content = resolvedDependency.Content
		// This allows us to ignore dependencies that have the same uri and digest.
		rd, err := json.Marshal(rDep)
		if err != nil {
			return nil, err
		}
		if seen[string(rd)] {
			// We dont want to remove the top level pipeline/task config from the resolved dependencies
			// because its critical to provide that information in the provenance. In SLSAv0.2 spec,
			// we would put this in invocation.ConfigSource. In order to ensure that it is present in
			// the resolved dependencies, we dont want to skip it if another resolved dependency from the same
			// uri+digest pair was already included before.
			if !(resolvedDependency.Name == TaskConfigName || resolvedDependency.Name == PipelineConfigName) {
				continue
			}
		}
		seen[string(rd)] = true
		out = append(out, resolvedDependency)
	}
	return out, nil
}
