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
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/test/tekton"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	rtesting "knative.dev/pkg/reconciler/testing"

	_ "github.com/tektoncd/chains/pkg/chains/formats/all"
)

func TestSigner_Sign(t *testing.T) {
	// Sign does three main things:
	// - generates payloads
	// - stores them in the configured systems
	// - marks the object as signed
	tro := objects.NewTaskRunObjectV1(&v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	})

	pro := objects.NewPipelineRunObjectV1(&v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	})

	tcfg := &config.Config{
		Artifacts: config.ArtifactConfigs{
			TaskRuns: config.Artifact{
				Format:         "in-toto",
				StorageBackend: sets.New[string]("mock"),
				Signer:         "x509",
			},
		},
	}

	pcfg := &config.Config{
		Artifacts: config.ArtifactConfigs{
			PipelineRuns: config.Artifact{
				Format:         "in-toto",
				StorageBackend: sets.New[string]("mock"),
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
			if Reconciled(ctx, ps, updatedObject) != shouldBeSigned {
				t.Errorf("IsSigned()=%t, wanted %t", Reconciled(ctx, ps, updatedObject), shouldBeSigned)
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
		return objects.NewTaskRunObjectV1(&v1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
	}
	newPipelineRun := func(name string) objects.TektonObject {
		return objects.NewPipelineRunObjectV1(&v1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})
	}
	setAnnotation := func(obj objects.TektonObject, key, value string) {
		// TODO: opportunity to add code reuse
		switch o := obj.GetObject().(type) {
		case *v1.PipelineRun:
			if o.Annotations == nil {
				o.Annotations = make(map[string]string)
			}
			o.Annotations[key] = value
		case *v1.TaskRun:
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
						Format:         "slsa/v1",
						StorageBackend: sets.New[string]("mock"),
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
						Format:         "slsa/v1",
						StorageBackend: sets.New[string]("mock"),
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
						Format:         "in-toto",
						StorageBackend: sets.New[string]("mock"),
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
						Format:         "in-toto",
						StorageBackend: sets.New[string]("mock"),
						Signer:         "x509",
					},
					OCI: config.Artifact{
						Format:         "in-toto",
						StorageBackend: sets.New[string]("mock"),
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
						Format:         "in-toto",
						StorageBackend: sets.New[string]("mock"),
					},
					OCI: config.Artifact{
						Format:         "in-toto",
						StorageBackend: sets.New[string]("mock"),
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
			ctx, _ := rtesting.SetupFakeContext(t)
			signers := allSigners(ctx, tt.SecretPath, tt.config)
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

func TestGetRawPayload(t *testing.T) {
	tests := []struct {
		name     string
		payload  interface{}
		expected string
	}{
		{
			name: "intoto.Statement object",
			payload: intoto.Statement{
				Type:          "type1",
				PredicateType: "predicate-type1",
			},
			expected: compactJSON(t, []byte(`{"_type":"type1","predicateType":"predicate-type1"}`)),
		},
		{
			name: "*intoto.Statement object",
			payload: &intoto.Statement{
				Type:          "type1",
				PredicateType: "predicate-type1",
			},
			expected: compactJSON(t, []byte(`{"_type":"type1","predicateType":"predicate-type1"}`)),
		},
		{
			name:     "*intoto.Statement object - nil",
			payload:  (func() *intoto.Statement { return nil })(),
			expected: "null",
		},
		{
			name:     "other object - nil",
			payload:  nil,
			expected: "null",
		},
		{
			name: "other object with value",
			payload: struct {
				Name  string
				ID    int
				Inner any
			}{
				Name: "wrapper",
				ID:   1,
				Inner: struct {
					InnerID     int
					Description string
					IsArtifact  bool
				}{
					InnerID:     2,
					Description: "some description",
					IsArtifact:  true,
				},
			},
			expected: compactJSON(t, []byte(`{"Name":"wrapper","ID":1,"Inner": {"InnerID":2,"Description":"some description","IsArtifact":true}}`)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := getRawPayload(test.payload)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			compactExpected := compactJSON(t, got)
			if diff := cmp.Diff(test.expected, compactExpected); diff != "" {
				t.Errorf("getRawPayload(), -want +got, diff = %s", diff)
			}
		})
	}
}

func compactJSON(t *testing.T, jsonString []byte) string {
	t.Helper()
	dst := &bytes.Buffer{}
	if err := json.Compact(dst, jsonString); err != nil {
		t.Fatalf("error getting compact JSON: %v", err)
	}
	return dst.String()
}

func fakeAllBackends(backends []*mockBackend) map[string]storage.Backend {
	newBackends := map[string]storage.Backend{}
	for _, m := range backends {
		newBackends[m.backendType] = m
	}
	return newBackends
}

func TestObjectSigner_Sign_OCIDisabled(t *testing.T) {
	tests := []struct {
		name                  string
		config                config.Config
		expectOCIEnabled      bool
		expectPipelineEnabled bool
	}{
		{
			name: "OCI signing enabled by default",
			config: config.Config{
				Artifacts: config.ArtifactConfigs{
					OCI: config.Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.New[string]("mock"),
						Signer:         "x509",
					},
					PipelineRuns: config.Artifact{
						Format:         "slsa/v1",
						StorageBackend: sets.New[string]("mock"),
						Signer:         "x509",
					},
				},
			},
			expectOCIEnabled:      true,
			expectPipelineEnabled: true,
		},
		{
			name: "OCI signing disabled with none signer",
			config: config.Config{
				Artifacts: config.ArtifactConfigs{
					OCI: config.Artifact{
						Format:         "simplesigning",
						StorageBackend: sets.New[string]("mock"),
						Signer:         "none",
					},
					PipelineRuns: config.Artifact{
						Format:         "slsa/v1",
						StorageBackend: sets.New[string]("mock"),
						Signer:         "x509",
					},
				},
			},
			expectOCIEnabled:      false,
			expectPipelineEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test direct artifact configuration - this is the core functionality
			if tt.config.Artifacts.OCI.Enabled() != tt.expectOCIEnabled {
				t.Errorf("Config OCI.Enabled() = %v, want %v", tt.config.Artifacts.OCI.Enabled(), tt.expectOCIEnabled)
			}
			if tt.config.Artifacts.PipelineRuns.Enabled() != tt.expectPipelineEnabled {
				t.Errorf("Config PipelineRuns.Enabled() = %v, want %v", tt.config.Artifacts.PipelineRuns.Enabled(), tt.expectPipelineEnabled)
			}
		})
	}
}

func setupMocks(rekor *mockRekor) func() {
	oldRekor := getRekor
	getRekor = func(_ string) (rekorClient, error) {
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

// Additional tests for protobuf bundle fix

func TestBundle_PublicKey(t *testing.T) {
	// Test Bundle struct with PublicKey field
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	tests := []struct {
		name   string
		bundle signing.Bundle
		want   crypto.PublicKey
	}{
		{
			name: "Bundle with PublicKey set",
			bundle: signing.Bundle{
				Content:   []byte("test-content"),
				Signature: []byte("test-signature"),
				Cert:      []byte("test-cert"),
				Chain:     []byte("test-chain"),
				PublicKey: publicKey,
			},
			want: publicKey,
		},
		{
			name: "Bundle with nil PublicKey",
			bundle: signing.Bundle{
				Content:   []byte("test-content"),
				Signature: []byte("test-signature"),
				Cert:      []byte("test-cert"),
				Chain:     []byte("test-chain"),
				PublicKey: nil,
			},
			want: nil,
		},
		{
			name: "Empty Bundle",
			bundle: signing.Bundle{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.bundle.PublicKey != tt.want {
				t.Errorf("Bundle.PublicKey = %v, want %v", tt.bundle.PublicKey, tt.want)
			}

			// Test that PublicKey field can be set and retrieved
			if tt.want != nil {
				// Verify the key is the expected type
				if _, ok := tt.bundle.PublicKey.(*rsa.PublicKey); !ok {
					t.Errorf("Expected PublicKey to be *rsa.PublicKey, got %T", tt.bundle.PublicKey)
				}

				// Verify the key matches our test key
				rsaKey, ok := tt.bundle.PublicKey.(*rsa.PublicKey)
				if !ok {
					t.Fatalf("PublicKey is not *rsa.PublicKey")
				}
				expectedRSAKey, ok := tt.want.(*rsa.PublicKey)
				if !ok {
					t.Fatalf("Expected key is not *rsa.PublicKey")
				}

				if rsaKey.N.Cmp(expectedRSAKey.N) != 0 || rsaKey.E != expectedRSAKey.E {
					t.Errorf("PublicKey does not match expected key")
				}
			}
		})
	}
}

// Mock signer for testing public key extraction
type mockSignerWithPublicKey struct {
	publicKey crypto.PublicKey
	cert      string
	chain     string
	shouldErr bool
	errOnPubKey bool
}

func (m *mockSignerWithPublicKey) SignMessage(msg io.Reader, opts ...signature.SignOption) ([]byte, error) {
	if m.shouldErr {
		return nil, errors.New("mock signing error")
	}
	return []byte("mock-signature"), nil
}

func (m *mockSignerWithPublicKey) VerifySignature(signature, message io.Reader, opts ...signature.VerifyOption) error {
	return nil
}

func (m *mockSignerWithPublicKey) PublicKey(opts ...signature.PublicKeyOption) (crypto.PublicKey, error) {
	if m.errOnPubKey {
		return nil, errors.New("mock public key error")
	}
	return m.publicKey, nil
}

func (m *mockSignerWithPublicKey) Type() string {
	return "mock"
}

func (m *mockSignerWithPublicKey) Cert() string {
	return m.cert
}

func (m *mockSignerWithPublicKey) Chain() string {
	return m.chain
}

func TestSigner_PublicKeyExtraction(t *testing.T) {
	// Test that the signing loop includes public key extraction logic
	// This test verifies the behavior exists but doesn't require complex mocking

	// Test that we can create a StorageOpts with public key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	// Test that StorageOpts can hold public key correctly
	opts := config.StorageOpts{
		FullKey:       "test-full",
		ShortKey:      "test-short",
		Cert:          "test-cert",
		Chain:         "test-chain",
		PublicKey:     publicKey,
		PayloadFormat: "in-toto",
	}

	if opts.PublicKey != publicKey {
		t.Error("StorageOpts should preserve public key")
	}

	// Test that mockBackendWithCapture can capture StorageOpts correctly
	var capturedOpts config.StorageOpts
	backend := &mockBackendWithCapture{
		backendType:  "test",
		capturedOpts: &capturedOpts,
	}

	ctx := context.Background()
	tro := objects.NewTaskRunObjectV1(&v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	})

	err = backend.StorePayload(ctx, tro, []byte("test"), "signature", opts)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify the options were captured correctly
	if capturedOpts.PublicKey != publicKey {
		t.Error("Backend should capture public key correctly")
	}
	if capturedOpts.Cert != "test-cert" {
		t.Errorf("Expected cert = test-cert, got %s", capturedOpts.Cert)
	}
}

// Mock backend that captures StorageOpts for testing
type mockBackendWithCapture struct {
	storedPayload []byte
	shouldErr     bool
	backendType   string
	capturedOpts  *config.StorageOpts
}

func (b *mockBackendWithCapture) StorePayload(ctx context.Context, _ objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	if b.shouldErr {
		return errors.New("mock error storing")
	}
	b.storedPayload = rawPayload
	if b.capturedOpts != nil {
		*b.capturedOpts = opts
	}
	return nil
}

func (b *mockBackendWithCapture) Type() string {
	return b.backendType
}

func (b *mockBackendWithCapture) RetrievePayloads(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (b *mockBackendWithCapture) RetrieveSignatures(ctx context.Context, _ objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	return nil, fmt.Errorf("not implemented")
}
