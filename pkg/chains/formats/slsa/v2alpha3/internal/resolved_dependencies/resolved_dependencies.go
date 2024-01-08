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
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/material"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

// used to toggle the fields in resolvedDependencies. see AddTektonTaskDescriptor
// and AddSLSATaskDescriptor
type addTaskDescriptorContent func(*objects.TaskRunObjectV1) (*slsa.ResourceDescriptor, error) //nolint:staticcheck

// the more verbose resolved dependency content. this adds the name, uri, digest
// and content if possible.
func AddTektonTaskDescriptor(tr *objects.TaskRunObjectV1) (*slsa.ResourceDescriptor, error) { //nolint:staticcheck
	rd := slsa.ResourceDescriptor{}
	storedTr, err := json.Marshal(tr)
	if err != nil {
		return nil, err
	}

	rd.Name = resolveddependencies.PipelineTaskConfigName
	rd.Content = storedTr
	if tr.Status.Provenance != nil && tr.Status.Provenance.RefSource != nil {
		rd.URI = tr.Status.Provenance.RefSource.URI
		rd.Digest = tr.Status.Provenance.RefSource.Digest
	}

	return &rd, nil
}

// resolved dependency content for the more generic slsa verifiers. just logs
// the name, uri and digest.
func AddSLSATaskDescriptor(tr *objects.TaskRunObjectV1) (*slsa.ResourceDescriptor, error) { //nolint:staticcheck
	if tr.Status.Provenance != nil && tr.Status.Provenance.RefSource != nil {
		return &slsa.ResourceDescriptor{
			Name:   resolveddependencies.PipelineTaskConfigName,
			URI:    tr.Status.Provenance.RefSource.URI,
			Digest: tr.Status.Provenance.RefSource.Digest,
		}, nil
	}
	return nil, nil
}

// fromPipelineTask adds the resolved dependencies from pipeline tasks
// such as pipeline task uri/digest for remote pipeline tasks and step and sidecar images.
func fromPipelineTask(logger *zap.SugaredLogger, pro *objects.PipelineRunObjectV1, addTasks addTaskDescriptorContent) ([]slsa.ResourceDescriptor, error) {
	pSpec := pro.Status.PipelineSpec
	resolvedDependencies := []slsa.ResourceDescriptor{}
	if pSpec != nil {
		pipelineTasks := append(pSpec.Tasks, pSpec.Finally...)
		for _, t := range pipelineTasks {
			tr := pro.GetTaskRunFromTask(t.Name)
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
				resolvedDependencies = append(resolvedDependencies, *rd)
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
			resolvedDependencies = append(resolvedDependencies, resolveddependencies.ConvertMaterialsToResolvedDependencies(mats, "")...)
		}
	}
	return resolvedDependencies, nil
}

