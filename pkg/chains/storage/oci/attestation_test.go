// Copyright 2025 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oci

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/config"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestAttestationStorer_Store(t *testing.T) {
	tests := []struct {
		name            string
		writeToRegistry func(*testing.T, string) name.Digest
		wantErr         error
	}{
		{
			name: "image manifest",
			writeToRegistry: func(t *testing.T, registryName string) name.Digest {
				t.Helper()
				img, err := random.Image(1024, 2)
				if err != nil {
					t.Fatalf("failed to create random image: %s", err)
				}
				imgDigest, err := img.Digest()
				if err != nil {
					t.Fatalf("failed to get image digest: %v", err)
				}
				ref, err := name.NewDigest(fmt.Sprintf("%s/test/img@%s", registryName, imgDigest))
				if err != nil {
					t.Fatalf("failed to parse digest: %v", err)
				}
				if err := remote.Write(ref, img); err != nil {
					t.Fatalf("failed to write image to mock registry: %v", err)
				}
				return ref
			},
		},
		{
			name: "image layer",
			writeToRegistry: func(t *testing.T, registryName string) name.Digest {
				t.Helper()
				layer, err := random.Layer(1024, types.OCILayer)
				if err != nil {
					t.Fatalf("failed to create random layer: %v", err)
				}
				layerDigest, err := layer.Digest()
				if err != nil {
					t.Fatalf("failed to get layer digest: %v", err)
				}
				ref, err := name.NewDigest(fmt.Sprintf("%s/test/img@%s", registryName, layerDigest))
				if err != nil {
					t.Fatalf("failed to parse digest: %v", err)
				}
				if err := remote.WriteLayer(ref.Repository, layer); err != nil {
					t.Fatalf("failed to write layer to mock registry: %v", err)
				}
				return ref
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(registry.New())
			defer s.Close()
			registryName := strings.TrimPrefix(s.URL, "http://")

			ref := tt.writeToRegistry(t, registryName)

			storer, err := NewAttestationStorer(WithTargetRepository(ref.Repository))
			if err != nil {
				t.Fatalf("failed to create storer: %v", err)
			}

			ctx := logtesting.TestContextWithLogger(t)
			_, err = storer.Store(ctx, &api.StoreRequest[name.Digest, *intoto.Statement]{
				Artifact: ref,
				Payload:  &intoto.Statement{},
				Bundle:   &signing.Bundle{},
			})

			if err != nil {
				t.Fatalf("error during Store(): %s", err)
			}
		})
	}
}

// Helper function to create a test certificate and key pair
func createTestCertAndKey(t *testing.T) (crypto.PublicKey, []byte) {
	t.Helper()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	// Create a test certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test Org"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  nil,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Encode certificate as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return publicKey, certPEM
}

