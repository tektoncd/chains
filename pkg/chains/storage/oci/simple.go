// Copyright 2023 The Tekton Authors
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
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	cbundle "github.com/sigstore/cosign/v2/pkg/cosign/bundle"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/cosign/v2/pkg/types"
	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	protodsse "github.com/sigstore/protobuf-specs/gen/pb-go/dsse"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/config"
	"google.golang.org/protobuf/encoding/protojson"
	"knative.dev/pkg/logging"
)

// SimpleStorer stores SimpleSigning payloads in OCI registries.
type SimpleStorer struct {
	// repo configures the repo where data should be stored.
	// If empty, the repo is inferred from the Artifact.
	repo *name.Repository
	// remoteOpts are additional remote options (i.e. auth) to use for client operations.
	remoteOpts []remote.Option
	// encodingFormat specifies the payload encoding ("dsse" tag-based or "sigstore-bundle" referrers).
	encodingFormat string
}

var (
	_ api.Storer[name.Digest, simple.SimpleContainerImage] = &SimpleStorer{}
)

func NewSimpleStorerFromConfig(opts ...SimpleStorerOption) (*SimpleStorer, error) {
	s := &SimpleStorer{}
	for _, o := range opts {
		if err := o.applySimpleStorer(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *SimpleStorer) Store(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage]) (*api.StoreResponse, error) {
	repo := req.Artifact.Repository
	if s.repo != nil {
		repo = *s.repo
	}

	if s.encodingFormat == config.OCIEncodingFormatSigstoreBundle {
		return s.storeReferrers(ctx, req, repo)
	}

	// Legacy path requires the signed entity; propagate non-404 errors so
	// TLS/auth failures surface immediately rather than at WriteSignatures.
	se, err := ociremote.SignedEntity(req.Artifact, ociremote.WithRemoteOptions(s.remoteOpts...))
	var entityNotFoundError *ociremote.EntityNotFoundError
	if errors.As(err, &entityNotFoundError) {
		se = ociremote.SignedUnknown(req.Artifact, ociremote.WithRemoteOptions(s.remoteOpts...))
	} else if err != nil {
		return nil, errors.Wrap(err, "getting signed entity")
	}
	return s.storeLegacy(ctx, req, se, repo)
}

// storeReferrers writes the signature via the OCI 1.1 Referrers API using the
// Sigstore protobuf-bundle format. When the registry has no native Referrers
// API, cosign/go-containerregistry transparently uses the OCI referrers tag
// schema; either way no .sig tags are created. The signature bundle carries
// CosignSignPredicateType so `cosign verify` can distinguish it from SLSA
// attestation bundles stored by the same API.
func (s *SimpleStorer) storeReferrers(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage], repo name.Repository) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx).With("image", req.Artifact.String())

	if referrersRepoOverrideIgnored(repo, req.Artifact.Repository) {
		logger.Warnf("storage.oci.repository override %q is ignored in sigstore-bundle mode; OCI 1.1 referrers are stored alongside their subject image in %q", repo.String(), req.Artifact.Repository.String())
	}

	return s.storeWithSigstoreBundle(ctx, req)
}

// storeLegacy is the default tag-based signature upload path.
func (s *SimpleStorer) storeLegacy(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage], se oci.SignedEntity, repo name.Repository) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx).With("image", req.Artifact.String())

	sigOpts := []static.Option{}
	if req.Bundle.Cert != nil {
		sigOpts = append(sigOpts, static.WithCertChain(req.Bundle.Cert, req.Bundle.Chain))
	}
	b64sig := base64.StdEncoding.EncodeToString(req.Bundle.Signature)
	sig, err := static.NewSignature(req.Bundle.Content, b64sig, sigOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "creating signature")
	}

	// Skip upload if an identical signature already exists.
	newDigest, err := sig.Digest()
	if err != nil {
		return nil, errors.Wrap(err, "getting new signature digest")
	}
	if existingSigs, err := se.Signatures(); err != nil {
		logger.Debugf("Could not fetch existing signatures, skipping dedup check: %v", err)
	} else if layers, err := existingSigs.Get(); err != nil {
		logger.Debugf("Could not get signature layers, skipping dedup check: %v", err)
	} else {
		for _, l := range layers {
			if d, err := l.Digest(); err == nil && d == newDigest {
				logger.Infof("Signature with digest %s already exists, skipping", newDigest)
				return &api.StoreResponse{}, nil
			}
		}
	}

	newSE, err := mutate.AttachSignatureToEntity(se, sig)
	if err != nil {
		return nil, errors.Wrap(err, "attaching signature to entity")
	}
	if err := ociremote.WriteSignatures(repo, newSE, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing signatures")
	}
	logger.Info("Successfully uploaded signature using legacy format")
	return &api.StoreResponse{}, nil
}

