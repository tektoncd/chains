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

package chains

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestMarkSigned(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	// Create a TR for testing
	c := fakepipelineclient.Get(ctx)
	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-taskrun",
		},
		Spec: v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name: "foo",
			},
		},
	}
	if _, err := c.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	// Now mark it as signed.
	if err := MarkSigned(ctx, tr, c, nil); err != nil {
		t.Errorf("MarkSigned() error = %v", err)
	}

	// Now check the signature.
	signed, err := c.TektonV1beta1().TaskRuns(tr.Namespace).Get(ctx, tr.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if _, ok := signed.Annotations[ChainsAnnotation]; !ok {
		t.Error("Taskrun not signed.")
	}

	// Try some extra annotations

	// Now mark it as signed.
	extra := map[string]string{
		"foo": "bar",
	}
	if err := MarkSigned(ctx, tr, c, extra); err != nil {
		t.Errorf("MarkSigned() error = %v", err)
	}

	// Now check the signature.
	signed, err = c.TektonV1beta1().TaskRuns(tr.Namespace).Get(ctx, tr.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if _, ok := signed.Annotations[ChainsAnnotation]; !ok {
		t.Error("Taskrun not signed.")
	}
	if signed.Annotations["foo"] != "bar" {
		t.Error("Extra annotations not applied")
	}
}

func TestMarkFailed(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	// Create a TR for testing
	c := fakepipelineclient.Get(ctx)
	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-taskrun",
			Annotations: map[string]string{RetryAnnotation: "3"},
		},
		Spec: v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name: "foo",
			},
		},
	}
	if _, err := c.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	// Test HandleRetry, should mark it as failed
	if err := HandleRetry(ctx, tr, c, nil); err != nil {
		t.Errorf("HandleRetry() error = %v", err)
	}

	failed, err := c.TektonV1beta1().TaskRuns(tr.Namespace).Get(ctx, tr.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}

	if failed.Annotations[ChainsAnnotation] != "failed" {
		t.Errorf("Taskrun not marked as 'failed', was: '%s'", failed.Annotations[ChainsAnnotation])
	}
}

func TestTaskRunSigner_SignTaskRun(t *testing.T) {
	// SignTaskRun does three main things:
	// - generates payloads
	// - stores them in the configured systems
	// - marks the taskrun as signed
	tests := []struct {
		name              string
		backends          []*mockBackend
		wantErr           bool
		configuredBackend string
	}{
		{
			name: "single system",
			backends: []*mockBackend{
				{backendType: "mock"},
			},
			configuredBackend: "mock",
		},
		{
			name: "multiple systems",
			backends: []*mockBackend{
				{backendType: "mock"},
				{backendType: "foo"},
			},
			configuredBackend: "mock",
		},
		{
			name: "multiple systems, error",
			backends: []*mockBackend{
				{backendType: "mock", shouldErr: true},
				{backendType: "foo"},
			},
			configuredBackend: "mock",
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupMocks(tt.backends, &mockRekor{})
			defer cleanup()

			ctx, _ := rtesting.SetupFakeContext(t)
			ps := fakepipelineclient.Get(ctx)

			ctx = config.ToContext(ctx, &config.Config{
				Artifacts: config.ArtifactConfigs{
					TaskRuns: config.Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString(tt.configuredBackend),
						Signer:         "x509",
					},
				},
			})

			ts := &TaskRunSigner{
				Pipelineclientset: ps,
				SecretPath:        "./signing/x509/testdata/",
			}

			tr := &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			}
			if _, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr, metav1.CreateOptions{}); err != nil {
				t.Errorf("error creating fake taskrun: %v", err)
			}
			if err := ts.SignTaskRun(ctx, tr); (err != nil) != tt.wantErr {
				t.Errorf("TaskRunSigner.SignTaskRun() error = %v", err)
			}

			// Fetch a new TR!
			tr, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Get(ctx, tr.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("error fetching fake taskrun: %v", err)
			}
			// Check it is marked as signed
			shouldBeSigned := !tt.wantErr
			if Reconciled(tr) != shouldBeSigned {
				t.Errorf("IsSigned()=%t, wanted %t", Reconciled(tr), shouldBeSigned)
			}
			// Check the payloads were stored in all the backends.
			for _, b := range tt.backends {
				if b.shouldErr {
					continue
				}
				if b.backendType != tt.configuredBackend {
					continue
				}
				// We don't actually need to check the signature and serialized formats here, just that
				// the payload was stored.
				if b.storedPayload == nil {
					t.Error("error, expected payload to be stored.")
				}
			}

		})
	}
}

