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
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	externalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/external_parameters"
	internalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/internal_parameters"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestMetadata(t *testing.T) {
	pr := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "my-namespace",
			Annotations: map[string]string{
				"chains.tekton.dev/reproducible": "true",
			},
			UID: "abhhf-12354-asjsdbjs23-3435353n",
		},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				StartTime:      &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)},
				CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
			},
		},
	}
	start := time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)
	end := time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)
	want := &slsa.BuildMetadata{
		InvocationId: "abhhf-12354-asjsdbjs23-3435353n",
		StartedOn:    timestamppb.New(start),
		FinishedOn:   timestamppb.New(end),
	}
	got := metadata(objects.NewPipelineRunObjectV1(pr))
	if d := cmp.Diff(want, got, protocmp.Transform()); d != "" {
		t.Fatalf("metadata (-want, +got):\n%s", d)
	}
}

func TestMetadataInTimeZone(t *testing.T) {
	tz := time.FixedZone("Test Time", int((12 * time.Hour).Seconds()))
	pr := &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "my-namespace",
			Annotations: map[string]string{
				"chains.tekton.dev/reproducible": "true",
			},
			UID: "abhhf-12354-asjsdbjs23-3435353n",
		},
		Status: v1.PipelineRunStatus{
			PipelineRunStatusFields: v1.PipelineRunStatusFields{
				StartTime:      &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, tz)},
				CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, tz)},
			},
		},
	}
	start := time.Date(1995, time.December, 24, 6, 12, 12, 12, tz).UTC()
	end := time.Date(1995, time.December, 24, 6, 12, 12, 24, tz).UTC()
	want := &slsa.BuildMetadata{
		InvocationId: "abhhf-12354-asjsdbjs23-3435353n",
		StartedOn:    timestamppb.New(start),
		FinishedOn:   timestamppb.New(end),
	}
	got := metadata(objects.NewPipelineRunObjectV1(pr))
	if d := cmp.Diff(want, got, protocmp.Transform()); d != "" {
		t.Fatalf("metadata (-want, +got):\n%s", d)
	}
}

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
			MediaType: JsonMediaType,
		},
	}
	got, err := byproducts(objects.NewPipelineRunObjectV1(pr))
	if err != nil {
		t.Fatalf("Could not extract byproducts: %s", err)
	}
	if d := cmp.Diff(want, got, protocmp.Transform()); d != "" {
		t.Fatalf("byproducts (-want, +got):\n%s", d)
	}
}

