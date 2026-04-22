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
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/sigstore/cosign/v2/pkg/cosign/bundle"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
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

func TestAttestationStorer_StoreWithRekorBundle(t *testing.T) {
	s := httptest.NewServer(registry.New())
	defer s.Close()
	registryName := strings.TrimPrefix(s.URL, "http://")

	// Push a random image to the registry.
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

	rekorBundle := &bundle.RekorBundle{
		SignedEntryTimestamp: []byte("test-signed-entry-timestamp"),
		Payload: bundle.RekorPayload{
			Body:           "dGVzdC1ib2R5",
			IntegratedTime: 1234567890,
			LogIndex:       0,
			LogID:          "test-log-id",
		},
	}

	storer, err := NewAttestationStorer(WithTargetRepository(ref.Repository))
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}

	ctx := logtesting.TestContextWithLogger(t)
	_, err = storer.Store(ctx, &api.StoreRequest[name.Digest, *intoto.Statement]{
		Artifact: ref,
		Payload:  &intoto.Statement{},
		Bundle: &signing.Bundle{
			Cert:        []byte("test-cert"),
			Chain:       []byte("test-chain"),
			RekorBundle: rekorBundle,
		},
	})
	if err != nil {
		t.Fatalf("error during Store(): %s", err)
	}

	// Verify the attestation was stored with the Rekor bundle annotation.
	se, err := ociremote.SignedEntity(ref)
	if err != nil {
		t.Fatalf("failed to get signed entity: %v", err)
	}

	atts, err := se.Attestations()
	if err != nil {
		t.Fatalf("failed to get attestations: %v", err)
	}

	sigs, err := atts.Get()
	if err != nil {
		t.Fatalf("failed to get signatures: %v", err)
	}

	if len(sigs) != 1 {
		t.Fatalf("expected 1 attestation, got %d", len(sigs))
	}

	annotations, err := sigs[0].Annotations()
	if err != nil {
		t.Fatalf("failed to get annotations: %v", err)
	}

	bundleAnnotation, ok := annotations[static.BundleAnnotationKey]
	if !ok {
		t.Fatal("expected dev.sigstore.cosign/bundle annotation to be present")
	}

	// Verify the bundle content can be deserialized back.
	var gotBundle bundle.RekorBundle
	if err := json.Unmarshal([]byte(bundleAnnotation), &gotBundle); err != nil {
		t.Fatalf("failed to unmarshal bundle annotation: %v", err)
	}

	if string(gotBundle.SignedEntryTimestamp) != "test-signed-entry-timestamp" {
		t.Errorf("unexpected SignedEntryTimestamp: got %s", string(gotBundle.SignedEntryTimestamp))
	}

	if gotBundle.Payload.IntegratedTime != 1234567890 {
		t.Errorf("unexpected IntegratedTime: got %d, want 1234567890", gotBundle.Payload.IntegratedTime)
	}

	if gotBundle.Payload.LogID != "test-log-id" {
		t.Errorf("unexpected LogID: got %s, want test-log-id", gotBundle.Payload.LogID)
	}

	// Also verify cert and chain annotations are present.
	if _, ok := annotations[static.CertificateAnnotationKey]; !ok {
		t.Error("expected dev.sigstore.cosign/certificate annotation to be present")
	}

	if _, ok := annotations[static.ChainAnnotationKey]; !ok {
		t.Error("expected dev.sigstore.cosign/chain annotation to be present")
	}
}

func TestAttestationStorer_StoreWithoutRekorBundle(t *testing.T) {
	s := httptest.NewServer(registry.New())
	defer s.Close()
	registryName := strings.TrimPrefix(s.URL, "http://")

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

	storer, err := NewAttestationStorer(WithTargetRepository(ref.Repository))
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}

	ctx := logtesting.TestContextWithLogger(t)
	_, err = storer.Store(ctx, &api.StoreRequest[name.Digest, *intoto.Statement]{
		Artifact: ref,
		Payload:  &intoto.Statement{},
		Bundle: &signing.Bundle{
			Cert:  []byte("test-cert"),
			Chain: []byte("test-chain"),
		},
	})
	if err != nil {
		t.Fatalf("error during Store(): %s", err)
	}

	// Verify no bundle annotation is present.
	se, err := ociremote.SignedEntity(ref)
	if err != nil {
		t.Fatalf("failed to get signed entity: %v", err)
	}

	atts, err := se.Attestations()
	if err != nil {
		t.Fatalf("failed to get attestations: %v", err)
	}

	sigs, err := atts.Get()
	if err != nil {
		t.Fatalf("failed to get signatures: %v", err)
	}

	if len(sigs) != 1 {
		t.Fatalf("expected 1 attestation, got %d", len(sigs))
	}

	annotations, err := sigs[0].Annotations()
	if err != nil {
		t.Fatalf("failed to get annotations: %v", err)
	}

	if _, ok := annotations[static.BundleAnnotationKey]; ok {
		t.Error("expected dev.sigstore.cosign/bundle annotation to NOT be present when RekorBundle is nil")
	}
}
