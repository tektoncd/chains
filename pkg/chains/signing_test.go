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
	"reflect"
	"testing"

	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/test/tekton"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/logging"
	rtesting "knative.dev/pkg/reconciler/testing"

	_ "github.com/tektoncd/chains/pkg/chains/formats/all"
)

func TestSigner_Sign(t *testing.T) {
	// Sign does three main things:
	// - generates payloads
	// - stores them in the configured systems
	// - marks the object as signed
	tro := objects.NewTaskRunObject(&v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	})

	pro := objects.NewPipelineRunObject(&v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	})

	tcfg := &config.Config{
		Artifacts: config.ArtifactConfigs{
			TaskRuns: config.Artifact{
				Format:         "tekton",
				StorageBackend: sets.NewString("mock"),
				Signer:         "x509",
			},
		},
	}

	pcfg := &config.Config{
		Artifacts: config.ArtifactConfigs{
			PipelineRuns: config.Artifact{
				Format:         "tekton",
				StorageBackend: sets.NewString("mock"),
				Signer:         "x509",
			},
		},
	}

	tests := []struct {
		name     string
		backends []*mockBackend
		wantErr  bool
		object   objects.TektonObject
		config   *config.Config
	}{
		{
			name: "taskrun single system",
			backends: []*mockBackend{
				{backendType: "mock"},
			},
			object: tro,
			config: tcfg,
		},
		{
			name: "taskrun multiple systems",
			backends: []*mockBackend{
				{backendType: "mock"},
				{backendType: "foo"},
			},
			object: tro,
			config: tcfg,
		},
		{
			name: "taskrun multiple systems, error",
			backends: []*mockBackend{
				{backendType: "mock", shouldErr: true},
				{backendType: "foo"},
			},
			wantErr: true,
			object:  tro,
			config:  tcfg,
		},
		{
			name: "pipelinerun single system",
			backends: []*mockBackend{
				{backendType: "mock"},
			},
			object: pro,
			config: pcfg,
		},
		{
			name: "pipelinerun multiple systems",
			backends: []*mockBackend{
				{backendType: "mock"},
				{backendType: "foo"},
			},
			object: pro,
			config: pcfg,
		},
		{
			name: "pipelinerun multiple systems, error",
			backends: []*mockBackend{
				{backendType: "mock", shouldErr: true},
				{backendType: "foo"},
			},
			wantErr: true,
			object:  pro,
			config:  pcfg,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx, _ := rtesting.SetupFakeContext(t)
			ps := fakepipelineclient.Get(ctx)

			ctx = config.ToContext(ctx, tt.config.DeepCopy())

			ts := &ObjectSigner{
				Backends:          fakeAllBackends(tt.backends),
				SecretPath:        "./signing/x509/testdata/",
				Pipelineclientset: ps,
			}

			tekton.CreateObject(t, ctx, ps, tt.object)

			if err := ts.Sign(ctx, tt.object); (err != nil) != tt.wantErr {
				t.Errorf("Signer.Sign() error = %v", err)
			}

			// Fetch the updated object
			updatedObject, err := tekton.GetObject(t, ctx, ps, tt.object)
			if err != nil {
				t.Errorf("error fetching fake object: %v", err)
			}

			// Check it is marked as signed
			shouldBeSigned := !tt.wantErr
			if Reconciled(updatedObject) != shouldBeSigned {
				t.Errorf("IsSigned()=%t, wanted %t", Reconciled(updatedObject), shouldBeSigned)
			}
			// Check the payloads were stored in all the backends.
			for _, b := range tt.backends {
				if b.shouldErr {
					continue
				}
				if b.backendType != "mock" {
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

func TestSigner_Transparency(t *testing.T) {
	newTaskRun := func(name string) objects.TektonObject {
		return objects.NewTaskRunObject(&v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
	}
	newPipelineRun := func(name string) objects.TektonObject {
		return objects.NewPipelineRunObject(&v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
	}
	setAnnotation := func(obj objects.TektonObject, key, value string) {
		// TODO: opportunity to add code reuse
		switch o := obj.GetObject().(type) {
		case *v1beta1.PipelineRun:
			if o.Annotations == nil {
				o.Annotations = make(map[string]string)
			}
			o.Annotations[key] = value
		case *v1beta1.TaskRun:
			if o.Annotations == nil {
				o.Annotations = make(map[string]string)
			}
			o.Annotations[key] = value
		}
	}

	tests := []struct {
		name         string
		cfg          *config.Config
		getNewObject func(string) objects.TektonObject
	}{
		{
			name: "taskrun in-toto",
			cfg: &config.Config{
				Artifacts: config.ArtifactConfigs{
					TaskRuns: config.Artifact{
						Format:         "in-toto",
						StorageBackend: sets.NewString("mock"),
						Signer:         "x509",
					},
				},
				Transparency: config.TransparencyConfig{
					Enabled: false,
				},
			},
			getNewObject: newTaskRun,
		},
		{
			name: "taskrun tekton",
			cfg: &config.Config{
				Artifacts: config.ArtifactConfigs{
					TaskRuns: config.Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("mock"),
						Signer:         "x509",
					},
				},
				Transparency: config.TransparencyConfig{
					Enabled: false,
				},
			},
			getNewObject: newTaskRun,
		},
		{
			name: "pipelinerun in-toto",
			cfg: &config.Config{
				Artifacts: config.ArtifactConfigs{
					PipelineRuns: config.Artifact{
						Format:         "in-toto",
						StorageBackend: sets.NewString("mock"),
						Signer:         "x509",
					},
				},
				Transparency: config.TransparencyConfig{
					Enabled: false,
				},
			},
			getNewObject: newPipelineRun,
		},
		{
			name: "pipelinerun tekton",
			cfg: &config.Config{
				Artifacts: config.ArtifactConfigs{
					PipelineRuns: config.Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("mock"),
						Signer:         "x509",
					},
				},
				Transparency: config.TransparencyConfig{
					Enabled: false,
				},
			},
			getNewObject: newPipelineRun,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			rekor := &mockRekor{}
			backends := []*mockBackend{{backendType: "mock"}}
			cleanup := setupMocks(rekor)
			defer cleanup()

			ctx, _ := rtesting.SetupFakeContext(t)
			ps := fakepipelineclient.Get(ctx)

			ctx = config.ToContext(ctx, tt.cfg.DeepCopy())

			os := &ObjectSigner{
				Backends:          fakeAllBackends(backends),
				SecretPath:        "./signing/x509/testdata/",
				Pipelineclientset: ps,
			}

			obj := tt.getNewObject("foo")

			tekton.CreateObject(t, ctx, ps, obj)

			if err := os.Sign(ctx, obj); err != nil {
				t.Errorf("Signer.Sign() error = %v", err)
			}

			if len(rekor.entries) != 0 {
				t.Error("expected no transparency log entries!")
			}

			// Now enable and try again!
			tt.cfg.Transparency.Enabled = true
			ctx = config.ToContext(ctx, tt.cfg.DeepCopy())

			obj = tt.getNewObject("foobar")

			tekton.CreateObject(t, ctx, ps, obj)

			if err := os.Sign(ctx, obj); err != nil {
				t.Errorf("Signer.Sign() error = %v", err)
			}

			if len(rekor.entries) != 1 {
				t.Error("expected transparency log entry!")
			}

			// Now enable verifying the annotation
			tt.cfg.Transparency.VerifyAnnotation = true
			ctx = config.ToContext(ctx, tt.cfg.DeepCopy())

			obj = tt.getNewObject("mytektonobject")

			tekton.CreateObject(t, ctx, ps, obj)

			if err := os.Sign(ctx, obj); err != nil {
				t.Errorf("Signer.Sign() error = %v", err)
			}

			if len(rekor.entries) != 1 {
				t.Error("expected new transparency log entries!")
			}

			// add in the annotation
			setAnnotation(obj, RekorAnnotation, "true")
			if err := os.Sign(ctx, obj); err != nil {
				t.Errorf("Signer.Sign() error = %v", err)
			}

			if len(rekor.entries) != 2 {
				t.Error("expected two transparency log entries!")
			}
		})
	}
}

func TestSigningObjects(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	logger := logging.FromContext(ctx)
	tests := []struct {
		name       string
		signers    []string
		config     config.Config
		SecretPath string
	}{
		{
			name:    "x509",
			signers: []string{signing.TypeX509},
			config: config.Config{
				Artifacts: config.ArtifactConfigs{
					TaskRuns: config.Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("mock"),
						Signer:         "x509",
					},
				},
			},
			SecretPath: "./signing/x509/testdata/",
		},
		{
			name:    "x509 twice",
			signers: []string{signing.TypeX509},
			config: config.Config{
				Artifacts: config.ArtifactConfigs{
					TaskRuns: config.Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("mock"),
						Signer:         "x509",
					},
					OCI: config.Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("mock"),
						Signer:         "x509",
					},
				},
			},
			SecretPath: "./signing/x509/testdata/",
		},
		{
			name:    "none",
			signers: nil,
			config: config.Config{
				Artifacts: config.ArtifactConfigs{
					TaskRuns: config.Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("mock"),
					},
					OCI: config.Artifact{
						Format:         "tekton",
						StorageBackend: sets.NewString("mock"),
					},
				},
				Transparency: config.TransparencyConfig{
					Enabled: false,
				},
			},
			SecretPath: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signers := allSigners(ctx, tt.SecretPath, tt.config, logger)
			var signerTypes []string
			for _, signer := range signers {
				signerTypes = append(signerTypes, signer.Type())
			}
			if !reflect.DeepEqual(tt.signers, signerTypes) {
				t.Errorf("Expected %q signers but got %q signers", tt.signers, signerTypes)
			}
		})
	}
}

func fakeAllBackends(backends []*mockBackend) map[string]storage.Backend {
	newBackends := map[string]storage.Backend{}
	for _, m := range backends {
		newBackends[m.backendType] = m
	}
	return newBackends
}

func setupMocks(rekor *mockRekor) func() {
	oldRekor := getRekor
	getRekor = func(_ string, _ *zap.SugaredLogger) (rekorClient, error) {
		return rekor, nil
	}
	return func() {
		getRekor = oldRekor
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
func (b *mockBackend) StorePayload(ctx context.Context, _ objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	if b.shouldErr {
		return errors.New("mock error storing")
	}
	b.storedPayload = rawPayload
	return nil
}

func (b *mockBackend) Type() string {
	return b.backendType
}

func (b *mockBackend) RetrievePayloads(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *mockBackend) RetrieveSignatures(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	return nil, fmt.Errorf("not implemented")
}
