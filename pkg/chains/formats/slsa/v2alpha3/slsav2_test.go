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

package v2alpha3

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/compare"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha3/internal/pipelinerun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/objectloader"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
	pipelineConfig "github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	logtesting "knative.dev/pkg/logging/testing"
)

var (
	e1BuildStart    = time.Unix(1617011400, 0)
	e1BuildFinished = time.Unix(1617011415, 0)
)

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
		} else {
			if err.Error() != "intoto does not support type: not a task ref" {
				t.Errorf("wrong error returned: '%s'", err.Error())
			}
		}
	})

}

func TestCorrectPayloadType(t *testing.T) {
	var i Slsa
	if i.Type() != formats.PayloadTypeSlsav2alpha3 {
		t.Errorf("Invalid type returned: %s", i.Type())
	}
}

func TestTaskRunCreatePayload1(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	tr, err := objectloader.TaskRunFromFile("../testdata/v2alpha3/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}

	resultValue := v1beta1.ParamValue{Type: "string", StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}
	resultBytesDigest, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}
	resultValue = v1beta1.ParamValue{Type: "string", StringVal: "gcr.io/my/image"}
	resultBytesUri, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}
	expected := in_toto.ProvenanceStatementSLSA1{
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
					"tekton-pipelines-feature-flags": pipelineConfig.FeatureFlags{EnableAPIFields: "beta", ResultExtractionMethod: "termination-message"},
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

	i, _ := NewFormatter(cfg)

	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
	}
}

func TestTaskRunCreatePayload2(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	tr, err := objectloader.TaskRunFromFile("../testdata/v2alpha3/taskrun2.json")
	if err != nil {
		t.Fatal(err)
	}

	resultValue := v1beta1.ParamValue{Type: "string", StringVal: "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"}
	resultBytesDigest, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}
	resultValue = v1beta1.ParamValue{Type: "string", StringVal: "pkg:deb/debian/curl@7.50.3-1"}
	resultBytesUri, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-2",
		},
	}
	expected := in_toto.ProvenanceStatementSLSA1{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       nil,
		},
		Predicate: slsa.ProvenancePredicate{
			BuildDefinition: slsa.ProvenanceBuildDefinition{
				BuildType: "https://tekton.dev/chains/v2/slsa",
				ExternalParameters: map[string]any{
					"runSpec": tr.Spec,
				},
				InternalParameters: map[string]any{},
				ResolvedDependencies: []slsa.ResourceDescriptor{
					{
						URI:    "git+https://github.com/catalog",
						Digest: common.DigestSet{"sha1": "x123"},
						Name:   "task",
					},
					{
						URI:    "oci://gcr.io/test1/test1",
						Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
					},
					{Name: "inputs/result", URI: "git+https://git.test.com.git", Digest: common.DigestSet{"sha1": "sha:taskdefault"}},
				},
			},
			RunDetails: slsa.ProvenanceRunDetails{
				Builder: slsa.Builder{
					ID: "test_builder-2",
				},
				BuildMetadata: slsa.BuildMetadata{
					InvocationID: "abhhf-12354-asjsdbjs23-3435353n",
					StartedOn:    &e1BuildStart,
					FinishedOn:   &e1BuildFinished,
				},
				Byproducts: []slsa.ResourceDescriptor{
					{
						Name:      "taskRunResults/some-uri_DIGEST",
						Content:   resultBytesDigest,
						MediaType: pipelinerun.JsonMediaType,
					},
					{
						Name:      "taskRunResults/some-uri",
						Content:   resultBytesUri,
						MediaType: pipelinerun.JsonMediaType,
					},
				},
			},
		},
	}

	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
	}
}

func TestMultipleSubjects(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	tr, err := objectloader.TaskRunFromFile("../testdata/v2alpha3/taskrun-multiple-subjects.json")
	if err != nil {
		t.Fatal(err)
	}

	resultValue := v1beta1.ParamValue{
		Type:      "string",
		StringVal: "gcr.io/myimage1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6,gcr.io/myimage2@sha256:daa1a56e13c85cf164e7d9e595006649e3a04c47fe4a8261320e18a0bf3b0367",
	}
	resultBytes, err := json.Marshal(resultValue)
	if err != nil {
		t.Fatalf("Could not marshal results: %s", err)
	}
	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-multiple",
		},
	}
	expected := in_toto.ProvenanceStatementSLSA1{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []in_toto.Subject{
				{
					Name: "gcr.io/myimage1",
					Digest: common.DigestSet{
						"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
					},
				}, {
					Name: "gcr.io/myimage2",
					Digest: common.DigestSet{
						"sha256": "daa1a56e13c85cf164e7d9e595006649e3a04c47fe4a8261320e18a0bf3b0367",
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
				InternalParameters: map[string]any{},
				ResolvedDependencies: []slsa.ResourceDescriptor{
					{
						URI:    "oci://gcr.io/test1/test1",
						Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
					},
				},
			},
			RunDetails: slsa.ProvenanceRunDetails{
				Builder: slsa.Builder{
					ID: "test_builder-multiple",
				},
				BuildMetadata: slsa.BuildMetadata{},
				Byproducts: []slsa.ResourceDescriptor{
					{
						Name:      "taskRunResults/IMAGES",
						Content:   resultBytes,
						MediaType: pipelinerun.JsonMediaType,
					},
				},
			},
		},
	}

	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1(tr))
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
	}
}

func createPro(path string) *objects.PipelineRunObjectV1 {
	pr, err := objectloader.PipelineRunFromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunFromFile("../testdata/v2alpha3/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunFromFile("../testdata/v2alpha3/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObjectV1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func TestPipelineRunCreatePayload1(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	pr := createPro("../testdata/v2alpha3/pipelinerun1.json")

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}
	expected := in_toto.ProvenanceStatementSLSA1{
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
						MediaType: pipelinerun.JsonMediaType,
					}, {
						Name:      "pipelineRunResults/CHAINS-GIT_URL",
						Content:   []uint8(`"https://git.test.com"`),
						MediaType: pipelinerun.JsonMediaType,
					}, {
						Name:      "pipelineRunResults/IMAGE_URL",
						Content:   []uint8(`"test.io/test/image"`),
						MediaType: pipelinerun.JsonMediaType,
					}, {
						Name:      "pipelineRunResults/IMAGE_DIGEST",
						Content:   []uint8(`"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"`),
						MediaType: pipelinerun.JsonMediaType,
					}, {
						Name:      "pipelineRunResults/img-ARTIFACT_INPUTS",
						Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7","uri":"abc"}`),
						MediaType: pipelinerun.JsonMediaType,
					}, {
						Name:      "pipelineRunResults/img2-ARTIFACT_OUTPUTS",
						Content:   []uint8(`{"digest":"sha256:","uri":"def"}`),
						MediaType: pipelinerun.JsonMediaType,
					}, {
						Name:      "pipelineRunResults/img_no_uri-ARTIFACT_OUTPUTS",
						Content:   []uint8(`{"digest":"sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}`),
						MediaType: pipelinerun.JsonMediaType,
					},
				},
			},
		},
	}

	i, _ := NewFormatter(cfg)

	got, err := i.CreatePayload(ctx, pr)

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got, compare.SLSAV1CompareOptions()...); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
	}
}
