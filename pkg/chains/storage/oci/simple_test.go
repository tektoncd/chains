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

// sigArtifactType is the artifactType cosign assigns to signature referrers
// (application/vnd.dev.cosign.artifact.sig.v1+json). `cosign verify` filters on
// exactly this value when discovering signatures in OCI 1.1 referrers mode.
const sigArtifactType = "application/vnd.dev.cosign.artifact.sig.v1+json"

// TestSimpleStorer_Store_ReferrersAPI verifies that the referrers-api distribution
// path writes a cosign-native signature manifest: config.mediaType set to the
// signature artifactType and a populated subject pointing back at the image. This
// is the manifest shape `cosign verify` reverse-discovers, so it guards against
// regressing back to the low-level WriteReferrer encoding.
func TestSimpleStorer_Store_ReferrersAPI(t *testing.T) {
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
		WithDistributionMethod(config.OCIDistributionReferrersAPI),
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

	// No legacy .sig tag should have been created in referrers mode.
	tags, err := remote.List(ref.Repository)
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}
	for _, tag := range tags {
		if strings.HasSuffix(tag, ".sig") {
			t.Errorf("unexpected legacy signature tag %q created in referrers mode", tag)
		}
	}

	// Discover the signature via the OCI 1.1 Referrers API, filtered by the cosign
	// signature artifactType.
	idx, err := ociremote.Referrers(ref, sigArtifactType)
	if err != nil {
		t.Fatalf("failed to list referrers: %v", err)
	}
	if len(idx.Manifests) == 0 {
		t.Fatalf("expected at least one signature referrer, got none")
	}

	// Fetch the referrer manifest and assert its cosign-native shape.
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

	if string(m.Config.MediaType) != sigArtifactType {
		t.Errorf("config.mediaType = %q, want %q", m.Config.MediaType, sigArtifactType)
	}
	if m.Subject == nil {
		t.Fatalf("referrer manifest has nil subject, want subject pointing at the image")
	}
	if m.Subject.Digest.String() != imgDigest.String() {
		t.Errorf("subject.digest = %q, want image digest %q", m.Subject.Digest, imgDigest)
	}
	if len(m.Layers) == 0 {
		t.Errorf("expected SimpleSigning signature layers, got none")
	}
}

// TestSimpleStorer_Store_ReferrersAPI_RepoOverrideIgnored verifies that a
// storage.oci.repository override is not honoured in referrers-api mode: the
// signature referrer is written alongside the subject image (its own repository),
// not the override repository, because OCI 1.1 referrers must be colocated with
// their subject. This guards the documented behaviour raised in PR review.
func TestSimpleStorer_Store_ReferrersAPI_RepoOverrideIgnored(t *testing.T) {
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
		WithDistributionMethod(config.OCIDistributionReferrersAPI),
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
	idx, err := ociremote.Referrers(ref, sigArtifactType)
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
	if overrideIdx, err := ociremote.Referrers(overrideDigest, sigArtifactType); err == nil && len(overrideIdx.Manifests) > 0 {
		t.Errorf("override repo %q unexpectedly received %d referrer(s); override must be ignored in referrers mode", overrideRepo.Name(), len(overrideIdx.Manifests))
	}
}
