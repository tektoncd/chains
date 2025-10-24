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
	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsaprov "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"

	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

const jsonMediaType = "application/json"

func TestByProducts(t *testing.T) {
	resultValue := v1.ResultValue{Type: "string", StringVal: "result-value"}
	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "taskRun-name",
		},
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
	want := []*intoto.ResourceDescriptor{
		{
			Name:      "taskRunResults/taskRun-name/result-name",
			Content:   resultBytes,
			MediaType: jsonMediaType,
		},
	}
	got, err := ByProducts(objects.NewTaskRunObjectV1(tr))
	if err != nil {
		t.Fatalf("Could not extract byproducts: %s", err)
	}
	if d := cmp.Diff(want, got, cmp.Options{protocmp.Transform()}); d != "" {
		t.Fatalf("byproducts (-want, +got):\n%s", d)
	}
}

func TestTaskRunGenerateAttestation(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	tr, err := objectloader.TaskRunV1FromFile("../../../testdata/slsa-v2alpha4/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}
	e1BuildStart := time.Unix(1617011400, 0)
	e1BuildFinished := time.Unix(1617011415, 0)

	externalParams := map[string]any{
		"runSpec": tr.Spec,
	}
	internalParams := map[string]any{
		"tekton-pipelines-feature-flags": config.FeatureFlags{EnableAPIFields: "beta", ResultExtractionMethod: "termination-message"},
	}

	slsaPredicate := slsa.Provenance{
		BuildDefinition: &slsa.BuildDefinition{
			BuildType:          "https://tekton.dev/chains/v2/slsa",
			ExternalParameters: getStruct(t, externalParams),
			InternalParameters: getStruct(t, internalParams),
			ResolvedDependencies: []*intoto.ResourceDescriptor{
				{
					Uri:    "git+https://github.com/test",
					Digest: common.DigestSet{"sha1": "ab123"},
					Name:   "task",
				},
				{
					Uri:    "oci://gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
				{
					Uri:    "oci://gcr.io/test2/test2",
					Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
				},
				{
					Uri:    "oci://gcr.io/test3/test3",
					Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
				},
				{
					Name:   "inputs/result",
					Uri:    "git+https://git.test.com.git",
					Digest: common.DigestSet{"sha1": "taskrun"},
				},
			},
		},
		RunDetails: &slsa.RunDetails{
			Builder: &slsa.Builder{
				Id: "test_builder-1",
			},
			Metadata: &slsa.BuildMetadata{
				InvocationId: "abhhf-12354-asjsdbjs23-3435353n",
				StartedOn:    timestamppb.New(e1BuildStart),
				FinishedOn:   timestamppb.New(e1BuildFinished),
			},
			Byproducts: []*intoto.ResourceDescriptor{
				{
					Name:      "stepResults/taskrun-build/step1_result1",
					MediaType: "application/json",
					Content:   []uint8(`"result-value"`),
				},
				{
					Name:      "stepResults/taskrun-build/step1_result1-ARTIFACT_OUTPUTS",
					MediaType: "application/json",
					Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7","uri":"gcr.io/my/image/fromstep2"}`),
				},
			},
		},
	}

	predicateStruct := getPredicateStruct(t, &slsaPredicate)

	want := intoto.Statement{
		Type:          intoto.StatementTypeUri,
		PredicateType: slsaprov.PredicateSLSAProvenance,
		Subject: []*intoto.ResourceDescriptor{
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
		Predicate: predicateStruct,
	}

	got, err := GenerateAttestation(ctx, objects.NewTaskRunObjectV1(tr), &slsaconfig.SlsaConfig{
		BuilderID: "test_builder-1",
		BuildType: "https://tekton.dev/chains/v2/slsa",
	})

	if err != nil {
		t.Errorf("unwant error: %s", err.Error())
	}
	if diff := cmp.Diff(&want, got, cmp.Options{protocmp.Transform()}); diff != "" {
		t.Errorf("GenerateAttestation(): -want +got: %s", diff)
	}
}

func getStruct(t *testing.T, data map[string]any) *structpb.Struct {
	t.Helper()
	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("error getting proto struct: %v", err)
	}

	protoStruct := &structpb.Struct{}
	err = protojson.Unmarshal(bytes, protoStruct)
	if err != nil {
		t.Fatalf("error getting proto struct: %v", err)
	}

	return protoStruct
}

func getPredicateStruct(t *testing.T, slsaPredicate *slsa.Provenance) *structpb.Struct {
	t.Helper()
	predicateJSON, err := protojson.Marshal(slsaPredicate)
	if err != nil {
		t.Fatalf("error getting SLSA predicate proto struct: %v", err)
	}

	predicateStruct := &structpb.Struct{}
	err = protojson.Unmarshal(predicateJSON, predicateStruct)
	if err != nil {
		t.Fatalf("error getting SLSA predicate proto struct: %v", err)
	}

	return predicateStruct
}
