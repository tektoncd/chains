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

func TestBuildConfigSource(t *testing.T) {
	digest := map[string]string{"alg1": "hex1", "alg2": "hex2"}
	provenance := &objects.ProvenanceV1{&v1.Provenance{
		RefSource: &v1.RefSource{
			Digest:     digest,
			URI:        "https://tekton.com",
			EntryPoint: "/path/to/entry",
		},
	}}

	want := map[string]string{
		"repository": "https://tekton.com",
		"path":       "/path/to/entry",
	}

	got := buildConfigSource(provenance)

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

func createPro(path string) *objects.PipelineRunObjectV1 {
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
	p := objects.NewPipelineRunObjectV1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func TestPipelineRun(t *testing.T) {
	pro := createPro("../../../testdata/v2alpha2/pipelinerun1.json")

	got := PipelineRun(pro)

	want := map[string]any{
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
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("PipelineRun(): -want +got: %s", diff)
	}
}

func TestTaskRun(t *testing.T) {
	tr, err := objectloader.TaskRunFromFile("../../../testdata/v2alpha2/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}
	got := TaskRun(objects.NewTaskRunObjectV1(tr))

	want := map[string]any{
		"runSpec": v1.TaskRunSpec{
			Params: v1.Params{
				{Name: "IMAGE", Value: v1.ParamValue{Type: "string", StringVal: "test.io/test/image"}},
				{Name: "CHAINS-GIT_COMMIT", Value: v1.ParamValue{Type: "string", StringVal: "taskrun"}},
				{Name: "CHAINS-GIT_URL", Value: v1.ParamValue{Type: "string", StringVal: "https://git.test.com"}},
			},
			ServiceAccountName: "default",
			TaskRef:            &v1.TaskRef{Name: "build", Kind: "Task"},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("TaskRun(): -want +got: %s", diff)
	}
}
