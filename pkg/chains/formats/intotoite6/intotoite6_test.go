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

package intotoite6

import (
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/attest"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/pipelinerun"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/objectloader"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	logtesting "knative.dev/pkg/logging/testing"
)

var e1BuildStart = time.Unix(1617011400, 0)
var e1BuildFinished = time.Unix(1617011415, 0)

func TestTaskRunCreatePayload1(t *testing.T) {
	tr, err := objectloader.TaskRunFromFile("testdata/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}
	expected := in_toto.ProvenanceStatement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []in_toto.Subject{
				{
					Name: "gcr.io/my/image",
					Digest: slsa.DigestSet{
						"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					},
				},
			},
		},
		Predicate: slsa.ProvenancePredicate{
			Metadata: &slsa.ProvenanceMetadata{
				BuildStartedOn:  &e1BuildStart,
				BuildFinishedOn: &e1BuildFinished,
			},
			Materials: []slsa.ProvenanceMaterial{
				{URI: "git+https://git.test.com.git", Digest: slsa.DigestSet{"sha1": "sha:taskrun"}},
			},
			Invocation: slsa.ProvenanceInvocation{
				Parameters: map[string]v1beta1.ArrayOrString{
					"IMAGE":             {Type: "string", StringVal: "test.io/test/image"},
					"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskrun"},
					"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
				},
			},
			Builder: slsa.ProvenanceBuilder{
				ID: "test_builder-1",
			},
			BuildType: "tekton.dev/v1beta1/TaskRun",
			BuildConfig: taskrun.BuildConfig{
				Steps: []attest.StepAttestation{
					{
						Arguments: []string(nil),
						Environment: map[string]interface{}{
							"container": string("step1"),
							"image":     string("docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"),
						},
					},
					{
						Arguments: []string(nil),
						Environment: map[string]interface{}{
							"container": string("step2"),
							"image":     string("docker-pullable://gcr.io/test2/test2@sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"),
						},
					},
					{
						Arguments: []string(nil),
						Environment: map[string]interface{}{
							"container": string("step3"),
							"image":     string("docker-pullable://gcr.io/test3/test3@sha256:f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"),
						},
					},
				},
			},
		},
	}
	i, _ := NewFormatter(cfg, logtesting.TestLogger(t))

	got, err := i.CreatePayload(objects.NewTaskRunObject(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestPipelineRunCreatePayload(t *testing.T) {
	pr, err := objectloader.PipelineRunFromFile("testdata/pipelinerun1.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}
	expected := in_toto.ProvenanceStatement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []in_toto.Subject{
				{
					Name: "test.io/test/image",
					Digest: slsa.DigestSet{
						"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					},
				},
			},
		},
		Predicate: slsa.ProvenancePredicate{
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
			Materials: []slsa.ProvenanceMaterial{
				{URI: "git+https://git.test.com.git", Digest: slsa.DigestSet{"sha1": "abcd"}},
			},
			Invocation: slsa.ProvenanceInvocation{
				ConfigSource: slsa.ConfigSource{},
				Parameters: map[string]v1beta1.ArrayOrString{
					"IMAGE": {Type: "string", StringVal: "test.io/test/image"},
				},
			},
			Builder: slsa.ProvenanceBuilder{
				ID: "test_builder-1",
			},
			BuildType: "https://tekton.dev/attestations/chains/pipelinerun@v2",
			BuildConfig: pipelinerun.BuildConfig{
				Tasks: []pipelinerun.TaskAttestation{
					{
						Name:  "git-clone",
						After: nil,
						Ref: v1beta1.TaskRef{
							Name: "git-clone",
							Kind: "ClusterTask",
						},
						StartedOn:  e1BuildStart,
						FinishedOn: e1BuildFinished,
						Status:     "Succeeded",
						Steps: []attest.StepAttestation{
							{
								EntryPoint: "git clone",
								Arguments:  []string(nil),
								Environment: map[string]interface{}{
									"container": "step1",
									"image":     "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
								},
								Annotations: nil,
							},
						},
						Invocation: slsa.ProvenanceInvocation{
							ConfigSource: slsa.ConfigSource{},
							Parameters: map[string]v1beta1.ArrayOrString{
								"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskdefault"},
								"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
								"revision":          {Type: "string", StringVal: ""},
								"url":               {Type: "string", StringVal: "https://git.test.com"},
							},
						},
						Results: []v1beta1.TaskRunResult{
							{
								Name: "some-uri_DIGEST",
								Value: v1beta1.ArrayOrString{
									Type:      v1beta1.ParamTypeString,
									StringVal: "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
								},
							},
							{
								Name: "some-uri",
								Value: v1beta1.ArrayOrString{
									Type:      v1beta1.ParamTypeString,
									StringVal: "pkg:deb/debian/curl@7.50.3-1",
								},
							},
						},
					},
					{
						Name:  "build",
						After: []string{"git-clone"},
						Ref: v1beta1.TaskRef{
							Name: "build",
							Kind: "ClusterTask",
						},
						StartedOn:  e1BuildStart,
						FinishedOn: e1BuildFinished,
						Status:     "Succeeded",
						Steps: []attest.StepAttestation{
							{
								EntryPoint: "",
								Arguments:  []string(nil),
								Environment: map[string]interface{}{
									"image":     "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
									"container": "step1",
								},
								Annotations: nil,
							},
							{
								EntryPoint: "",
								Arguments:  []string(nil),
								Environment: map[string]interface{}{
									"image":     "docker-pullable://gcr.io/test2/test2@sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
									"container": "step2",
								},
								Annotations: nil,
							},
							{
								EntryPoint: "",
								Arguments:  []string(nil),
								Environment: map[string]interface{}{
									"image":     "docker-pullable://gcr.io/test3/test3@sha256:f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478",
									"container": "step3",
								},
								Annotations: nil,
							},
						},
						Invocation: slsa.ProvenanceInvocation{
							ConfigSource: slsa.ConfigSource{},
							Parameters: map[string]v1beta1.ArrayOrString{
								"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskrun"},
								"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
								"IMAGE":             {Type: "string", StringVal: "test.io/test/image"},
							},
						},
						Results: []v1beta1.TaskRunResult{
							{
								Name: "IMAGE_DIGEST",
								Value: v1beta1.ArrayOrString{
									Type:      v1beta1.ParamTypeString,
									StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
								},
							},
							{
								Name: "IMAGE_URL",
								Value: v1beta1.ArrayOrString{
									Type:      v1beta1.ParamTypeString,
									StringVal: "gcr.io/my/image",
								},
							},
						},
					},
				},
			},
		},
	}

	tr1, err := objectloader.TaskRunFromFile("testdata/taskrun1.json")
	if err != nil {
		t.Errorf("error reading taskrun1: %s", err.Error())
	}
	tr2, err := objectloader.TaskRunFromFile("testdata/taskrun2.json")
	if err != nil {
		t.Errorf("error reading taskrun: %s", err.Error())
	}
	pro := objects.NewPipelineRunObject(pr)
	pro.AppendTaskRun(tr1)
	pro.AppendTaskRun(tr2)

	i, _ := NewFormatter(cfg, logtesting.TestLogger(t))

	got, err := i.CreatePayload(pro)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}
