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

package formats_test

import (
	"testing"

	"encoding/json"

	"github.com/google/go-cmp/cmp"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha3"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha4"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestFormatterPipelinesVersionDoesNotLeak(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		newFormatter           formats.PayloaderInit
		includeBuilderMetadata bool
	}{
		{name: "v2alpha3", newFormatter: v2alpha3.NewFormatter},
		{name: "v2alpha4", newFormatter: v2alpha4.NewFormatter, includeBuilderMetadata: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := tt.newFormatter(config.Config{Builder: config.BuilderConfig{ID: "https://example.com/builder"}})
			if err != nil {
				t.Fatal(err)
			}
			run, err := objectloader.TaskRunV1FromFile("slsa/testdata/slsa-v2alpha4/taskrun1.json")
			if err != nil {
				t.Fatal(err)
			}
			for _, version := range []string{"v1.18.0", "", "v1.19.0"} {
				ctx := logtesting.TestContextWithLogger(t)
				if version != "" {
					ctx = formats.WithPipelinesVersion(ctx, version)
				}
				payload, err := formatter.CreatePayload(ctx, objects.NewTaskRunObjectV1(run))
				if err != nil {
					t.Fatal(err)
				}
				payloadJSON, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("encoding payload: %v", err)
				}

				var statement intoto.Statement
				if err := json.Unmarshal(payloadJSON, &statement); err != nil {
					t.Fatalf("decoding payload: %v", err)
				}
				builder := statement.Predicate.AsMap()["runDetails"].(map[string]any)["builder"].(map[string]any)
				if !tt.includeBuilderMetadata {
					if _, ok := builder["version"]; ok {
						t.Error("older formatter unexpectedly includes builder.version")
					}
				}
				var want any
				if tt.includeBuilderMetadata && version != "" {
					want = []any{
						map[string]any{
							"name": "tekton-pipelines",
							"uri":  "https://github.com/tektoncd/pipeline/releases/tag/" + version,
						},
					}
				}
				if diff := cmp.Diff(want, builder["builderDependencies"]); diff != "" {
					t.Errorf("version %q dependencies (-want +got):\n%s", version, diff)
				}
			}
		})
	}
}
