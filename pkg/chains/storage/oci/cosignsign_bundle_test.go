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
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	intoto "github.com/in-toto/attestation/go/v1"
	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/cosignsign"
	"github.com/tektoncd/chains/pkg/chains/objects"
	chainssigning "github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/config"
	remotetest "github.com/tektoncd/pipeline/test"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/client-go/kubernetes"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	bundleContentAnnotation       = "dev.sigstore.bundle.content"
	bundlePredicateTypeAnnotation = "dev.sigstore.bundle.predicateType"
	cosignSignPredicateType       = "https://sigstore.dev/cosign/sign/v1"
)

// testSigner adapts a sigstore signer to the chains signing.Signer interface.
type testSigner struct {
	signature.SignerVerifier
}

func (testSigner) Type() string  { return "x509" }
func (testSigner) Cert() string  { return "" }
func (testSigner) Chain() string { return "" }

type cosignSignFixture struct {
	registry string
	ref      name.Digest
	pub      crypto.PublicKey
	// rawPayload is the in-toto Statement that was signed; envelope is the DSSE envelope JSON.
	rawPayload []byte
	envelope   []byte
}

// newCosignSignFixture pushes an image to an in-memory registry with referrers
// support and signs the cosign-sign/v1 payload with the real wrapped DSSE signer.
func newCosignSignFixture(t *testing.T) *cosignSignFixture {
	t.Helper()

	s := httptest.NewServer(registry.New(registry.WithReferrersSupport(true)))
	t.Cleanup(s.Close)
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}

	imgRefStr, err := remotetest.CreateImage(u.Host+"/test/cosign-sign-img", tr)
	if err != nil {
		t.Fatalf("CreateImage: %v", err)
	}
	ref, err := name.NewDigest(imgRefStr)
	if err != nil {
		t.Fatalf("name.NewDigest: %v", err)
	}

	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey: %v", err)
	}
	sv, err := signature.LoadECDSASignerVerifier(privKey, crypto.SHA256)
	if err != nil {
		t.Fatalf("LoadECDSASignerVerifier: %v", err)
	}
	wrapped, err := chainssigning.Wrap(testSigner{sv})
	if err != nil {
		t.Fatalf("signing.Wrap: %v", err)
	}

	payloader, err := cosignsign.NewFormatter(config.Config{})
	if err != nil {
		t.Fatalf("NewFormatter: %v", err)
	}
	if !payloader.Wrap() {
		t.Fatal("cosign-sign/v1 payloader must request the wrapped DSSE signer")
	}
	stmt, err := payloader.CreatePayload(context.Background(), ref)
	if err != nil {
		t.Fatalf("CreatePayload: %v", err)
	}
	// Same marshalling as pkg/chains/signing.go getRawPayload for in-toto statements.
	rawPayload, err := protojson.Marshal(stmt.(*intoto.Statement))
	if err != nil {
		t.Fatalf("protojson.Marshal: %v", err)
	}
	envelope, err := wrapped.SignMessage(strings.NewReader(string(rawPayload)))
	if err != nil {
		t.Fatalf("SignMessage: %v", err)
	}

	return &cosignSignFixture{
		registry:   u.Host,
		ref:        ref,
		pub:        privKey.Public(),
		rawPayload: rawPayload,
		envelope:   envelope,
	}
}

func (f *cosignSignFixture) store(t *testing.T, rekorEntry *models.LogEntryAnon) {
	t.Helper()
	b := &Backend{
		cfg: config.Config{
			Storage: config.StorageConfigs{
				OCI: config.OCIStorageConfig{EncodingFormat: config.OCIEncodingFormatSigstoreBundle},
			},
		},
		getAuthenticator: func(context.Context, objects.TektonObject, kubernetes.Interface) (remote.Option, error) {
			return remote.WithAuthFromKeychain(authn.DefaultKeychain), nil
		},
	}
	ctx := logtesting.TestContextWithLogger(t)
	if err := b.StorePayload(ctx, objects.NewTaskRunObjectV1(tr), f.rawPayload, string(f.envelope), config.StorageOpts{
		PayloadFormat: formats.PayloadTypeCosignSignV1,
		PublicKey:     f.pub,
		RekorEntry:    rekorEntry,
	}); err != nil {
		t.Fatalf("StorePayload: %v", err)
	}
}

