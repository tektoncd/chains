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

package oci

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto"

	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	remotetest "github.com/tektoncd/pipeline/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/webhook/certificates/resources"
)

const namespace = "oci-test"

var (
	tr = &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: namespace,
		},
	}
	pr = &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: namespace,
		},
	}
)

func TestBackend_StorePayload(t *testing.T) {
	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, _ := url.Parse(s.URL)

	// Push image to server
	ref, err := remotetest.CreateImage(u.Host+"/task/"+tr.Name, tr)
	if err != nil {
		t.Fatalf("failed to push img: %v", err)
	}
	digest := strings.Split(ref, "@")[1]
	digSplits := strings.Split(digest, ":")
	algo, hex := digSplits[0], digSplits[1]

	simple := simple.SimpleContainerImage{
		Critical: payload.Critical{
			Identity: payload.Identity{
				DockerReference: u.Host + "/task/" + tr.Name,
			},
			Image: payload.Image{
				DockerManifestDigest: digest,
			},
			Type: payload.CosignSignatureType,
		},
	}

	intotoStatement := &intoto.Statement{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: slsa.PredicateSLSAProvenance,
		Subject: []*intoto.ResourceDescriptor{
			{
				Name: u.Host + "/task/" + tr.Name,
				Digest: common.DigestSet{
					algo: hex,
				},
			},
		},
		Predicate: &structpb.Struct{},
	}

	type fields struct {
		object objects.TektonObject
	}
	type args struct {
		payload     interface{}
		signature   string
		storageOpts config.StorageOpts
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{{
		name: "simplesigning payload",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   simple,
			signature: "simplesigning",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSimpleSigning,
			},
		},
		wantErr: false,
	}, {
		name: "into-to payload",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   intotoStatement,
			signature: "into-to",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}, {
		name: "no subject",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   intoto.Statement{},
			signature: "",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}, {
		name: "simplesigning payload",
		fields: fields{
			object: objects.NewPipelineRunObjectV1(pr),
		},
		args: args{
			payload:   simple,
			signature: "simplesigning",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSimpleSigning,
			},
		},
		wantErr: false,
	}, {
		name: "into-to payload",
		fields: fields{
			object: objects.NewPipelineRunObjectV1(pr),
		},
		args: args{
			payload:   intotoStatement,
			signature: "into-to",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}, {
		name: "in-toto-and-simple-payload",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   simple,
			signature: "",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}, {
		name: "tekton-and-simple-payload",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   simple,
			signature: "",
			storageOpts: config.StorageOpts{
				PayloadFormat: "tekton",
			},
		},
		wantErr: false,
	}, {
		name: "no subject",
		fields: fields{
			object: objects.NewPipelineRunObjectV1(pr),
		},
		args: args{
			payload:   intoto.Statement{},
			signature: "",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			b := &Backend{
				getAuthenticator: func(context.Context, objects.TektonObject, kubernetes.Interface) (remote.Option, error) {
					return remote.WithAuthFromKeychain(authn.DefaultKeychain), nil
				},
			}
			rawPayload, err := json.Marshal(tt.args.payload)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			if err := b.StorePayload(ctx, tt.fields.object, rawPayload, tt.args.signature, tt.args.storageOpts); (err != nil) != tt.wantErr {
				t.Errorf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestBackend_StorePayload_Insecure tests the StorePayload functionality with both secure and insecure configurations.
// It verifies that:
// 1. In secure mode, the backend should reject connections to untrusted registries due to TLS certificate verification failure
// 2. In insecure mode, the backend should successfully connect and upload signatures, bypassing TLS verification
func TestBackend_StorePayload_Insecure(t *testing.T) {
	// Setup test registry with self-signed certificate
	s, registryURL := setupTestRegistry(t)
	defer s.Close()

	testCases := []struct {
		name        string
		insecure    bool
		wantErr     bool
		wantErrMsg  string
		description string
	}{
		{
			name:        "secure mode with untrusted certificate",
			insecure:    false,
			wantErr:     true,
			wantErrMsg:  "tls: failed to verify certificate: x509:",
			description: "Should reject connection to registry with self-signed certificate",
		},
		{
			name:        "insecure mode bypassing TLS verification",
			insecure:    true,
			wantErr:     false,
			wantErrMsg:  "",
			description: "Should successfully connect and upload signature despite untrusted certificate",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Initialize backend with test configuration
			b := &Backend{
				cfg: config.Config{
					Storage: config.StorageConfigs{
						OCI: config.OCIStorageConfig{
							Insecure: tc.insecure,
						},
					},
				},
				getAuthenticator: func(context.Context, objects.TektonObject, kubernetes.Interface) (remote.Option, error) {
					return remote.WithAuthFromKeychain(authn.DefaultKeychain), nil
				},
			}

			// Create test reference and payload
			ref := registryURL + "/task/test@sha256:0000000000000000000000000000000000000000000000000000000000000000"
			simple := simple.SimpleContainerImage{
				Critical: payload.Critical{
					Identity: payload.Identity{
						DockerReference: registryURL + "/task/test",
					},
					Image: payload.Image{
						DockerManifestDigest: strings.Split(ref, "@")[1],
					},
					Type: payload.CosignSignatureType,
				},
			}

			rawPayload, err := json.Marshal(simple)
			if err != nil {
				t.Fatalf("failed to marshal payload: %v", err)
			}

			// Test StorePayload functionality
			ctx := logtesting.TestContextWithLogger(t)
			err = b.StorePayload(ctx, objects.NewTaskRunObjectV1(tr), rawPayload, "test", config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSimpleSigning,
			})

			// Validate test results based on expected outcome
			if tc.wantErr {
				if err == nil {
					t.Errorf("%s: expected error but got nil", tc.description)
					return
				}
				if tc.wantErrMsg != "" && !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("%s: error message mismatch\ngot: %v\nwant: %v", tc.description, err, tc.wantErrMsg)
				}
			} else if err != nil {
				t.Errorf("%s: expected success but got error: %v", tc.description, err)
			}
		})
	}
}

