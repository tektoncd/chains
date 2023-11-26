/*
Copyright 2022 The Tekton Authors
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

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/attest"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	materialv1beta1 "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/material/v1beta1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func GenerateAttestation(ctx context.Context, tro *objects.TaskRunObjectV1Beta1, slsaConfig *slsaconfig.SlsaConfig) (interface{}, error) {
	subjects := extract.SubjectDigests(ctx, tro, slsaConfig)

	mat, err := materialv1beta1.TaskMaterials(ctx, tro)
	if err != nil {
		return nil, err
	}
	att := intoto.ProvenanceStatement{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       subjects,
		},
		Predicate: slsa.ProvenancePredicate{
			Builder: common.ProvenanceBuilder{
				ID: slsaConfig.BuilderID,
			},
			BuildType:   tro.GetGVK(),
			Invocation:  invocation(tro),
			BuildConfig: buildConfig(tro),
			Metadata:    Metadata(tro),
			Materials:   mat,
		},
	}
	return att, nil
}

// invocation describes the event that kicked off the build
// we currently don't set ConfigSource because we don't know
// which material the Task definition came from
func invocation(tro *objects.TaskRunObjectV1Beta1) slsa.ProvenanceInvocation {
	var paramSpecs []v1beta1.ParamSpec
	if ts := tro.Status.TaskSpec; ts != nil {
		paramSpecs = ts.Params
	}
	return attest.InvocationV1Beta1(tro, tro.Spec.Params, paramSpecs)
}

// Metadata adds taskrun's start time, completion time and reproducibility labels
// to the metadata section of the generated provenance.
func Metadata(tro *objects.TaskRunObjectV1Beta1) *slsa.ProvenanceMetadata {
	m := &slsa.ProvenanceMetadata{}
	if tro.Status.StartTime != nil {
		utc := tro.Status.StartTime.Time.UTC()
		m.BuildStartedOn = &utc
	}
	if tro.Status.CompletionTime != nil {
		utc := tro.Status.CompletionTime.Time.UTC()
		m.BuildFinishedOn = &utc
	}
	for label, value := range tro.Labels {
		if label == attest.ChainsReproducibleAnnotation && value == "true" {
			m.Reproducible = true
		}
	}
	return m
}
