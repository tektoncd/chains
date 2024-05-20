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

	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	buildtypes "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/build_types"
	externalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/external_parameters"
	internalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/internal_parameters"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	pipelineRunResults = "pipelineRunResults/%s"
	// JsonMediaType is the media type of json encoded content used in resource descriptors
	JsonMediaType = "application/json"
)

// GenerateAttestation generates a provenance statement with SLSA v1.0 predicate for a pipeline run.
func GenerateAttestation(ctx context.Context, pro *objects.PipelineRunObjectV1, slsaconfig *slsaconfig.SlsaConfig) (interface{}, error) {
	bp, err := byproducts(pro)
	if err != nil {
		return nil, err
	}

	bd, err := getBuildDefinition(ctx, slsaconfig, pro)
	if err != nil {
		return nil, err
	}

	predicate := &slsa.Provenance{
		BuildDefinition: &bd,
		RunDetails: &slsa.RunDetails{
			Builder: &slsa.Builder{
				Id: slsaconfig.BuilderID,
			},
			Metadata:   metadata(pro),
			Byproducts: bp,
		},
	}

	predicateStruct := &structpb.Struct{}
	predicateJSON, err := protojson.Marshal(predicate)
	if err != nil {
		return nil, err
	}

	err = protojson.Unmarshal(predicateJSON, predicateStruct)
	if err != nil {
		return nil, err
	}

	att := &intoto.Statement{
		Type:          intoto.StatementTypeUri,
		PredicateType: "https://slsa.dev/provenance/v1",
		Subject:       extract.SubjectDigests(ctx, pro, slsaconfig),
		Predicate:     predicateStruct,
	}
	return att, nil
}

func metadata(pro *objects.PipelineRunObjectV1) *slsa.BuildMetadata {
	m := &slsa.BuildMetadata{
		InvocationId: string(pro.ObjectMeta.UID),
	}
	if pro.Status.StartTime != nil {
		utc := pro.Status.StartTime.Time.UTC()
		m.StartedOn = timestamppb.New(utc)
	}
	if pro.Status.CompletionTime != nil {
		utc := pro.Status.CompletionTime.Time.UTC()
		m.FinishedOn = timestamppb.New(utc)
	}
	return m
}

// byproducts contains the pipelineRunResults
func byproducts(pro *objects.PipelineRunObjectV1) ([]*intoto.ResourceDescriptor, error) {
	byProd := []*intoto.ResourceDescriptor{}
	for _, key := range pro.Status.Results {
		content, err := json.Marshal(key.Value)
		if err != nil {
			return nil, err
		}
		bp := &intoto.ResourceDescriptor{
			Name:      fmt.Sprintf(pipelineRunResults, key.Name),
			Content:   content,
			MediaType: JsonMediaType,
		}
		byProd = append(byProd, bp)
	}
	return byProd, nil
}

// getBuildDefinition get the buildDefinition based on the configured buildType. This will default to the slsa buildType
func getBuildDefinition(ctx context.Context, slsaconfig *slsaconfig.SlsaConfig, pro *objects.PipelineRunObjectV1) (slsa.BuildDefinition, error) {
	// if buildType is not set in the chains-config, default to slsa build type
	buildDefinitionType := slsaconfig.BuildType
	if slsaconfig.BuildType == "" {
		buildDefinitionType = buildtypes.SlsaBuildType
	}

	switch buildDefinitionType {
	case buildtypes.SlsaBuildType:
		rd, err := resolveddependencies.PipelineRun(ctx, pro, slsaconfig, resolveddependencies.AddSLSATaskDescriptor)
		if err != nil {
			return slsa.BuildDefinition{}, err
		}
		extParamsStruct, err := getStruct(externalparameters.PipelineRun(pro))
		if err != nil {
			return slsa.BuildDefinition{}, err
		}

		intParamsStruct, err := getStruct(internalparameters.SLSAInternalParameters(pro))
		if err != nil {
			return slsa.BuildDefinition{}, err
		}

		return slsa.BuildDefinition{
			BuildType:            buildDefinitionType,
			ExternalParameters:   extParamsStruct,
			InternalParameters:   intParamsStruct,
			ResolvedDependencies: rd,
		}, nil
	case buildtypes.TektonBuildType:
		rd, err := resolveddependencies.PipelineRun(ctx, pro, slsaconfig, resolveddependencies.AddTektonTaskDescriptor)
		if err != nil {
			return slsa.BuildDefinition{}, err
		}
		extParamsStruct, err := getStruct(externalparameters.PipelineRun(pro))
		if err != nil {
			return slsa.BuildDefinition{}, err
		}

		intParamsStruct, err := getStruct(internalparameters.TektonInternalParameters(pro))
		if err != nil {
			return slsa.BuildDefinition{}, err
		}

		return slsa.BuildDefinition{
			BuildType:            buildDefinitionType,
			ExternalParameters:   extParamsStruct,
			InternalParameters:   intParamsStruct,
			ResolvedDependencies: rd,
		}, nil
	default:
		return slsa.BuildDefinition{}, fmt.Errorf("unsupported buildType %v", buildDefinitionType)
	}
}

func getStruct(data map[string]any) (*structpb.Struct, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	protoStruct := &structpb.Struct{}
	err = protojson.Unmarshal(bytes, protoStruct)
	if err != nil {
		return nil, err
	}

	return protoStruct, nil
}
