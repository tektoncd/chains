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
	"encoding/json"
	"fmt"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	buildtypes "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/build_types"
	externalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/external_parameters"
	internalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/internal_parameters"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/objects"
)

const (
	pipelineRunResults = "pipelineRunResults/%s"
	// JsonMediaType is the media type of json encoded content used in resource descriptors
	JsonMediaType = "application/json"
)

// GenerateAttestation generates a provenance statement with SLSA v1.0 predicate for a pipeline run.
func GenerateAttestation(ctx context.Context, pro *objects.PipelineRunObjectV1Beta1, slsaconfig *slsaconfig.SlsaConfig) (interface{}, error) {
	bp, err := byproducts(pro)
	if err != nil {
		return nil, err
	}

	bd, err := getBuildDefinition(ctx, slsaconfig, pro)
	if err != nil {
		return nil, err
	}

	att := intoto.ProvenanceStatementSLSA1{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       extract.SubjectDigests(ctx, pro, slsaconfig),
		},
		Predicate: slsa.ProvenancePredicate{
			BuildDefinition: bd,
			RunDetails: slsa.ProvenanceRunDetails{
				Builder: slsa.Builder{
					ID: slsaconfig.BuilderID,
				},
				BuildMetadata: metadata(pro),
				Byproducts:    bp,
			},
		},
	}
	return att, nil
}

func metadata(pro *objects.PipelineRunObjectV1Beta1) slsa.BuildMetadata {
	m := slsa.BuildMetadata{
		InvocationID: string(pro.ObjectMeta.UID),
	}
	if pro.Status.StartTime != nil {
		utc := pro.Status.StartTime.Time.UTC()
		m.StartedOn = &utc
	}
	if pro.Status.CompletionTime != nil {
		utc := pro.Status.CompletionTime.Time.UTC()
		m.FinishedOn = &utc
	}
	return m
}

// byproducts contains the pipelineRunResults
func byproducts(pro *objects.PipelineRunObjectV1Beta1) ([]slsa.ResourceDescriptor, error) {
	byProd := []slsa.ResourceDescriptor{}
	for _, key := range pro.Status.PipelineResults {
		content, err := json.Marshal(key.Value)
		if err != nil {
			return nil, err
		}
		bp := slsa.ResourceDescriptor{
			Name:      fmt.Sprintf(pipelineRunResults, key.Name),
			Content:   content,
			MediaType: JsonMediaType,
		}
		byProd = append(byProd, bp)
	}
	return byProd, nil
}

// getBuildDefinition get the buildDefinition based on the configured buildType. This will default to the slsa buildType
func getBuildDefinition(ctx context.Context, slsaconfig *slsaconfig.SlsaConfig, pro *objects.PipelineRunObjectV1Beta1) (slsa.ProvenanceBuildDefinition, error) {
	// if buildType is not set in the chains-config, default to slsa build type
	buildDefinitionType := slsaconfig.BuildType
	if slsaconfig.BuildType == "" {
		buildDefinitionType = buildtypes.SlsaBuildType
	}

	switch buildDefinitionType {
	case buildtypes.SlsaBuildType:
		rd, err := resolveddependencies.PipelineRun(ctx, pro, slsaconfig, resolveddependencies.AddSLSATaskDescriptor)
		if err != nil {
			return slsa.ProvenanceBuildDefinition{}, err
		}
		return slsa.ProvenanceBuildDefinition{
			BuildType:            buildDefinitionType,
			ExternalParameters:   externalparameters.PipelineRun(pro),
			InternalParameters:   internalparameters.SLSAInternalParameters(pro),
			ResolvedDependencies: rd,
		}, nil
	case buildtypes.TektonBuildType:
		rd, err := resolveddependencies.PipelineRun(ctx, pro, slsaconfig, resolveddependencies.AddTektonTaskDescriptor)
		if err != nil {
			return slsa.ProvenanceBuildDefinition{}, err
		}
		return slsa.ProvenanceBuildDefinition{
			BuildType:            buildDefinitionType,
			ExternalParameters:   externalparameters.PipelineRun(pro),
			InternalParameters:   internalparameters.TektonInternalParameters(pro),
			ResolvedDependencies: rd,
		}, nil
	default:
		return slsa.ProvenanceBuildDefinition{}, fmt.Errorf("unsupported buildType %v", buildDefinitionType)
	}
}
