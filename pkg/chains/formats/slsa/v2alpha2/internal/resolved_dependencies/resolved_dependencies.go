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
	"context"
	"encoding/json"
	"fmt"

	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	v1 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/material"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

const (
	// pipelineConfigName is the name of the resolved dependency of the pipelineRef.
	pipelineConfigName = "pipeline"
	// taskConfigName is the name of the resolved dependency of the top level taskRef.
	taskConfigName = "task"
	// pipelineTaskConfigName is the name of the resolved dependency of the pipeline task.
	pipelineTaskConfigName = "pipelineTask"
	// inputResultName is the name of the resolved dependency generated from Type hinted parameters or results.
	inputResultName = "inputs/result"
	// pipelineResourceName is the name of the resolved dependency of pipeline resource.
	pipelineResourceName = "pipelineResource"
	// JsonMediaType is the media type of json encoded content used in resource descriptors
	JsonMediaType = "application/json"
)

// TaskRun constructs `predicate.resolvedDependencies` section by collecting all the artifacts that influence a taskrun such as source code repo and step&sidecar base images.
func TaskRun(ctx context.Context, tro *objects.TaskRunObject) ([]v1.ResourceDescriptor, error) {
	var resolvedDependencies []v1.ResourceDescriptor
	var err error

	// add top level task config
	if p := tro.Status.Provenance; p != nil && p.RefSource != nil {
		rd := v1.ResourceDescriptor{
			Name:   taskConfigName,
			URI:    p.RefSource.URI,
			Digest: p.RefSource.Digest,
		}
		resolvedDependencies = append(resolvedDependencies, rd)
	}

	mats := []common.ProvenanceMaterial{}

	// add step and sidecar images
	if err := material.AddStepImagesToMaterials(tro.Status.Steps, &mats); err != nil {
		return nil, err
	}
	if err := material.AddSidecarImagesToMaterials(tro.Status.Sidecars, &mats); err != nil {
		return nil, err
	}
	resolvedDependencies = append(resolvedDependencies, convertMaterialsToResolvedDependencies(mats, "")...)

	mats = material.AddMaterialsFromTaskParamsAndResults(ctx, tro)
	// convert materials to resolved dependencies
	resolvedDependencies = append(resolvedDependencies, convertMaterialsToResolvedDependencies(mats, inputResultName)...)

	// add task resources
	mats = material.AddTaskResourcesToMaterials(ctx, tro, []common.ProvenanceMaterial{})
	// convert materials to resolved dependencies
	resolvedDependencies = append(resolvedDependencies, convertMaterialsToResolvedDependencies(mats, pipelineResourceName)...)

	// remove duplicate resolved dependencies
	resolvedDependencies, err = removeDuplicateResolvedDependencies(resolvedDependencies)
	if err != nil {
		return nil, err
	}
	// add on-cluster task
	rd, err := getLocalTaskRunRef(tro)
	if err != nil {
		return nil, err
	}
	if rd.Content != nil {
		resolvedDependencies = append(resolvedDependencies, rd)
	}
	return resolvedDependencies, nil
}

// PipelineRun constructs `predicate.resolvedDependencies` section by collecting all the artifacts that influence a pipeline run such as source code repo and step&sidecar base images.
func PipelineRun(ctx context.Context, pro *objects.PipelineRunObject) ([]v1.ResourceDescriptor, error) {
	var err error
	var resolvedDependencies []v1.ResourceDescriptor
	logger := logging.FromContext(ctx)

	// add pipeline config to resolved dependencies
	if p := pro.Status.Provenance; p != nil && p.RefSource != nil {
		rd := v1.ResourceDescriptor{
			Name:   pipelineConfigName,
			URI:    p.RefSource.URI,
			Digest: p.RefSource.Digest,
		}
		resolvedDependencies = append(resolvedDependencies, rd)
	}

	// add resolved dependencies from pipeline tasks
	resolvedDependencies, err = addPipelineTask(logger, pro, resolvedDependencies)
	if err != nil {
		return nil, err
	}

	// add resolved dependencies from pipeline results
	mats := material.AddMaterialsFromPipelineParamsAndResults(ctx, pro, []common.ProvenanceMaterial{})
	// convert materials to resolved dependencies
	resolvedDependencies = append(resolvedDependencies, convertMaterialsToResolvedDependencies(mats, inputResultName)...)

	// remove duplicate resolved dependencies
	resolvedDependencies, err = removeDuplicateResolvedDependencies(resolvedDependencies)
	if err != nil {
		return nil, err
	}

	// add on-cluster pipeline and tasks
	rd, err := getLocalPipelineRunRefs(pro)
	if err != nil {
		return nil, err
	}
	resolvedDependencies = append(resolvedDependencies, rd...)
	return resolvedDependencies, nil
}

// getLocalTaskRunRef adds the spec of on-cluster referenced task used by the taskrun
func getLocalTaskRunRef(tro *objects.TaskRunObject) (v1.ResourceDescriptor, error) {
	// resolvedDependencies := []v1.ResourceDescriptor{}
	tSpec := tro.Status.TaskSpec
	if tSpec != nil && tro.Spec.TaskRef != nil && tro.Spec.TaskRef.Resolver == "" {
		content, err := json.Marshal(tSpec)
		if err != nil {
			return v1.ResourceDescriptor{}, err
		}
		return v1.ResourceDescriptor{
			Name:      fmt.Sprintf("%s/%s", taskConfigName, tro.Spec.TaskRef.Name),
			Content:   content,
			MediaType: JsonMediaType,
		}, nil
	}
	return v1.ResourceDescriptor{}, nil
}

