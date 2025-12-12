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
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	cbundle "github.com/sigstore/cosign/v2/pkg/cosign/bundle"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/cosign/v2/pkg/types"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/config"
	"knative.dev/pkg/logging"
)

// SimpleStorer stores SimpleSigning payloads in OCI registries.
type SimpleStorer struct {
	// repo configures the repo where data should be stored.
	// If empty, the repo is inferred from the Artifact.
	repo *name.Repository
	// remoteOpts are additional remote options (i.e. auth) to use for client operations.
	remoteOpts []remote.Option
	// format specifies the storage format (legacy, referrers-api, protobuf-bundle)
	format string
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
	logger := logging.FromContext(ctx).With("image", req.Artifact.String())

	// Get or create signed entity
	se, err := ociremote.SignedEntity(req.Artifact, ociremote.WithRemoteOptions(s.remoteOpts...))
	var entityNotFoundError *ociremote.EntityNotFoundError
	if errors.As(err, &entityNotFoundError) {
		se = ociremote.SignedUnknown(req.Artifact)
	} else if err != nil {
		return nil, errors.Wrap(err, "getting signed entity")
	}

	// Determine repository
	repo := req.Artifact.Repository
	if s.repo != nil {
		repo = *s.repo
	}

	// Route to appropriate storage implementation
	switch s.format {
	case config.OCIFormatLegacy, "": // Default to legacy
		return s.storeLegacy(ctx, req, se, repo)
	case config.OCIFormatReferrersAPI:
		return s.storeWithReferrersAPI(ctx, req, se, repo)
	case config.OCIFormatProtobuf:
		return s.storeWithProtobufBundle(ctx, req, se, repo)
	default:
		logger.Warnf("Unknown OCI format %s, defaulting to legacy", s.format)
		return s.storeLegacy(ctx, req, se, repo)
	}
}

func (s *SimpleStorer) storeLegacy(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage], se oci.SignedEntity, repo name.Repository) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)
	logger.Info("Using legacy tag-based signature storage")

	// Create signature
	sigOpts := []static.Option{}
	if req.Bundle.Cert != nil {
		sigOpts = append(sigOpts, static.WithCertChain(req.Bundle.Cert, req.Bundle.Chain))
	}

	b64sig := base64.StdEncoding.EncodeToString(req.Bundle.Signature)
	sig, err := static.NewSignature(req.Bundle.Content, b64sig, sigOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "creating signature")
	}

	newSE, err := mutate.AttachSignatureToEntity(se, sig)
	if err != nil {
		return nil, errors.Wrap(err, "attaching signature to entity")
	}

	// Use traditional WriteSignatures (tag-based)
	if err := ociremote.WriteSignatures(repo, newSE, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing signatures")
	}

	logger.Info("Successfully uploaded signature using legacy format")
	return &api.StoreResponse{}, nil
}

func (s *SimpleStorer) storeWithReferrersAPI(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage], se oci.SignedEntity, repo name.Repository) (*api.StoreResponse, error) {
	_ = repo // repo parameter unused in referrers API - uses req.Artifact directly
	logger := logging.FromContext(ctx)
	logger.Info("Using OCI 1.1 referrers API for signature storage with proper artifact type")

	// Create signature (same as legacy)
	sigOpts := []static.Option{}
	if req.Bundle.Cert != nil {
		sigOpts = append(sigOpts, static.WithCertChain(req.Bundle.Cert, req.Bundle.Chain))
	}

	b64sig := base64.StdEncoding.EncodeToString(req.Bundle.Signature)
	sig, err := static.NewSignature(req.Bundle.Content, b64sig, sigOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "creating signature")
	}

	newSE, err := mutate.AttachSignatureToEntity(se, sig)
	if err != nil {
		return nil, errors.Wrap(err, "attaching signature to entity")
	}

	// Extract signature layers for WriteReferrer
	sigs, err := newSE.Signatures()
	if err != nil {
		return nil, errors.Wrap(err, "getting signatures from entity")
	}

	layers, err := sigs.Layers()
	if err != nil {
		return nil, errors.Wrap(err, "getting signature layers")
	}

	// Create annotations with creation timestamp
	annotations := map[string]string{
		"org.opencontainers.image.created": time.Now().UTC().Format(time.RFC3339),
	}

	// Use WriteReferrer with proper signature artifact type
	// This is equivalent to ociexperimental.ArtifactType("sig") which returns "application/vnd.dev.cosign.artifact.sig.v1+json"
	artifactType := "application/vnd.dev.cosign.artifact.sig.v1+json"
	if err := ociremote.WriteReferrer(req.Artifact, artifactType, layers, annotations, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing signature referrer with proper artifact type")
	}

	logger.Info("Successfully uploaded signature using referrers API with proper artifact type")
	return &api.StoreResponse{}, nil
}

