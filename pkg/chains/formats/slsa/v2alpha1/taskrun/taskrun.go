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
	"fmt"
	"reflect"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	materialv1beta1 "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/material/v1beta1"
	slsav1 "github.com/tektoncd/chains/pkg/chains/formats/slsa/v1/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// BuildConfig is the custom Chains format to fill out the
// "buildConfig" section of the slsa-provenance predicate
type BuildConfig struct {
	TaskSpec       *v1beta1.TaskSpec       `json:"taskSpec"`
	TaskRunResults []v1beta1.TaskRunResult `json:"taskRunResults"`
}

func GenerateAttestation(ctx context.Context, builderID string, payloadType config.PayloadType, tro *objects.TaskRunObjectV1Beta1) (interface{}, error) {
	subjects := extract.SubjectDigests(ctx, tro, nil)
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
				ID: builderID,
			},
			BuildType:   fmt.Sprintf("https://chains.tekton.dev/format/%v/type/%s", payloadType, tro.GetGVK()),
			Invocation:  invocation(tro),
			BuildConfig: BuildConfig{TaskSpec: tro.Status.TaskSpec, TaskRunResults: tro.Status.TaskRunResults},
			Metadata:    metadata(tro),
			Materials:   mat,
		},
	}
	return att, nil
}

func metadata(tro *objects.TaskRunObjectV1Beta1) *slsa.ProvenanceMetadata {
	m := slsav1.Metadata(tro)
	m.Completeness = slsa.ProvenanceComplete{
		Parameters: true,
	}
	return m
}

// invocation describes the event that kicked off the build
// we currently don't set ConfigSource because we don't know
// which material the Task definition came from
func invocation(tro *objects.TaskRunObjectV1Beta1) slsa.ProvenanceInvocation {
	i := slsa.ProvenanceInvocation{}
	if p := tro.Status.Provenance; p != nil && p.RefSource != nil {
		i.ConfigSource = slsa.ConfigSource{
			URI:        p.RefSource.URI,
			Digest:     p.RefSource.Digest,
			EntryPoint: p.RefSource.EntryPoint,
		}
	}
	i.Parameters = invocationParams(tro)
	env := invocationEnv(tro)
	if len(env) > 0 {
		i.Environment = env
	}
	return i
}

// invocationEnv adds the tekton feature flags that were enabled
// for the taskrun. In the future, we can populate versioning information
// here as well.
func invocationEnv(tro *objects.TaskRunObjectV1Beta1) map[string]any {
	var iEnv map[string]any = make(map[string]any)
	if tro.Status.Provenance != nil && tro.Status.Provenance.FeatureFlags != nil {
		iEnv["tekton-pipelines-feature-flags"] = tro.Status.Provenance.FeatureFlags
	}
	return iEnv
}

// invocationParams adds all fields from the task run object except
// TaskRef or TaskSpec since they are in the ConfigSource or buildConfig.
func invocationParams(tro *objects.TaskRunObjectV1Beta1) map[string]any {
	var iParams map[string]any = make(map[string]any)
	skipFields := sets.NewString("TaskRef", "TaskSpec")
	v := reflect.ValueOf(tro.Spec)
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		if !skipFields.Has(v.Type().Field(i).Name) {
			iParams[fieldName] = v.Field(i).Interface()
		}
	}
	return iParams
}
