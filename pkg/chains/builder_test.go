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

// Package chains coordinates signing and storage of Tekton run attestations.
package chains

import (
	"encoding/json"
	"testing"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/objectloader"
	"github.com/tektoncd/chains/pkg/test/tekton"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	fakekubeclient "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestSignerPipelinesBuilderDependency(t *testing.T) {
	const (
		testSigner         = "x509"
		testDisabledSigner = "none"
		testBackend        = "mock"
		testSecretPath     = "./signing/x509/testdata/"
	)

	tests := []struct {
		name        string
		version     string
		configMap   bool
		forbidden   bool
		wantVersion string
	}{
		{name: "installed release", version: " v1.18.0\n", configMap: true, wantVersion: "v1.18.0"},
		{name: "missing ConfigMap"},
		{name: "empty version", configMap: true},
		{name: "forbidden ConfigMap", forbidden: true},
	}
	for _, format := range []string{"slsa/v2alpha3", "slsa/v2alpha4"} {
		for _, kind := range []string{"TaskRun", "PipelineRun"} {
			for _, tt := range tests {
				t.Run(format+"/"+kind+"/"+tt.name, func(t *testing.T) {
					ctx, _ := rtesting.SetupFakeContext(t)
					pipelineClient := fakepipelineclient.Get(ctx)
					kubeClient := fakekubeclient.NewSimpleClientset()
					const namespace = "tekton-pipelines"
					if tt.configMap {
						_, err := kubeClient.CoreV1().ConfigMaps(namespace).Create(ctx, &corev1.ConfigMap{
							ObjectMeta: metav1.ObjectMeta{Name: "pipelines-info", Namespace: namespace},
							Data:       map[string]string{"version": tt.version},
						}, metav1.CreateOptions{})
						if err != nil {
							t.Fatal(err)
						}
					}
					if tt.forbidden {
						kubeClient.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
							return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "pipelines-info", nil)
						})
					}
					kubeClient.ClearActions()

					var obj objects.TektonObject
					if kind == "TaskRun" {
						run, err := objectloader.TaskRunV1FromFile("formats/slsa/testdata/slsa-v2alpha4/taskrun1.json")
						if err != nil {
							t.Fatal(err)
						}
						obj = objects.NewTaskRunObjectV1(run)
					} else {
						run, err := objectloader.PipelineRunV1FromFile("formats/slsa/testdata/slsa-v2alpha4/pipelinerun1.json")
						if err != nil {
							t.Fatal(err)
						}
						obj = objects.NewPipelineRunObjectV1(run)
					}
					cfg := &config.Config{
						Builder: config.BuilderConfig{
							ID: "https://example.com/builder",
						},
						Artifacts: config.ArtifactConfigs{
							TaskRuns:     config.Artifact{Format: format, Signer: testSigner, StorageBackend: sets.New[string](testBackend)},
							PipelineRuns: config.Artifact{Format: format, Signer: testSigner, StorageBackend: sets.New[string](testBackend)},
							OCI:          config.Artifact{Signer: testDisabledSigner},
						},
					}
					ctx = config.ToContext(ctx, cfg)
					backend := &mockBackend{backendType: testBackend}
					signer := &ObjectSigner{
						Backends:          fakeAllBackends([]*mockBackend{backend}),
						SecretPath:        testSecretPath,
						Pipelineclientset: pipelineClient,
						KubeClient:        kubeClient,
					}
					tekton.CreateObject(t, ctx, pipelineClient, obj)
					if err := signer.Sign(ctx, obj); err != nil {
						t.Fatalf("Sign() = %v", err)
					}
					var statement struct {
						Predicate struct {
							RunDetails struct {
								Builder struct {
									ID           string `json:"id"`
									Dependencies []struct {
										Name string `json:"name"`
										URI  string `json:"uri"`
									} `json:"builderDependencies"`
								} `json:"builder"`
							} `json:"runDetails"`
						} `json:"predicate"`
					}
					if err := json.Unmarshal(backend.storedPayload, &statement); err != nil {
						t.Fatalf("decoding stored attestation: %v", err)
					}
					builder := statement.Predicate.RunDetails.Builder
					if builder.ID != cfg.Builder.ID {
						t.Errorf("builder.id = %q, want %q", builder.ID, cfg.Builder.ID)
					}
					wantVersion := ""
					wantLookups := 0
					if format == "slsa/v2alpha4" {
						wantVersion = tt.wantVersion
						wantLookups = 1
					}
					if wantVersion == "" {
						if len(builder.Dependencies) != 0 {
							t.Errorf("unexpected dependencies: %v", builder.Dependencies)
						}
					} else {
						if len(builder.Dependencies) != 1 {
							t.Fatalf("got %d dependencies, want 1", len(builder.Dependencies))
						}
						dep := builder.Dependencies[0]
						if dep.Name != "tekton-pipelines" || dep.URI != "https://github.com/tektoncd/pipeline/releases/tag/"+wantVersion {
							t.Errorf("unexpected Pipelines dependency: %+v", dep)
						}
					}
					actions := kubeClient.Actions()
					if len(actions) != wantLookups {
						t.Fatalf("got %d ConfigMap requests, want %d", len(actions), wantLookups)
					}
					if wantLookups == 1 && (actions[0].GetVerb() != "get" || actions[0].GetNamespace() != namespace) {
						t.Errorf("expected a ConfigMap get in %s, got %v", namespace, actions)
					}
				})
			}
		}
	}
}
