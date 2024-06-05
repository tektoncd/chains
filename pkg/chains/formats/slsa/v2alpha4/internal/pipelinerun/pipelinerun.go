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

package pipelinerun

import (
	"context"

	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/artifact"
	builddefinition "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/build_definition"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/provenance"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/results"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha4/internal/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
)

const (
	pipelineRunResults = "pipelineRunResults/%s"
	// JSONMediaType is the media type of json encoded content used in resource descriptors
	JSONMediaType = "application/json"
)

// GenerateAttestation generates a provenance statement with SLSA v1.0 predicate for a pipeline run.
func GenerateAttestation(ctx context.Context, pro *objects.PipelineRunObjectV1, slsaconfig *slsaconfig.SlsaConfig) (interface{}, error) {
	bp, err := byproducts(pro, slsaconfig)
	if err != nil {
		return nil, err
	}

	opts := resolveddependencies.ResolveOptions{WithStepActionsResults: true}
	bd, err := builddefinition.GetPipelineRunBuildDefinition(ctx, pro, slsaconfig, opts)
	if err != nil {
		return nil, err
	}

	sub := subjectDigests(ctx, pro, slsaconfig)

	return provenance.GetSLSA1Statement(pro, sub, &bd, bp, slsaconfig)
}

// byproducts contains the pipelineRunResults that are not subjects.
func byproducts(pro *objects.PipelineRunObjectV1, slsaconfig *slsaconfig.SlsaConfig) ([]*intoto.ResourceDescriptor, error) {
	byProd, err := results.GetResultsWithoutBuildArtifacts(pro.GetResults(), pipelineRunResults)
	if err != nil {
		return nil, err
	}

	if !slsaconfig.DeepInspectionEnabled {
		return byProd, nil
	}

	for _, tro := range pro.GetExecutedTasks() {
		taskProds, err := taskrun.ByProducts(tro)
		if err != nil {
			return nil, err
		}
		byProd = append(byProd, taskProds...)
	}

	return byProd, nil
}

func subjectDigests(ctx context.Context, pro *objects.PipelineRunObjectV1, slsaconfig *slsaconfig.SlsaConfig) []*intoto.ResourceDescriptor {
	subjects := extract.SubjectsFromBuildArtifact(ctx, pro.GetResults())

	if !slsaconfig.DeepInspectionEnabled {
		return subjects
	}

	for _, task := range pro.GetExecutedTasks() {
		subjects = artifact.AppendSubjects(subjects, taskrun.SubjectDigests(ctx, task)...)
	}

	return subjects
}
