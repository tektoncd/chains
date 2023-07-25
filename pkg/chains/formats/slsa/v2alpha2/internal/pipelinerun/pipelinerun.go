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
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/resolved_dependencies"
	rd "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/resource_descriptor"
	"github.com/tektoncd/chains/pkg/chains/objects"
)

// GenerateAttestation generates a provenance statement with SLSA v1.0 predicate for a pipeline run.
func GenerateAttestation(ctx context.Context, builderID string, pro *objects.PipelineRunObject) (interface{}, error) {
	rd, err := resolveddependencies.PipelineRun(ctx, pro)
	if err != nil {
		return nil, err
	}
	bp, err := byproducts(pro)
	if err != nil {
		return nil, err
	}
	att := intoto.ProvenanceStatementSLSA1{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       extract.SubjectDigests(ctx, pro),
		},
		Predicate: slsa.ProvenancePredicate{
			BuildDefinition: slsa.ProvenanceBuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa",
				ExternalParameters:   externalParameters(pro),
				InternalParameters:   internalParameters(pro),
				ResolvedDependencies: rd,
			},
			RunDetails: slsa.ProvenanceRunDetails{
				Builder: slsa.Builder{
					ID: builderID,
				},
				BuildMetadata: metadata(pro),
				Byproducts:    bp,
			},
		},
	}
	return att, nil
}

func metadata(pro *objects.PipelineRunObject) slsa.BuildMetadata {
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

// internalParameters adds the tekton feature flags that were enabled
// for the pipelinerun.
func internalParameters(pro *objects.PipelineRunObject) map[string]any {
	internalParams := make(map[string]any)
	if pro.Status.Provenance != nil && pro.Status.Provenance.FeatureFlags != nil {
		internalParams["tekton-pipelines-feature-flags"] = *pro.Status.Provenance.FeatureFlags
	}
	return internalParams
}

// externalParameters adds the pipeline run spec
func externalParameters(pro *objects.PipelineRunObject) map[string]any {
	externalParams := make(map[string]any)

	// add the origin of top level pipeline config
	// isRemotePipeline checks if the pipeline was fetched using a remote resolver
	isRemotePipeline := false
	if pro.Spec.PipelineRef != nil {
		if pro.Spec.PipelineRef.Resolver != "" && pro.Spec.PipelineRef.Resolver != "Cluster" {
			isRemotePipeline = true
		}
	}

	if p := pro.Status.Provenance; p != nil && p.RefSource != nil && isRemotePipeline {
		ref := ""
		for alg, hex := range p.RefSource.Digest {
			ref = fmt.Sprintf("%s:%s", alg, hex)
			break
		}
		buildConfigSource := map[string]string{
			"ref":        ref,
			"repository": p.RefSource.URI,
			"path":       p.RefSource.EntryPoint,
		}
		externalParams["buildConfigSource"] = buildConfigSource
	}
	externalParams["runSpec"] = pro.Spec
	return externalParams
}

// byproducts contains the pipelineRunResults
func byproducts(pro *objects.PipelineRunObject) ([]slsa.ResourceDescriptor, error) {
	byProd := []slsa.ResourceDescriptor{}
	for _, key := range pro.Status.PipelineResults {
		content, err := json.Marshal(key.Value)
		if err != nil {
			return nil, err
		}
		bp := slsa.ResourceDescriptor{
			Name:      fmt.Sprintf(string(rd.PipelineRunResults), key.Name),
			Content:   content,
			MediaType: string(rd.JsonMediaType),
		}
		byProd = append(byProd, bp)
	}
	return byProd, nil
}
