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

package config

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/configmap/informer"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
	"knative.dev/pkg/system"
)

func TestNewConfigStore(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)

	ns := system.Namespace()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "chains-config",
			Namespace: ns,
		},
	}
	fakekubeclient := fakek8s.NewSimpleClientset(cm)
	cmw := informer.NewInformedWatcher(fakekubeclient, system.Namespace())

	cs := NewConfigStore(logtesting.TestLogger(t))
	cs.WatchConfigs(cmw)
	if err := cmw.Start(ctx.Done()); err != nil {
		t.Fatalf("Error starting configmap.Watcher %v", err)
	}

	// Check that with an empty configmap we get the default values.
	if diff := cmp.Diff(cs.Load(), defaultConfig()); diff != "" {
		t.Errorf("unexpected data: %v", diff)
	}

	// Setup some config
	cm.Data = map[string]string{}
	cm.Data[taskrunSignerKey] = "x509"

	var err error
	if cm, err = fakekubeclient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		t.Errorf("error updating configmap: %v", err)
	}

	// It should be updated by then...
	time.Sleep(100 * time.Millisecond)
	// Test that the values are set!
	if diff := cmp.Diff("x509", cs.Load().Artifacts.TaskRuns.Signer); diff != "" {
		t.Error(diff)
	}

	// Change it again
	cm.Data[taskrunSignerKey] = "kms"

	if _, err := fakekubeclient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		t.Errorf("error updating configmap: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	// Test that the values are set!
	if diff := cmp.Diff("kms", cs.Load().Artifacts.TaskRuns.Signer); diff != "" {
		t.Error(diff)
	}
}

var defaultSigners = SignerConfigs{
	X509: X509Signer{
		FulcioAddr:       "https://fulcio.sigstore.dev",
		FulcioOIDCIssuer: "https://oauth2.sigstore.dev/auth",
	},
}

func TestParse(t *testing.T) {
	tests := []struct {
		name           string
		data           map[string]string
		taskrunEnabled bool
		ociEnbaled     bool
		want           Config
	}{
		{
			name:           "empty",
			data:           map[string]string{},
			taskrunEnabled: true,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("tekton"),
						Signer:         "x509",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci"),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name:           "taskrun multi backend",
			data:           map[string]string{taskrunStorageKey: "tekton,oci"},
			taskrunEnabled: true,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("tekton", "oci"),
						Signer:         "x509",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci"),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name:           "taskrun multi backend disabled",
			data:           map[string]string{taskrunStorageKey: ""},
			taskrunEnabled: false,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString(""),
						Signer:         "x509",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci"),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name:           "oci multi backend",
			data:           map[string]string{ociStorageKey: "oci,tekton"},
			taskrunEnabled: true,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("tekton"),
						Signer:         "x509",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci", "tekton"),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name:           "oci multi backend disabled",
			data:           map[string]string{ociStorageKey: ""},
			taskrunEnabled: true,
			ociEnbaled:     false,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("tekton"),
						Signer:         "x509",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString(""),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name:           "taskrun multi backend, oci disabled",
			data:           map[string]string{taskrunStorageKey: "tekton,oci", ociStorageKey: ""},
			taskrunEnabled: true,
			ociEnbaled:     false,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("tekton", "oci"),
						Signer:         "x509",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString(""),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name:           "oci multi backend, tekton disabled",
			data:           map[string]string{ociStorageKey: "oci,tekton", taskrunStorageKey: ""},
			taskrunEnabled: false,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString(""),
						Signer:         "x509",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci", "tekton"),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name:           "single",
			data:           map[string]string{taskrunSignerKey: "x509"},
			taskrunEnabled: true,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						Signer:         "x509",
						StorageBackend: sets.NewString("tekton"),
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci"),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name:           "manual transparency",
			data:           map[string]string{transparencyEnabledKey: "manual"},
			taskrunEnabled: true,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						Signer:         "x509",
						StorageBackend: sets.NewString("tekton"),
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci"),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					Enabled:          true,
					VerifyAnnotation: true,
					URL:              "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name:           "extra",
			data:           map[string]string{taskrunSignerKey: "x509", "other-key": "foo"},
			taskrunEnabled: true,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						Signer:         "x509",
						StorageBackend: sets.NewString("tekton"),
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci"),
						Signer:         "x509",
					},
				},
				Signers: defaultSigners,
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		}, {
			name: "fulcio",
			data: map[string]string{
				taskrunSignerKey:              "x509",
				"signers.x509.fulcio.enabled": "true",
				"signers.x509.fulcio.address": "fulcio-address",
			},
			taskrunEnabled: true,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						Signer:         "x509",
						StorageBackend: sets.NewString("tekton"),
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci"),
						Signer:         "x509",
					},
				},
				Signers: SignerConfigs{
					X509: X509Signer{
						FulcioEnabled:    true,
						FulcioAddr:       "fulcio-address",
						FulcioOIDCIssuer: "https://oauth2.sigstore.dev/auth",
					},
				},
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		}, {
			name: "rekor - true",
			data: map[string]string{
				"transparency.enabled": "true",
			},
			taskrunEnabled: true,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						Signer:         "x509",
						StorageBackend: sets.NewString("tekton"),
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci"),
						Signer:         "x509",
					},
				},
				Signers: SignerConfigs{
					X509: X509Signer{
						FulcioAddr:       "https://fulcio.sigstore.dev",
						FulcioOIDCIssuer: "https://oauth2.sigstore.dev/auth",
					},
				},
				Transparency: TransparencyConfig{
					Enabled: true,
					URL:     "https://rekor.sigstore.dev",
				},
			},
		}, {
			name: "rekor - manual",
			data: map[string]string{
				"transparency.enabled": "manual",
			},
			taskrunEnabled: true,
			ociEnbaled:     true,
			want: Config{
				Builder: BuilderConfig{
					"https://tekton.dev/chains/v2",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						Signer:         "x509",
						StorageBackend: sets.NewString("tekton"),
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.NewString("oci"),
						Signer:         "x509",
					},
				},
				Signers: SignerConfigs{
					X509: X509Signer{
						FulcioAddr:       "https://fulcio.sigstore.dev",
						FulcioOIDCIssuer: "https://oauth2.sigstore.dev/auth",
					},
				},
				Transparency: TransparencyConfig{
					Enabled:          true,
					VerifyAnnotation: true,
					URL:              "https://rekor.sigstore.dev",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfigFromMap(tt.data)
			if err != nil {
				t.Fatalf("NewConfigFromMap() = %v", err)
			}
			if got.Artifacts.OCI.Enabled() != tt.ociEnbaled {
				t.Errorf("OCI artifact enable mismatch")
			}
			if got.Artifacts.TaskRuns.Enabled() != tt.taskrunEnabled {
				t.Errorf("Taskrun artifact enable mismatch")
			}
			if diff := cmp.Diff(*got, tt.want); diff != "" {
				t.Errorf("parse() = %v", diff)
			}
		})
	}
}