// setupTestRegistry sets up a test registry with TLS configuration
func setupTestRegistry(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("failed to generate self-signed cert: %v", err)
	}

	reg := registry.New()
	s := httptest.NewUnstartedServer(reg)
	s.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	s.StartTLS()

	u, _ := url.Parse(s.URL)
	return s, u.Host
}

// generateSelfSignedCert generates a self-signed certificate for testing purposes
// It uses knative's certificate generation utilities to create a proper certificate chain
func generateSelfSignedCert() (tls.Certificate, error) {
	// Generate certificates with 24 hour validity
	notAfter := time.Now().Add(24 * time.Hour)

	// Use test service name and namespace
	serverKey, serverCert, _, err := resources.CreateCerts(context.Background(), "test-registry", "test-namespace", notAfter)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate certificates: %w", err)
	}

	// Parse the generated certificates
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}

// TestWithEncodingFormat_AttestationStorer verifies that WithEncodingFormat
// correctly sets the encodingFormat field on an AttestationStorer.
func TestWithEncodingFormat_AttestationStorer(t *testing.T) {
	repo, err := name.NewRepository("example.com/test")
	if err != nil {
		t.Fatalf("name.NewRepository: %v", err)
	}
	for _, format := range []string{config.OCIEncodingFormatDSSE, config.OCIEncodingFormatSigstoreBundle} {
		t.Run(format, func(t *testing.T) {
			storer, err := NewAttestationStorer(
				WithTargetRepository(repo),
				WithEncodingFormat(format),
			)
			if err != nil {
				t.Fatalf("NewAttestationStorer: %v", err)
			}
			if storer.encodingFormat != format {
				t.Errorf("encodingFormat = %q, want %q", storer.encodingFormat, format)
			}
		})
	}
}

// TestWithEncodingFormat_SimpleStorer verifies that WithEncodingFormat
// correctly sets the encodingFormat field on a SimpleStorer.
func TestWithEncodingFormat_SimpleStorer(t *testing.T) {
	repo, err := name.NewRepository("example.com/test")
	if err != nil {
		t.Fatalf("name.NewRepository: %v", err)
	}
	for _, format := range []string{config.OCIEncodingFormatDSSE, config.OCIEncodingFormatSigstoreBundle} {
		t.Run(format, func(t *testing.T) {
			storer, err := NewSimpleStorerFromConfig(
				WithTargetRepository(repo),
				WithEncodingFormat(format),
			)
			if err != nil {
				t.Fatalf("NewSimpleStorerFromConfig: %v", err)
			}
			if storer.encodingFormat != format {
				t.Errorf("encodingFormat = %q, want %q", storer.encodingFormat, format)
			}
		})
	}
}

