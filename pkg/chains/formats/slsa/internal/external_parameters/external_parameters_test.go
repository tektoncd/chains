/*
Copyright 2023 The Tekton Authors

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

package externalparameters

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func createPro(path string) *objects.PipelineRunObjectV1 {
	pr, err := objectloader.PipelineRunV1FromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunV1FromFile("../../testdata/slsa-v2alpha3/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunV1FromFile("../../testdata/slsa-v2alpha3/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObjectV1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func TestBuildConfigSource(t *testing.T) {
	digest := map[string]string{"alg1": "hex1", "alg2": "hex2"}
	provenance := &v1.Provenance{
		RefSource: &v1.RefSource{
			Digest:     digest,
			URI:        "https://tekton.com",
			EntryPoint: "/path/to/entry",
		},
	}

	want := map[string]string{
		"repository": "https://tekton.com",
		"path":       "/path/to/entry",
	}

	got := BuildConfigSource(provenance)

	gotRef := strings.Split(got["ref"], ":")
	if len(gotRef) != 2 {
		t.Errorf("buildConfigSource() does not return the proper ref: want one of: %s got: %s", digest, got["ref"])
	}
	refValue, ok := digest[gotRef[0]]
	if !ok {
		t.Errorf("buildConfigSource() does not contain correct ref: want one of: %s got: %s:%s", digest, gotRef[0], gotRef[1])
	}

	if refValue != gotRef[1] {
		t.Errorf("buildConfigSource() does not contain correct ref: want one of: %s got: %s:%s", digest, gotRef[0], gotRef[1])
	}

	if got["repository"] != want["repository"] {
		t.Errorf("buildConfigSource() does not contain correct repository: want: %s got: %s", want["repository"], want["repository"])
	}

	if got["path"] != want["path"] {
		t.Errorf("buildConfigSource() does not contain correct path: want: %s got: %s", want["path"], got["path"])
	}
}

func TestPipelineRun(t *testing.T) {
	tests := []struct {
		name                       string
		inputPipelineRunFile       string
		expectedExternalParameters map[string]any
	}{
		{
			name:                 "simple pipelinerun",
			inputPipelineRunFile: "../../testdata/slsa-v2alpha3/pipelinerun1.json",
			expectedExternalParameters: map[string]any{
				"runSpec": v1.PipelineRunSpec{
					PipelineRef: &v1.PipelineRef{Name: "test-pipeline"},
					Params: v1.Params{
						{
							Name:  "IMAGE",
							Value: v1.ParamValue{Type: "string", StringVal: "test.io/test/image"},
						},
					},
					TaskRunTemplate: v1.PipelineTaskRunTemplate{
						ServiceAccountName: "pipeline",
					},
				},
			},
		},
		{
			name:                 "pipelinerun with remote resolver",
			inputPipelineRunFile: "../../testdata/slsa-v2alpha3/pipelinerun-remote-resolver.json",
			expectedExternalParameters: map[string]any{
				"runSpec": v1.PipelineRunSpec{
					PipelineRef: &v1.PipelineRef{
						ResolverRef: v1.ResolverRef{
							Resolver: "git",
							Params: v1.Params{
								{Name: "url", Value: v1.ParamValue{Type: "string", StringVal: "https://github.com/tektoncd/catalog"}},
								{Name: "revision", Value: v1.ParamValue{Type: "string", StringVal: "main"}},
								{Name: "pathInRepo", Value: v1.ParamValue{Type: "string", StringVal: "pipeline/build-push-gke-deploy/0.1/build-push-gke-deploy.yaml"}},
							},
						},
					},
					Params: v1.Params{
						{Name: "pathToContext", Value: v1.ParamValue{Type: "string", StringVal: "gke-deploy/example/app"}},
						{Name: "pathToKubernetesConfigs", Value: v1.ParamValue{Type: "string", StringVal: "gke-deploy/example/app/config"}},
					},
					TaskRunTemplate: v1.PipelineTaskRunTemplate{
						ServiceAccountName: "default",
					},
				},
				"buildConfigSource": map[string]string{
					"path":       "pipeline/build-push-gke-deploy/0.1/build-push-gke-deploy.yaml",
					"ref":        "sha1:4df486f198c3c2616ab129186fb30a74f580b5a1",
					"repository": "git+https://github.com/tektoncd/catalog",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pro := createPro(test.inputPipelineRunFile)
			got := PipelineRun(pro)
			if diff := cmp.Diff(test.expectedExternalParameters, got); diff != "" {
				t.Errorf("PipelineRun(): -want +got: %s", diff)
			}
		})
	}
}

func TestTaskRun(t *testing.T) {
	tests := []struct {
		name                       string
		inputTaskRunFile           string
		expectedExternalParameters map[string]any
	}{
		{
			name:             "simple taskrun",
			inputTaskRunFile: "../../testdata/slsa-v2alpha3/taskrun1.json",
			expectedExternalParameters: map[string]any{
				"runSpec": v1.TaskRunSpec{
					Params: v1.Params{
						{Name: "IMAGE", Value: v1.ParamValue{Type: "string", StringVal: "test.io/test/image"}},
						{Name: "CHAINS-GIT_COMMIT", Value: v1.ParamValue{Type: "string", StringVal: "taskrun"}},
						{Name: "CHAINS-GIT_URL", Value: v1.ParamValue{Type: "string", StringVal: "https://git.test.com"}},
					},
					ServiceAccountName: "default",
					TaskRef:            &v1.TaskRef{Name: "build", Kind: "Task"},
				},
			},
		},
		{
			name:             "taskrun with remote resolver",
			inputTaskRunFile: "../../testdata/slsa-v2alpha3/taskrun-remote-resolver.json",
			expectedExternalParameters: map[string]any{
				"runSpec": v1.TaskRunSpec{
					Params: v1.Params{
						{Name: "ARGS", Value: v1.ParamValue{Type: "array", ArrayVal: []string{"help"}}},
					},
					ServiceAccountName: "default",
					TaskRef: &v1.TaskRef{
						Kind: "Task",
						ResolverRef: v1.ResolverRef{
							Resolver: "git",
							Params: []v1.Param{
								{Name: "url", Value: v1.ParamValue{Type: "string", StringVal: "https://github.com/tektoncd/catalog.git"}},
								{Name: "revision", Value: v1.ParamValue{Type: "string", StringVal: "main"}},
								{Name: "pathInRepo", Value: v1.ParamValue{Type: "string", StringVal: "task/gcloud/0.3/gcloud.yaml"}},
							},
						},
					},
				},
				"buildConfigSource": map[string]string{
					"ref":        "sha1:4df486f198c3c2616ab129186fb30a74f580b5a1",
					"repository": "git+https://github.com/tektoncd/catalog.git",
					"path":       "task/gcloud/0.3/gcloud.yaml",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tr, err := objectloader.TaskRunV1FromFile(test.inputTaskRunFile)
			if err != nil {
				t.Fatal(err)
			}

			got := TaskRun(objects.NewTaskRunObjectV1(tr))
			if diff := cmp.Diff(test.expectedExternalParameters, got); diff != "" {
				t.Errorf("TaskRun(): -want +got: %s", diff)
			}
		})
	}
}