// taskDependencies gather all dependencies in a task and adds them to resolvedDependencies
func taskDependencies(ctx context.Context, tro *objects.TaskRunObjectV1) ([]slsa.ResourceDescriptor, error) {
	var resolvedDependencies []slsa.ResourceDescriptor
	var err error
	mats := []common.ProvenanceMaterial{}

	// add step and sidecar images
	stepMaterials, err := material.FromStepImages(tro)
	mats = append(mats, stepMaterials...)
	if err != nil {
		return nil, err
	}
	sidecarMaterials, err := material.FromSidecarImages(tro)
	if err != nil {
		return nil, err
	}
	mats = append(mats, sidecarMaterials...)
	resolvedDependencies = append(resolvedDependencies, resolveddependencies.ConvertMaterialsToResolvedDependencies(mats, "")...)

	mats = material.FromTaskParamsAndResults(ctx, tro)
	// convert materials to resolved dependencies
	resolvedDependencies = append(resolvedDependencies, resolveddependencies.ConvertMaterialsToResolvedDependencies(mats, resolveddependencies.InputResultName)...)

	// add task resources
	// =====
	// convert to v1beta1 and add any task resources
	serializedResources := tro.Annotations["tekton.dev/v1beta1-spec-resources"]
	var resources v1beta1.TaskRunResources //nolint:staticcheck
	shouldReplace := false
	if err := json.Unmarshal([]byte(serializedResources), &resources); err == nil {
		shouldReplace = true

	}
	trV1Beta1 := &v1beta1.TaskRun{} //nolint:staticcheck
	fmt.Printf("%v", tro.GetObject().(*v1.TaskRun))
	if err := trV1Beta1.ConvertFrom(ctx, tro.GetObject().(*v1.TaskRun)); err == nil {
		if shouldReplace {
			trV1Beta1.Spec.Resources = &resources //nolint:staticcheck
		}
		mats = material.FromTaskResources(ctx, trV1Beta1)

	}

	// convert materials to resolved dependencies
	resolvedDependencies = append(resolvedDependencies,
		resolveddependencies.ConvertMaterialsToResolvedDependencies(mats, resolveddependencies.PipelineResourceName)...)

	// remove duplicate resolved dependencies
	resolvedDependencies, err = resolveddependencies.RemoveDuplicateResolvedDependencies(resolvedDependencies)
	if err != nil {
		return nil, err
	}

	return resolvedDependencies, nil
}

// TaskRun constructs `predicate.resolvedDependencies` section by collecting all the artifacts that influence a taskrun such as source code repo and step&sidecar base images.
func TaskRun(ctx context.Context, tro *objects.TaskRunObjectV1) ([]slsa.ResourceDescriptor, error) {
	var resolvedDependencies []slsa.ResourceDescriptor
	var err error

	// add top level task config
	if p := tro.Status.Provenance; p != nil && p.RefSource != nil {
		rd := slsa.ResourceDescriptor{
			Name:   resolveddependencies.TaskConfigName,
			URI:    p.RefSource.URI,
			Digest: p.RefSource.Digest,
		}
		resolvedDependencies = append(resolvedDependencies, rd)
	}

	rds, err := taskDependencies(ctx, tro)
	if err != nil {
		return nil, err
	}
	resolvedDependencies = append(resolvedDependencies, rds...)

	return resolvedDependencies, nil
}

// PipelineRun constructs `predicate.resolvedDependencies` section by collecting all the artifacts that influence a pipeline run such as source code repo and step&sidecar base images.
func PipelineRun(ctx context.Context, pro *objects.PipelineRunObjectV1, slsaconfig *slsaconfig.SlsaConfig, addTasks addTaskDescriptorContent) ([]slsa.ResourceDescriptor, error) {
	var err error
	var resolvedDependencies []slsa.ResourceDescriptor
	logger := logging.FromContext(ctx)

	// add pipeline config to resolved dependencies
	if p := pro.Status.Provenance; p != nil && p.RefSource != nil {
		rd := slsa.ResourceDescriptor{
			Name:   resolveddependencies.PipelineConfigName,
			URI:    p.RefSource.URI,
			Digest: p.RefSource.Digest,
		}
		resolvedDependencies = append(resolvedDependencies, rd)
	}

	// add resolved dependencies from pipeline tasks
	rds, err := fromPipelineTask(logger, pro, addTasks)
	if err != nil {
		return nil, err
	}
	resolvedDependencies = append(resolvedDependencies, rds...)

	// add resolved dependencies from pipeline results
	mats := material.FromPipelineParamsAndResults(ctx, pro, slsaconfig)
	// convert materials to resolved dependencies
	resolvedDependencies = append(resolvedDependencies, resolveddependencies.ConvertMaterialsToResolvedDependencies(mats, resolveddependencies.InputResultName)...)

	// remove duplicate resolved dependencies
	resolvedDependencies, err = resolveddependencies.RemoveDuplicateResolvedDependencies(resolvedDependencies)
	if err != nil {
		return nil, err
	}
	return resolvedDependencies, nil
}
