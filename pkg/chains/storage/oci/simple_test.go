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
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
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
	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	protocommon "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	sigstoresignature "github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/config"
	"google.golang.org/protobuf/encoding/protojson"
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

// TestSimpleStorer_Store_SigstoreBundle verifies that the sigstore-bundle path
// writes a protobuf bundle referrer with the expected annotations and a subject
// pointing back at the image.
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

	// Discover the signature via the OCI 1.1 Referrers API. Empty filter: the mock
	// registry derives ArtifactType from config.MediaType, not the manifest field.
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
	// Image signatures are MessageSignature bundles, so the content annotation
	// must be "message-signature".
	if got := m.Annotations["dev.sigstore.bundle.content"]; got != "message-signature" {
		t.Errorf("dev.sigstore.bundle.content = %q, want %q", got, "message-signature")
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
// storage.oci.repository override is ignored in sigstore-bundle mode: the
// referrer is written alongside the subject image, not the override repository,
// because OCI 1.1 referrers must be colocated with their subject.
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

// TestMakeSigBundleBytes_MessageSignatureContent verifies that the signature
// bundle produced for a simplesigning payload is a MessageSignature bundle, not
// a DsseEnvelope. A DsseEnvelope would require a dsse/intoto Rekor entry, but
// Chains creates a hashedrekord entry for simplesigning.
func TestMakeSigBundleBytes_MessageSignatureContent(t *testing.T) {
	payload := []byte("payload")
	rawSig := []byte("sig")

	bundleBytes, err := makeSigBundleBytes(nil, nil, payload, rawSig, nil)
	if err != nil {
		t.Fatalf("makeSigBundleBytes failed: %v", err)
	}

	var bundle protobundle.Bundle
	if err := protojson.Unmarshal(bundleBytes, &bundle); err != nil {
		t.Fatalf("failed to unmarshal protobuf bundle: %v", err)
	}

	// The content must be a MessageSignature, never a DsseEnvelope.
	if bundle.GetDsseEnvelope() != nil {
		t.Fatalf("bundle content is a DsseEnvelope; want MessageSignature")
	}
	msg := bundle.GetMessageSignature()
	if msg == nil {
		t.Fatalf("bundle content is not a MessageSignature")
	}

	// The raw signature bytes must round-trip unchanged.
	if string(msg.GetSignature()) != string(rawSig) {
		t.Errorf("MessageSignature.Signature = %q, want %q", msg.GetSignature(), rawSig)
	}

	// The message digest must be sha256(payload), the value the hashedrekord
	// Rekor entry commits to.
	wantDigest := sha256.Sum256(payload)
	if got := msg.GetMessageDigest(); got == nil {
		t.Fatalf("MessageSignature.MessageDigest is nil")
	} else {
		if got.GetAlgorithm() != protocommon.HashAlgorithm_SHA2_256 {
			t.Errorf("MessageDigest.Algorithm = %v, want SHA2_256", got.GetAlgorithm())
		}
		if !bytes.Equal(got.GetDigest(), wantDigest[:]) {
			t.Errorf("MessageDigest.Digest = %x, want %x", got.GetDigest(), wantDigest[:])
		}
	}
}

// TestMakeSigBundleBytes_SignatureVerifies cryptographically verifies the
// signature in the generated bundle against the signed payload. Chains signs the
// raw simplesigning payload (ECDSA over SHA256), so the MessageSignature must
// verify against that payload. The negative case confirms it does NOT verify
// against the DSSE PAE, which a DsseEnvelope bundle would require.
func TestMakeSigBundleBytes_SignatureVerifies(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	sv, err := sigstoresignature.LoadECDSASignerVerifier(privKey, crypto.SHA256)
	if err != nil {
		t.Fatalf("failed to load signer/verifier: %v", err)
	}

	payload := []byte(`{"critical":{"identity":{"docker-reference":"example/image"}}}`)
	rawSig, err := sv.SignMessage(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to sign payload: %v", err)
	}

	bundleBytes, err := makeSigBundleBytes(privKey.Public(), nil, payload, rawSig, nil)
	if err != nil {
		t.Fatalf("makeSigBundleBytes failed: %v", err)
	}

	var bundle protobundle.Bundle
	if err := protojson.Unmarshal(bundleBytes, &bundle); err != nil {
		t.Fatalf("failed to unmarshal protobuf bundle: %v", err)
	}
	msg := bundle.GetMessageSignature()
	if msg == nil {
		t.Fatalf("bundle content is not a MessageSignature")
	}

	// The signature carried in the bundle must verify against the raw payload.
	if err := sv.VerifySignature(bytes.NewReader(msg.GetSignature()), bytes.NewReader(payload)); err != nil {
		t.Fatalf("bundle signature failed to verify against the signed payload: %v", err)
	}

	// Negative control: the signature must NOT verify against the DSSE PAE of the
	// payload. If it did, a DsseEnvelope bundle would be acceptable; it is not.
	pae := dssePAE("application/vnd.in-toto+json", payload)
	if err := sv.VerifySignature(bytes.NewReader(msg.GetSignature()), bytes.NewReader(pae)); err == nil {
		t.Error("signature unexpectedly verified against the DSSE PAE; a MessageSignature must be over the raw payload, not the PAE")
	}
}

// dssePAE builds the DSSE Pre-Authentication Encoding for a payload, the bytes a
// DsseEnvelope signature would be computed over.
func dssePAE(payloadType string, payload []byte) []byte {
	return []byte(fmt.Sprintf("DSSEv1 %d %s %d %s",
		len(payloadType), payloadType, len(payload), payload))
}
