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

	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	buildtypes "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/build_types"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/material"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"knative.dev/pkg/logging"
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
	// v1beta1SpecResourceLabel is the name of the label that contains information about TaskRun resources.
	v1beta1SpecResourceLabel = "tekton.dev/v1beta1-spec-resources"
)

// AddTaskDescriptorContent is used to toggle the fields in  see AddTektonTaskDescriptor and AddSLSATaskDescriptor
type AddTaskDescriptorContent func(*objects.TaskRunObjectV1) (*intoto.ResourceDescriptor, error)

// ResolveOptions represents the configuration to be use to resolve dependencies.
type ResolveOptions struct {
	// Indicates if StepActions type-hinted results should be read to resolve dependecies.
	WithStepActionsResults bool
}

// ConvertMaterialsToResolvedDependencies converts a SLSAv0.2 Material to a resolved dependency
func ConvertMaterialsToResolvedDependencies(mats []common.ProvenanceMaterial, name string) []*intoto.ResourceDescriptor {
	rds := []*intoto.ResourceDescriptor{}
	for _, mat := range mats {
		rd := intoto.ResourceDescriptor{}
		rd.Uri = mat.URI
		rd.Digest = mat.Digest
		if len(name) > 0 {
			rd.Name = name
		}
		rds = append(rds, &rd)
	}
	return rds
}

