/*
Copyright 2023 The Tekton Authors
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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"k8s.io/client-go/kubernetes"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/webhook/certificates/resources"
)

func TestNewRepo(t *testing.T) {
	t.Run("Use any registry in storage oci repository", func(t *testing.T) {
		cfg := config.Config{}
		cfg.Storage.OCI.Repository = "example.com/foo"
		tests := []struct {
			imageName        string
			expectedRepoName string
		}{
			{
				imageName:        "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:bc4f7468f87486e3835b09098c74cd7f54db2cf697cbb9b824271b95a2d0871e",
				expectedRepoName: "example.com/foo",
			},
			{
				imageName:        "foo.io/bar/kaniko-chains@sha256:bc4f7468f87486e3835b09098c74cd7f54db2cf697cbb9b824271b95a2d0871e",
				expectedRepoName: "example.com/foo",
			},
			{
				imageName:        "registry.com/spam/spam/spam/spam/spam/spam@sha256:bc4f7468f87486e3835b09098c74cd7f54db2cf697cbb9b824271b95a2d0871e",
				expectedRepoName: "example.com/foo",
			},
		}

		for _, test := range tests {
			ref, err := name.NewDigest(test.imageName)
			if err != nil {
				t.Error(err)
			}
			repo, err := newRepo(cfg, ref)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, repo.Name(), test.expectedRepoName)
		}
	})
}

// TestBackend_StorePayload_Insecure tests the StorePayload functionality with both secure and insecure configurations.
// It verifies that:
// 1. In secure mode, the backend should reject connections to untrusted registries
// 2. In insecure mode, the backend should attempt to connect but fail due to missing image
func TestBackend_StorePayload_Insecure(t *testing.T) {
	// Setup test registry with self-signed certificate
	s, registryURL := setupTestRegistry(t)
	defer s.Close()

	testCases := []struct {
		name       string
		insecure   bool
		wantErrMsg string
	}{
		{
			name:       "secure mode - should reject untrusted registry",
			insecure:   false,
			wantErrMsg: "tls: failed to verify certificate: x509: certificate signed by unknown authority",
		},
		{
			name:       "insecure mode - should attempt connection but fail due to missing image",
			insecure:   true,
			wantErrMsg: "getting signed image: entity not found in registry",
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
			err = b.StorePayload(ctx, objects.NewTaskRunObjectV1Beta1(tr), rawPayload, "test", config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSimpleSigning,
			})

			if err == nil {
				t.Error("expected error but got nil")
				return
			}
			if !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Errorf("error message mismatch\ngot: %v\nwant: %v", err, tc.wantErrMsg)
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
		return tls.Certificate{}, fmt.Errorf("failed to generate certificates: %v", err)
	}

	// Parse the generated certificates
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to parse certificate: %v", err)
	}

	return cert, nil
}
