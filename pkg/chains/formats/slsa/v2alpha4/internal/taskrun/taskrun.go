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

package taskrun

import (
	"context"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	builddefinition "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/build_definition"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/provenance"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/results"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
)

const (
	taskRunResults     = "taskRunResults/%s"
	taskRunStepResults = "stepResults/%s"
)

// GenerateAttestation returns the provenance for the given taskrun in SALSA 1.0 format.
func GenerateAttestation(ctx context.Context, tro *objects.TaskRunObjectV1, slsaConfig *slsaconfig.SlsaConfig) (interface{}, error) {
	bp, err := byproducts(tro)
	if err != nil {
		return nil, err
	}

	resOpts := resolveddependencies.ResolveOptions{WithStepActionsResults: true}
	bd, err := builddefinition.GetTaskRunBuildDefinition(ctx, tro, slsaConfig.BuildType, resOpts)
	if err != nil {
		return nil, err
	}

	results := append(tro.GetResults(), tro.GetStepResults()...)
	sub := extract.SubjectsFromBuildArtifact(ctx, results)

	return provenance.GetSLSA1Statement(tro, sub, bd, bp, slsaConfig), nil
}

func byproducts(tro *objects.TaskRunObjectV1) ([]slsa.ResourceDescriptor, error) {
	byProd := []slsa.ResourceDescriptor{}

	res, err := results.GetResultsWithoutBuildArtifacts(tro.GetResults(), taskRunResults)
	if err != nil {
		return nil, err
	}
	byProd = append(byProd, res...)

	res, err = results.GetResultsWithoutBuildArtifacts(tro.GetStepResults(), taskRunStepResults)
	if err != nil {
		return nil, err
	}
	byProd = append(byProd, res...)

	return byProd, nil
}