func TestTaskRunSigner_Transparency(t *testing.T) {
	for _, format := range []string{"in-toto", "tekton"} {
		rekor := &mockRekor{}
		backends := []*mockBackend{{backendType: "mock"}}
		cleanup := setupMocks(backends, rekor)
		defer cleanup()

		ctx, _ := rtesting.SetupFakeContext(t)
		ps := fakepipelineclient.Get(ctx)

		cfg := &config.Config{
			Artifacts: config.ArtifactConfigs{
				TaskRuns: config.Artifact{
					Format:         format,
					StorageBackend: sets.NewString("mock"),
					Signer:         "x509",
				},
			},
			Transparency: config.TransparencyConfig{
				Enabled: false,
			},
		}
		ctx = config.ToContext(ctx, cfg.DeepCopy())

		ts := &TaskRunSigner{
			Pipelineclientset: ps,
			SecretPath:        "./signing/x509/testdata/",
		}

		tr := &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
			},
		}
		if _, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr, metav1.CreateOptions{}); err != nil {
			t.Errorf("error creating fake taskrun: %v", err)
		}
		if err := ts.SignTaskRun(ctx, tr); err != nil {
			t.Errorf("TaskRunSigner.SignTaskRun() error = %v", err)
		}

		if len(rekor.entries) != 0 {
			t.Error("expected no transparency log entries!")
		}

		// Now enable and try again!
		cfg.Transparency.Enabled = true
		ctx = config.ToContext(ctx, cfg.DeepCopy())

		tr2 := &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: "foobar",
			},
		}
		if _, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr2, metav1.CreateOptions{}); err != nil {
			t.Errorf("error creating fake taskrun: %v", err)
		}
		if err := ts.SignTaskRun(ctx, tr2); err != nil {
			t.Errorf("TaskRunSigner.SignTaskRun() error = %v", err)
		}

		if len(rekor.entries) != 1 {
			t.Error("expected transparency log entry!")
		}

		// Now enable verifying the annotation
		cfg.Transparency.VerifyAnnotation = true
		ctx = config.ToContext(ctx, cfg.DeepCopy())

		tr3 := &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mytaskrun",
			},
		}

		if _, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr3, metav1.CreateOptions{}); err != nil {
			t.Errorf("error creating fake taskrun: %v", err)
		}
		if err := ts.SignTaskRun(ctx, tr3); err != nil {
			t.Errorf("TaskRunSigner.SignTaskRun() error = %v", err)
		}

		if len(rekor.entries) != 1 {
			t.Error("expected no new transparency log entries!")
		}
		// add in the annotation
		tr3.Annotations = map[string]string{RekorAnnotation: "true"}
		if err := ts.SignTaskRun(ctx, tr3); err != nil {
			t.Errorf("TaskRunSigner.SignTaskRun() error = %v", err)
		}

		if len(rekor.entries) != 2 {
			t.Error("expected two transparency log entries!")
		}
	}
}

func setupMocks(backends []*mockBackend, rekor *mockRekor) func() {
	oldGet := getBackends
	getBackends = func(ctx context.Context, ps versioned.Interface, _ kubernetes.Interface, logger *zap.SugaredLogger, _ *v1beta1.TaskRun, _ config.Config) (map[string]storage.Backend, error) {
		newBackends := map[string]storage.Backend{}
		for _, m := range backends {
			newBackends[m.backendType] = m
		}
		return newBackends, nil
	}

	oldRekor := getRekor
	getRekor = func(_ string, _ *zap.SugaredLogger) (rekorClient, error) {
		return rekor, nil
	}

	return func() {
		getRekor = oldRekor
		getBackends = oldGet
	}
}

type mockRekor struct {
	entries [][]byte
}

func (r *mockRekor) UploadTlog(ctx context.Context, signer signing.Signer, signature, rawPayload []byte, cert, payloadFormat string) (*models.LogEntryAnon, error) {
	r.entries = append(r.entries, signature)
	index := int64(len(r.entries) - 1)
	return &models.LogEntryAnon{
		LogIndex: &index,
	}, nil
}

type mockBackend struct {
	storedPayload []byte
	shouldErr     bool
	backendType   string
}

// StorePayload implements the Payloader interface.
func (b *mockBackend) StorePayload(ctx context.Context, signed []byte, signature string, opts config.StorageOpts) error {
	if b.shouldErr {
		return errors.New("mock error storing")
	}
	b.storedPayload = signed
	return nil
}

func (b *mockBackend) Type() string {
	return b.backendType
}

func (b *mockBackend) RetrievePayloads(ctx context.Context, opts config.StorageOpts) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *mockBackend) RetrieveSignatures(ctx context.Context, opts config.StorageOpts) (map[string][]string, error) {
	return nil, fmt.Errorf("not implemented")
}
