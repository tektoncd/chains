/*
Copyright 2020 The Tekton Authors
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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-containerregistry/pkg/name"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/attest"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/compare"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/selection"
	logtesting "knative.dev/pkg/logging/testing"
)

// Global pro is only read from, never modified
var pro *objects.PipelineRunObjectV1Beta1
var proStructuredResults *objects.PipelineRunObjectV1Beta1
var e1BuildStart = time.Unix(1617011400, 0)
var e1BuildFinished = time.Unix(1617011415, 0)

func init() {
	pro = createPro("../../testdata/v1beta1/pipelinerun1.json")
	proStructuredResults = createPro("../../testdata/v1beta1/pipelinerun_structured_results.json")
}

func createPro(path string) *objects.PipelineRunObjectV1Beta1 {
	var err error
	pr, err := objectloader.PipelineRunV1Beta1FromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunV1Beta1FromFile("../../testdata/v1beta1/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunV1Beta1FromFile("../../testdata/v1beta1/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObjectV1Beta1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func TestInvocation(t *testing.T) {
	expected := slsa.ProvenanceInvocation{
		ConfigSource: slsa.ConfigSource{
			URI:        "github.com/test",
			Digest:     map[string]string{"sha1": "28b123"},
			EntryPoint: "pipeline.yaml",
		},
		Parameters: map[string]v1beta1.ParamValue{
			"IMAGE": {Type: "string", StringVal: "test.io/test/image"},
		},
	}
	got := invocation(pro)
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("invocation(): -want +got: %s", diff)
	}
}

func TestBuildConfig(t *testing.T) {
	expected := BuildConfig{
		Tasks: []TaskAttestation{
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
					Parameters: map[string]v1beta1.ParamValue{
						"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskdefault"},
						"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
						"revision":          {Type: "string", StringVal: ""},
						"url":               {Type: "string", StringVal: "https://git.test.com"},
					},
					Environment: map[string]map[string]string{
						"labels": {"tekton.dev/pipelineTask": "git-clone"},
					},
				},
				Results: []v1beta1.TaskRunResult{
					{
						Name: "some-uri_DIGEST",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
						},
					},
					{
						Name: "some-uri",
						Value: v1beta1.ParamValue{
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
					Parameters: map[string]v1beta1.ParamValue{
						"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskrun"},
						"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
						"IMAGE":             {Type: "string", StringVal: "test.io/test/image"},
					},
					Environment: map[string]map[string]string{
						"labels": {"tekton.dev/pipelineTask": "build"},
					},
				},
				Results: []v1beta1.TaskRunResult{
					{
						Name: "IMAGE_DIGEST",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
						},
					},
					{
						Name: "IMAGE_URL",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "gcr.io/my/image",
						},
					},
				},
			},
		},
	}
	ctx := logtesting.TestContextWithLogger(t)
	got := buildConfig(ctx, pro)
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("buildConfig(): -want +got: %s", diff)
	}
}

func TestBuildConfigTaskOrder(t *testing.T) {
	BUILD_TASK := 1
	tests := []struct {
		name            string
		params          []v1beta1.Param
		whenExpressions v1beta1.WhenExpressions
		runAfter        []string
	}{
		{
			name: "Referencing previous task via parameter",
			params: []v1beta1.Param{
				{
					Name:  "CHAINS-GIT_COMMIT",
					Value: v1beta1.ParamValue{Type: "string", StringVal: "$(tasks.git-clone.results.commit)"},
				},
				{
					Name:  "CHAINS-GIT_URL",
					Value: v1beta1.ParamValue{Type: "string", StringVal: "$(tasks.git-clone.results.url)"},
				},
			},
			whenExpressions: nil,
			runAfter:        []string{},
		},
		{
			name:     "Referencing previous task via runAfter",
			params:   []v1beta1.Param{},
			runAfter: []string{"git-clone"},
		},
		{
			name:   "Referencing previous task via when.Input",
			params: []v1beta1.Param{},
			whenExpressions: v1beta1.WhenExpressions{
				{
					Input:    "$(tasks.git-clone.results.commit)",
					Operator: selection.Equals,
					Values:   []string{"abcd"},
				},
			},
			runAfter: []string{},
		},
		{
			name:   "Referencing previous task via when.Value",
			params: []v1beta1.Param{},
			whenExpressions: v1beta1.WhenExpressions{
				{
					Input:    "abcd",
					Operator: selection.Equals,
					Values:   []string{"$(tasks.git-clone.results.commit)"},
				},
			},
			runAfter: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expected := BuildConfig{
				Tasks: []TaskAttestation{
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
							Parameters: map[string]v1beta1.ParamValue{
								"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskdefault"},
								"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
								"url":               {Type: "string", StringVal: "https://git.test.com"},
								"revision":          {Type: "string", StringVal: ""},
							},
							Environment: map[string]map[string]string{
								"labels": {
									"tekton.dev/pipelineTask": "git-clone",
								},
							},
						},
						Results: []v1beta1.TaskRunResult{
							{
								Name: "some-uri_DIGEST",
								Value: v1beta1.ParamValue{
									Type:      v1beta1.ParamTypeString,
									StringVal: "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
								},
							},
							{
								Name: "some-uri",
								Value: v1beta1.ParamValue{
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
							Parameters: map[string]v1beta1.ParamValue{
								// TODO: Is this right?
								// "CHAINS-GIT_COMMIT": {Type: "string", StringVal: "abcd"},
								"CHAINS-GIT_COMMIT": {Type: "string", StringVal: "sha:taskrun"},
								"CHAINS-GIT_URL":    {Type: "string", StringVal: "https://git.test.com"},
								"IMAGE":             {Type: "string", StringVal: "test.io/test/image"},
							},
							Environment: map[string]map[string]string{
								"labels": {
									"tekton.dev/pipelineTask": "build",
								},
							},
						},
						Results: []v1beta1.TaskRunResult{
							{
								Name: "IMAGE_DIGEST",
								Value: v1beta1.ParamValue{
									Type:      v1beta1.ParamTypeString,
									StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
								},
							},
							{
								Name: "IMAGE_URL",
								Value: v1beta1.ParamValue{
									Type:      v1beta1.ParamTypeString,
									StringVal: "gcr.io/my/image",
								},
							},
						},
					},
				},
			}
			pt := v1beta1.PipelineTask{
				Name: "build",
				TaskRef: &v1beta1.TaskRef{
					Kind: "ClusterTask",
					Name: "build",
				},
				Params:          tt.params,
				WhenExpressions: tt.whenExpressions,
				RunAfter:        tt.runAfter,
			}
			pro := createPro("../../testdata/v1beta1/pipelinerun1.json")
			pro.Status.PipelineSpec.Tasks[BUILD_TASK] = pt
			ctx := logtesting.TestContextWithLogger(t)
			got := buildConfig(ctx, pro)
			if diff := cmp.Diff(expected, got); diff != "" {
				t.Errorf("buildConfig(): -want +got: %s", diff)
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	expected := &slsa.ProvenanceMetadata{
		BuildStartedOn:  &e1BuildStart,
		BuildFinishedOn: &e1BuildFinished,
		Completeness: slsa.ProvenanceComplete{
			Parameters:  false,
			Environment: false,
			Materials:   false,
		},
		Reproducible: false,
	}

	got := metadata(pro)
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("metadata(): -want +got: %s", diff)
	}
}

func TestMetadataInTimeZone(t *testing.T) {
	expected := &slsa.ProvenanceMetadata{
		BuildStartedOn:  &e1BuildStart,
		BuildFinishedOn: &e1BuildFinished,
		Completeness: slsa.ProvenanceComplete{
			Parameters:  false,
			Environment: false,
			Materials:   false,
		},
		Reproducible: false,
	}

	zoned := objects.NewPipelineRunObjectV1Beta1(pro.DeepCopy())
	tz := time.FixedZone("Test Time", int((12 * time.Hour).Seconds()))
	zoned.Status.StartTime.Time = zoned.Status.StartTime.Time.In(tz)
	zoned.Status.CompletionTime.Time = zoned.Status.CompletionTime.Time.In(tz)
	got := metadata(zoned)
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("metadata(): -want +got: %s", diff)
	}
}

var ignore = []cmp.Option{cmpopts.IgnoreUnexported(name.Registry{}, name.Repository{}, name.Digest{})}

func TestSubjectDigests(t *testing.T) {
	wantSubjects := []intoto.Subject{
		{
			Name:   "test.io/test/image",
			Digest: common.DigestSet{"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"},
		},
	}

	ctx := logtesting.TestContextWithLogger(t)
	gotSubjects := extract.SubjectDigests(ctx, pro, &slsaconfig.SlsaConfig{DeepInspectionEnabled: false})
	opts := append(ignore, compare.SubjectCompareOption())
	if diff := cmp.Diff(gotSubjects, wantSubjects, opts...); diff != "" {
		t.Errorf("Differences in subjects: -want +got: %s", diff)
	}
}