func TestPipelineRunCreatePayloadChildRefs(t *testing.T) {
	pr, err := objectloader.PipelineRunFromFile("testdata/pipelinerun-childrefs.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}
	expected := in_toto.ProvenanceStatement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []in_toto.Subject{
				{
					Name: "test.io/test/image",
					Digest: slsa.DigestSet{
						"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					},
				},
			},
		},
		Predicate: slsa.ProvenancePredicate{
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
			Materials: []slsa.ProvenanceMaterial{
				{URI: "git+https://git.test.com.git", Digest: slsa.DigestSet{"sha1": "abcd"}},
			},
			Invocation: slsa.ProvenanceInvocation{
				ConfigSource: slsa.ConfigSource{},
				Parameters: map[string]v1beta1.ArrayOrString{
					"IMAGE": {Type: "string", StringVal: "test.io/test/image"},
				},
			},
			Builder: slsa.ProvenanceBuilder{
				ID: "test_builder-1",
			},
			BuildType: "https://tekton.dev/attestations/chains/pipelinerun@v2",
			BuildConfig: pipelinerun.BuildConfig{
				Tasks: []pipelinerun.TaskAttestation{
					{
						Name:  "git-clone",
						After: nil,
						Ref: v1beta1.TaskRef{
							Name: "git-clone",
							Kind: "ClusterTask",
						},
						StartedOn:  e1BuildStart,
						FinishedOn: e1BuildFinished,
						Status:     "Succeeded",
						Steps: []attest.StepAttestation{
							{
								EntryPoint: "git clone",
								Arguments:  []string(nil),
								Environment: map[string]interface{}{
									"container": "step1",
									"image":     "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
								},
								Annotations: nil,
							},
						},
						Invocation: slsa.ProvenanceInvocation{
							ConfigSource: slsa.ConfigSource{},
							Parameters: map[string]v1beta1.ArrayOrString{
								"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskdefault"},
								"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
								"revision":          {Type: "string", StringVal: ""},
								"url":               {Type: "string", StringVal: "https://git.test.com"},
							},
						},
						Results: []v1beta1.TaskRunResult{
							{
								Name: "some-uri_DIGEST",
								Value: v1beta1.ArrayOrString{
									Type:      v1beta1.ParamTypeString,
									StringVal: "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
								},
							},
							{
								Name: "some-uri",
								Value: v1beta1.ArrayOrString{
									Type:      v1beta1.ParamTypeString,
									StringVal: "pkg:deb/debian/curl@7.50.3-1",
								},
							},
						},
					},
					{
						Name:  "build",
						After: []string{"git-clone"},
						Ref: v1beta1.TaskRef{
							Name: "build",
							Kind: "ClusterTask",
						},
						StartedOn:  e1BuildStart,
						FinishedOn: e1BuildFinished,
						Status:     "Succeeded",
						Steps: []attest.StepAttestation{
							{
								EntryPoint: "",
								Arguments:  []string(nil),
								Environment: map[string]interface{}{
									"image":     "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
									"container": "step1",
								},
								Annotations: nil,
							},
							{
								EntryPoint: "",
								Arguments:  []string(nil),
								Environment: map[string]interface{}{
									"image":     "docker-pullable://gcr.io/test2/test2@sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
									"container": "step2",
								},
								Annotations: nil,
							},
							{
								EntryPoint: "",
								Arguments:  []string(nil),
								Environment: map[string]interface{}{
									"image":     "docker-pullable://gcr.io/test3/test3@sha256:f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478",
									"container": "step3",
								},
								Annotations: nil,
							},
						},
						Invocation: slsa.ProvenanceInvocation{
							ConfigSource: slsa.ConfigSource{},
							Parameters: map[string]v1beta1.ArrayOrString{
								"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskrun"},
								"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
								"IMAGE":             {Type: "string", StringVal: "test.io/test/image"},
							},
						},
						Results: []v1beta1.TaskRunResult{
							{
								Name: "IMAGE_DIGEST",
								Value: v1beta1.ArrayOrString{
									Type:      v1beta1.ParamTypeString,
									StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
								},
							},
							{
								Name: "IMAGE_URL",
								Value: v1beta1.ArrayOrString{
									Type:      v1beta1.ParamTypeString,
									StringVal: "gcr.io/my/image",
								},
							},
						},
					},
				},
			},
		},
	}

	tr1, err := objectloader.TaskRunFromFile("testdata/taskrun1.json")
	if err != nil {
		t.Errorf("error reading taskrun1: %s", err.Error())
	}
	tr2, err := objectloader.TaskRunFromFile("testdata/taskrun2.json")
	if err != nil {
		t.Errorf("error reading taskrun: %s", err.Error())
	}
	pro := objects.NewPipelineRunObject(pr)
	pro.AppendTaskRun(tr1)
	pro.AppendTaskRun(tr2)

	i, _ := NewFormatter(cfg, logtesting.TestLogger(t))
	got, err := i.CreatePayload(pro)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestTaskRunCreatePayload2(t *testing.T) {
	tr, err := objectloader.TaskRunFromFile("testdata/taskrun2.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-2",
		},
	}
	expected := in_toto.ProvenanceStatement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       nil,
		},
		Predicate: slsa.ProvenancePredicate{
			Metadata: &slsa.ProvenanceMetadata{
				BuildStartedOn:  &e1BuildStart,
				BuildFinishedOn: &e1BuildFinished,
			},
			Builder: slsa.ProvenanceBuilder{
				ID: "test_builder-2",
			},
			Materials: []slsa.ProvenanceMaterial{
				{URI: "git+https://git.test.com.git", Digest: slsa.DigestSet{"sha1": "sha:taskdefault"}},
			},
			Invocation: slsa.ProvenanceInvocation{
				Parameters: map[string]v1beta1.ArrayOrString{
					"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskdefault"},
					"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
					"revision":          {Type: "string"},
					"url":               {Type: "string", StringVal: "https://git.test.com"},
				},
			},
			BuildType: "tekton.dev/v1beta1/TaskRun",
			BuildConfig: taskrun.BuildConfig{
				Steps: []attest.StepAttestation{
					{
						EntryPoint: "git clone",
						Arguments:  []string(nil),
						Environment: map[string]interface{}{
							"container": string("step1"),
							"image":     string("docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"),
						},
					},
				},
			},
		},
	}
	i, _ := NewFormatter(cfg, logtesting.TestLogger(t))
	got, err := i.CreatePayload(objects.NewTaskRunObject(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestCreatePayloadNilTaskRef(t *testing.T) {
	tr, err := objectloader.TaskRunFromFile("testdata/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}

	tr.Spec.TaskRef = nil
	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "testid",
		},
	}
	f, _ := NewFormatter(cfg, logtesting.TestLogger(t))

	p, err := f.CreatePayload(objects.NewTaskRunObject(tr))
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	ps := p.(in_toto.ProvenanceStatement)
	if diff := cmp.Diff("", ps.Predicate.Invocation.ConfigSource.EntryPoint); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestMultipleSubjects(t *testing.T) {
	tr, err := objectloader.TaskRunFromFile("testdata/taskrun-multiple-subjects.json")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-multiple",
		},
	}
	expected := in_toto.ProvenanceStatement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []in_toto.Subject{
				{
					Name: "gcr.io/myimage",
					Digest: slsa.DigestSet{
						"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
					},
				}, {
					Name: "gcr.io/myimage",
					Digest: slsa.DigestSet{
						"sha256": "daa1a56e13c85cf164e7d9e595006649e3a04c47fe4a8261320e18a0bf3b0367",
					},
				},
			},
		},
		Predicate: slsa.ProvenancePredicate{
			BuildType: "tekton.dev/v1beta1/TaskRun",
			Metadata:  &slsa.ProvenanceMetadata{},
			Builder: slsa.ProvenanceBuilder{
				ID: "test_builder-multiple",
			},
			Invocation: slsa.ProvenanceInvocation{
				Parameters: map[string]v1beta1.ArrayOrString{},
			},
			BuildConfig: taskrun.BuildConfig{
				Steps: []attest.StepAttestation{
					{
						Arguments: []string(nil),
						Environment: map[string]interface{}{
							"container": string("step1"),
							"image":     string("docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"),
						},
					},
				},
			},
		},
	}

	i, _ := NewFormatter(cfg, logtesting.TestLogger(t))
	got, err := i.CreatePayload(objects.NewTaskRunObject(tr))
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
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
		f, err := NewFormatter(cfg, logtesting.TestLogger(t))
		if f == nil {
			t.Error("Failed to create formatter")
		}
		if err != nil {
			t.Errorf("Error creating formatter: %s", err)
		}
	})
}

func TestCreatePayloadError(t *testing.T) {
	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "testid",
		},
	}
	f, _ := NewFormatter(cfg, logtesting.TestLogger(t))

	t.Run("Invalid type", func(t *testing.T) {
		p, err := f.CreatePayload("not a task ref")

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
	if i.Type() != formats.PayloadTypeInTotoIte6 {
		t.Errorf("Invalid type returned: %s", i.Type())
	}
}
