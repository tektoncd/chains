// Copyright 2026 The Tekton Authors
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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

func TestAttestationStorer_Store_Dedup(t *testing.T) {
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
	req := &api.StoreRequest[name.Digest, *intoto.Statement]{
		Artifact: ref,
		Payload:  &intoto.Statement{},
		Bundle:   &signing.Bundle{Signature: []byte("sig1")},
	}

	// Store the same attestation twice.
	if _, err := storer.Store(ctx, req); err != nil {
		t.Fatalf("first Store() failed: %s", err)
	}
	if _, err := storer.Store(ctx, req); err != nil {
		t.Fatalf("second Store() failed: %s", err)
	}

	// Verify only one attestation layer exists.
	se, err := ociremote.SignedEntity(ref)
	if err != nil {
		t.Fatalf("failed to get signed entity: %v", err)
	}
	atts, err := se.Attestations()
	if err != nil {
		t.Fatalf("failed to get attestations: %v", err)
	}
	layers, err := atts.Get()
	if err != nil {
		t.Fatalf("failed to get attestation layers: %v", err)
	}
	if got := len(layers); got != 1 {
		t.Errorf("expected 1 attestation layer, got %d", got)
	}
}

func TestAttestationStorer_Store_DistinctNotDeduped(t *testing.T) {
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

	// Store two attestations with different signatures (different layer content).
	req1 := &api.StoreRequest[name.Digest, *intoto.Statement]{
		Artifact: ref,
		Payload:  &intoto.Statement{},
		Bundle:   &signing.Bundle{Signature: []byte("sig1")},
	}
	req2 := &api.StoreRequest[name.Digest, *intoto.Statement]{
		Artifact: ref,
		Payload:  &intoto.Statement{},
		Bundle:   &signing.Bundle{Signature: []byte("sig2")},
	}

	if _, err := storer.Store(ctx, req1); err != nil {
		t.Fatalf("first Store() failed: %s", err)
	}
	if _, err := storer.Store(ctx, req2); err != nil {
		t.Fatalf("second Store() failed: %s", err)
	}

	// Verify both attestation layers are kept.
	se, err := ociremote.SignedEntity(ref)
	if err != nil {
		t.Fatalf("failed to get signed entity: %v", err)
	}
	atts, err := se.Attestations()
	if err != nil {
		t.Fatalf("failed to get attestations: %v", err)
	}
	layers, err := atts.Get()
	if err != nil {
		t.Fatalf("failed to get attestation layers: %v", err)
	}
	if got := len(layers); got != 2 {
		t.Errorf("expected 2 distinct attestation layers, got %d", got)
	}
}

// testDSSEEnvelope builds a minimal DSSE envelope JSON accepted by cbundle.MakeNewBundle.
// The payload and signature values are arbitrary; this is used only to exercise the
// storage path, not to produce a cryptographically valid attestation.
func testDSSEEnvelope(t *testing.T, payload []byte) []byte {
	t.Helper()
	type sig struct {
		Sig string `json:"sig"`
	}
	type envelope struct {
		Payload     string `json:"payload"`
		PayloadType string `json:"payloadType"`
		Signatures  []sig  `json:"signatures"`
	}
	env := envelope{
		Payload:     base64.StdEncoding.EncodeToString(payload),
		PayloadType: "application/vnd.in-toto+json",
		Signatures:  []sig{{Sig: base64.StdEncoding.EncodeToString([]byte("test-sig-bytes"))}},
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal test DSSE envelope: %v", err)
	}
	return b
}

