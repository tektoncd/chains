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
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"

	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

const jsonMediaType = "application/json"

func TestByProducts(t *testing.T) {
	resultValue := v1.ResultValue{Type: "string", StringVal: "result-value"}
	tr := &v1.TaskRun{ //nolint:staticcheck
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				Results: []v1.TaskRunResult{
					{
						Name:  "result-name",
						Value: resultValue,
					},
				},
			},
		},
	}

	resultBytes, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}
	want := []slsa.ResourceDescriptor{
		{
			Name:      "taskRunResults/result-name",
			Content:   resultBytes,
			MediaType: jsonMediaType,
		},
	}
	got, err := byproducts(objects.NewTaskRunObjectV1(tr))
	if err != nil {
		t.Fatalf("Could not extract byproducts: %s", err)
	}
	if d := cmp.Diff(want, got); d != "" {
		t.Fatalf("byproducts (-want, +got):\n%s", d)
	}
}

func TestTaskRunGenerateAttestation(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	tr, err := objectloader.TaskRunFromFile("../../../testdata/slsa-v2alpha4/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}
	e1BuildStart := time.Unix(1617011400, 0)
	e1BuildFinished := time.Unix(1617011415, 0)

	want := in_toto.ProvenanceStatementSLSA1{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []in_toto.Subject{
				{
					Name: "gcr.io/my/image/fromstep3",
					Digest: common.DigestSet{
						"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					},
				},
				{
					Name: "gcr.io/my/image",
					Digest: common.DigestSet{
						"sha256": "d31cc8328054de2bd93735f9cbf0ccfb6e0ee8f4c4225da7d8f8cb3900eaf466",
					},
				},
			},
		},
		Predicate: slsa.ProvenancePredicate{
			BuildDefinition: slsa.ProvenanceBuildDefinition{
				BuildType: "https://tekton.dev/chains/v2/slsa",
				ExternalParameters: map[string]any{
					"runSpec": tr.Spec,
				},
				InternalParameters: map[string]any{
					"tekton-pipelines-feature-flags": config.FeatureFlags{EnableAPIFields: "beta", ResultExtractionMethod: "termination-message"},
				},
				ResolvedDependencies: []slsa.ResourceDescriptor{
					{
						URI:    "git+https://github.com/test",
						Digest: common.DigestSet{"sha1": "ab123"},
						Name:   "task",
					},
					{
						URI:    "oci://gcr.io/test1/test1",
						Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
					},
					{
						URI:    "oci://gcr.io/test2/test2",
						Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
					},
					{
						URI:    "oci://gcr.io/test3/test3",
						Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
					},
					{Name: "inputs/result", URI: "git+https://git.test.com.git", Digest: common.DigestSet{"sha1": "taskrun"}},
				},
			},
			RunDetails: slsa.ProvenanceRunDetails{
				Builder: slsa.Builder{
					ID: "test_builder-1",
				},
				BuildMetadata: slsa.BuildMetadata{
					InvocationID: "abhhf-12354-asjsdbjs23-3435353n",
					StartedOn:    &e1BuildStart,
					FinishedOn:   &e1BuildFinished,
				},
				Byproducts: []slsa.ResourceDescriptor{
					{
						Name:      "stepResults/step1_result1",
						MediaType: "application/json",
						Content:   []uint8(`"result-value"`),
					},
					{
						Name:      "stepResults/step1_result1-ARTIFACT_OUTPUTS",
						MediaType: "application/json",
						Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7","uri":"gcr.io/my/image/fromstep2"}`),
					},
				},
			},
		},
	}

	got, err := GenerateAttestation(ctx, objects.NewTaskRunObjectV1(tr), &slsaconfig.SlsaConfig{
		BuilderID: "test_builder-1",
		BuildType: "https://tekton.dev/chains/v2/slsa",
	})

	if err != nil {
		t.Errorf("unwant error: %s", err.Error())
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("GenerateAttestation(): -want +got: %s", diff)
	}
}
