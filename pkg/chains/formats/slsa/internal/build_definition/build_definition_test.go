/*
Copyright 2024 The Tekton Authors
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

package builddefinition

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	intoto "github.com/in-toto/attestation/go/v1"
	externalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/external_parameters"
	internalparameters "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/internal_parameters"
	resolveddependencies "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/resolved_dependencies"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestGetTaskRunBuildDefinition(t *testing.T) {
	tr, err := objectloader.TaskRunFromFile("../../testdata/slsa-v2alpha4/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}

	tr.Annotations = map[string]string{
		"annotation1": "annotation1",
	}
	tr.Labels = map[string]string{
		"label1": "label1",
	}

	ctx := context.Background()

	tro := objects.NewTaskRunObjectV1(tr)
	tests := []struct {
		name      string
		buildType string
		want      slsa.BuildDefinition
		err       error
	}{
		{
			name:      "test slsa build type",
			buildType: "https://tekton.dev/chains/v2/slsa",
			want: slsa.BuildDefinition{
				BuildType:          "https://tekton.dev/chains/v2/slsa",
				ExternalParameters: getProtoStruct(t, externalparameters.TaskRun(tro)),
				InternalParameters: getProtoStruct(t, internalparameters.SLSAInternalParameters(tro)),
			},
			err: nil,
		},
		{
			name:      "test default build type",
			buildType: "",
			want: slsa.BuildDefinition{
				BuildType:          "https://tekton.dev/chains/v2/slsa",
				ExternalParameters: getProtoStruct(t, externalparameters.TaskRun(tro)),
				InternalParameters: getProtoStruct(t, internalparameters.SLSAInternalParameters(tro)),
			},
			err: nil,
		},
		{
			name:      "test tekton build type",
			buildType: "https://tekton.dev/chains/v2/slsa-tekton",
			want: slsa.BuildDefinition{
				BuildType:          "https://tekton.dev/chains/v2/slsa-tekton",
				ExternalParameters: getProtoStruct(t, externalparameters.TaskRun(tro)),
				InternalParameters: getProtoStruct(t, internalparameters.TektonInternalParameters(tro)),
			},
			err: nil,
		},
	}

	for i := range tests {
		tc := &tests[i]
		t.Run(tc.name, func(t *testing.T) {
			rd, err := resolveddependencies.TaskRun(ctx, resolveddependencies.ResolveOptions{}, tro)
			if err != nil {
				t.Fatalf("Error resolving dependencies: %v", err)
			}
			tc.want.ResolvedDependencies = rd

			bd, err := GetTaskRunBuildDefinition(context.Background(), tro, tc.buildType, resolveddependencies.ResolveOptions{})
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}
			bd.ProtoReflect()
			if diff := cmp.Diff(&tc.want, &bd, cmp.Options{protocmp.Transform()}); diff != "" {
				t.Errorf("getBuildDefinition(): -want +got: %v", diff)
			}
		})
	}
}

func TestTaskRunUnsupportedBuildType(t *testing.T) {
	tr, err := objectloader.TaskRunFromFile("../../testdata/slsa-v2alpha4/taskrun1.json")
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetTaskRunBuildDefinition(context.Background(), objects.NewTaskRunObjectV1(tr), "bad-buildType", resolveddependencies.ResolveOptions{})
	if err == nil {
		t.Error("getBuildDefinition(): expected error got nil")
	}
	if diff := cmp.Diff(&slsa.BuildDefinition{}, &got, protocmp.Transform()); diff != "" {
		t.Errorf("getBuildDefinition(): -want +got: %s", diff)
	}
}

func getProtoStruct(t *testing.T, data map[string]any) *structpb.Struct {
	t.Helper()
	protoStruct, err := getStruct(data)
	if err != nil {
		t.Fatalf("error getting proto struct from data: %v", err)
	}

	return protoStruct
}

func TestGetPipelineRunBuildDefinition(t *testing.T) {
	pr := createPro("../../testdata/slsa-v2alpha3/pipelinerun1.json")
	pr.Annotations = map[string]string{
		"annotation1": "annotation1",
	}
	pr.Labels = map[string]string{
		"label1": "label1",
	}
	tests := []struct {
		name   string
		config *slsaconfig.SlsaConfig
		want   slsa.BuildDefinition
	}{
		{
			name:   "test slsa build type",
			config: &slsaconfig.SlsaConfig{BuildType: "https://tekton.dev/chains/v2/slsa"},
			want: slsa.BuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa",
				ExternalParameters:   getProtoStruct(t, externalparameters.PipelineRun(pr)),
				InternalParameters:   getProtoStruct(t, internalparameters.SLSAInternalParameters(pr)),
				ResolvedDependencies: getResolvedDependencies(pr, resolveddependencies.AddSLSATaskDescriptor),
			},
		},
		{
			name:   "test tekton build type",
			config: &slsaconfig.SlsaConfig{BuildType: "https://tekton.dev/chains/v2/slsa-tekton"},
			want: slsa.BuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa-tekton",
				ExternalParameters:   getProtoStruct(t, externalparameters.PipelineRun(pr)),
				InternalParameters:   getProtoStruct(t, internalparameters.TektonInternalParameters(pr)),
				ResolvedDependencies: getResolvedDependencies(pr, resolveddependencies.AddTektonTaskDescriptor),
			},
		},
		{
			name:   "test default build type",
			config: &slsaconfig.SlsaConfig{BuildType: "https://tekton.dev/chains/v2/slsa"},
			want: slsa.BuildDefinition{
				BuildType:            "https://tekton.dev/chains/v2/slsa",
				ExternalParameters:   getProtoStruct(t, externalparameters.PipelineRun(pr)),
				InternalParameters:   getProtoStruct(t, internalparameters.SLSAInternalParameters(pr)),
				ResolvedDependencies: getResolvedDependencies(pr, resolveddependencies.AddSLSATaskDescriptor),
			},
		},
	}

	for i := range tests {
		tc := &tests[i]
		t.Run(tc.name, func(t *testing.T) {
			bd, err := GetPipelineRunBuildDefinition(context.TODO(), pr, tc.config, resolveddependencies.ResolveOptions{})
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}

			if diff := cmp.Diff(&tc.want, &bd, protocmp.Transform()); diff != "" {
				t.Errorf("getBuildDefinition(): -want +got: %v", diff)
			}
		})
	}
}

func createPro(path string) *objects.PipelineRunObjectV1 {
	pr, err := objectloader.PipelineRunFromFile(path)
	if err != nil {
		panic(err)
	}
	tr1, err := objectloader.TaskRunFromFile("../../testdata/slsa-v2alpha3/taskrun1.json")
	if err != nil {
		panic(err)
	}
	tr2, err := objectloader.TaskRunFromFile("../../testdata/slsa-v2alpha3/taskrun2.json")
	if err != nil {
		panic(err)
	}
	p := objects.NewPipelineRunObjectV1(pr)
	p.AppendTaskRun(tr1)
	p.AppendTaskRun(tr2)
	return p
}

func getResolvedDependencies(pr *objects.PipelineRunObjectV1, addTasks func(*objects.TaskRunObjectV1) (*intoto.ResourceDescriptor, error)) []*intoto.ResourceDescriptor {
	rd, err := resolveddependencies.PipelineRun(context.Background(), pr, &slsaconfig.SlsaConfig{}, resolveddependencies.ResolveOptions{}, addTasks)
	if err != nil {
		return []*intoto.ResourceDescriptor{}
	}
	return rd
}

func TestPipelineRunUnsupportedBuildType(t *testing.T) {
	pr := createPro("../../testdata/slsa-v2alpha3/pipelinerun1.json")

	got, err := GetPipelineRunBuildDefinition(context.Background(), pr, &slsaconfig.SlsaConfig{BuildType: "bad-buildtype"}, resolveddependencies.ResolveOptions{})
	if err == nil {
		t.Error("getBuildDefinition(): expected error got nil")
	}
	if diff := cmp.Diff(&slsa.BuildDefinition{}, &got, protocmp.Transform()); diff != "" {
		t.Errorf("getBuildDefinition(): -want +got: %s", diff)
	}
}
