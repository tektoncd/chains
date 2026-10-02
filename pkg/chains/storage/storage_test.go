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

package storage

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tektoncd/chains/pkg/config"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/util/sets"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestInitializeBackends(t *testing.T) {

	tests := []struct {
		name string
		cfg  config.Config
		want []string
	}{
		{
			name: "none",
			want: []string{},
		},
		{
			name: "tekton",
			want: []string{"tekton"},
			cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.New[string]("tekton")}}},
		},
		// TODO: Re-enable this test when it doesn't rely on ambient GCP credentials.
		//{
		//	name: "gcs",
		//	want: []string{"gcs"},
		//	cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.New[string]("gcs")}}},
		//},
		{
			name: "oci",
			want: []string{"oci"},
			cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.New[string]("oci")}}},
		},
		// TODO: Re-enable this test when it doesn't rely on ambient GCP credentials.
		// {
		// 	name: "grafeas",
		// 	want: []string{"grafeas"},
		// 	cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.New[string]("grafeas")}}},
		// },
		{
			name: "multi",
			want: []string{"oci", "tekton"},
			cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.New[string]("oci", "tekton")}}},
		},
		{
			name: "pubsub",
			want: []string{"pubsub"},
			cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.New[string]("pubsub")}}}},
	}
	ctx, _ := rtesting.SetupFakeContext(t)
	ps := fakepipelineclient.Get(ctx)
	kc := fakekubeclient.Get(ctx)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.WithLogger(ctx, logtesting.TestLogger(t))
			got, err := InitializeBackends(ctx, ps, kc, tt.cfg)
			if err != nil {
				t.Errorf("InitializeBackends() error = %v", err)
				return
			}
			t.Logf("Backend: %v", got)
			gotTypes := []string{}
			for _, g := range got {
				gotTypes = append(gotTypes, g.Type())
			}
			sort.Strings(gotTypes)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(gotTypes, tt.want) {
				t.Errorf("InitializeBackends() = %v, want %v", gotTypes, tt.want)
			}
		})
	}
}

func TestInitializeBackendsWarnsForResultDerivedOCIRegistry(t *testing.T) {
	defaultCfg, err := config.NewConfigFromMap(nil)
	if err != nil {
		t.Fatalf("NewConfigFromMap() error = %v", err)
	}

	tests := []struct {
		name         string
		cfg          config.Config
		wantWarnings int
	}{
		{
			name:         "default configuration",
			cfg:          *defaultCfg,
			wantWarnings: 1,
		},
		{
			name: "taskrun oci storage",
			cfg: config.Config{Artifacts: config.ArtifactConfigs{
				TaskRuns: config.Artifact{StorageBackend: sets.New[string]("oci")},
			}},
			wantWarnings: 1,
		},
		{
			name: "pipelinerun oci storage",
			cfg: config.Config{Artifacts: config.ArtifactConfigs{
				PipelineRuns: config.Artifact{StorageBackend: sets.New[string]("oci")},
			}},
			wantWarnings: 1,
		},
		{
			name: "oci artifact storage",
			cfg: config.Config{Artifacts: config.ArtifactConfigs{
				OCI: config.Artifact{StorageBackend: sets.New[string]("oci")},
			}},
			wantWarnings: 1,
		},
		{
			name: "multiple artifact types use oci storage",
			cfg: config.Config{Artifacts: config.ArtifactConfigs{
				TaskRuns:     config.Artifact{StorageBackend: sets.New[string]("oci")},
				PipelineRuns: config.Artifact{StorageBackend: sets.New[string]("oci")},
				OCI:          config.Artifact{StorageBackend: sets.New[string]("oci")},
			}},
			wantWarnings: 1,
		},
		{
			name: "oci storage disabled by signer",
			cfg: config.Config{Artifacts: config.ArtifactConfigs{
				OCI: config.Artifact{
					StorageBackend: sets.New[string]("oci"),
					Signer:         "none",
				},
			}},
		},
		{
			name: "non-oci storage",
			cfg: config.Config{Artifacts: config.ArtifactConfigs{
				TaskRuns: config.Artifact{StorageBackend: sets.New[string]("tekton")},
			}},
		},
	}

	// Keep the expected messages independent of the production constants so
	// changes to the security guidance must be reviewed explicitly.
	const dsseWarning = "OCI storage with DSSE is using result-derived repositories for storage. Set storage.oci.repository to control the DSSE upload destination. Chains may still contact the result-derived source artifact registry; in multi-tenant deployments, use network egress controls to restrict controller outbound access."
	const bundleWarning = "OCI storage with sigstore-bundle uses the result-derived artifact repository; storage.oci.repository does not override the destination in this mode. In multi-tenant deployments, use network egress controls to restrict controller outbound access."

	ctx, _ := rtesting.SetupFakeContext(t)
	ps := fakepipelineclient.Get(ctx)
	kc := fakekubeclient.Get(ctx)
	for _, tt := range tests {
		for _, encoding := range []string{"", config.OCIEncodingFormatDSSE, config.OCIEncodingFormatSigstoreBundle} {
			for _, repository := range []string{"", "registry.example.com/chains"} {
				t.Run(tt.name+"/encoding="+encoding+"/repository="+repository, func(t *testing.T) {
					cfg := tt.cfg
					cfg.Storage.OCI.EncodingFormat = encoding
					cfg.Storage.OCI.Repository = repository
					var logs bytes.Buffer
					core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(&logs), zap.WarnLevel)
					ctx := logging.WithLogger(ctx, zap.New(core).Sugar())

					if _, err := InitializeBackends(ctx, ps, kc, cfg); err != nil {
						t.Fatalf("InitializeBackends() error = %v", err)
					}

					wantWarnings := tt.wantWarnings
					wantMessage := dsseWarning
					if encoding == config.OCIEncodingFormatSigstoreBundle {
						wantMessage = bundleWarning
					} else if repository != "" {
						wantWarnings = 0
					}
					var entries []string
					if output := strings.TrimSpace(logs.String()); output != "" {
						entries = strings.Split(output, "\n")
					}
					if len(entries) != wantWarnings {
						t.Fatalf("warnings = %v, want %d", entries, wantWarnings)
					}
					for _, entry := range entries {
						var record struct {
							Level   string `json:"level"`
							Message string `json:"msg"`
						}
						if err := json.Unmarshal([]byte(entry), &record); err != nil {
							t.Fatalf("decode log entry: %v", err)
						}
						if record.Level != "warn" || record.Message != wantMessage {
							t.Errorf("log = %v, want WARN %q", entry, wantMessage)
						}
					}
				})
			}
		}
	}
}
