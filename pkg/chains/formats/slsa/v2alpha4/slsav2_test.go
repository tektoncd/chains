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

package v2alpha4

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsaprov "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	pipelineConfig "github.com/tektoncd/pipeline/pkg/apis/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

var (
	e1BuildStart    = time.Unix(1617011400, 0)
	e1BuildFinished = time.Unix(1617011415, 0)
)

const jsonMediaType = "application/json"

func TestNewFormatter(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		cfg := config.Config{
			Builder: config.BuilderConfig{
				ID: "testid",
			},
		}
		f, err := NewFormatter(cfg)
		if err != nil {
			t.Errorf("Error creating formatter: %s", err)
		}
		if f == nil {
			t.Error("Failed to create formatter")
		}
	})
}

func TestCreatePayloadError(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "testid",
		},
	}
	f, _ := NewFormatter(cfg)

	t.Run("Invalid type", func(t *testing.T) {
		p, err := f.CreatePayload(ctx, "not a task ref")

		if p != nil {
			t.Errorf("Unexpected payload")
		}
		if err == nil {
			t.Errorf("Expected error")
		} else if err.Error() != "intoto does not support type: not a task ref" {
			t.Errorf("wrong error returned: '%s'", err.Error())
		}
	})
}

func TestCorrectPayloadType(t *testing.T) {
	var i Slsa
	if i.Type() != formats.PayloadTypeSlsav2alpha4 {
		t.Errorf("Invalid type returned: %s", i.Type())
	}
}

func TestTaskRunCreatePayload1(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	tr, err := objectloader.TaskRunV1FromFile("../testdata/slsa-v2alpha4/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}

	resultValue := v1.ParamValue{Type: "string", StringVal: "result-value"}
	resultBytesStepResult, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}

	resultValue = v1.ParamValue{Type: "object", ObjectVal: map[string]string{
		"uri":    "gcr.io/my/image/fromstep2",
		"digest": "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
	}}
	resultBytesStepResultObj, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}

	slsaPredicate := slsa.Provenance{
		BuildDefinition: &slsa.BuildDefinition{
			BuildType: "https://tekton.dev/chains/v2/slsa",
			ExternalParameters: getStruct(t, map[string]any{
				"runSpec": tr.Spec,
			}),
			InternalParameters: getStruct(t, map[string]any{
				"tekton-pipelines-feature-flags": pipelineConfig.FeatureFlags{EnableAPIFields: "beta", ResultExtractionMethod: "termination-message"},
			}),
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
					Content:   resultBytesStepResult,
					MediaType: jsonMediaType,
				},
				{
					Name:      "stepResults/taskrun-build/step1_result1-ARTIFACT_OUTPUTS",
					Content:   resultBytesStepResultObj,
					MediaType: jsonMediaType,
				},
			},
		},
	}

	expected := &intoto.Statement{
		Type:          intoto.StatementTypeUri,
		PredicateType: slsaprov.PredicateSLSAProvenance,
		Subject: []*intoto.ResourceDescriptor{
			{
				Name:   "gcr.io/my/image/fromstep3",
				Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"},
			},
			{
				Name:   "gcr.io/my/image",
				Digest: common.DigestSet{"sha256": "d31cc8328054de2bd93735f9cbf0ccfb6e0ee8f4c4225da7d8f8cb3900eaf466"},
			},
		},
		Predicate: getPredicateStruct(t, &slsaPredicate),
	}

	i, _ := NewFormatter(cfg)

	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got, cmp.Options{protocmp.Transform()}); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
	}
}

