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

package internalparameters

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	buildtypes "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/build_types"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"github.com/tektoncd/pipeline/pkg/apis/config"
)

func TestGetInternalParamters(t *testing.T) {
	tests := []struct {
		name                string
		shouldErr           bool
		buildDefinitionType string
		expected            map[string]any
	}{
		{
			name:                "SLSA build type",
			buildDefinitionType: buildtypes.SlsaBuildType,
			expected: map[string]any{
				"tekton-pipelines-feature-flags": config.FeatureFlags{EnableAPIFields: "beta", ResultExtractionMethod: "termination-message"},
			},
		},
		{
			name:                "Tekton build type",
			buildDefinitionType: buildtypes.TektonBuildType,
			expected: map[string]any{
				"labels":                         map[string]string{"tekton.dev/pipelineTask": "build"},
				"annotations":                    map[string]string(nil),
				"tekton-pipelines-feature-flags": config.FeatureFlags{EnableAPIFields: "beta", ResultExtractionMethod: "termination-message"},
			},
		},
		{
			name:                "Invalid build type",
			buildDefinitionType: "invalid-type",
			shouldErr:           true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tr, err := objectloader.TaskRunV1FromFile("../../testdata/slsa-v2alpha2/taskrun1.json")
			if err != nil {
				t.Fatal(err)
			}
			tro := objects.NewTaskRunObjectV1(tr)

			got, err := GetInternalParamters(tro, test.buildDefinitionType)

			didError := err != nil
			if didError != test.shouldErr {
				t.Fatalf("Unexpected error behavior, shouldErr: %v, didError: %v, error: %v", test.shouldErr, didError, err)
			}

			if diff := cmp.Diff(test.expected, got); diff != "" {
				t.Errorf("TaskRun(): -want +got: %s", diff)
			}
		})
	}
}
