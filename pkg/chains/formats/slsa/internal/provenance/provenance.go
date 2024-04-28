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
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/metadata"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
)

// GetSLSA1Statement returns a predicate in SLSA v1.0 format using the given data.
func GetSLSA1Statement(obj objects.TektonObject, sub []intoto.Subject, bd slsa.ProvenanceBuildDefinition, bp []slsa.ResourceDescriptor, slsaConfig *slsaconfig.SlsaConfig) intoto.ProvenanceStatementSLSA1 {
	return intoto.ProvenanceStatementSLSA1{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       sub,
		},
		Predicate: slsa.ProvenancePredicate{
			BuildDefinition: bd,
			RunDetails: slsa.ProvenanceRunDetails{
				Builder: slsa.Builder{
					ID: slsaConfig.BuilderID,
				},
				BuildMetadata: metadata.GetBuildMetadata(obj),
				Byproducts:    bp,
			},
		},
	}
}