// TestDefaultsAreEmpty verifies that omitting the option leaves encodingFormat
// empty (which the Store methods treat as dsse).
func TestDefaultsAreEmpty(t *testing.T) {
	repo, err := name.NewRepository("example.com/test")
	if err != nil {
		t.Fatalf("name.NewRepository: %v", err)
	}

	attestStorer, err := NewAttestationStorer(WithTargetRepository(repo))
	if err != nil {
		t.Fatalf("NewAttestationStorer: %v", err)
	}
	if attestStorer.encodingFormat != "" {
		t.Errorf("AttestationStorer.encodingFormat without option = %q, want empty", attestStorer.encodingFormat)
	}

	simpleStorer, err := NewSimpleStorerFromConfig(WithTargetRepository(repo))
	if err != nil {
		t.Fatalf("NewSimpleStorerFromConfig: %v", err)
	}
	if simpleStorer.encodingFormat != "" {
		t.Errorf("SimpleStorer.encodingFormat without option = %q, want empty", simpleStorer.encodingFormat)
	}
}

// TestOCIBackend_EncodingFormatConfig verifies that the Backend struct properly
// exposes the encoding-format OCI configuration.
func TestOCIBackend_EncodingFormatConfig(t *testing.T) {
	for _, format := range []string{config.OCIEncodingFormatDSSE, config.OCIEncodingFormatSigstoreBundle} {
		t.Run(format, func(t *testing.T) {
			backend := &Backend{
				cfg: config.Config{
					Storage: config.StorageConfigs{
						OCI: config.OCIStorageConfig{
							Repository:     "example.com/repo",
							EncodingFormat: format,
						},
					},
				},
			}
			if backend.cfg.Storage.OCI.EncodingFormat != format {
				t.Errorf("EncodingFormat = %q, want %q",
					backend.cfg.Storage.OCI.EncodingFormat, format)
			}
		})
	}
}

// TestReferrersRepoOverrideIgnored verifies the helper that flags when a
// storage.oci.repository override cannot be honoured in sigstore-bundle mode.
// Referrers are colocated with their subject image, so an override pointing at a
// different repository is reported as ignored, while an override that matches the
// artifact repository (the no-op case) is not.
func TestReferrersRepoOverrideIgnored(t *testing.T) {
	artifact, err := name.NewRepository("registry.example.com/team/app")
	if err != nil {
		t.Fatalf("name.NewRepository: %v", err)
	}
	differentRepo, err := name.NewRepository("registry.example.com/team/signatures")
	if err != nil {
		t.Fatalf("name.NewRepository: %v", err)
	}
	differentRegistry, err := name.NewRepository("other.example.com/team/app")
	if err != nil {
		t.Fatalf("name.NewRepository: %v", err)
	}

	tests := []struct {
		name     string
		override name.Repository
		want     bool
	}{
		{name: "same repository is not ignored", override: artifact, want: false},
		{name: "different repository in same registry is ignored", override: differentRepo, want: true},
		{name: "different registry is ignored", override: differentRegistry, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := referrersRepoOverrideIgnored(tc.override, artifact); got != tc.want {
				t.Errorf("referrersRepoOverrideIgnored(%q, %q) = %v, want %v", tc.override, artifact, got, tc.want)
			}
		})
	}
}