func (s *SimpleStorer) storeWithProtobufBundle(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage], se oci.SignedEntity, repo name.Repository) (*api.StoreResponse, error) {
	_ = se   // unused in bundle format
	_ = repo // unused in bundle format
	logger := logging.FromContext(ctx)
	logger.Info("Using cosign's MakeNewBundle for signature storage")

	// Use public key from StorageOpts (extracted from signer)
	var pubKey crypto.PublicKey
	if req.Bundle.PublicKey != nil {
		pubKey = req.Bundle.PublicKey
		logger.Info("Using public key provided from signer")
	} else if req.Bundle.Cert != nil && len(req.Bundle.Cert) > 0 {
		logger.Info("Extracting public key from certificate for bundle creation")

		// Try to parse as PEM first
		block, _ := pem.Decode(req.Bundle.Cert)
		var certBytes []byte
		if block != nil {
			certBytes = block.Bytes
		} else {
			// Assume DER format if PEM decode fails
			certBytes = req.Bundle.Cert
		}

		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			logger.Warnf("Failed to parse certificate for public key extraction: %v", err)
		} else {
			pubKey = cert.PublicKey
			logger.Info("Successfully extracted public key from certificate")
		}
	}

	if pubKey == nil {
		return nil, errors.New("no public key available: neither from signer nor from certificate")
	}

	// For signatures, create DSSE envelope with signature payload
	// MakeNewBundle expects the DSSE envelope as JSON bytes
	dsseEnvelope := map[string]interface{}{
		"payload":     base64.StdEncoding.EncodeToString(req.Bundle.Content),
		"payloadType": "application/vnd.dev.cosign.simple.signing.v1+json",
		"signatures": []map[string]interface{}{
			{
				"sig": base64.StdEncoding.EncodeToString(req.Bundle.Signature),
			},
		},
	}

	signedPayload, err := json.Marshal(dsseEnvelope)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling DSSE envelope for signature bundle")
	}

	// Use cosign's MakeNewBundle to create proper protobuf bundle
	// Following the pattern from cosign CLI: MakeNewBundle(pubKey, rekorEntry, payload, signedPayload, signerBytes, timestampBytes)
	var rekorEntry *models.LogEntryAnon // nil for x509 static key signing
	var signerBytes []byte              // certificate bytes for verification material
	var timestampBytes []byte           // nil for x509 static key signing

	if req.Bundle.Cert != nil {
		signerBytes = req.Bundle.Cert
	}

	bundleBytes, err := cbundle.MakeNewBundle(pubKey, rekorEntry, req.Bundle.Content, signedPayload, signerBytes, timestampBytes)
	if err != nil {
		return nil, errors.Wrap(err, "creating signature bundle with cosign's MakeNewBundle")
	}

	// Store the bundle using WriteAttestationNewBundleFormat (same function cosign uses for signatures)
	// Use CosignSignPredicateType for signatures (same as cosign CLI)
	if err := ociremote.WriteAttestationNewBundleFormat(req.Artifact, bundleBytes, types.CosignSignPredicateType, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing signature bundle with WriteAttestationNewBundleFormat")
	}

	logger.Info("Successfully uploaded signature using cosign protobuf bundle")
	return &api.StoreResponse{}, nil
}
