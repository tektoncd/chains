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

package v2alpha1

import (
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha1/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	logtesting "knative.dev/pkg/logging/testing"
)

var (
	e1BuildStart    = time.Unix(1617011400, 0)
	e1BuildFinished = time.Unix(1617011415, 0)
)

func TestTaskRunCreatePayload1(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	tr, err := objectloader.TaskRunV1Beta1FromFile("../testdata/pipeline-v1beta1/taskrun1.json")
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
					Digest: common.DigestSet{
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
					Parameters: true,
				},
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
				Parameters: map[string]any{
					"ComputeResources": (*corev1.ResourceRequirements)(nil),
					"Debug":            (*v1beta1.TaskRunDebug)(nil),
					"Params": v1beta1.Params{
						{
							Name:  "IMAGE",
							Value: v1beta1.ParamValue{Type: "string", StringVal: "test.io/test/image"},
						},
						{
							Name:  "CHAINS-GIT_COMMIT",
							Value: v1beta1.ParamValue{Type: "string", StringVal: "sha:taskrun"},
						},
						{
							Name:  "CHAINS-GIT_URL",
							Value: v1beta1.ParamValue{Type: "string", StringVal: "https://git.test.com"},
						},
					},
					"PodTemplate":        (*pod.Template)(nil),
					"Resources":          (*v1beta1.TaskRunResources)(nil), //nolint:staticcheck
					"Retries":            0,
					"ServiceAccountName": "default",
					"SidecarOverrides":   []v1beta1.TaskRunSidecarOverride(nil),
					"Status":             v1beta1.TaskRunSpecStatus(""),
					"StatusMessage":      v1beta1.TaskRunSpecStatusMessage(""),
					"StepOverrides":      []v1beta1.TaskRunStepOverride(nil),
					"Timeout":            (*metav1.Duration)(nil),
					"Workspaces":         []v1beta1.WorkspaceBinding(nil),
				},
			},
			Builder: common.ProvenanceBuilder{
				ID: "test_builder-1",
			},
			BuildType: "https://chains.tekton.dev/format/slsa/v2alpha1/type/tekton.dev/v1beta1/TaskRun",
			BuildConfig: taskrun.BuildConfig{
				TaskSpec: &v1beta1.TaskSpec{
					Params: []v1beta1.ParamSpec{
						{Name: "IMAGE", Type: "string"}, {Name: "filename", Type: "string"},
						{Name: "DOCKERFILE", Type: "string"}, {Name: "CONTEXT", Type: "string"},
						{Name: "EXTRA_ARGS", Type: "string"}, {Name: "BUILDER_IMAGE", Type: "string"},
						{Name: "CHAINS-GIT_COMMIT", Type: "string", Default: &v1beta1.ParamValue{Type: "string", StringVal: "sha:task"}},
						{Name: "CHAINS-GIT_URL", Type: "string", Default: &v1beta1.ParamValue{Type: "string", StringVal: "https://defaultgit.test.com"}},
					},
					Steps: []v1beta1.Step{{Name: "step1"}, {Name: "step2"}, {Name: "step3"}},
					Results: []v1beta1.TaskResult{
						{Name: "IMAGE_DIGEST", Description: "Digest of the image just built."},
						{Name: "filename_DIGEST", Description: "Digest of the file just built."},
					},
				},
				TaskRunResults: []v1beta1.TaskRunResult{
					{
						Name:  "IMAGE_DIGEST",
						Value: v1beta1.ParamValue{Type: "string", StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"},
					},
					{
						Name:  "IMAGE_URL",
						Value: v1beta1.ParamValue{Type: "string", StringVal: "gcr.io/my/image"},
					},
				},
			},
		},
	}
	i, _ := NewFormatter(cfg)

	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1Beta1(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
	}
}

