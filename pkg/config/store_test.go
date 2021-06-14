package config

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestNewConfigStore(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)

	ns := "my-namespace"
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "chains-config",
			Namespace: ns,
		},
	}
	fakekubeclient := fakek8s.NewSimpleClientset(cm)

	cs, err := NewConfigStore(fakekubeclient, ns, logtesting.TestLogger(t))
	if err != nil {
		t.Errorf("error creating new config store: %v", err)
	}

	// Test that there's nothing with an empty configmap
	if cs.Config().Artifacts.TaskRuns.Format != "" {
		t.Errorf("unexpected data: %v", cs.Config().Artifacts.TaskRuns.Format)
	}

	// Setup some config
	cm.Data = map[string]string{}
	cm.Data[taskrunSignerKey] = "pgp"

	if cm, err = fakekubeclient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		t.Errorf("error updating configmap: %v", err)
	}

	// It should be updated by then...
	time.Sleep(100 * time.Millisecond)
	// Test that the values are set!
	if diff := cmp.Diff("pgp", cs.Config().Artifacts.TaskRuns.Signer); diff != "" {
		t.Error(diff)
	}

	// Change it again
	cm.Data[taskrunSignerKey] = "kms"

	if _, err := fakekubeclient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
		t.Errorf("error updating configmap: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	// Test that the values are set!
	if diff := cmp.Diff("kms", cs.Config().Artifacts.TaskRuns.Signer); diff != "" {
		t.Error(diff)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string
		want Config
	}{
		{
			name: "empty",
			data: map[string]string{},
			want: Config{
				Builder: BuilderConfig{
					"tekton-chains",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						StorageBackend: "tekton",
						Signer:         "x509",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: "oci",
						Signer:         "x509",
					},
				},
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name: "single",
			data: map[string]string{taskrunSignerKey: "pgp"},
			want: Config{
				Builder: BuilderConfig{
					"tekton-chains",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						Signer:         "pgp",
						StorageBackend: "tekton",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: "oci",
						Signer:         "x509",
					},
				},
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
		{
			name: "extra",
			data: map[string]string{taskrunSignerKey: "pgp", "other-key": "foo"},
			want: Config{
				Builder: BuilderConfig{
					"tekton-chains",
				},
				Artifacts: ArtifactConfigs{
					TaskRuns: Artifact{
						Format:         "tekton",
						Signer:         "pgp",
						StorageBackend: "tekton",
					},
					OCI: Artifact{
						Format:         "simplesigning",
						StorageBackend: "oci",
						Signer:         "x509",
					},
				},
				Transparency: TransparencyConfig{
					URL: "https://rekor.sigstore.dev",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parse(tt.data, logtesting.TestLogger(t))
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("parse() = %v", diff)
			}
		})
	}
}

func TestValueOrDefault(t *testing.T) {
	tests := []struct {
		description string
		key         string
		value       string
		expected    string
	}{
		{
			description: "valid key set to default",
			key:         ociFormatKey,
			value:       "simplesigning",
			expected:    "simplesigning",
		}, {
			description: "valid key not set to default",
			key:         ociFormatKey,
			value:       "tekton",
			expected:    "tekton",
		}, {
			description: "invalid value with default",
			key:         ociFormatKey,
			value:       "invalid",
			expected:    "simplesigning",
		}, {
			description: "key with no default",
			key:         gcsBucketKey,
			value:       "bucket",
			expected:    "bucket",
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			got := valueOrDefault(test.key, map[string]string{test.key: test.value}, logtesting.TestLogger(t))
			if got != test.expected {
				t.Fatalf("got (%s) is not expected (%s)", got, test.expected)
			}
		})
	}
}
