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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto"

	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
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
