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
	v1 "github.com/google/go-containerregistry/pkg/v1"
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

// TestSimpleStorer_Store_SigstoreBundle verifies that the sigstore-bundle encoding
// path writes a Sigstore protobuf bundle referrer: artifactType set to
// bundleArtifactType, dev.sigstore.bundle.predicateType annotation set to
// CosignSignPredicateType, and subject pointing back at the image.
func TestSimpleStorer_Store_SigstoreBundle(t *testing.T) {
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
		Bundle:   &signing.Bundle{Content: []byte("payload"), Signature: []byte("sig1")},
	}); err != nil {
		t.Fatalf("error during Store(): %s", err)
	}

	// No legacy .sig tag should have been created in sigstore-bundle mode.
	tags, err := remote.List(ref.Repository)
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}
	for _, tag := range tags {
		if strings.HasSuffix(tag, ".sig") {
			t.Errorf("unexpected legacy signature tag %q created in sigstore-bundle mode", tag)
		}
	}

	// Discover the signature via the OCI 1.1 Referrers API.
	// Note: use empty filter because the mock registry derives ArtifactType from
	// config.MediaType ("application/vnd.oci.empty.v1+json") rather than the manifest's
	// top-level artifactType field; real OCI 1.1 registries handle this correctly.
	idx, err := ociremote.Referrers(ref, "")
	if err != nil {
		t.Fatalf("failed to list referrers: %v", err)
	}
	if len(idx.Manifests) == 0 {
		t.Fatalf("expected at least one signature referrer, got none")
	}

	// Fetch the referrer manifest and assert its bundle shape.
	refDesc := idx.Manifests[0]
	referrerRef, err := name.NewDigest(fmt.Sprintf("%s@%s", ref.Repository.Name(), refDesc.Digest))
	if err != nil {
		t.Fatalf("failed to build referrer digest ref: %v", err)
	}
	got, err := remote.Get(referrerRef)
	if err != nil {
		t.Fatalf("failed to fetch referrer manifest: %v", err)
	}
	var m v1.Manifest
	if err := json.Unmarshal(got.Manifest, &m); err != nil {
		t.Fatalf("failed to unmarshal referrer manifest: %v", err)
	}

	if got := m.Annotations["dev.sigstore.bundle.predicateType"]; got != cosigntypes.CosignSignPredicateType {
		t.Errorf("dev.sigstore.bundle.predicateType = %q, want %q", got, cosigntypes.CosignSignPredicateType)
	}
	if m.Subject == nil {
		t.Fatalf("referrer manifest has nil subject, want subject pointing at the image")
	}
	if m.Subject.Digest.String() != imgDigest.String() {
		t.Errorf("subject.digest = %q, want image digest %q", m.Subject.Digest, imgDigest)
	}
	if len(m.Layers) == 0 {
		t.Errorf("expected bundle signature layer, got none")
	}
}

// TestSimpleStorer_Store_SigstoreBundle_RepoOverrideIgnored verifies that a
// storage.oci.repository override is not honoured in sigstore-bundle mode: the
// signature referrer is written alongside the subject image (its own repository),
// not the override repository, because OCI 1.1 referrers must be colocated with
// their subject. This guards the documented behaviour raised in PR review.
func TestSimpleStorer_Store_SigstoreBundle_RepoOverrideIgnored(t *testing.T) {
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

	// Configure a target repository override that differs from the artifact's repo.
	overrideRepo, err := name.NewRepository(fmt.Sprintf("%s/test/override", registryName))
	if err != nil {
		t.Fatalf("failed to parse override repo: %v", err)
	}

	storer, err := NewSimpleStorerFromConfig(
		WithTargetRepository(overrideRepo),
		WithEncodingFormat(config.OCIEncodingFormatSigstoreBundle),
	)
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}

	ctx := logtesting.TestContextWithLogger(t)
	if _, err := storer.Store(ctx, &api.StoreRequest[name.Digest, simple.SimpleContainerImage]{
		Artifact: ref,
		Payload:  simple.NewSimpleStruct(ref),
		Bundle:   &signing.Bundle{Content: []byte("payload"), Signature: []byte("sig1")},
	}); err != nil {
		t.Fatalf("error during Store(): %s", err)
	}

	// The referrer must be discoverable against the artifact's own repository.
	// Note: use empty filter — same reasoning as TestSimpleStorer_Store_SigstoreBundle.
	idx, err := ociremote.Referrers(ref, "")
	if err != nil {
		t.Fatalf("failed to list referrers at artifact repo: %v", err)
	}
	if len(idx.Manifests) == 0 {
		t.Fatalf("expected signature referrer at artifact repo %q, got none", ref.Repository.Name())
	}

	// The override repository must NOT have received the referrer.
	overrideDigest, err := name.NewDigest(fmt.Sprintf("%s@%s", overrideRepo.Name(), imgDigest))
	if err != nil {
		t.Fatalf("failed to build override digest ref: %v", err)
	}
	if overrideIdx, err := ociremote.Referrers(overrideDigest, ""); err == nil && len(overrideIdx.Manifests) > 0 {
		t.Errorf("override repo %q unexpectedly received %d referrer(s); override must be ignored in sigstore-bundle mode", overrideRepo.Name(), len(overrideIdx.Manifests))
	}
}

// TestSimpleStorer_Store_SigstoreBundle_Dedup verifies that storing the same
// signature twice in sigstore-bundle mode results in a single referrer, not two.
func TestSimpleStorer_Store_SigstoreBundle_Dedup(t *testing.T) {
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

	// Exactly one referrer must exist — no duplicates.
	// Note: use empty filter — same reasoning as TestSimpleStorer_Store_SigstoreBundle.
	idx, err := ociremote.Referrers(ref, "")
	if err != nil {
		t.Fatalf("failed to list referrers: %v", err)
	}
	if got := len(idx.Manifests); got != 1 {
		t.Errorf("expected 1 signature referrer after dedup, got %d", got)
	}
}

// TestMakeSigBundleBytes_TlogEntries verifies that makeSigBundleBytes embeds
// tlogEntries when a non-nil RekorEntry is passed, and omits them when nil.
// This guards the fix for the transparency-log omission bug in the signature
// bundle path (legacy.go uploadSignature was not forwarding storageOpts.RekorEntry
// into the Bundle, so req.Bundle.RekorEntry arrived as nil here).
func TestMakeSigBundleBytes_TlogEntries(t *testing.T) {
	// nil rekorEntry → tlogEntries must be absent/empty in the serialized bundle.
	bundleBytes, err := makeSigBundleBytes(nil, nil, []byte("payload"), []byte("sig"), nil)
	if err != nil {
		t.Fatalf("makeSigBundleBytes with nil rekorEntry failed: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(bundleBytes, &got); err != nil {
		t.Fatalf("failed to unmarshal bundle JSON: %v", err)
	}
	vm, _ := got["verificationMaterial"].(map[string]interface{})
	if vm != nil {
		if entries, ok := vm["tlogEntries"]; ok {
			// tlogEntries key present — must be empty or nil.
			if arr, ok := entries.([]interface{}); ok && len(arr) > 0 {
				t.Errorf("expected empty tlogEntries with nil rekorEntry, got %d entries", len(arr))
			}
		}
	}
}