// fetchBundle returns the single referrer's manifest annotations and bundle layer bytes.
func (f *cosignSignFixture) fetchBundle(t *testing.T) (map[string]string, []byte) {
	t.Helper()
	// Empty artifactType filter: the in-memory registry does not index the manifest artifactType.
	idx, err := remote.Referrers(f.ref.Context().Digest(f.ref.DigestStr()))
	if err != nil {
		t.Fatalf("remote.Referrers: %v", err)
	}
	im, err := idx.IndexManifest()
	if err != nil {
		t.Fatalf("IndexManifest: %v", err)
	}
	if len(im.Manifests) != 1 {
		t.Fatalf("expected exactly 1 referrer, got %d", len(im.Manifests))
	}

	referrerRef := f.ref.Context().Digest(im.Manifests[0].Digest.String())
	desc, err := remote.Get(referrerRef)
	if err != nil {
		t.Fatalf("remote.Get(referrer): %v", err)
	}
	var m v1.Manifest
	if err := json.Unmarshal(desc.Manifest, &m); err != nil {
		t.Fatalf("unmarshal referrer manifest: %v", err)
	}
	if m.Subject == nil || m.Subject.Digest.String() != f.ref.DigestStr() {
		t.Fatalf("referrer subject = %v, want image digest %s", m.Subject, f.ref.DigestStr())
	}

	img, err := desc.Image()
	if err != nil {
		t.Fatalf("referrer Image(): %v", err)
	}
	layers, err := img.Layers()
	if err != nil || len(layers) != 1 {
		t.Fatalf("expected 1 bundle layer, got %d (err: %v)", len(layers), err)
	}
	rc, err := layers[0].Compressed()
	if err != nil {
		t.Fatalf("layer.Compressed: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading bundle layer: %v", err)
	}
	return m.Annotations, data
}

func (f *cosignSignFixture) verifier(t *testing.T) *verify.Verifier {
	t.Helper()
	sv, err := signature.LoadVerifier(f.pub, crypto.SHA256)
	if err != nil {
		t.Fatalf("LoadVerifier: %v", err)
	}
	tm := root.NewTrustedPublicKeyMaterial(func(string) (root.TimeConstrainedVerifier, error) {
		return root.NewExpiringKey(sv, time.Time{}, time.Time{}), nil
	})
	v, err := verify.NewVerifier(tm, verify.WithNoObserverTimestamps())
	if err != nil {
		t.Fatalf("verify.NewVerifier: %v", err)
	}
	return v
}

func (f *cosignSignFixture) policy(t *testing.T, digest string) verify.PolicyBuilder {
	t.Helper()
	algo, hexDigest, _ := strings.Cut(digest, ":")
	raw, err := hex.DecodeString(hexDigest)
	if err != nil {
		t.Fatalf("decoding digest: %v", err)
	}
	return verify.NewPolicy(verify.WithArtifactDigest(algo, raw), verify.WithKey())
}

func TestCosignSignBundle_VerifiableWithSigstoreGo(t *testing.T) {
	f := newCosignSignFixture(t)
	f.store(t, nil)
	annotations, bundleBytes := f.fetchBundle(t)
	t.Logf("sigstore bundle written for %s:\n%s", f.ref, bundleBytes)

	if got := annotations[bundleContentAnnotation]; got != "dsse-envelope" {
		t.Errorf("%s = %q, want dsse-envelope", bundleContentAnnotation, got)
	}
	if got := annotations[bundlePredicateTypeAnnotation]; got != cosignSignPredicateType {
		t.Errorf("%s = %q, want %s", bundlePredicateTypeAnnotation, got, cosignSignPredicateType)
	}

	b := &bundle.Bundle{}
	if err := b.UnmarshalJSON(bundleBytes); err != nil {
		t.Fatalf("parsing bundle with sigstore-go: %v", err)
	}
	env := b.GetDsseEnvelope()
	if env == nil {
		t.Fatal("bundle content is not a dsseEnvelope")
	}
	if env.PayloadType != "application/vnd.in-toto+json" {
		t.Errorf("payloadType = %q, want application/vnd.in-toto+json", env.PayloadType)
	}
	var stmt intoto.Statement
	if err := protojson.Unmarshal(env.Payload, &stmt); err != nil {
		t.Fatalf("payload is not an in-toto statement: %v", err)
	}
	if stmt.PredicateType != cosignSignPredicateType {
		t.Errorf("predicateType = %q, want %s", stmt.PredicateType, cosignSignPredicateType)
	}

	v := f.verifier(t)
	policy := f.policy(t, f.ref.DigestStr())
	if _, err := v.Verify(b, policy); err != nil {
		t.Fatalf("sigstore-go failed to verify the image signature bundle: %v", err)
	}

	t.Run("wrong image digest", func(t *testing.T) {
		wrong := "sha256:" + strings.Repeat("0", 64)
		if _, err := v.Verify(b, f.policy(t, wrong)); err == nil {
			t.Error("verification must fail for a different image digest")
		}
	})

	t.Run("tampered payload", func(t *testing.T) {
		tampered := &protobundle.Bundle{}
		if err := protojson.Unmarshal(bundleBytes, tampered); err != nil {
			t.Fatalf("protojson.Unmarshal: %v", err)
		}
		dsseEnv := tampered.GetDsseEnvelope()
		dsseEnv.Payload = []byte(strings.Replace(string(dsseEnv.Payload), cosignSignPredicateType, "https://example.com/other", 1))
		if string(dsseEnv.Payload) == string(env.Payload) {
			t.Fatal("test bug: payload was not modified")
		}
		tb, err := bundle.NewBundle(tampered)
		if err != nil {
			t.Fatalf("bundle.NewBundle: %v", err)
		}
		if _, err := v.Verify(tb, policy); err == nil {
			t.Error("verification must fail when the payload is tampered with")
		}
	})
}