func TestTaskRunCreatePayload2(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	tr, err := objectloader.TaskRunV1FromFile("../testdata/slsa-v2alpha4/taskrun2.json")
	if err != nil {
		t.Fatal(err)
	}

	resultValue := v1.ParamValue{Type: "string", StringVal: "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"}
	resultBytesDigest, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}
	resultValue = v1.ParamValue{Type: "string", StringVal: "pkg:deb/debian/curl@7.50.3-1"}
	resultBytesURI, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}
	resultValue = v1.ParamValue{Type: "object", ObjectVal: map[string]string{
		"uri":    "https://github.com/tektoncd/pipeline",
		"digest": "sha1:7f2f46e1b97df36b2b82d1b1d87c81b8b3d21601",
	}}
	resultBytesObj, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-2",
		},
	}

	slsaPredicate := slsa.Provenance{
		BuildDefinition: &slsa.BuildDefinition{
			BuildType: "https://tekton.dev/chains/v2/slsa",
			ExternalParameters: getStruct(t, map[string]any{
				"runSpec": tr.Spec,
			}),
			InternalParameters: getStruct(t, map[string]any{}),
			ResolvedDependencies: []*intoto.ResourceDescriptor{
				{
					Uri:    "git+https://github.com/catalog",
					Digest: common.DigestSet{"sha1": "x123"},
					Name:   "task",
				},
				{
					Uri:    "oci://gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
				{
					Name:   "inputs/result",
					Uri:    "https://github.com/tektoncd/pipeline",
					Digest: common.DigestSet{"sha1": "7f2f46e1b97df36b2b82d1b1d87c81b8b3d21601"},
				},
				{
					Name:   "inputs/result",
					Uri:    "git+https://git.test.com.git",
					Digest: common.DigestSet{"sha1": "sha:taskdefault"},
				},
			},
		},
		RunDetails: &slsa.RunDetails{
			Builder: &slsa.Builder{
				Id: "test_builder-2",
			},
			Metadata: &slsa.BuildMetadata{
				InvocationId: "abhhf-12354-asjsdbjs23-3435353n",
				StartedOn:    timestamppb.New(e1BuildStart),
				FinishedOn:   timestamppb.New(e1BuildFinished),
			},
			Byproducts: []*intoto.ResourceDescriptor{
				{
					Name:      "taskRunResults/git-clone/some-uri_DIGEST",
					Content:   resultBytesDigest,
					MediaType: jsonMediaType,
				},
				{
					Name:      "taskRunResults/git-clone/some-uri",
					Content:   resultBytesURI,
					MediaType: jsonMediaType,
				},
				{
					Name:      "stepResults/git-clone/step1_result1-ARTIFACT_INPUTS",
					Content:   resultBytesObj,
					MediaType: jsonMediaType,
				},
			},
		},
	}

	expected := &intoto.Statement{
		Type:          intoto.StatementTypeUri,
		PredicateType: slsaprov.PredicateSLSAProvenance,
		Subject:       nil,
		Predicate:     getPredicateStruct(t, &slsaPredicate),
	}

	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got, cmp.Options{protocmp.Transform()}); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
	}
}

func TestMultipleSubjects(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	tr, err := objectloader.TaskRunV1FromFile("../testdata/slsa-v2alpha4/taskrun-multiple-subjects.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-multiple",
		},
	}

	slsaPredicate := slsa.Provenance{
		BuildDefinition: &slsa.BuildDefinition{
			BuildType: "https://tekton.dev/chains/v2/slsa",
			ExternalParameters: getStruct(t, map[string]any{
				"runSpec": tr.Spec,
			}),
			InternalParameters: getStruct(t, map[string]any{}),
			ResolvedDependencies: []*intoto.ResourceDescriptor{
				{
					Uri:    "oci://gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
			},
		},
		RunDetails: &slsa.RunDetails{
			Builder: &slsa.Builder{
				Id: "test_builder-multiple",
			},
			Metadata:   &slsa.BuildMetadata{},
			Byproducts: []*intoto.ResourceDescriptor{},
		},
	}

	expected := &intoto.Statement{
		Type:          intoto.StatementTypeUri,
		PredicateType: slsaprov.PredicateSLSAProvenance,
		Subject: []*intoto.ResourceDescriptor{
			{
				Name: "gcr.io/foo/bar",
				Digest: common.DigestSet{
					"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
			{
				Name: "gcr.io/myimage2",
				Digest: common.DigestSet{
					"sha256": "9f036c6170dd7aba07e45cf2fe414c7ca792e5ede3bc3a78609e3aab4fa2ff2d",
				},
			},
			{
				Name: "gcr.io/myimage1",
				Digest: common.DigestSet{
					"sha256": "db546e77d11cf34199d965d28b1107f98bcbb7630182b7d847cc31d5d21b47b0",
				},
			},
			{
				Name: "gcr.io/myimage3",
				Digest: common.DigestSet{
					"sha256": "8d14f5ded713f263742d371279586b264bde42ee8de97b808d1f5e205f376ade",
				},
			},
		},
		Predicate: getPredicateStruct(t, &slsaPredicate),
	}

	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1(tr))
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got, cmp.Options{protocmp.Transform()}); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
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
