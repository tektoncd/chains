/*
Copyright 2021 The Tekton Authors

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
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsaprov "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/compare"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestByProducts(t *testing.T) {
	resultValue := v1.ResultValue{Type: "string", StringVal: "result-value"}
	pr := &v1.PipelineRun{
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				Results: []v1.PipelineRunResult{
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
			Name:      "pipelineRunResults/result-name",
			Content:   resultBytes,
			MediaType: JSONMediaType,
		},
	}
	got, err := byproducts(objects.NewPipelineRunObjectV1(pr), &slsaconfig.SlsaConfig{})
	if err != nil {
		t.Fatalf("Could not extract byproducts: %s", err)
	}
	if d := cmp.Diff(&want, &got, protocmp.Transform()); d != "" {
		t.Fatalf("byproducts (-want, +got):\n%s", d)
	}
}

func TestGenerateAttestation(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	pr := createPro("../../../testdata/slsa-v2alpha4/pipelinerun1.json")

	e1BuildStart := time.Unix(1617011400, 0)
	e1BuildFinished := time.Unix(1617011415, 0)

	tests := []struct {
		name                         string
		expectedStatement            *intoto.Statement
		expectedPredicate            *slsa.Provenance
		expectedSubjects             []*intoto.ResourceDescriptor
		expectedResolvedDependencies []*intoto.ResourceDescriptor
		expectedByProducts           []*intoto.ResourceDescriptor
		withDeepInspection           bool
	}{
		{
			name: "attestation without deepinspection",
			expectedSubjects: []*intoto.ResourceDescriptor{
				{
					Name: "abc",
					Digest: common.DigestSet{
						"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					},
				},
				{
					Name: "test.io/test/image",
					Digest: common.DigestSet{
						"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					},
				},
			},
			expectedResolvedDependencies: []*intoto.ResourceDescriptor{
				{
					Uri:    "git+https://github.com/test",
					Digest: common.DigestSet{"sha1": "28b123"},
					Name:   "pipeline",
				},
				{
					Uri:    "git+https://github.com/catalog",
					Digest: common.DigestSet{"sha1": "x123"},
					Name:   "pipelineTask",
				},
				{
					Uri:    "oci://gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
				{
					Uri:    "git+https://github.com/test",
					Digest: common.DigestSet{"sha1": "ab123"},
					Name:   "pipelineTask",
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
					Uri:    "abc",
					Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"},
					Name:   "inputs/result",
				},
				{
					Name:   "inputs/result",
					Uri:    "git+https://git.test.com.git",
					Digest: common.DigestSet{"sha1": "abcd"},
				},
			},
			expectedByProducts: []*intoto.ResourceDescriptor{
				{
					Name:      "pipelineRunResults/CHAINS-GIT_COMMIT",
					Content:   []uint8(`"abcd"`),
					MediaType: JSONMediaType,
				}, {
					Name:      "pipelineRunResults/CHAINS-GIT_URL",
					Content:   []uint8(`"https://git.test.com"`),
					MediaType: JSONMediaType,
				}, {
					Name:      "pipelineRunResults/img-ARTIFACT_INPUTS",
					Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7","uri":"abc"}`),
					MediaType: JSONMediaType,
				}, {
					Name:      "pipelineRunResults/img_no_uri-ARTIFACT_OUTPUTS",
					Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}`),
					MediaType: JSONMediaType,
				},
			},
		},
		{
			name:               "attestation with deepinspection",
			withDeepInspection: true,
			expectedSubjects: []*intoto.ResourceDescriptor{
				{
					Name: "abc",
					Digest: common.DigestSet{
						"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					},
				},
				{
					Name: "test.io/test/image",
					Digest: common.DigestSet{
						"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					},
				},
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
			expectedResolvedDependencies: []*intoto.ResourceDescriptor{
				{
					Uri:    "git+https://github.com/test",
					Digest: common.DigestSet{"sha1": "28b123"},
					Name:   "pipeline",
				},
				{
					Uri:    "git+https://github.com/catalog",
					Digest: common.DigestSet{"sha1": "x123"},
					Name:   "pipelineTask",
				},
				{
					Uri:    "oci://gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
				{
					Uri:    "git+https://github.com/test",
					Digest: common.DigestSet{"sha1": "ab123"},
					Name:   "pipelineTask",
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
					Uri:    "https://github.com/tektoncd/pipeline",
					Digest: common.DigestSet{"sha1": "7f2f46e1b97df36b2b82d1b1d87c81b8b3d21601"},
				},
				{
					Uri:    "abc",
					Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"},
					Name:   "inputs/result",
				},
				{
					Name:   "inputs/result",
					Uri:    "git+https://git.test.com.git",
					Digest: common.DigestSet{"sha1": "sha:taskdefault"},
				},
				{
					Name:   "inputs/result",
					Uri:    "git+https://git.test.com.git",
					Digest: common.DigestSet{"sha1": "taskrun"},
				},
				{
					Name:   "inputs/result",
					Uri:    "git+https://git.test.com.git",
					Digest: common.DigestSet{"sha1": "abcd"},
				},
			},
			expectedByProducts: []*intoto.ResourceDescriptor{
				{
					Name:      "pipelineRunResults/CHAINS-GIT_COMMIT",
					Content:   []uint8(`"abcd"`),
					MediaType: JSONMediaType,
				}, {
					Name:      "pipelineRunResults/CHAINS-GIT_URL",
					Content:   []uint8(`"https://git.test.com"`),
					MediaType: JSONMediaType,
				}, {
					Name:      "pipelineRunResults/img-ARTIFACT_INPUTS",
					Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7","uri":"abc"}`),
					MediaType: JSONMediaType,
				}, {
					Name:      "pipelineRunResults/img_no_uri-ARTIFACT_OUTPUTS",
					Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}`),
					MediaType: JSONMediaType,
				}, {
					Name:      "taskRunResults/some-uri_DIGEST",
					Content:   []uint8(`"sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"`),
					MediaType: JSONMediaType,
				}, {
					Name:      "taskRunResults/some-uri",
					Content:   []uint8(`"pkg:deb/debian/curl@7.50.3-1"`),
					MediaType: JSONMediaType,
				}, {
					Name:      "stepResults/step1_result1-ARTIFACT_INPUTS",
					Content:   []uint8(`{"digest":"sha1:7f2f46e1b97df36b2b82d1b1d87c81b8b3d21601","uri":"https://github.com/tektoncd/pipeline"}`),
					MediaType: JSONMediaType,
				}, {
					Name:      "stepResults/step1_result1",
					Content:   []uint8(`"result-value"`),
					MediaType: JSONMediaType,
				}, {
					Name:      "stepResults/step1_result1-ARTIFACT_OUTPUTS",
					Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7","uri":"gcr.io/my/image/fromstep2"}`),
					MediaType: JSONMediaType,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := GenerateAttestation(ctx, pr, &slsaconfig.SlsaConfig{
				BuilderID:             "test_builder-1",
				DeepInspectionEnabled: test.withDeepInspection,
				BuildType:             "https://tekton.dev/chains/v2/slsa",
			})

			if err != nil {
				t.Errorf("unwant error: %s", err.Error())
			}

			expectedPredicate := &slsa.Provenance{
				BuildDefinition: &slsa.BuildDefinition{
					BuildType: "https://tekton.dev/chains/v2/slsa",
					ExternalParameters: getStruct(t, map[string]any{
						"runSpec": pr.Spec,
					}),
					InternalParameters:   getStruct(t, map[string]any{}),
					ResolvedDependencies: test.expectedResolvedDependencies,
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
					Byproducts: test.expectedByProducts,
				},
			}
			expectedStatement := &intoto.Statement{
				Type:          intoto.StatementTypeUri,
				PredicateType: slsaprov.PredicateSLSAProvenance,
				Subject:       test.expectedSubjects,
				Predicate:     getPredicateStruct(t, expectedPredicate),
			}

			opts := compare.SLSAV1CompareOptions()
			opts = append(opts, protocmp.Transform())

			if diff := cmp.Diff(expectedStatement, got, opts...); diff != "" {
				t.Errorf("GenerateAttestation(): -want +got: %s", diff)
			}
		})
	}
}

func createPro(path string) *objects.PipelineRunObjectV1 {
	pr, err := objectloader.PipelineRunFromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunFromFile("../../../testdata/slsa-v2alpha4/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunFromFile("../../../testdata/slsa-v2alpha4/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObjectV1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
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