// TestAttestationStorer_Store_SigstoreBundle verifies that the sigstore-bundle path
// writes the attestation as a protobuf bundle referrer: the bundle artifactType is
// set, a subject pointing at the image is present, no legacy .att tag is created,
// and the referrer is discoverable via the OCI 1.1 Referrers API.
func TestAttestationStorer_Store_SigstoreBundle(t *testing.T) {
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

	// Generate a test key pair; PublicKey is required by storeWithProtobufBundle.
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	payload := []byte(`{"_type":"https://in-toto.io/Statement/v0.1"}`)
	dsseEnv := testDSSEEnvelope(t, payload)

	storer, err := NewAttestationStorer(
		WithTargetRepository(ref.Repository),
		WithEncodingFormat(config.OCIEncodingFormatSigstoreBundle),
	)
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}

	ctx := logtesting.TestContextWithLogger(t)
	if _, err := storer.Store(ctx, &api.StoreRequest[name.Digest, *intoto.Statement]{
		Artifact: ref,
		Payload:  &intoto.Statement{PredicateType: "https://slsa.dev/provenance/v0.2"},
		Bundle: &signing.Bundle{
			Content:   payload,
			Signature: dsseEnv,
			PublicKey: privKey.Public(),
		},
	}); err != nil {
		t.Fatalf("Store() returned unexpected error: %v", err)
	}

	// No legacy .att tag should exist in sigstore-bundle mode.
	tags, err := remote.List(ref.Repository)
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}
	for _, tag := range tags {
		if strings.HasSuffix(tag, ".att") {
			t.Errorf("unexpected legacy attestation tag %q created in sigstore-bundle mode", tag)
		}
	}

	// The attestation must be discoverable as a referrer.
	// Note: we use empty filter because the mock registry derives ArtifactType from
	// config.MediaType ("application/vnd.oci.empty.v1+json") rather than the manifest's
	// top-level artifactType field; real OCI 1.1 registries handle this correctly.
	idx, err := ociremote.Referrers(ref, "")
	if err != nil {
		t.Fatalf("failed to list referrers: %v", err)
	}
	if len(idx.Manifests) == 0 {
		t.Fatalf("expected at least one attestation referrer, got none")
	}

	// Fetch the referrer manifest and verify its layer mediaType.
	desc := idx.Manifests[0]
	refRef, err := name.NewDigest(ref.Repository.Name() + "@" + desc.Digest.String())
	if err != nil {
		t.Fatalf("failed to build referrer digest ref: %v", err)
	}
	refImg, err := remote.Image(refRef)
	if err != nil {
		t.Fatalf("failed to fetch referrer image: %v", err)
	}
	refLayers, err := refImg.Layers()
	if err != nil {
		t.Fatalf("failed to get referrer layers: %v", err)
	}
	if len(refLayers) == 0 {
		t.Fatalf("expected bundle layer in referrer manifest, got none")
	}
	layerMT, err := refLayers[0].MediaType()
	if err != nil {
		t.Fatalf("failed to get layer media type: %v", err)
	}
	if string(layerMT) != bundleArtifactType {
		t.Errorf("layer mediaType = %q, want %q", layerMT, bundleArtifactType)
	}

	// Assert the referrer manifest subject points at the image digest.
	manifest, err := refImg.Manifest()
	if err != nil {
		t.Fatalf("failed to get referrer manifest: %v", err)
	}
	if manifest.Subject == nil {
		t.Fatal("referrer manifest has nil subject, want subject pointing at the image")
	}
	if manifest.Subject.Digest.String() != imgDigest.String() {
		t.Errorf("subject.digest = %q, want image digest %q", manifest.Subject.Digest, imgDigest)
	}

	// Assert the layer content is valid sigstore bundle JSON containing a dsseEnvelope.
	rc, err := refLayers[0].Compressed()
	if err != nil {
		t.Fatalf("failed to open layer: %v", err)
	}
	defer rc.Close()
	bundleBytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read bundle layer: %v", err)
	}
	var bundleJSON struct {
		DsseEnvelope map[string]interface{} `json:"dsseEnvelope"`
	}
	if err := json.Unmarshal(bundleBytes, &bundleJSON); err != nil {
		t.Fatalf("bundle layer is not valid JSON: %v", err)
	}
	if bundleJSON.DsseEnvelope == nil {
		t.Error("bundle layer JSON missing 'dsseEnvelope' key")
	}
}

// TestAttestationStorer_Store_SigstoreBundle_Dedup verifies that storing the same
// attestation bundle twice in sigstore-bundle mode results in a single referrer.
func TestAttestationStorer_Store_SigstoreBundle_Dedup(t *testing.T) {
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

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	payload := []byte(`{"_type":"https://in-toto.io/Statement/v0.1"}`)
	dsseEnv := testDSSEEnvelope(t, payload)

	storer, err := NewAttestationStorer(
		WithTargetRepository(ref.Repository),
		WithEncodingFormat(config.OCIEncodingFormatSigstoreBundle),
	)
	if err != nil {
		t.Fatalf("failed to create storer: %v", err)
	}

	ctx := logtesting.TestContextWithLogger(t)
	req := &api.StoreRequest[name.Digest, *intoto.Statement]{
		Artifact: ref,
		Payload:  &intoto.Statement{PredicateType: "https://slsa.dev/provenance/v0.2"},
		Bundle: &signing.Bundle{
			Content:   payload,
			Signature: dsseEnv,
			PublicKey: privKey.Public(),
		},
	}

	// Store the same attestation twice.
	if _, err := storer.Store(ctx, req); err != nil {
		t.Fatalf("first Store() failed: %v", err)
	}
	if _, err := storer.Store(ctx, req); err != nil {
		t.Fatalf("second Store() failed: %v", err)
	}

	// Exactly one referrer must exist — no duplicates.
	// Note: we use empty filter; see TestAttestationStorer_Store_SigstoreBundle for rationale.
	idx, err := ociremote.Referrers(ref, "")
	if err != nil {
		t.Fatalf("failed to list referrers: %v", err)
	}
	if got := len(idx.Manifests); got != 1 {
		t.Errorf("expected 1 attestation referrer after dedup, got %d", got)
	}
}

// TestResolvePubKey_ExplicitWinsOverCert verifies that when both an explicit PublicKey
// and a certificate are provided, resolvePubKey returns the explicit key without
// even parsing the certificate (white-box: the function short-circuits on non-nil explicit).
func TestResolvePubKey_ExplicitWinsOverCert(t *testing.T) {
	explicitKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate explicit key: %v", err)
	}

	// certPEM is intentionally not a valid certificate — the function must not reach
	// cert-parsing code when explicit is non-nil.
	fakeCertPEM := []byte("-----BEGIN CERTIFICATE-----\nnot-a-real-cert\n-----END CERTIFICATE-----")

	got, err := resolvePubKey(explicitKey.Public(), fakeCertPEM)
	if err != nil {
		t.Fatalf("resolvePubKey() returned unexpected error: %v", err)
	}
	if got != explicitKey.Public() {
		t.Errorf("resolvePubKey() returned cert-derived key, want explicit key")
	}
}
