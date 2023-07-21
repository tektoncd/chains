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
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"

	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/compare"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestMetadata(t *testing.T) {
	pr := &v1beta1.PipelineRun{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "my-namespace",
			Annotations: map[string]string{
				"chains.tekton.dev/reproducible": "true",
			},
			UID: "abhhf-12354-asjsdbjs23-3435353n",
		},
		Status: v1beta1.PipelineRunStatus{
			PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
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
	got := metadata(objects.NewPipelineRunObject(pr))
	if d := cmp.Diff(want, got); d != "" {
		t.Fatalf("metadata (-want, +got):\n%s", d)
	}
}

func TestMetadataInTimeZone(t *testing.T) {
	tz := time.FixedZone("Test Time", int((12 * time.Hour).Seconds()))
	pr := &v1beta1.PipelineRun{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "my-namespace",
			Annotations: map[string]string{
				"chains.tekton.dev/reproducible": "true",
			},
			UID: "abhhf-12354-asjsdbjs23-3435353n",
		},
		Status: v1beta1.PipelineRunStatus{
			PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
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
	got := metadata(objects.NewPipelineRunObject(pr))
	if d := cmp.Diff(want, got); d != "" {
		t.Fatalf("metadata (-want, +got):\n%s", d)
	}
}

func TestExternalParameters(t *testing.T) {
	pr := &v1beta1.PipelineRun{
		Spec: v1beta1.PipelineRunSpec{
			Params: v1beta1.Params{
				{
					Name:  "my-param",
					Value: v1beta1.ResultValue{Type: "string", StringVal: "string-param"},
				},
				{
					Name:  "my-array-param",
					Value: v1beta1.ResultValue{Type: "array", ArrayVal: []string{"my", "array"}},
				},
				{
					Name:  "my-empty-string-param",
					Value: v1beta1.ResultValue{Type: "string"},
				},
				{
					Name:  "my-empty-array-param",
					Value: v1beta1.ResultValue{Type: "array", ArrayVal: []string{}},
				},
			},
			PipelineRef: &v1beta1.PipelineRef{
				ResolverRef: v1beta1.ResolverRef{
					Resolver: "git",
				},
			},
		},
		Status: v1beta1.PipelineRunStatus{
			PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
				Provenance: &v1beta1.Provenance{
					RefSource: &v1beta1.RefSource{
						URI: "hello",
						Digest: map[string]string{
							"sha1": "abc123",
						},
						EntryPoint: "pipeline.yaml",
					},
				},
			},
		},
	}

	want := map[string]any{
		"buildConfigSource": map[string]string{
			"path":       "pipeline.yaml",
			"ref":        "sha1:abc123",
			"repository": "hello",
		},
		"runSpec": pr.Spec,
	}
	got := externalParameters(objects.NewPipelineRunObject(pr))
	if d := cmp.Diff(want, got); d != "" {
		t.Fatalf("externalParameters (-want, +got):\n%s", d)
	}
}

func TestInternalParameters(t *testing.T) {
	pr := &v1beta1.PipelineRun{
		Status: v1beta1.PipelineRunStatus{
			PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
				Provenance: &v1beta1.Provenance{
					FeatureFlags: &config.FeatureFlags{
						RunningInEnvWithInjectedSidecars: true,
						EnableAPIFields:                  "stable",
						AwaitSidecarReadiness:            true,
						VerificationNoMatchPolicy:        "skip",
						EnableProvenanceInStatus:         true,
						ResultExtractionMethod:           "termination-message",
						MaxResultSize:                    4096,
					},
				},
			},
		},
	}

	want := map[string]any{
		"tekton-pipelines-feature-flags": config.FeatureFlags{
			RunningInEnvWithInjectedSidecars: true,
			EnableAPIFields:                  "stable",
			AwaitSidecarReadiness:            true,
			VerificationNoMatchPolicy:        "skip",
			EnableProvenanceInStatus:         true,
			ResultExtractionMethod:           "termination-message",
			MaxResultSize:                    4096,
		},
	}
	got := internalParameters(objects.NewPipelineRunObject(pr))
	if d := cmp.Diff(want, got); d != "" {
		t.Fatalf("internalParameters (-want, +got):\n%s", d)
	}
}

func TestByProducts(t *testing.T) {
	resultValue := v1beta1.ResultValue{Type: "string", StringVal: "result-value"}
	pr := &v1beta1.PipelineRun{
		Status: v1beta1.PipelineRunStatus{
			PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
				PipelineResults: []v1beta1.PipelineRunResult{
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
			Name:      "pipelineRunResults/result-name",
			Content:   resultBytes,
			MediaType: JsonMediaType,
		},
	}
	got, err := byproducts(objects.NewPipelineRunObject(pr))
	if err != nil {
		t.Fatalf("Could not extract byproducts: %s", err)
	}
	if d := cmp.Diff(want, got); d != "" {
		t.Fatalf("byproducts (-want, +got):\n%s", d)
	}
}

func createPro(path string) *objects.PipelineRunObject {
	pr, err := objectloader.PipelineRunFromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunFromFile("../../../testdata/v2alpha2/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunFromFile("../../../testdata/v2alpha2/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObject(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func TestGenerateAttestation(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	pr := createPro("../../../testdata/v2alpha2/pipelinerun1.json")

	e1BuildStart := time.Unix(1617011400, 0)
	e1BuildFinished := time.Unix(1617011415, 0)

	want := in_toto.ProvenanceStatementSLSA1{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []in_toto.Subject{
				{
					Name: "test.io/test/image",
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
					"runSpec": pr.Spec,
				},
				InternalParameters: map[string]any{},
				ResolvedDependencies: []slsa.ResourceDescriptor{
					{
						URI:    "git+https://github.com/test",
						Digest: common.DigestSet{"sha1": "28b123"},
						Name:   "pipeline",
					},
					{
						URI:    "git+https://github.com/catalog",
						Digest: common.DigestSet{"sha1": "x123"},
						Name:   "pipelineTask",
					},
					{
						URI:    "oci://gcr.io/test1/test1",
						Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
					},
					{
						URI:    "git+https://github.com/test",
						Digest: common.DigestSet{"sha1": "ab123"},
						Name:   "pipelineTask",
					},
					{
						URI:    "oci://gcr.io/test2/test2",
						Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
					},
					{
						URI:    "oci://gcr.io/test3/test3",
						Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
					},
					{
						URI:    "abc",
						Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"},
						Name:   "inputs/result",
					},
					{Name: "inputs/result", URI: "git+https://git.test.com.git", Digest: common.DigestSet{"sha1": "abcd"}},
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
		},
	}

	got, err := GenerateAttestation(ctx, "test_builder-1", pr)

	if err != nil {
		t.Errorf("unwant error: %s", err.Error())
	}
	if diff := cmp.Diff(want, got, compare.SLSAV1CompareOptions()...); diff != "" {
		t.Errorf("GenerateAttestation(): -want +got: %s", diff)
	}
}
