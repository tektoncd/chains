/*
Copyright 2026 The Tekton Authors

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

package provenance

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func TestGetSLSA1StatementBuilder(t *testing.T) {
	const (
		testChainsVersion    = "v0.26.0"
		testPipelinesVersion = "v1.18.0"
	)
	originalVersion := chainsVersion
	t.Cleanup(func() { chainsVersion = originalVersion })

	tests := []struct {
		name                   string
		chainsVersion          string
		pipelinesVersion       string
		includeBuilderMetadata bool
	}{
		{name: "versions unavailable", includeBuilderMetadata: true},
		{name: "Chains version", chainsVersion: testChainsVersion, includeBuilderMetadata: true},
		{name: "Pipelines version", pipelinesVersion: testPipelinesVersion, includeBuilderMetadata: true},
		{name: "both versions", chainsVersion: testChainsVersion, pipelinesVersion: testPipelinesVersion, includeBuilderMetadata: true},
		{name: "metadata disabled for older formats", chainsVersion: testChainsVersion, pipelinesVersion: testPipelinesVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainsVersion = tt.chainsVersion
			obj := objects.NewTaskRunObjectV1(&v1.TaskRun{})
			statement, err := GetSLSA1Statement(obj, nil, &slsa.BuildDefinition{}, nil, &slsaconfig.SlsaConfig{
				BuilderID:              "https://example.com/builder",
				PipelinesVersion:       tt.pipelinesVersion,
				IncludeBuilderMetadata: tt.includeBuilderMetadata,
			})
			if err != nil {
				t.Fatal(err)
			}

			got := statement.Predicate.AsMap()["runDetails"].(map[string]any)["builder"]
			want := map[string]any{"id": "https://example.com/builder"}
			if tt.includeBuilderMetadata && tt.chainsVersion != "" {
				want["version"] = map[string]any{"tekton-chains": tt.chainsVersion}
			}
			if tt.includeBuilderMetadata && tt.pipelinesVersion != "" {
				want["builderDependencies"] = []any{
					map[string]any{
						"name": "tekton-pipelines",
						"uri":  "https://github.com/tektoncd/pipeline/releases/tag/" + tt.pipelinesVersion,
					},
				}
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("builder (-want +got):\n%s", diff)
			}
		})
	}
}