func TestTaskRunCreatePayload2(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	tr, err := objectloader.TaskRunV1Beta1FromFile("../testdata/pipeline-v1beta1/taskrun2.json")
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
				Completeness: slsa.ProvenanceComplete{
					Parameters: true,
				},
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
				Parameters: map[string]any{
					"ComputeResources": (*corev1.ResourceRequirements)(nil),
					"Debug":            (*v1beta1.TaskRunDebug)(nil),
					"Params": v1beta1.Params{
						{
							Name:  "url",
							Value: v1beta1.ParamValue{Type: "string", StringVal: "https://git.test.com"},
						},
						{Name: "revision", Value: v1beta1.ParamValue{Type: "string"}},
					},
					"PodTemplate":        (*pod.Template)(nil),
					"Resources":          (*v1beta1.TaskRunResources)(nil), //nolint:staticcheck
					"Retries":            0,
					"ServiceAccountName": "default",
					"SidecarOverrides":   []v1beta1.TaskRunSidecarOverride(nil),
					"Status":             v1beta1.TaskRunSpecStatus(""),
					"StatusMessage":      v1beta1.TaskRunSpecStatusMessage(""),
					"StepOverrides":      []v1beta1.TaskRunStepOverride(nil),
					"Timeout":            (*metav1.Duration)(nil),
					"Workspaces":         []v1beta1.WorkspaceBinding(nil),
				},
			},
			BuildType: "https://chains.tekton.dev/format/slsa/v2alpha1/type/tekton.dev/v1beta1/TaskRun",
			BuildConfig: taskrun.BuildConfig{
				TaskSpec: &v1beta1.TaskSpec{
					Params: []v1beta1.ParamSpec{
						{Name: "CHAINS-GIT_COMMIT", Type: "string", Default: &v1beta1.ParamValue{Type: "string", StringVal: "sha:taskdefault"}},
						{Name: "CHAINS-GIT_URL", Type: "string", Default: &v1beta1.ParamValue{Type: "string", StringVal: "https://git.test.com"}},
					},
					Steps: []v1beta1.Step{{Name: "step1", Env: []v1.EnvVar{{Name: "HOME", Value: "$(params.userHome)"}, {Name: "PARAM_URL", Value: "$(params.url)"}}, Script: "git clone"}},
					Results: []v1beta1.TaskResult{
						{Name: "some-uri_DIGEST", Description: "Digest of a file to push."},
						{Name: "some-uri", Description: "some calculated uri"},
					},
				},
				TaskRunResults: []v1beta1.TaskRunResult{
					{
						Name:  "some-uri_DIGEST",
						Value: v1beta1.ParamValue{Type: "string", StringVal: "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
					},
					{
						Name:  "some-uri",
						Value: v1beta1.ParamValue{Type: "string", StringVal: "pkg:deb/debian/curl@7.50.3-1"},
					},
				},
			},
		},
	}
	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1Beta1(tr))

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
	}
}

func TestMultipleSubjects(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)

	tr, err := objectloader.TaskRunV1Beta1FromFile("../testdata/pipeline-v1beta1/taskrun-multiple-subjects.json")
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
			BuildType: "https://chains.tekton.dev/format/slsa/v2alpha1/type/tekton.dev/v1beta1/TaskRun",
			Metadata: &slsa.ProvenanceMetadata{
				Completeness: slsa.ProvenanceComplete{
					Parameters: true,
				},
			},
			Builder: common.ProvenanceBuilder{
				ID: "test_builder-multiple",
			},
			Materials: []common.ProvenanceMaterial{
				{
					URI:    artifacts.OCIScheme + "gcr.io/test1/test1",
					Digest: common.DigestSet{"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"},
				},
			},
			Invocation: slsa.ProvenanceInvocation{
				Parameters: map[string]any{
					"ComputeResources":   (*corev1.ResourceRequirements)(nil),
					"Debug":              (*v1beta1.TaskRunDebug)(nil),
					"Params":             v1beta1.Params{},
					"PodTemplate":        (*pod.Template)(nil),
					"Resources":          (*v1beta1.TaskRunResources)(nil), //nolint:staticcheck
					"Retries":            0,
					"ServiceAccountName": "default",
					"SidecarOverrides":   []v1beta1.TaskRunSidecarOverride(nil),
					"Status":             v1beta1.TaskRunSpecStatus(""),
					"StatusMessage":      v1beta1.TaskRunSpecStatusMessage(""),
					"StepOverrides":      []v1beta1.TaskRunStepOverride(nil),
					"Timeout":            (*metav1.Duration)(nil),
					"Workspaces":         []v1beta1.WorkspaceBinding(nil),
				},
			},
			BuildConfig: taskrun.BuildConfig{
				TaskSpec: &v1beta1.TaskSpec{
					Params: []v1beta1.ParamSpec{},
					Results: []v1beta1.TaskResult{
						{Name: "file1_DIGEST", Description: "Digest of a file to push."},
						{Name: "file1", Description: "some assembled file"},
						{Name: "file2_DIGEST", Description: "Digest of a file to push."},
						{Name: "file2", Description: "some assembled file"},
					},
				},
				TaskRunResults: []v1beta1.TaskRunResult{
					{
						Name: "IMAGES",
						Value: v1beta1.ParamValue{
							Type:      "string",
							StringVal: "gcr.io/myimage1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6,gcr.io/myimage2@sha256:daa1a56e13c85cf164e7d9e595006649e3a04c47fe4a8261320e18a0bf3b0367",
						},
					},
				},
			},
		},
	}

	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(ctx, objects.NewTaskRunObjectV1Beta1(tr))
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected, got); diff != "" {
		t.Errorf("Slsa.CreatePayload(): -want +got: %s", diff)
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
	var i Slsa
	if i.Type() != formats.PayloadTypeSlsav2alpha1 {
		t.Errorf("Invalid type returned: %s", i.Type())
	}
}