// storeWithSigstoreBundle uploads the image signature as a Sigstore protobuf
// bundle (v0.3) referrer. The bundle uses a DsseEnvelope wrapping the
// SimpleSigning payload — the same format cosign 3.x `cosign sign` produces.
// WriteAttestationNewBundleFormat hardcodes the "dev.sigstore.bundle.content":
// "dsse-envelope" annotation, so the bundle content must be a DsseEnvelope for
// cosign verify to accept it. The predicateType annotation is set to
// CosignSignPredicateType so cosign can distinguish signature bundles from SLSA
// attestation bundles stored alongside them.
func (s *SimpleStorer) storeWithSigstoreBundle(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage]) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx).With("image", req.Artifact.String())
	logger.Info("Using sigstore bundle format for signature storage")

	bundleBytes, err := makeSigBundleBytes(req.Bundle.PublicKey, req.Bundle.Cert, req.Bundle.Content, req.Bundle.Signature, req.Bundle.RekorEntry)
	if err != nil {
		return nil, errors.Wrap(err, "creating signature bundle")
	}

	// Dedup scan: O(referrers × layers) serial registry calls.
	// Acceptable for typical bundle counts (1–3 per artifact); the scan
	// short-circuits on the first digest match. Optimize to parallel
	// fetches if referrer counts grow large in practice.
	//
	// Dedup: skip if an identical bundle layer already exists as a referrer.
	// static.NewLayer (used by WriteAttestationNewBundleFormat) stores bytes
	// uncompressed, so sha256(bundleBytes) == the stored layer's Digest.
	bundleHash := sha256.Sum256(bundleBytes)
	newLayerDigest := v1.Hash{Algorithm: "sha256", Hex: hex.EncodeToString(bundleHash[:])}
	// Use empty artifactType filter to list ALL referrers; the mock registry (and some real
	// registries) derive the descriptor's ArtifactType from config.MediaType rather than
	// the manifest's top-level artifactType field, so filtering by bundleArtifactType would
	// return 0 results even when an identical bundle already exists. Dedup is based on the
	// layer content digest, so fetching all referrers is safe and correct.
	if idx, listErr := ociremote.Referrers(req.Artifact, "", ociremote.WithRemoteOptions(s.remoteOpts...)); listErr != nil {
		logger.Debugf("Could not list referrers for dedup check, will attempt write: %v", listErr)
	} else {
		for _, desc := range idx.Manifests {
			refRef, nameErr := name.NewDigest(req.Artifact.Repository.Name() + "@" + desc.Digest.String())
			if nameErr != nil {
				continue
			}
			refImg, imgErr := remote.Image(refRef, s.remoteOpts...)
			if imgErr != nil {
				continue
			}
			layers, layerErr := refImg.Layers()
			if layerErr != nil {
				continue
			}
			for _, l := range layers {
				if d, dErr := l.Digest(); dErr == nil && d == newLayerDigest {
					logger.Infof("Identical signature bundle with layer digest %s already exists as a referrer, skipping", newLayerDigest)
					return &api.StoreResponse{}, nil
				}
			}
		}
	}

	if err := ociremote.WriteAttestationNewBundleFormat(req.Artifact, bundleBytes, types.CosignSignPredicateType, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing signature bundle referrer")
	}
	logger.Info("Successfully uploaded signature using sigstore bundle format")
	return &api.StoreResponse{}, nil
}

// makeSigBundleBytes constructs a Sigstore protobuf bundle (v0.3) JSON for an
// OCI image signature. The format exactly matches cosign 3.x `cosign sign`:
// the bundle content is a DsseEnvelope with payloadType set to
// SimpleSigningMediaType. WriteAttestationNewBundleFormat hardcodes the
// "dev.sigstore.bundle.content": "dsse-envelope" annotation, so the bundle
// content MUST be a DsseEnvelope — using MessageSignature here would make the
// annotation inconsistent and cause cosign verify to fail.
//
// We build the bundle directly (rather than calling MakeNewBundle) to avoid a
// nil-pubkey crash in the test path: MakeNewBundle calls x509.MarshalPKIXPublicKey
// unconditionally when no cert is provided, which panics with a nil key.
func makeSigBundleBytes(pubKey interface{}, certPEM []byte, payload []byte, rawSig []byte, rekorEntry *models.LogEntryAnon) ([]byte, error) {
	var hint string
	var rawCert []byte

	if len(certPEM) > 0 {
		block, _ := pem.Decode(certPEM)
		der := certPEM
		if block != nil {
			der = block.Bytes
		}
		if cert, err := x509.ParseCertificate(der); err == nil {
			rawCert = cert.Raw
		}
	}
	if rawCert == nil && pubKey != nil {
		if pkixKey, err := x509.MarshalPKIXPublicKey(pubKey); err == nil {
			h := sha256.Sum256(pkixKey)
			hint = base64.StdEncoding.EncodeToString(h[:])
		}
	}

	bundle, err := cbundle.MakeProtobufBundle(hint, rawCert, rekorEntry, nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating protobuf bundle")
	}

	bundle.Content = &protobundle.Bundle_DsseEnvelope{
		DsseEnvelope: &protodsse.Envelope{
			Payload:     payload,
			PayloadType: types.SimpleSigningMediaType,
			Signatures:  []*protodsse.Signature{{Sig: rawSig}},
		},
	}

	out, err := protojson.Marshal(bundle)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling bundle")
	}
	return out, nil
}