func TestAttestationStorer_StoreWithProtobufBundle(t *testing.T) {
	// Setup test registry
	s := httptest.NewServer(registry.New())
	defer s.Close()
	registryName := strings.TrimPrefix(s.URL, "http://")

	// Create test image
	img, err := random.Image(1024, 2)
	if err != nil {
		t.Fatalf("failed to create random image: %s", err)
	}
	imgDigest, err := img.Digest()
	if err != nil {
		t.Fatalf("failed to get image digest: %v", err)
	}
	ref, err := name.NewDigest(fmt.Sprintf("%s/test/img@%s", registryName, imgDigest))
	if err != nil {
		t.Fatalf("failed to parse digest: %v", err)
	}
	if err := remote.Write(ref, img); err != nil {
		t.Fatalf("failed to write image to mock registry: %v", err)
	}

	publicKey, certPEM := createTestCertAndKey(t)

	tests := []struct {
		name        string
		bundle      *signing.Bundle
		wantErr     bool
		errContains string
		format      string
	}{
		{
			name: "protobuf bundle with public key priority",
			bundle: &signing.Bundle{
				Content:   []byte(`{"test": "payload"}`),
				Signature: []byte("test-signature"),
				Cert:      certPEM,
				Chain:     []byte("test-chain"),
				PublicKey: publicKey,
			},
			format: config.OCIFormatProtobuf,
		},
		{
			name: "protobuf bundle with certificate fallback",
			bundle: &signing.Bundle{
				Content:   []byte(`{"test": "payload"}`),
				Signature: []byte("test-signature"),
				Cert:      certPEM,
				Chain:     []byte("test-chain"),
				PublicKey: nil, // Test fallback to certificate
			},
			format: config.OCIFormatProtobuf,
		},
		{
			name: "protobuf bundle with no public key or certificate",
			bundle: &signing.Bundle{
				Content:   []byte(`{"test": "payload"}`),
				Signature: []byte("test-signature"),
				Cert:      nil,
				Chain:     []byte("test-chain"),
				PublicKey: nil,
			},
			format:      config.OCIFormatProtobuf,
			wantErr:     true,
			errContains: "no public key available",
		},
		{
			name: "protobuf bundle with invalid certificate",
			bundle: &signing.Bundle{
				Content:   []byte(`{"test": "payload"}`),
				Signature: []byte("test-signature"),
				Cert:      []byte("invalid-cert-data"),
				Chain:     []byte("test-chain"),
				PublicKey: nil,
			},
			format:      config.OCIFormatProtobuf,
			wantErr:     true,
			errContains: "no public key available",
		},
		{
			name: "protobuf bundle without predicate type",
			bundle: &signing.Bundle{
				Content:   []byte(`{"test": "payload"}`),
				Signature: []byte("test-signature"),
				Cert:      certPEM,
				Chain:     []byte("test-chain"),
				PublicKey: publicKey,
			},
			format:      config.OCIFormatProtobuf,
			wantErr:     true,
			errContains: "PredicateType is required",
		},
		{
			name: "legacy format should work normally",
			bundle: &signing.Bundle{
				Content:   []byte(`{"test": "payload"}`),
				Signature: []byte("test-signature"),
				Cert:      certPEM,
				Chain:     []byte("test-chain"),
				PublicKey: publicKey,
			},
			format: config.OCIFormatLegacy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var storer *AttestationStorer
			var err error

			// Create storer with specific format
			if tt.format == config.OCIFormatProtobuf {
				storer, err = NewAttestationStorer(
					WithTargetRepository(ref.Repository),
					WithFormat(config.OCIFormatProtobuf),
				)
			} else {
				storer, err = NewAttestationStorer(
					WithTargetRepository(ref.Repository),
					WithFormat(config.OCIFormatLegacy),
				)
			}
			if err != nil {
				t.Fatalf("failed to create storer: %v", err)
			}

			ctx := logtesting.TestContextWithLogger(t)

			// Create payload with or without predicate type based on test
			var payload *intoto.Statement
			if tt.name == "protobuf bundle without predicate type" {
				payload = &intoto.Statement{
					// Missing PredicateType
					Type: "https://in-toto.io/Statement/v0.1",
				}
			} else {
				payload = &intoto.Statement{
					Type:          "https://in-toto.io/Statement/v0.1",
					PredicateType: "https://slsa.dev/provenance/v0.2",
				}
			}

			_, err = storer.Store(ctx, &api.StoreRequest[name.Digest, *intoto.Statement]{
				Artifact: ref,
				Payload:  payload,
				Bundle:   tt.bundle,
			})

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAttestationStorer_CertificateParsing(t *testing.T) {
	// Test different certificate formats
	_, certPEM := createTestCertAndKey(t)

	// Parse the certificate to get DER bytes
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatalf("Failed to decode PEM certificate")
	}
	certDER := block.Bytes

	tests := []struct {
		name     string
		certData []byte
		wantErr  bool
	}{
		{
			name:     "PEM encoded certificate",
			certData: certPEM,
			wantErr:  false,
		},
		{
			name:     "DER encoded certificate",
			certData: certDER,
			wantErr:  false,
		},
		{
			name:     "invalid certificate data",
			certData: []byte("invalid-cert-data"),
			wantErr:  true,
		},
		{
			name:     "empty certificate data",
			certData: []byte{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test registry
			s := httptest.NewServer(registry.New())
			defer s.Close()
			registryName := strings.TrimPrefix(s.URL, "http://")

			// Create test image
			img, err := random.Image(1024, 2)
			if err != nil {
				t.Fatalf("failed to create random image: %s", err)
			}
			imgDigest, err := img.Digest()
			if err != nil {
				t.Fatalf("failed to get image digest: %v", err)
			}
			ref, err := name.NewDigest(fmt.Sprintf("%s/test/img@%s", registryName, imgDigest))
			if err != nil {
				t.Fatalf("failed to parse digest: %v", err)
			}
			if err := remote.Write(ref, img); err != nil {
				t.Fatalf("failed to write image to mock registry: %v", err)
			}

			storer, err := NewAttestationStorer(
				WithTargetRepository(ref.Repository),
				WithFormat(config.OCIFormatProtobuf),
			)
			if err != nil {
				t.Fatalf("failed to create storer: %v", err)
			}

			ctx := logtesting.TestContextWithLogger(t)

			payload := &intoto.Statement{
				Type:          "https://in-toto.io/Statement/v0.1",
				PredicateType: "https://slsa.dev/provenance/v0.2",
			}

			bundle := &signing.Bundle{
				Content:   []byte(`{"test": "payload"}`),
				Signature: []byte("test-signature"),
				Cert:      tt.certData,
				Chain:     []byte("test-chain"),
				PublicKey: nil, // Force certificate parsing
			}

			_, err = storer.Store(ctx, &api.StoreRequest[name.Digest, *intoto.Statement]{
				Artifact: ref,
				Payload:  payload,
				Bundle:   bundle,
			})

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, but got nil")
				} else if !strings.Contains(err.Error(), "no public key available") {
					t.Errorf("Expected 'no public key available' error, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAttestationStorer_PublicKeyPriority(t *testing.T) {
	// Test that public key takes priority over certificate
	_, cert1PEM := createTestCertAndKey(t)
	publicKey2, _ := createTestCertAndKey(t) // Different key

	// Setup test registry
	s := httptest.NewServer(registry.New())
	defer s.Close()
	registryName := strings.TrimPrefix(s.URL, "http://")

	// Create test image
	img, err := random.Image(1024, 2)
	if err != nil {
		t.Fatalf("failed to create random image: %s", err)
	}
	imgDigest, err := img.Digest()
	if err != nil {
		t.Fatalf("failed to get image digest: %v", err)
	}
	ref, err := name.NewDigest(fmt.Sprintf("%s/test/img@%s", registryName, imgDigest))
	if err != nil {
		t.Fatalf("failed to parse digest: %v", err)
	}
	if err := remote.Write(ref, img); err != nil {
		t.Fatalf("failed to write image to mock registry: %v", err)
	}

	storer, err := NewAttestationStorer(
		WithTargetRepository(ref.Repository),
		WithFormat(config.OCIFormatProtobuf),
	)
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}

	ctx := logtesting.TestContextWithLogger(t)

	payload := &intoto.Statement{
		Type:          "https://in-toto.io/Statement/v0.1",
		PredicateType: "https://slsa.dev/provenance/v0.2",
	}

	// Bundle with both public key and certificate - should use public key
	bundle := &signing.Bundle{
		Content:   []byte(`{"test": "payload"}`),
		Signature: []byte("test-signature"),
		Cert:      cert1PEM,     // This has publicKey1
		Chain:     []byte("test-chain"),
		PublicKey: publicKey2,   // This should take priority
	}

	// This should succeed and use publicKey2, not publicKey1 from certificate
	_, err = storer.Store(ctx, &api.StoreRequest[name.Digest, *intoto.Statement]{
		Artifact: ref,
		Payload:  payload,
		Bundle:   bundle,
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
