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

package taskrun

import (
	"context"
	"encoding/json"
	"fmt"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	buildtypes "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha3/internal/build_types"
	externalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha3/internal/external_parameters"
	internalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha3/internal/internal_parameters"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha3/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/objects"
)

const taskRunResults = "taskRunResults/%s"

// GenerateAttestation generates a provenance statement with SLSA v1.0 predicate for a task run.
func GenerateAttestation(ctx context.Context, tro *objects.TaskRunObjectV1, slsaConfig *slsaconfig.SlsaConfig) (interface{}, error) {
	bp, err := byproducts(tro)
	if err != nil {
		return nil, err
	}

	bd, err := getBuildDefinition(ctx, slsaConfig.BuildType, tro)
	if err != nil {
		return nil, err
	}

	att := intoto.ProvenanceStatementSLSA1{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       extract.SubjectDigests(ctx, tro, slsaConfig),
		},
		Predicate: slsa.ProvenancePredicate{
			BuildDefinition: bd,
			RunDetails: slsa.ProvenanceRunDetails{
				Builder: slsa.Builder{
					ID: slsaConfig.BuilderID,
				},
				BuildMetadata: metadata(tro),
				Byproducts:    bp,
			},
		},
	}
	return att, nil
}

func metadata(tro *objects.TaskRunObjectV1) slsa.BuildMetadata {
	m := slsa.BuildMetadata{
		InvocationID: string(tro.ObjectMeta.UID),
	}
	if tro.Status.StartTime != nil {
		utc := tro.Status.StartTime.Time.UTC()
		m.StartedOn = &utc
	}
	if tro.Status.CompletionTime != nil {
		utc := tro.Status.CompletionTime.Time.UTC()
		m.FinishedOn = &utc
	}
	return m
}

// byproducts contains the taskRunResults
func byproducts(tro *objects.TaskRunObjectV1) ([]slsa.ResourceDescriptor, error) {
	byProd := []slsa.ResourceDescriptor{}
	for _, key := range tro.Status.Results {
		content, err := json.Marshal(key.Value)
		if err != nil {
			return nil, err
		}
		bp := slsa.ResourceDescriptor{
			Name:      fmt.Sprintf(taskRunResults, key.Name),
			Content:   content,
			MediaType: "application/json",
		}
		byProd = append(byProd, bp)
	}
	return byProd, nil
}

// getBuildDefinition get the buildDefinition based on the configured buildType. This will default to the slsa buildType
func getBuildDefinition(ctx context.Context, buildType string, tro *objects.TaskRunObjectV1) (slsa.ProvenanceBuildDefinition, error) {
	// if buildType is not set in the chains-config, default to slsa build type
	buildDefinitionType := buildType
	if buildType == "" {
		buildDefinitionType = buildtypes.SlsaBuildType
	}

	switch buildDefinitionType {
	case buildtypes.SlsaBuildType:
		rd, err := resolveddependencies.TaskRun(ctx, tro)
		if err != nil {
			return slsa.ProvenanceBuildDefinition{}, err
		}
		return slsa.ProvenanceBuildDefinition{
			BuildType:            buildDefinitionType,
			ExternalParameters:   externalparameters.TaskRun(tro),
			InternalParameters:   internalparameters.SLSAInternalParameters(tro),
			ResolvedDependencies: rd,
		}, nil
	case buildtypes.TektonBuildType:
		rd, err := resolveddependencies.TaskRun(ctx, tro)
		if err != nil {
			return slsa.ProvenanceBuildDefinition{}, err
		}
		return slsa.ProvenanceBuildDefinition{
			BuildType:            buildDefinitionType,
			ExternalParameters:   externalparameters.TaskRun(tro),
			InternalParameters:   internalparameters.TektonInternalParameters(tro),
			ResolvedDependencies: rd,
		}, nil
	default:
		return slsa.ProvenanceBuildDefinition{}, fmt.Errorf("unsupported buildType %v", buildType)
	}
}
