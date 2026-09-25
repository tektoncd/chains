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

	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/artifact"
	builddefinition "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/build_definition"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/provenance"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/results"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	taskRunResults     = "taskRunResults/%s/%s"
	taskRunStepResults = "stepResults/%s/%s"
)

// GenerateAttestation returns the provenance for the given taskrun in SALSA 1.0 format.
func GenerateAttestation(ctx context.Context, tro *objects.TaskRunObjectV1, slsaConfig *slsaconfig.SlsaConfig) (interface{}, error) {
	bp, err := ByProducts(tro)
	if err != nil {
		return nil, err
	}

	resOpts := resolveddependencies.ResolveOptions{WithStepActionsResults: true}
	bd, err := builddefinition.GetTaskRunBuildDefinition(ctx, tro, slsaConfig.BuildType, resOpts)
	if err != nil {
		return nil, err
	}

	sub := SubjectDigests(ctx, tro)

	return provenance.GetSLSA1Statement(tro, sub, &bd, bp, slsaConfig)
}

// ByProducts returns the results categorized as byproduct from the given TaskRun.
func ByProducts(tro *objects.TaskRunObjectV1) ([]*intoto.ResourceDescriptor, error) {
	byProd := []*intoto.ResourceDescriptor{}

	res, err := results.GetResultsWithoutBuildArtifacts(tro.GetName(), tro.GetResults(), taskRunResults)
	if err != nil {
		return nil, err
	}
	byProd = append(byProd, res...)

	res, err = results.GetResultsWithoutBuildArtifacts(tro.GetName(), tro.GetStepResults(), taskRunStepResults)
	if err != nil {
		return nil, err
	}
	byProd = append(byProd, res...)
	byProd = append(byProd, byProductsFromNativeArtifacts(tro)...)

	return byProd, nil
}

func byProductsFromNativeArtifacts(tro *objects.TaskRunObjectV1) []*intoto.ResourceDescriptor {
	var byProds []*intoto.ResourceDescriptor

	addNonBuildOutputs := func(outputs []v1.Artifact) {
		for _, output := range outputs {
			if output.BuildOutput {
				continue
			}
			for _, val := range output.Values {
				if val.Uri == "" || len(val.Digest) == 0 {
					continue
				}
				digestSet := common.DigestSet{}
				for algo, hex := range val.Digest {
					digestSet[string(algo)] = hex
				}
				byProds = append(byProds, &intoto.ResourceDescriptor{
					Name:   val.Uri,
					Digest: digestSet,
				})
			}
		}
	}

	if arts := tro.GetArtifacts(); arts != nil {
		addNonBuildOutputs(arts.Outputs)
	} else {
		for _, step := range tro.GetStepArtifacts() {
			addNonBuildOutputs(step.Outputs)
		}
	}
	return byProds
}

// SubjectDigests returns the subjects detected in the given TaskRun. It takes into account taskrun and step results.
func SubjectDigests(ctx context.Context, tro *objects.TaskRunObjectV1) []*intoto.ResourceDescriptor {
	var subjects []*intoto.ResourceDescriptor
	for _, step := range tro.Status.Steps {
		res := getObjectResults(step.Results)
		stepSubjects := extract.SubjectsFromBuildArtifact(ctx, res)
		subjects = artifact.AppendSubjects(subjects, stepSubjects...)
	}

	taskSubjects := extract.SubjectsFromBuildArtifact(ctx, tro.GetResults())
	subjects = artifact.AppendSubjects(subjects, taskSubjects...)
	subjects = artifact.AppendSubjects(subjects, subjectsFromNativeArtifacts(tro)...)

	return subjects
}

func getObjectResults(tresults []v1.TaskRunResult) (res []objects.Result) {
	for _, r := range tresults {
		res = append(res, objects.Result{
			Name:  r.Name,
			Value: r.Value,
		})
	}
	return
}

// subjectsFromNativeArtifacts add native artifact outputs (where BuildOutput=true) as subjects
func subjectsFromNativeArtifacts(tro *objects.TaskRunObjectV1) []*intoto.ResourceDescriptor {
	var subjects []*intoto.ResourceDescriptor

	addOutputSubjects := func(outputs []v1.Artifact) {
		for _, output := range outputs {
			if !output.BuildOutput {
				continue
			}
			for _, val := range output.Values {
				if val.Uri == "" || len(val.Digest) == 0 {
					continue
				}
				digestSet := common.DigestSet{}
				for algo, hex := range val.Digest {
					digestSet[string(algo)] = hex
				}
				subjects = artifact.AppendSubjects(subjects, &intoto.ResourceDescriptor{
					Name:   val.Uri,
					Digest: digestSet,
				})
			}
		}
	}

	if arts := tro.GetArtifacts(); arts != nil {
		addOutputSubjects(arts.Outputs)
	} else {
		for _, step := range tro.GetStepArtifacts() {
			addOutputSubjects(step.Outputs)
		}
	}
	return subjects
}
