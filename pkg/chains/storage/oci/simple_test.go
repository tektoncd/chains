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
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	cosigntypes "github.com/sigstore/cosign/v2/pkg/types"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/config"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestSimpleStorer_Store(t *testing.T) {
	tests := []struct {
		name            string
		writeToRegistry func(*testing.T, string) name.Digest
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
					t.Fatalf("failed to create random layer: %s", err)
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

			storer, err := NewSimpleStorerFromConfig(WithTargetRepository(ref.Repository))
			if err != nil {
				t.Fatalf("failed to create storer: %v", err)
			}

			ctx := logtesting.TestContextWithLogger(t)
			_, err = storer.Store(ctx, &api.StoreRequest[name.Digest, simple.SimpleContainerImage]{
				Artifact: ref,
				Payload:  simple.NewSimpleStruct(ref),
				Bundle:   &signing.Bundle{},
			})

			if err != nil {
				t.Fatalf("error during Store(): %s", err)
			}
		})
	}
}

func TestSimpleStorer_Store_Dedup(t *testing.T) {
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

	storer, err := NewSimpleStorerFromConfig(WithTargetRepository(ref.Repository))
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}

	ctx := logtesting.TestContextWithLogger(t)
	req := &api.StoreRequest[name.Digest, simple.SimpleContainerImage]{
		Artifact: ref,
		Payload:  simple.NewSimpleStruct(ref),
		Bundle:   &signing.Bundle{Content: []byte("payload"), Signature: []byte("sig1")},
	}

	// Store the same signature twice.
	if _, err := storer.Store(ctx, req); err != nil {
		t.Fatalf("first Store() failed: %s", err)
	}
	if _, err := storer.Store(ctx, req); err != nil {
		t.Fatalf("second Store() failed: %s", err)
	}

	// Verify only one signature layer exists.
	se, err := ociremote.SignedEntity(ref)
	if err != nil {
		t.Fatalf("failed to get signed entity: %v", err)
	}
	sigs, err := se.Signatures()
	if err != nil {
		t.Fatalf("failed to get signatures: %v", err)
	}
	layers, err := sigs.Get()
	if err != nil {
		t.Fatalf("failed to get signature layers: %v", err)
	}
	if got := len(layers); got != 1 {
		t.Errorf("expected 1 signature layer, got %d", got)
	}
}

func TestSimpleStorer_Store_DistinctNotDeduped(t *testing.T) {
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

	storer, err := NewSimpleStorerFromConfig(WithTargetRepository(ref.Repository))
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}

	ctx := logtesting.TestContextWithLogger(t)

	// Store two signatures with different content (different layer digests).
	req1 := &api.StoreRequest[name.Digest, simple.SimpleContainerImage]{
		Artifact: ref,
		Payload:  simple.NewSimpleStruct(ref),
		Bundle:   &signing.Bundle{Content: []byte("payload1"), Signature: []byte("sig1")},
	}
	req2 := &api.StoreRequest[name.Digest, simple.SimpleContainerImage]{
		Artifact: ref,
		Payload:  simple.NewSimpleStruct(ref),
		Bundle:   &signing.Bundle{Content: []byte("payload2"), Signature: []byte("sig2")},
	}

	if _, err := storer.Store(ctx, req1); err != nil {
		t.Fatalf("first Store() failed: %s", err)
	}
	if _, err := storer.Store(ctx, req2); err != nil {
		t.Fatalf("second Store() failed: %s", err)
	}

	// Verify both signature layers are kept.
	se, err := ociremote.SignedEntity(ref)
	if err != nil {
		t.Fatalf("failed to get signed entity: %v", err)
	}
	sigs, err := se.Signatures()
	if err != nil {
		t.Fatalf("failed to get signatures: %v", err)
	}
	layers, err := sigs.Get()
	if err != nil {
		t.Fatalf("failed to get signature layers: %v", err)
	}
	if got := len(layers); got != 2 {
		t.Errorf("expected 2 distinct signature layers, got %d", got)
	}
}

// TestSimpleStorer_Store_DSSEWritesSigTag is a regression test: in dsse mode the
// signature is still a simplesigning layer stored under the legacy .sig tag.
func TestSimpleStorer_Store_DSSEWritesSigTag(t *testing.T) {
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

	payload := simple.NewSimpleStruct(ref)
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	storer, err := NewSimpleStorerFromConfig(
		WithTargetRepository(ref.Repository),
		WithEncodingFormat(config.OCIEncodingFormatDSSE),
	)
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}
	ctx := logtesting.TestContextWithLogger(t)
	if _, err := storer.Store(ctx, &api.StoreRequest[name.Digest, simple.SimpleContainerImage]{
		Artifact: ref,
		Payload:  payload,
		Bundle:   &signing.Bundle{Content: rawPayload, Signature: []byte("c2lnbmF0dXJl")},
	}); err != nil {
		t.Fatalf("error during Store(): %s", err)
	}

	sigTag := strings.Replace(imgDigest.String(), ":", "-", 1) + ".sig"
	tags, err := remote.List(ref.Repository)
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}
	found := false
	for _, tag := range tags {
		if tag == sigTag {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected tag %q in %v", sigTag, tags)
	}

	sigImg, err := remote.Image(ref.Tag(sigTag))
	if err != nil {
		t.Fatalf("failed to fetch signature image: %v", err)
	}
	layers, err := sigImg.Layers()
	if err != nil || len(layers) != 1 {
		t.Fatalf("expected 1 signature layer, got %d (err: %v)", len(layers), err)
	}
	mt, err := layers[0].MediaType()
	if err != nil {
		t.Fatalf("failed to get layer media type: %v", err)
	}
	if string(mt) != cosigntypes.SimpleSigningMediaType {
		t.Errorf("layer media type = %q, want %q", mt, cosigntypes.SimpleSigningMediaType)
	}
}

// TestSimpleStorer_Store_SigstoreBundleRejected verifies the simplesigning storer
// refuses sigstore-bundle mode instead of writing an unverifiable bundle.
func TestSimpleStorer_Store_SigstoreBundleRejected(t *testing.T) {
	s := httptest.NewServer(registry.New(registry.WithReferrersSupport(true)))
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

	storer, err := NewSimpleStorerFromConfig(
		WithTargetRepository(ref.Repository),
		WithEncodingFormat(config.OCIEncodingFormatSigstoreBundle),
	)
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}
	ctx := logtesting.TestContextWithLogger(t)
	if _, err := storer.Store(ctx, &api.StoreRequest[name.Digest, simple.SimpleContainerImage]{
		Artifact: ref,
		Payload:  simple.NewSimpleStruct(ref),
		Bundle:   &signing.Bundle{},
	}); err == nil {
		t.Fatal("expected an error in sigstore-bundle mode, got nil")
	}

	idx, err := ociremote.Referrers(ref, "")
	if err == nil && len(idx.Manifests) > 0 {
		t.Errorf("expected no referrers to be written, got %d", len(idx.Manifests))
	}
}
