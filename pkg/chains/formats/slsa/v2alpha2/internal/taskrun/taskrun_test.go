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

package taskrun

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"

	v1resourcedescriptor "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	externalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/external_parameters"
	internalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/internal_parameters"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/pipelinerun"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha2/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestMetadata(t *testing.T) {
	tr := &v1beta1.TaskRun{ //nolint:staticcheck
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "my-namespace",
			Annotations: map[string]string{
				"chains.tekton.dev/reproducible": "true",
			},
			UID: "abhhf-12354-asjsdbjs23-3435353n",
		},
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				StartTime:      &v1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)},
				CompletionTime: &v1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
			},
		},
	}
	start := time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)
	end := time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)
	want := slsa.BuildMetadata{
		InvocationID: "abhhf-12354-asjsdbjs23-3435353n",
		StartedOn:    &start,
		FinishedOn:   &end,
	}
	got := metadata(objects.NewTaskRunObjectV1Beta1(tr))
	if d := cmp.Diff(want, got); d != "" {
		t.Fatalf("metadata (-want, +got):\n%s", d)
	}
}

func TestMetadataInTimeZone(t *testing.T) {
	tz := time.FixedZone("Test Time", int((12 * time.Hour).Seconds()))
	tr := &v1beta1.TaskRun{ //nolint:staticcheck
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "my-namespace",
			Annotations: map[string]string{
				"chains.tekton.dev/reproducible": "true",
			},
			UID: "abhhf-12354-asjsdbjs23-3435353n",
		},
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				StartTime:      &v1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, tz)},
				CompletionTime: &v1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, tz)},
			},
		},
	}
	start := time.Date(1995, time.December, 24, 6, 12, 12, 12, tz).UTC()
	end := time.Date(1995, time.December, 24, 6, 12, 12, 24, tz).UTC()
	want := slsa.BuildMetadata{
		InvocationID: "abhhf-12354-asjsdbjs23-3435353n",
		StartedOn:    &start,
		FinishedOn:   &end,
	}
	got := metadata(objects.NewTaskRunObjectV1Beta1(tr))
	if d := cmp.Diff(want, got); d != "" {
		t.Fatalf("metadata (-want, +got):\n%s", d)
	}
}

func TestByProducts(t *testing.T) {
	resultValue := v1beta1.ResultValue{Type: "string", StringVal: "result-value"}
	tr := &v1beta1.TaskRun{ //nolint:staticcheck
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				TaskRunResults: []v1beta1.TaskRunResult{
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
			MediaType: pipelinerun.JsonMediaType,
		},
	}
	got, err := byproducts(objects.NewTaskRunObjectV1Beta1(tr))
	if err != nil {
		t.Fatalf("Could not extract byproducts: %s", err)
	}
	if d := cmp.Diff(want, got); d != "" {
		t.Fatalf("byproducts (-want, +got):\n%s", d)
	}
}

func TestTaskRunGenerateAttestation(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	tr, err := objectloader.TaskRunV1Beta1FromFile("../../../testdata/v2alpha2/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}
	e1BuildStart := time.Unix(1617011400, 0)
	e1BuildFinished := time.Unix(1617011415, 0)

	resultValue := v1beta1.ResultValue{Type: "string", StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}
	resultBytesDigest, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}
	resultValue = v1beta1.ResultValue{Type: "string", StringVal: "gcr.io/my/image"}
	resultBytesUri, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}

	want := in_toto.ProvenanceStatementSLSA1{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []in_toto.Subject{
				{
					Name: "gcr.io/my/image",
					Digest: common.DigestSet{
						"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
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
						Name:      "taskRunResults/IMAGE_DIGEST",
						Content:   resultBytesDigest,
						MediaType: pipelinerun.JsonMediaType,
					},
					{
						Name:      "taskRunResults/IMAGE_URL",
						Content:   resultBytesUri,
						MediaType: pipelinerun.JsonMediaType,
					},
				},
			},
		},
	}

	got, err := GenerateAttestation(ctx, objects.NewTaskRunObjectV1Beta1(tr), &slsaconfig.SlsaConfig{
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

func getResolvedDependencies(tro *objects.TaskRunObjectV1Beta1) []v1resourcedescriptor.ResourceDescriptor {
	rd, err := resolveddependencies.TaskRun(context.Background(), tro)
	if err != nil {
		return []v1resourcedescriptor.ResourceDescriptor{}
	}
	return rd
}

func TestGetBuildDefinition(t *testing.T) {
	tr, err := objectloader.TaskRunV1Beta1FromFile("../../../testdata/v2alpha2/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}

	tr.Annotations = map[string]string{
		"annotation1": "annotation1",
	}
	tr.Labels = map[string]string{
		"label1": "label1",
	}

	tro := objects.NewTaskRunObjectV1Beta1(tr)
	tests := []struct {
		name      string
		buildType string
		want      slsa.ProvenanceBuildDefinition
		err       error
	}{
		{
			name:      "test slsa build type",
			buildType: "https://tekton.dev/chains/v2/slsa",
			want: slsa.ProvenanceBuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa",
				ExternalParameters:   externalparameters.TaskRun(tro),
				InternalParameters:   internalparameters.SLSAInternalParameters(tro),
				ResolvedDependencies: getResolvedDependencies(tro),
			},
			err: nil,
		},
		{
			name:      "test default build type",
			buildType: "",
			want: slsa.ProvenanceBuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa",
				ExternalParameters:   externalparameters.TaskRun(tro),
				InternalParameters:   internalparameters.SLSAInternalParameters(tro),
				ResolvedDependencies: getResolvedDependencies(tro),
			},
			err: nil,
		},
		{
			name:      "test tekton build type",
			buildType: "https://tekton.dev/chains/v2/slsa-tekton",
			want: slsa.ProvenanceBuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa-tekton",
				ExternalParameters:   externalparameters.TaskRun(tro),
				InternalParameters:   internalparameters.TektonInternalParameters(tro),
				ResolvedDependencies: getResolvedDependencies(tro),
			},
			err: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bd, err := getBuildDefinition(context.Background(), tc.buildType, tro)
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}

			if diff := cmp.Diff(tc.want, bd); diff != "" {
				t.Errorf("getBuildDefinition(): -want +got: %v", diff)
			}

		})
	}
}

func TestUnsupportedBuildType(t *testing.T) {
	tr, err := objectloader.TaskRunV1Beta1FromFile("../../../testdata/v2alpha2/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}

	got, err := getBuildDefinition(context.Background(), "bad-buildType", objects.NewTaskRunObjectV1Beta1(tr))
	if err == nil {
		t.Error("getBuildDefinition(): expected error got nil")
	}
	if diff := cmp.Diff(slsa.ProvenanceBuildDefinition{}, got); diff != "" {
		t.Errorf("getBuildDefinition(): -want +got: %s", diff)
	}
}
