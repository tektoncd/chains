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

package v1

import (
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/attest"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/compare"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v1/internal/protos"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v1/pipelinerun"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v1/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/google/go-cmp/cmp"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

var e1BuildStart = time.Unix(1617011400, 0).UTC()
var e1BuildFinished = time.Unix(1617011415, 0).UTC()

func TestTaskRunCreatePayload1(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	tr, err := objectloader.TaskRunV1FromFile("../testdata/pipeline-v1/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}

	predicate := slsa.ProvenancePredicate{
		Metadata: &slsa.ProvenanceMetadata{
			BuildStartedOn:  &e1BuildStart,
			BuildFinishedOn: &e1BuildFinished,
		},
		Materials: []common.ProvenanceMaterial{
			{
				URI:    artifacts.OCIScheme + "gcr.io/test1/test1",
				Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
			},
			{
				URI:    artifacts.OCIScheme + "gcr.io/test2/test2",
				Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
			},
			{
				URI:    artifacts.OCIScheme + "gcr.io/test3/test3",
				Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
			},
			{URI: artifacts.GitSchemePrefix + "https://git.test.com.git", Digest: common.DigestSet{"sha1": "sha:taskrun"}},
		},
		Invocation: slsa.ProvenanceInvocation{
			ConfigSource: slsa.ConfigSource{
				URI:        "github.com/test",
				Digest:     map[string]string{"sha1": "ab123"},
				EntryPoint: "build.yaml",
			},
			Parameters: map[string]v1.ParamValue{
				"IMAGE":             {Type: "string", StringVal: "test.io/test/image"},
				"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskrun"},
				"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
			},
			Environment: map[string]map[string]string{
				"labels": {"tekton.dev/pipelineTask": "build"},
			},
		},
		Builder: common.ProvenanceBuilder{
			ID: "test_builder-1",
		},
		BuildType: "tekton.dev/v1/TaskRun",
		BuildConfig: taskrun.BuildConfig{
			Steps: []attest.StepAttestation{
				{
					Arguments: []string(nil),
					Environment: map[string]interface{}{
						"container": string("step1"),
						"image":     artifacts.OCIScheme + "gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
					},
				},
				{
					Arguments: []string(nil),
					Environment: map[string]interface{}{
						"container": string("step2"),
						"image":     artifacts.OCIScheme + "gcr.io/test2/test2@sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
					},
				},
				{
					Arguments: []string(nil),
					Environment: map[string]interface{}{
						"container": string("step3"),
						"image":     artifacts.OCIScheme + "gcr.io/test3/test3@sha256:f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478",
					},
				},
			},
		},
	}

	predicateStruct, err := protos.GetPredicateStruct(&predicate)
	if err != nil {
		t.Fatalf("error creating proto struct from predicate: %v", err)
	}

	expected := &intoto.Statement{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: slsa.PredicateSLSAProvenance,
		Subject: []*intoto.ResourceDescriptor{
			{
				Name: "gcr.io/my/image",
				Digest: common.DigestSet{
					"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
				},
			},
		},
		Predicate: predicateStruct,
	}
	i, _ := NewFormatter(cfg)

	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	if diff := cmp.Diff(expected, got, cmp.Options{protocmp.Transform()}); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestPipelineRunCreatePayload(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	pr, err := objectloader.PipelineRunV1FromFile("../testdata/pipeline-v1/pipelinerun1.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}

	predicate := &slsa.ProvenancePredicate{
		Metadata: &slsa.ProvenanceMetadata{
			BuildStartedOn:  &e1BuildStart,
			BuildFinishedOn: &e1BuildFinished,
			Completeness: slsa.ProvenanceComplete{
				Parameters:  false,
				Environment: false,
				Materials:   false,
			},
			Reproducible: false,
		},
		Materials: []common.ProvenanceMaterial{
			{URI: "github.com/test", Digest: common.DigestSet{"sha1": "28b123"}},
			{
				URI:    artifacts.OCIScheme + "gcr.io/test1/test1",
				Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
			},
			{URI: "github.com/catalog", Digest: common.DigestSet{"sha1": "x123"}},
			{
				URI:    artifacts.OCIScheme + "gcr.io/test2/test2",
				Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
			},
			{
				URI:    artifacts.OCIScheme + "gcr.io/test3/test3",
				Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
			},
			{URI: "github.com/test", Digest: common.DigestSet{"sha1": "ab123"}},
			{URI: "abc", Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"}},
			{URI: artifacts.GitSchemePrefix + "https://git.test.com.git", Digest: common.DigestSet{"sha1": "abcd"}},
		},
		Invocation: slsa.ProvenanceInvocation{
			ConfigSource: slsa.ConfigSource{
				URI:        "github.com/test",
				Digest:     map[string]string{"sha1": "28b123"},
				EntryPoint: "pipeline.yaml",
			},
			Parameters: map[string]v1.ParamValue{
				"IMAGE": {Type: "string", StringVal: "test.io/test/image"},
			},
		},
		Builder: common.ProvenanceBuilder{
			ID: "test_builder-1",
		},
		BuildType:   "tekton.dev/v1/PipelineRun",
		BuildConfig: getBuildPipelineRun(),
	}

	predicateStruct, err := protos.GetPredicateStruct(predicate)
	if err != nil {
		t.Fatalf("error getting predicate struct: %v", err)
	}

	expected := &intoto.Statement{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: slsa.PredicateSLSAProvenance,
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

	tr1, err := objectloader.TaskRunV1FromFile("../testdata/pipeline-v1/taskrun1.json")
	if err != nil {
		t.Errorf("error reading taskrun1: %s", err.Error())
	}
	tr2, err := objectloader.TaskRunV1FromFile("../testdata/pipeline-v1/taskrun2.json")
	if err != nil {
		t.Errorf("error reading taskrun: %s", err.Error())
	}
	pro := objects.NewPipelineRunObjectV1(pr)
	pro.AppendTaskRun(tr1)
	pro.AppendTaskRun(tr2)

	i, _ := NewFormatter(cfg)

	got, err := i.CreatePayload(ctx, pro)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	// Sort Materials since their order can vary and result in flakes
	if diff := cmp.Diff(expected, got, compare.MaterialsCompareOption(), protocmp.Transform()); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}
func TestPipelineRunCreatePayloadChildRefs(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	pr, err := objectloader.PipelineRunV1FromFile("../testdata/pipeline-v1/pipelinerun-childrefs.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}

	predicate := &slsa.ProvenancePredicate{
		Metadata: &slsa.ProvenanceMetadata{
			BuildStartedOn:  &e1BuildStart,
			BuildFinishedOn: &e1BuildFinished,
			Completeness: slsa.ProvenanceComplete{
				Parameters:  false,
				Environment: false,
				Materials:   false,
			},
			Reproducible: false,
		},
		Materials: []common.ProvenanceMaterial{
			{
				URI:    artifacts.OCIScheme + "gcr.io/test1/test1",
				Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
			},
			{URI: "github.com/catalog", Digest: common.DigestSet{"sha1": "x123"}},
			{
				URI:    artifacts.OCIScheme + "gcr.io/test2/test2",
				Digest: common.DigestSet{"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"},
			},
			{
				URI:    artifacts.OCIScheme + "gcr.io/test3/test3",
				Digest: common.DigestSet{"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"},
			},
			{URI: "github.com/test", Digest: common.DigestSet{"sha1": "ab123"}},
			{URI: artifacts.GitSchemePrefix + "https://git.test.com.git", Digest: common.DigestSet{"sha1": "abcd"}},
		},
		Invocation: slsa.ProvenanceInvocation{
			ConfigSource: slsa.ConfigSource{},
			Parameters: map[string]v1.ParamValue{
				"IMAGE": {Type: "string", StringVal: "test.io/test/image"},
			},
		},
		Builder: common.ProvenanceBuilder{
			ID: "test_builder-1",
		},
		BuildType:   "tekton.dev/v1/PipelineRun",
		BuildConfig: getBuildPipelineRun(),
	}

	predicateStruct, err := protos.GetPredicateStruct(predicate)
	if err != nil {
		t.Fatalf("error getting predicate struct: %v", err)
	}

	expected := &intoto.Statement{

		Type:          in_toto.StatementInTotoV01,
		PredicateType: slsa.PredicateSLSAProvenance,
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

	tr1, err := objectloader.TaskRunV1FromFile("../testdata/pipeline-v1/taskrun1.json")
	if err != nil {
		t.Errorf("error reading taskrun1: %s", err.Error())
	}
	tr2, err := objectloader.TaskRunV1FromFile("../testdata/pipeline-v1/taskrun2.json")
	if err != nil {
		t.Errorf("error reading taskrun: %s", err.Error())
	}
	pro := objects.NewPipelineRunObjectV1(pr)
	pro.AppendTaskRun(tr1)
	pro.AppendTaskRun(tr2)

	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(ctx, pro)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	// Sort Materials since their order can vary and result in flakes
	if diff := cmp.Diff(expected, got, compare.MaterialsCompareOption(), protocmp.Transform()); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestTaskRunCreatePayload2(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	tr, err := objectloader.TaskRunV1FromFile("../testdata/pipeline-v1/taskrun2.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-2",
		},
	}

	predicate := &slsa.ProvenancePredicate{
		Metadata: &slsa.ProvenanceMetadata{
			BuildStartedOn:  &e1BuildStart,
			BuildFinishedOn: &e1BuildFinished,
		},
		Builder: common.ProvenanceBuilder{
			ID: "test_builder-2",
		},
		Materials: []common.ProvenanceMaterial{
			{
				URI:    artifacts.OCIScheme + "gcr.io/test1/test1",
				Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
			},
			{URI: artifacts.GitSchemePrefix + "https://git.test.com.git", Digest: common.DigestSet{"sha1": "sha:taskdefault"}},
		},
		Invocation: slsa.ProvenanceInvocation{
			ConfigSource: slsa.ConfigSource{
				URI:        "github.com/catalog",
				Digest:     common.DigestSet{"sha1": "x123"},
				EntryPoint: "git-clone.yaml",
			},
			Parameters: map[string]v1.ParamValue{
				"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskdefault"},
				"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
				"revision":          {Type: "string"},
				"url":               {Type: "string", StringVal: "https://git.test.com"},
			},
			Environment: map[string]map[string]string{
				"labels": {"tekton.dev/pipelineTask": "git-clone"},
			},
		},
		BuildType: "tekton.dev/v1/TaskRun",
		BuildConfig: taskrun.BuildConfig{
			Steps: []attest.StepAttestation{
				{
					EntryPoint: "git clone",
					Arguments:  []string(nil),
					Environment: map[string]interface{}{
						"container": string("step1"),
						"image":     artifacts.OCIScheme + "gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
					},
				},
			},
		},
	}

	predicateStruct, err := protos.GetPredicateStruct(predicate)
	if err != nil {
		t.Fatalf("error getting predicate struct: %v", err)
	}

	expected := &intoto.Statement{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: slsa.PredicateSLSAProvenance,
		Subject:       nil,
		Predicate:     predicateStruct,
	}
	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got, protocmp.Transform()); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestMultipleSubjects(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	tr, err := objectloader.TaskRunV1FromFile("../testdata/pipeline-v1/taskrun-multiple-subjects.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-multiple",
		},
	}

	predicate := &slsa.ProvenancePredicate{
		BuildType: "tekton.dev/v1/TaskRun",
		Metadata:  &slsa.ProvenanceMetadata{},
		Builder: common.ProvenanceBuilder{
			ID: "test_builder-multiple",
		},
		Materials: []common.ProvenanceMaterial{
			{
				URI:    artifacts.OCIScheme + "gcr.io/test1/test1",
				Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
			},
			{
				URI:    artifacts.OCIScheme + "gcr.io/test1/test2",
				Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b5"},
			},
		},
		Invocation: slsa.ProvenanceInvocation{
			Parameters: map[string]v1.ParamValue{},
		},
		BuildConfig: taskrun.BuildConfig{
			Steps: []attest.StepAttestation{
				{
					Arguments: []string(nil),
					Environment: map[string]interface{}{
						"container": string("step1"),
						"image":     artifacts.OCIScheme + "gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
					},
				},
				{
					Arguments: []string(nil),
					Environment: map[string]interface{}{
						"container": string("step2"),
						"image":     artifacts.OCIScheme + "gcr.io/test1/test2@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b5",
					},
				},
			},
		},
	}

	predicateStruct, err := protos.GetPredicateStruct(predicate)
	if err != nil {
		t.Fatalf("error getting predicate struct: %v", err)
	}

	expected := &intoto.Statement{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: slsa.PredicateSLSAProvenance,
		Subject: []*intoto.ResourceDescriptor{
			{
				Name: "gcr.io/myimage1",
				Digest: common.DigestSet{
					"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
			{
				Name: "gcr.io/myimage2",
				Digest: common.DigestSet{
					"sha256": "daa1a56e13c85cf164e7d9e595006649e3a04c47fe4a8261320e18a0bf3b0367",
				},
			},
		},
		Predicate: predicateStruct,
	}

	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1(tr))
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got, protocmp.Transform()); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestNewFormatter(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		cfg := config.Config{
			Builder: config.BuilderConfig{
				ID: "testid",
			},
		}
		f, err := NewFormatter(cfg)
		if f == nil {
			t.Error("Failed to create formatter")
		}
		if err != nil {
			t.Errorf("Error creating formatter: %s", err)
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
	var i InTotoIte6
	if i.Type() != formats.PayloadTypeSlsav1 {
		t.Errorf("Invalid type returned: %s", i.Type())
	}
}

func getBuildPipelineRun() pipelinerun.BuildConfig {
	return pipelinerun.BuildConfig{
		Tasks: []pipelinerun.TaskAttestation{
			{
				Name:  "git-clone",
				After: nil,
				Ref: v1.TaskRef{
					Name: "git-clone",
					Kind: "ClusterTask",
				},
				StartedOn:          e1BuildStart,
				FinishedOn:         e1BuildFinished,
				ServiceAccountName: "pipeline",
				Status:             "Succeeded",
				Steps: []attest.StepAttestation{
					{
						EntryPoint: "git clone",
						Arguments:  []string(nil),
						Environment: map[string]interface{}{
							"container": "step1",
							"image":     artifacts.OCIScheme + "gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
						},
						Annotations: nil,
					},
				},
				Invocation: slsa.ProvenanceInvocation{
					ConfigSource: slsa.ConfigSource{
						URI:        "github.com/catalog",
						Digest:     common.DigestSet{"sha1": "x123"},
						EntryPoint: "git-clone.yaml",
					},
					Parameters: map[string]v1.ParamValue{
						"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskdefault"},
						"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
						"revision":          {Type: "string", StringVal: ""},
						"url":               {Type: "string", StringVal: "https://git.test.com"},
					},
					Environment: map[string]map[string]string{
						"labels": {"tekton.dev/pipelineTask": "git-clone"},
					},
				},
				Results: []v1.TaskRunResult{
					{
						Name: "some-uri_DIGEST",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
						},
					},
					{
						Name: "some-uri",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "pkg:deb/debian/curl@7.50.3-1",
						},
					},
				},
			},
			{
				Name:  "build",
				After: []string{"git-clone"},
				Ref: v1.TaskRef{
					Name: "build",
					Kind: "ClusterTask",
				},
				StartedOn:          e1BuildStart,
				FinishedOn:         e1BuildFinished,
				ServiceAccountName: "pipeline",
				Status:             "Succeeded",
				Steps: []attest.StepAttestation{
					{
						EntryPoint: "",
						Arguments:  []string(nil),
						Environment: map[string]interface{}{
							"image":     artifacts.OCIScheme + "gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
							"container": "step1",
						},
						Annotations: nil,
					},
					{
						EntryPoint: "",
						Arguments:  []string(nil),
						Environment: map[string]interface{}{
							"image":     artifacts.OCIScheme + "gcr.io/test2/test2@sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
							"container": "step2",
						},
						Annotations: nil,
					},
					{
						EntryPoint: "",
						Arguments:  []string(nil),
						Environment: map[string]interface{}{
							"image":     artifacts.OCIScheme + "gcr.io/test3/test3@sha256:f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478",
							"container": "step3",
						},
						Annotations: nil,
					},
				},
				Invocation: slsa.ProvenanceInvocation{
					ConfigSource: slsa.ConfigSource{
						URI:        "github.com/test",
						Digest:     map[string]string{"sha1": "ab123"},
						EntryPoint: "build.yaml",
					},
					Parameters: map[string]v1.ParamValue{
						"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskrun"},
						"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
						"IMAGE":             {Type: "string", StringVal: "test.io/test/image"},
					},
					Environment: map[string]map[string]string{
						"labels": {"tekton.dev/pipelineTask": "build"},
					},
				},
				Results: []v1.TaskRunResult{
					{
						Name: "IMAGE_DIGEST",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
						},
					},
					{
						Name: "IMAGE_URL",
						Value: v1.ParamValue{
							Type:      v1.ParamTypeString,
							StringVal: "gcr.io/my/image",
						},
					},
				},
			},
		},
	}
}