// RemoveDuplicateResolvedDependencies removes duplicate resolved dependencies from the slice of resolved dependencies.
// Original order of resolved dependencies is retained.
func RemoveDuplicateResolvedDependencies(resolvedDependencies []*intoto.ResourceDescriptor) ([]*intoto.ResourceDescriptor, error) {
	out := make([]*intoto.ResourceDescriptor, 0, len(resolvedDependencies))

	// make map to store seen resolved dependencies
	seen := map[string]bool{}
	for _, resolvedDependency := range resolvedDependencies {
		// Since resolvedDependencies contain names, we want to ignore those while checking for duplicates.
		// Therefore, make a copy of the resolved dependency that only contains the uri and digest fields.
		rDep := intoto.ResourceDescriptor{}
		rDep.Uri = resolvedDependency.Uri
		rDep.Digest = resolvedDependency.Digest
		// pipelinTasks store content with the slsa-tekton buildType
		rDep.Content = resolvedDependency.Content
		// This allows us to ignore dependencies that have the same uri and digest.
		rd, err := protojson.Marshal(&rDep)
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

// AddTektonTaskDescriptor returns the more verbose resolved dependency content. this adds the name, uri, digest
// and content if possible.
func AddTektonTaskDescriptor(tr *objects.TaskRunObjectV1) (*intoto.ResourceDescriptor, error) {
	rd := intoto.ResourceDescriptor{}
	storedTr, err := json.Marshal(tr)
	if err != nil {
		return nil, err
	}

	rd.Name = PipelineTaskConfigName
	rd.Content = storedTr
	if tr.Status.Provenance != nil && tr.Status.Provenance.RefSource != nil {
		rd.Uri = tr.Status.Provenance.RefSource.URI
		rd.Digest = tr.Status.Provenance.RefSource.Digest
	}

	return &rd, nil
}

// AddSLSATaskDescriptor resolves dependency content for the more generic slsa verifiers. just logs
// the name, uri and digest.
func AddSLSATaskDescriptor(tr *objects.TaskRunObjectV1) (*intoto.ResourceDescriptor, error) {
	if tr.Status.Provenance != nil && tr.Status.Provenance.RefSource != nil {
		return &intoto.ResourceDescriptor{
			Name:   PipelineTaskConfigName,
			Uri:    tr.Status.Provenance.RefSource.URI,
			Digest: tr.Status.Provenance.RefSource.Digest,
		}, nil
	}
	return nil, nil
}

// fromPipelineTask adds the resolved dependencies from pipeline tasks
// such as pipeline task uri/digest for remote pipeline tasks and step and sidecar images.
func fromPipelineTask(logger *zap.SugaredLogger, pro *objects.PipelineRunObjectV1, addTasks AddTaskDescriptorContent) ([]*intoto.ResourceDescriptor, error) {
	pSpec := pro.Status.PipelineSpec
	resolvedDependencies := []*intoto.ResourceDescriptor{}
	if pSpec != nil {
		pipelineTasks := pSpec.Tasks
		pipelineTasks = append(pipelineTasks, pSpec.Finally...)
		for _, t := range pipelineTasks {
			taskRuns := pro.GetTaskRunsFromTask(t.Name)
			if len(taskRuns) == 0 {
				logger.Infof("no taskruns found for task %s", t.Name)
				continue
			}
			for _, tr := range taskRuns {
				// Ignore Tasks that did not execute during the PipelineRun.
				if tr == nil || tr.Status.CompletionTime == nil {
					logger.Infof("taskrun status not found for task %s", t.Name)
					continue
				}
				rd, err := addTasks(tr)
				if err != nil {
					logger.Errorf("error storing taskRun %s, error: %s", t.Name, err)
					continue
				}
				if rd != nil {
					resolvedDependencies = append(resolvedDependencies, rd)
				}

				mats := []common.ProvenanceMaterial{}

				// add step images
				stepMaterials, err := material.FromStepImages(tr)
				if err != nil {
					return nil, err
				}
				mats = append(mats, stepMaterials...)

				// add sidecar images
				sidecarMaterials, err := material.FromSidecarImages(tr)
				if err != nil {
					return nil, err
				}
				mats = append(mats, sidecarMaterials...)

				// convert materials to resolved dependencies
				resolvedDependencies = append(resolvedDependencies, ConvertMaterialsToResolvedDependencies(mats, "")...)

			}
		}
	}
	return resolvedDependencies, nil
}

// taskDependencies gather all dependencies in a task and adds them to resolvedDependencies
func taskDependencies(ctx context.Context, opts ResolveOptions, tro *objects.TaskRunObjectV1) ([]*intoto.ResourceDescriptor, error) {
	var resolvedDependencies []*intoto.ResourceDescriptor
	var err error
	mats := []common.ProvenanceMaterial{}

	// add step and sidecar images
	stepMaterials, err := material.FromStepImages(tro)
	if err != nil {
		return nil, err
	}
	mats = append(mats, stepMaterials...)

	sidecarMaterials, err := material.FromSidecarImages(tro)
	if err != nil {
		return nil, err
	}
	mats = append(mats, sidecarMaterials...)
	resolvedDependencies = append(resolvedDependencies, ConvertMaterialsToResolvedDependencies(mats, "")...)

	if opts.WithStepActionsResults {
		mats = material.FromStepActionsResults(ctx, tro)
		resolvedDependencies = append(resolvedDependencies, ConvertMaterialsToResolvedDependencies(mats, InputResultName)...)
	}

	mats = material.FromTaskParamsAndResults(ctx, tro)
	// convert materials to resolved dependencies
	resolvedDependencies = append(resolvedDependencies, ConvertMaterialsToResolvedDependencies(mats, InputResultName)...)

	// add task resources
	// =====
	// convert to v1beta1 and add any task resources
	serializedResources := tro.Annotations[v1beta1SpecResourceLabel]
	var resources v1beta1.TaskRunResources //nolint:staticcheck
	shouldReplace := false
	if err := json.Unmarshal([]byte(serializedResources), &resources); err == nil {
		shouldReplace = true

	}
	trV1Beta1 := &v1beta1.TaskRun{} //nolint:staticcheck
	if err := trV1Beta1.ConvertFrom(ctx, tro.GetObject().(*v1.TaskRun)); err == nil {
		if shouldReplace {
			trV1Beta1.Spec.Resources = &resources //nolint:staticcheck
		}
		mats = material.FromTaskResources(ctx, trV1Beta1)

	}

	// convert materials to resolved dependencies
	resolvedDependencies = append(resolvedDependencies,
		ConvertMaterialsToResolvedDependencies(mats, PipelineResourceName)...)

	// remove duplicate resolved dependencies
	resolvedDependencies, err = RemoveDuplicateResolvedDependencies(resolvedDependencies)
	if err != nil {
		return nil, err
	}

	return resolvedDependencies, nil
}

// TaskRun constructs `predicate.resolvedDependencies` section by collecting all the artifacts that influence a taskrun such as source code repo and step&sidecar base images.
func TaskRun(ctx context.Context, opts ResolveOptions, tro *objects.TaskRunObjectV1) ([]*intoto.ResourceDescriptor, error) {
	var resolvedDependencies []*intoto.ResourceDescriptor
	var err error

	// add top level task config
	if p := tro.Status.Provenance; p != nil && p.RefSource != nil {
		rd := intoto.ResourceDescriptor{
			Name:   TaskConfigName,
			Uri:    p.RefSource.URI,
			Digest: p.RefSource.Digest,
		}
		resolvedDependencies = append(resolvedDependencies, &rd)
	}

	rds, err := taskDependencies(ctx, opts, tro)
	if err != nil {
		return nil, err
	}
	resolvedDependencies = append(resolvedDependencies, rds...)

	return resolvedDependencies, nil
}

// PipelineRun constructs `predicate.resolvedDependencies` section by collecting all the artifacts that influence a pipeline run such as source code repo and step&sidecar base images.
func PipelineRun(ctx context.Context, pro *objects.PipelineRunObjectV1, slsaconfig *slsaconfig.SlsaConfig, opts ResolveOptions, addTasks AddTaskDescriptorContent) ([]*intoto.ResourceDescriptor, error) {
	var err error
	var resolvedDependencies []*intoto.ResourceDescriptor
	logger := logging.FromContext(ctx)

	// add pipeline config to resolved dependencies
	if p := pro.Status.Provenance; p != nil && p.RefSource != nil {
		rd := intoto.ResourceDescriptor{
			Name:   PipelineConfigName,
			Uri:    p.RefSource.URI,
			Digest: p.RefSource.Digest,
		}
		resolvedDependencies = append(resolvedDependencies, &rd)
	}

	// add resolved dependencies from pipeline tasks
	rds, err := fromPipelineTask(logger, pro, addTasks)
	if err != nil {
		return nil, err
	}
	resolvedDependencies = append(resolvedDependencies, rds...)

	if slsaconfig.DeepInspectionEnabled && opts.WithStepActionsResults {
		execTasks := pro.GetExecutedTasks()
		for _, task := range execTasks {
			stepActionMat := material.FromStepActionsResults(ctx, task)
			resolvedDependencies = append(resolvedDependencies, ConvertMaterialsToResolvedDependencies(stepActionMat, InputResultName)...)
		}
	}

	// add resolved dependencies from pipeline results
	mats := material.FromPipelineParamsAndResults(ctx, pro, slsaconfig)
	// convert materials to resolved dependencies
	resolvedDependencies = append(resolvedDependencies, ConvertMaterialsToResolvedDependencies(mats, InputResultName)...)

	// remove duplicate resolved dependencies
	resolvedDependencies, err = RemoveDuplicateResolvedDependencies(resolvedDependencies)
	if err != nil {
		return nil, err
	}
	return resolvedDependencies, nil
}

// GetTaskDescriptor returns the conrresponding addTaskDescriptor function according to the given build type.
func GetTaskDescriptor(buildDefinition string) (AddTaskDescriptorContent, error) {
	switch buildDefinition {
	case buildtypes.SlsaBuildType:
		return AddSLSATaskDescriptor, nil
	case buildtypes.TektonBuildType:
		return AddTektonTaskDescriptor, nil
	default:
		return nil, fmt.Errorf("unsupported buildType %v", buildDefinition)
	}
}