// getLocalPipelineRunRefs adds the spec of on-cluster referenced pipelines and pipeline tasks used by the pipeline run
func getLocalPipelineRunRefs(pro *objects.PipelineRunObject) ([]v1.ResourceDescriptor, error) {
	resolvedDependencies := []v1.ResourceDescriptor{}
	pSpec := pro.Status.PipelineSpec
	if pSpec != nil {
		if pro.Spec.PipelineRef != nil && pro.Spec.PipelineRef.Resolver == "" {
			content, err := json.Marshal(pSpec)
			if err != nil {
				return nil, err
			}
			rd := v1.ResourceDescriptor{
				Name:      fmt.Sprintf("%s/%s", pipelineConfigName, pro.Spec.PipelineRef.Name),
				Content:   content,
				MediaType: JsonMediaType,
			}
			resolvedDependencies = append(resolvedDependencies, rd)
		}
		pipelineTasks := append(pSpec.Tasks, pSpec.Finally...)
		for _, t := range pipelineTasks {
			tr := pro.GetTaskRunFromTask(t.Name)
			tSpec := tr.Status.TaskSpec
			if tSpec != nil {
				if tr.Spec.TaskRef != nil && tr.Spec.TaskRef.Resolver == "" {
					content, err := json.Marshal(tSpec)
					if err != nil {
						return nil, err
					}
					rd := v1.ResourceDescriptor{
						Name:      fmt.Sprintf("%s/%s", pipelineTaskConfigName, t.Name),
						Content:   content,
						MediaType: JsonMediaType,
					}
					resolvedDependencies = append(resolvedDependencies, rd)
				}
			}
		}
	}
	return resolvedDependencies, nil
}

// convertMaterialToResolvedDependency converts a SLSAv0.2 Material to a resolved dependency
func convertMaterialsToResolvedDependencies(mats []common.ProvenanceMaterial, name string) []v1.ResourceDescriptor {
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

// removeDuplicateResolvedDependencies removes duplicate resolved dependencies from the slice of resolved dependencies.
// Original order of resolved dependencies is retained.
func removeDuplicateResolvedDependencies(resolvedDependencies []v1.ResourceDescriptor) ([]v1.ResourceDescriptor, error) {
	out := make([]v1.ResourceDescriptor, 0, len(resolvedDependencies))

	// make map to store seen resolved dependencies
	seen := map[string]bool{}
	for _, resolvedDependency := range resolvedDependencies {
		// Since resolvedDependencies contain names, we want to igmore those while checking for duplicates.
		// Therefore, make a copy of the resolved dependency that only contains the uri and digest fields.
		rDep := v1.ResourceDescriptor{}
		rDep.URI = resolvedDependency.URI
		rDep.Digest = resolvedDependency.Digest
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
			if !(resolvedDependency.Name == taskConfigName || resolvedDependency.Name == pipelineConfigName) {
				continue
			}
		}
		seen[string(rd)] = true
		out = append(out, resolvedDependency)
	}
	return out, nil
}

// addPipelineTask adds the resolved dependencies from pipeline tasks
// such as pipeline task uri/digest for remote pipeline tasks and step and sidecar images.
func addPipelineTask(logger *zap.SugaredLogger, pro *objects.PipelineRunObject, resolvedDependencies []v1.ResourceDescriptor) ([]v1.ResourceDescriptor, error) {
	pSpec := pro.Status.PipelineSpec
	if pSpec != nil {
		pipelineTasks := append(pSpec.Tasks, pSpec.Finally...)
		for _, t := range pipelineTasks {
			tr := pro.GetTaskRunFromTask(t.Name)
			// Ignore Tasks that did not execute during the PipelineRun.
			if tr == nil || tr.Status.CompletionTime == nil {
				logger.Infof("taskrun status not found for task %s", t.Name)
				continue
			}
			// add remote task configsource information in materials
			if tr.Status.Provenance != nil && tr.Status.Provenance.RefSource != nil {
				rd := v1.ResourceDescriptor{
					Name:   pipelineTaskConfigName,
					URI:    tr.Status.Provenance.RefSource.URI,
					Digest: tr.Status.Provenance.RefSource.Digest,
				}
				resolvedDependencies = append(resolvedDependencies, rd)
			}

			mats := []common.ProvenanceMaterial{}

			// add step images
			if err := material.AddStepImagesToMaterials(tr.Status.Steps, &mats); err != nil {
				return nil, err
			}

			// add sidecar images
			if err := material.AddSidecarImagesToMaterials(tr.Status.Sidecars, &mats); err != nil {
				return nil, err
			}

			// convert materials to resolved dependencies
			resolvedDependencies = append(resolvedDependencies, convertMaterialsToResolvedDependencies(mats, "")...)
		}
	}
	return resolvedDependencies, nil
}
