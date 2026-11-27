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

package provenance

import (
	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	intoto "github.com/in-toto/attestation/go/v1"
	slsaprov "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/metadata"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// chainsVersion is set by the release build using linker flags.
var chainsVersion string

// GetSLSA1Statement returns a predicate in SLSA v1.0 format using the given data.
func GetSLSA1Statement(obj objects.TektonObject, sub []*intoto.ResourceDescriptor, bd *slsa.BuildDefinition, bp []*intoto.ResourceDescriptor, slsaConfig *slsaconfig.SlsaConfig) (intoto.Statement, error) {
	predicate := slsa.Provenance{
		BuildDefinition: bd,
		RunDetails: &slsa.RunDetails{
			Builder: &slsa.Builder{
				Id: slsaConfig.BuilderID,
			},
			Metadata:   metadata.GetBuildMetadata(obj),
			Byproducts: bp,
		},
	}

	if slsaConfig.IncludeBuilderMetadata && chainsVersion != "" {
		predicate.RunDetails.Builder.Version = map[string]string{
			"tekton-chains": chainsVersion,
		}
	}

	if slsaConfig.IncludeBuilderMetadata && slsaConfig.PipelinesVersion != "" {
		predicate.RunDetails.Builder.BuilderDependencies = []*intoto.ResourceDescriptor{
			{
				Name: "tekton-pipelines",
				Uri: "https://github.com/tektoncd/pipeline/releases/tag/" +
					slsaConfig.PipelinesVersion,
			},
		}
	}

	predicateStruct, err := getProtoStruct(&predicate)
	if err != nil {
		return intoto.Statement{}, err
	}

	return intoto.Statement{
		Type:          intoto.StatementTypeUri,
		PredicateType: slsaprov.PredicateSLSAProvenance,
		Subject:       sub,
		Predicate:     predicateStruct,
	}, nil
}

func getProtoStruct(predicate *slsa.Provenance) (*structpb.Struct, error) {
	protoStruct := &structpb.Struct{}
	predicateJSON, err := protojson.Marshal(predicate)
	if err != nil {
		return nil, err
	}

	err = protojson.Unmarshal(predicateJSON, protoStruct)
	if err != nil {
		return nil, err
	}

	return protoStruct, nil
}