func TestCosignSignBundle_DSSERekorEntry(t *testing.T) {
	f := newCosignSignFixture(t)

	pubPEM, err := cryptoutils.MarshalPublicKeyToPEM(f.pub)
	if err != nil {
		t.Fatalf("MarshalPublicKeyToPEM: %v", err)
	}
	var env struct {
		Signatures []struct {
			Sig string `json:"sig"`
		} `json:"signatures"`
	}
	if err := json.Unmarshal(f.envelope, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	payloadHash := sha256.Sum256(f.rawPayload)
	envelopeHash := sha256.Sum256(f.envelope)
	body := fmt.Sprintf(`{"apiVersion":"0.0.1","kind":"dsse","spec":{`+
		`"envelopeHash":{"algorithm":"sha256","value":"%x"},`+
		`"payloadHash":{"algorithm":"sha256","value":"%x"},`+
		`"signatures":[{"signature":"%s","verifier":"%s"}]}}`,
		envelopeHash, payloadHash, env.Signatures[0].Sig, base64.StdEncoding.EncodeToString(pubPEM))

	entry := &models.LogEntryAnon{
		Body:           base64.StdEncoding.EncodeToString([]byte(body)),
		IntegratedTime: ptr(int64(1700000000)),
		LogID:          ptr(hex.EncodeToString(payloadHash[:])),
		LogIndex:       ptr(int64(42)),
		Verification: &models.LogEntryAnonVerification{
			SignedEntryTimestamp: []byte("set"),
			InclusionProof: &models.InclusionProof{
				Checkpoint: ptr("rekor.example - 1\n1\nroot\n"),
				Hashes:     []string{hex.EncodeToString(envelopeHash[:])},
				LogIndex:   ptr(int64(7)),
				RootHash:   ptr(hex.EncodeToString(payloadHash[:])),
				TreeSize:   ptr(int64(8)),
			},
		},
	}

	f.store(t, entry)
	_, bundleBytes := f.fetchBundle(t)

	var got struct {
		VerificationMaterial struct {
			TlogEntries []struct {
				KindVersion struct {
					Kind    string `json:"kind"`
					Version string `json:"version"`
				} `json:"kindVersion"`
			} `json:"tlogEntries"`
		} `json:"verificationMaterial"`
	}
	if err := json.Unmarshal(bundleBytes, &got); err != nil {
		t.Fatalf("unmarshal bundle: %v", err)
	}
	if len(got.VerificationMaterial.TlogEntries) != 1 {
		t.Fatalf("expected 1 tlog entry, got %d", len(got.VerificationMaterial.TlogEntries))
	}
	if kind := got.VerificationMaterial.TlogEntries[0].KindVersion.Kind; kind != "dsse" {
		t.Errorf("tlogEntries[0].kindVersion.kind = %q, want dsse", kind)
	}
}

func ptr[T any](v T) *T { return &v }

func TestCosignSignV1_PayloadFormatInIntotoSet(t *testing.T) {
	if _, ok := formats.IntotoAttestationSet[formats.PayloadTypeCosignSignV1]; !ok {
		t.Error("cosign-sign/v1 must be in IntotoAttestationSet so Rekor gets a dsse entry and storage uses the attestation path")
	}
}