func createPro(path string) *objects.PipelineRunObjectV1 {
	pr, err := objectloader.PipelineRunFromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunFromFile("../../../testdata/slsa-v2alpha3/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunFromFile("../../../testdata/slsa-v2alpha3/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObjectV1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func TestGenerateAttestation(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	pr := createPro("../../../testdata/slsa-v2alpha3/pipelinerun1.json")

	e1BuildStart := time.Unix(1617011400, 0)
	e1BuildFinished := time.Unix(1617011415, 0)

	externalParams := map[string]any{
		"runSpec": pr.Spec,
	}

	slsaPredicate := slsa.Provenance{
		BuildDefinition: &slsa.BuildDefinition{
			BuildType:          "https://tekton.dev/chains/v2/slsa",
			ExternalParameters: getProtoStruct(t, externalParams),
			InternalParameters: getProtoStruct(t, map[string]any{}),
			ResolvedDependencies: []*intoto.ResourceDescriptor{
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
					Name:      "pipelineRunResults/CHAINS-GIT_COMMIT",
					Content:   []uint8(`"abcd"`),
					MediaType: JsonMediaType,
				}, {
					Name:      "pipelineRunResults/CHAINS-GIT_URL",
					Content:   []uint8(`"https://git.test.com"`),
					MediaType: JsonMediaType,
				}, {
					Name:      "pipelineRunResults/IMAGE_URL",
					Content:   []uint8(`"test.io/test/image"`),
					MediaType: JsonMediaType,
				}, {
					Name:      "pipelineRunResults/IMAGE_DIGEST",
					Content:   []uint8(`"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"`),
					MediaType: JsonMediaType,
				}, {
					Name:      "pipelineRunResults/img-ARTIFACT_INPUTS",
					Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7","uri":"abc"}`),
					MediaType: JsonMediaType,
				}, {
					Name:      "pipelineRunResults/img2-ARTIFACT_OUTPUTS",
					Content:   []uint8(`{"digest":"sha256:","uri":"def"}`),
					MediaType: JsonMediaType,
				}, {
					Name:      "pipelineRunResults/img_no_uri-ARTIFACT_OUTPUTS",
					Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}`),
					MediaType: JsonMediaType,
				},
			},
		},
	}

	predicateJSON, err := protojson.Marshal(&slsaPredicate)
	if err != nil {
		t.Fatalf("error getting SLSA predicate proto struct: %v", err)
	}

	predicateStruct := &structpb.Struct{}
	err = protojson.Unmarshal(predicateJSON, predicateStruct)
	if err != nil {
		t.Fatalf("error getting SLSA predicate proto struct: %v", err)
	}

	want := intoto.Statement{
		Type:          intoto.StatementTypeUri,
		PredicateType: "https://slsa.dev/provenance/v1",
		Subject: []*intoto.ResourceDescriptor{
			{
				Name: "test.io/test/image",
				Digest: common.DigestSet{
					"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
				},
			},
		},
		Predicate: predicateStruct,
	}

	got, err := GenerateAttestation(ctx, pr, &slsaconfig.SlsaConfig{
		BuilderID:             "test_builder-1",
		DeepInspectionEnabled: false,
		BuildType:             "https://tekton.dev/chains/v2/slsa",
	})

	if err != nil {
		t.Errorf("unwant error: %s", err.Error())
	}

	if diff := cmp.Diff(&want, got, cmp.Options{protocmp.Transform()}); diff != "" {
		t.Errorf("GenerateAttestation(): -want +got: %s", diff)
	}
}

func getResolvedDependencies(addTasks func(*objects.TaskRunObjectV1) (*intoto.ResourceDescriptor, error)) []*intoto.ResourceDescriptor {
	pr := createPro("../../../testdata/slsa-v2alpha3/pipelinerun1.json")
	rd, err := resolveddependencies.PipelineRun(context.Background(), pr, &slsaconfig.SlsaConfig{DeepInspectionEnabled: false}, addTasks)
	if err != nil {
		return nil
	}
	return rd
}

func TestGetBuildDefinition(t *testing.T) {
	pr := createPro("../../../testdata/slsa-v2alpha3/pipelinerun1.json")
	pr.Annotations = map[string]string{
		"annotation1": "annotation1",
	}
	pr.Labels = map[string]string{
		"label1": "label1",
	}
	tests := []struct {
		name        string
		taskContent func(*objects.TaskRunObjectV1) (*intoto.ResourceDescriptor, error)
		config      *slsaconfig.SlsaConfig
		want        slsa.BuildDefinition
	}{
		{
			name:        "test slsa build type",
			taskContent: resolveddependencies.AddSLSATaskDescriptor,
			config:      &slsaconfig.SlsaConfig{BuildType: "https://tekton.dev/chains/v2/slsa"},
			want: slsa.BuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa",
				ExternalParameters:   getProtoStruct(t, externalparameters.PipelineRun(pr)),
				InternalParameters:   getProtoStruct(t, internalparameters.SLSAInternalParameters(pr)),
				ResolvedDependencies: getResolvedDependencies(resolveddependencies.AddSLSATaskDescriptor),
			},
		},
		{
			name:        "test tekton build type",
			config:      &slsaconfig.SlsaConfig{BuildType: "https://tekton.dev/chains/v2/slsa-tekton"},
			taskContent: resolveddependencies.AddSLSATaskDescriptor,
			want: slsa.BuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa-tekton",
				ExternalParameters:   getProtoStruct(t, externalparameters.PipelineRun(pr)),
				InternalParameters:   getProtoStruct(t, internalparameters.TektonInternalParameters(pr)),
				ResolvedDependencies: getResolvedDependencies(resolveddependencies.AddTektonTaskDescriptor),
			},
		},
		{
			name:        "test default build type",
			config:      &slsaconfig.SlsaConfig{BuildType: "https://tekton.dev/chains/v2/slsa"},
			taskContent: resolveddependencies.AddSLSATaskDescriptor,
			want: slsa.BuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa",
				ExternalParameters:   getProtoStruct(t, externalparameters.PipelineRun(pr)),
				InternalParameters:   getProtoStruct(t, internalparameters.SLSAInternalParameters(pr)),
				ResolvedDependencies: getResolvedDependencies(resolveddependencies.AddSLSATaskDescriptor),
			},
		},
	}

	for i := range tests {
		tc := &tests[i]
		t.Run(tc.name, func(t *testing.T) {
			bd, err := getBuildDefinition(context.Background(), tc.config, pr)
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}

			if diff := cmp.Diff(&tc.want, &bd, cmp.Options{protocmp.Transform()}); diff != "" {
				t.Errorf("getBuildDefinition(): -want +got: %v", diff)
			}
		})
	}
}

func TestUnsupportedBuildType(t *testing.T) {
	pr := createPro("../../../testdata/slsa-v2alpha3/pipelinerun1.json")

	got, err := getBuildDefinition(context.Background(), &slsaconfig.SlsaConfig{BuildType: "bad-buildtype"}, pr)
	if err == nil {
		t.Error("getBuildDefinition(): expected error got nil")
	}
	if diff := cmp.Diff(&slsa.BuildDefinition{}, &got, cmp.Options{protocmp.Transform()}); diff != "" {
		t.Errorf("getBuildDefinition(): -want +got: %s", diff)
	}
}

func getProtoStruct(t *testing.T, data map[string]any) *structpb.Struct {
	t.Helper()
	protoStruct, err := getStruct(data)
	if err != nil {
		t.Fatalf("error getting proto struct: %v", err)
	}

	return protoStruct
}