// TestBackend_StorePayload_SigstoreBundle_BundlePayloadPreserved is a regression
// test for uploadAttestation omitting Content from signing.Bundle when constructing
// the sigstore-bundle call. If Content is nil, cbundle.MakeNewBundle produces a
// protobuf bundle whose dsseEnvelope.payload field is absent, making the attestation
// unverifiable. The fix: uploadAttestation must set Content: rawPayload in Bundle.
func TestBackend_StorePayload_SigstoreBundle_BundlePayloadPreserved(t *testing.T) {
	s := httptest.NewServer(registry.New(registry.WithReferrersSupport(true)))
	defer s.Close()
	u, _ := url.Parse(s.URL)

	imgRefStr, err := remotetest.CreateImage(u.Host+"/test/attestation-img", tr)
	if err != nil {
		t.Fatalf("CreateImage: %v", err)
	}
	imgRef, err := name.NewDigest(imgRefStr)
	if err != nil {
		t.Fatalf("name.NewDigest: %v", err)
	}
	// DigestStr() = "sha256:HEX" — split into algo and hex for the in-toto subject.
	digestParts := strings.SplitN(imgRef.DigestStr(), ":", 2)
	algo, digestHex := digestParts[0], digestParts[1]

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey: %v", err)
	}

	// Build an in-toto statement pointing at the image; rawPayload is the content that
	// must appear in the stored bundle's dsseEnvelope.payload field.
	// Use protojson.Marshal (not encoding/json) so field names are camelCase
	// (e.g. "predicateType"), matching what uploadAttestation in legacy.go expects.
	stmt := &intoto.Statement{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: slsa.PredicateSLSAProvenance,
		Subject: []*intoto.ResourceDescriptor{{
			Name:   imgRef.Repository.String(),
			Digest: common.DigestSet{algo: digestHex},
		}},
		Predicate: &structpb.Struct{},
	}
	rawPayload, err := protojson.Marshal(stmt)
	if err != nil {
		t.Fatalf("protojson.Marshal(statement): %v", err)
	}

	// dsseEnv is the DSSE envelope JSON that StorePayload receives as its signature
	// arg. MakeNewBundle extracts the inner signature bytes from it; the Content field
	// (rawPayload) is what fills dsseEnvelope.payload in the output bundle.
	dsseEnv := testDSSEEnvelope(t, rawPayload)

	b := &Backend{
		cfg: config.Config{
			Storage: config.StorageConfigs{
				OCI: config.OCIStorageConfig{
					EncodingFormat: config.OCIEncodingFormatSigstoreBundle,
				},
			},
		},
		getAuthenticator: func(context.Context, objects.TektonObject, kubernetes.Interface) (remote.Option, error) {
			return remote.WithAuthFromKeychain(authn.DefaultKeychain), nil
		},
	}

	ctx := logtesting.TestContextWithLogger(t)
	if err := b.StorePayload(ctx, objects.NewTaskRunObjectV1(tr), rawPayload, string(dsseEnv), config.StorageOpts{
		PayloadFormat: formats.PayloadTypeSlsav1,
		PublicKey:     privKey.Public(),
	}); err != nil {
		t.Fatalf("StorePayload: %v", err)
	}

	// Discover the referrer written by uploadAttestation.
	idx, err := ociremote.Referrers(imgRef, "")
	if err != nil {
		t.Fatalf("ociremote.Referrers: %v", err)
	}
	if len(idx.Manifests) == 0 {
		t.Fatalf("expected at least one referrer, got none; bundle was not stored")
	}

	refRef, err := name.NewDigest(fmt.Sprintf("%s@%s", imgRef.Repository.Name(), idx.Manifests[0].Digest))
	if err != nil {
		t.Fatalf("referrer digest ref: %v", err)
	}
	refImg, err := remote.Image(refRef)
	if err != nil {
		t.Fatalf("remote.Image(referrer): %v", err)
	}
	layers, err := refImg.Layers()
	if err != nil || len(layers) == 0 {
		t.Fatalf("referrer has no layers: %v", err)
	}
	rc, err := layers[0].Compressed()
	if err != nil {
		t.Fatalf("layer.Compressed: %v", err)
	}
	defer rc.Close()
	bundleBytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("io.ReadAll(bundle layer): %v", err)
	}

	// The protobuf bundle is serialised by protojson: bytes fields are standard base64.
	// Assert dsseEnvelope.payload is present and round-trips back to the original payload.
	var bundle struct {
		DsseEnvelope struct {
			Payload string `json:"payload"`
		} `json:"dsseEnvelope"`
	}
	if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
		t.Fatalf("unmarshal bundle JSON: %v", err)
	}
	if bundle.DsseEnvelope.Payload == "" {
		t.Fatal("bundle.dsseEnvelope.payload is empty: " +
			"uploadAttestation must set Content: rawPayload in signing.Bundle; " +
			"if nil, cbundle.MakeNewBundle omits the payload key from the bundle JSON")
	}
	got, err := base64.StdEncoding.DecodeString(bundle.DsseEnvelope.Payload)
	if err != nil {
		t.Fatalf("base64-decode bundle.dsseEnvelope.payload: %v", err)
	}
	if string(got) != string(rawPayload) {
		t.Errorf("bundle.dsseEnvelope.payload decoded to %q, want rawPayload %q", got, rawPayload)
	}
}
