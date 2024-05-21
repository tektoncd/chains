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

package builddefinition

import (
	"context"
	"encoding/json"

	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	buildtypes "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/build_types"
	externalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/external_parameters"
	internalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/internal_parameters"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetTaskRunBuildDefinition returns the buildDefinition for the given TaskRun based on the configured buildType. This will default to the slsa buildType
func GetTaskRunBuildDefinition(ctx context.Context, tro *objects.TaskRunObjectV1, buildType string, resolveOpts resolveddependencies.ResolveOptions) (slsa.BuildDefinition, error) {
	rd, err := resolveddependencies.TaskRun(ctx, resolveOpts, tro)
	if err != nil {
		return slsa.BuildDefinition{}, err
	}

	externalParams := externalparameters.TaskRun(tro)
	structExternalParams, err := getStruct(externalParams)
	if err != nil {
		return slsa.BuildDefinition{}, err
	}

	buildDefinitionType := buildType
	if buildDefinitionType == "" {
		buildDefinitionType = buildtypes.SlsaBuildType
	}

	internalParams, err := internalparameters.GetInternalParamters(tro, buildDefinitionType)
	if err != nil {
		return slsa.BuildDefinition{}, err
	}
	structInternalParams, err := getStruct(internalParams)
	if err != nil {
		return slsa.BuildDefinition{}, err
	}

	return slsa.BuildDefinition{
		BuildType:            buildDefinitionType,
		ExternalParameters:   structExternalParams,
		InternalParameters:   structInternalParams,
		ResolvedDependencies: rd,
	}, nil
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
