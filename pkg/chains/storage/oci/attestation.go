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

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/pkg/errors"
	cbundle "github.com/sigstore/cosign/v2/pkg/cosign/bundle"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/cosign/v2/pkg/types"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/config"
	"knative.dev/pkg/logging"
)

var (
	_ api.Storer[name.Digest, *intoto.Statement] = &AttestationStorer{}
)

// AttestationStorer stores in-toto Attestation payloads in OCI registries.
type AttestationStorer struct {
	// repo configures the repo where data should be stored.
	// If empty, the repo is inferred from the Artifact.
	repo *name.Repository
	// remoteOpts are additional remote options (i.e. auth) to use for client operations.
	remoteOpts []remote.Option
	// format specifies the storage format (legacy, referrers-api, protobuf-bundle)
	format string
}

func NewAttestationStorer(opts ...AttestationStorerOption) (*AttestationStorer, error) {
	s := &AttestationStorer{}
	for _, o := range opts {
		if err := o.applyAttestationStorer(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Store saves the given statement.
func (s *AttestationStorer) Store(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement]) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)

	// Determine repository
	repo := req.Artifact.Repository
	if s.repo != nil {
		repo = *s.repo
	}

	// Get or create signed entity
	se, err := ociremote.SignedEntity(req.Artifact, ociremote.WithRemoteOptions(s.remoteOpts...))
	var entityNotFoundError *ociremote.EntityNotFoundError
	if errors.As(err, &entityNotFoundError) {
		se = ociremote.SignedUnknown(req.Artifact)
	} else if err != nil {
		return nil, errors.Wrap(err, "getting signed entity")
	}

	// Route to appropriate storage implementation based on format
	switch s.format {
	case config.OCIFormatLegacy, "": // Default to legacy
		return s.storeLegacy(ctx, req, se, repo)
	case config.OCIFormatReferrersAPI:
		return s.storeWithReferrersAPI(ctx, req, se, repo)
	case config.OCIFormatProtobuf:
		return s.storeWithProtobufBundle(ctx, req, repo)
	default:
		logger.Warnf("Unknown OCI format %s, defaulting to legacy", s.format)
		return s.storeLegacy(ctx, req, se, repo)
	}
}

// Legacy tag-based storage (current default implementation)
func (s *AttestationStorer) storeLegacy(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement], se oci.SignedEntity, repo name.Repository) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)
	logger.Info("Using legacy tag-based attestation storage")

	// Create attestation with DSSE format
	attOpts := []static.Option{static.WithLayerMediaType(types.DssePayloadType)}
	if req.Bundle.Cert != nil {
		attOpts = append(attOpts, static.WithCertChain(req.Bundle.Cert, req.Bundle.Chain))
	}

	att, err := static.NewAttestation(req.Bundle.Signature, attOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "creating attestation")
	}

	newImage, err := mutate.AttachAttestationToEntity(se, att)
	if err != nil {
		return nil, errors.Wrap(err, "attaching attestation to entity")
	}

	// Use traditional WriteAttestations (tag-based)
	if err := ociremote.WriteAttestations(repo, newImage, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing attestations")
	}

	logger.Infof("Successfully uploaded attestation using legacy format for %s", req.Artifact.String())
	return &api.StoreResponse{}, nil
}

// Referrers API storage with DSSE format
func (s *AttestationStorer) storeWithReferrersAPI(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement], se oci.SignedEntity, repo name.Repository) (*api.StoreResponse, error) {
	_ = repo // repo parameter unused in referrers API - uses req.Artifact directly
	logger := logging.FromContext(ctx)
	logger.Info("Using OCI 1.1 referrers API with DSSE format")

	// Create attestation with DSSE format (same as legacy)
	attOpts := []static.Option{static.WithLayerMediaType(types.DssePayloadType)}
	if req.Bundle.Cert != nil {
		attOpts = append(attOpts, static.WithCertChain(req.Bundle.Cert, req.Bundle.Chain))
	}

	att, err := static.NewAttestation(req.Bundle.Signature, attOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "creating attestation")
	}

	newImage, err := mutate.AttachAttestationToEntity(se, att)
	if err != nil {
		return nil, errors.Wrap(err, "attaching attestation to entity")
	}

	// Use WriteAttestationsReferrer from cosign PR #4357
	if err := ociremote.WriteAttestationsReferrer(req.Artifact, newImage, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing attestations with referrers API")
	}

	logger.Infof("Successfully uploaded attestation using referrers API for %s", req.Artifact.String())
	return &api.StoreResponse{}, nil
}

// Protobuf bundle storage using cosign's MakeNewBundle
func (s *AttestationStorer) storeWithProtobufBundle(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement], repo name.Repository) (*api.StoreResponse, error) {
	_ = repo // repo parameter unused in protobuf bundle - uses req.Artifact directly
	logger := logging.FromContext(ctx)
	logger.Info("Using cosign's MakeNewBundle for attestation storage")

	// Extract predicate type for annotations
	predicateType := req.Payload.PredicateType
	if predicateType == "" {
		return nil, errors.New("PredicateType is required for protobuf bundle format")
	}

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

	// Create DSSE envelope from existing signed data (Chains already has DSSE components)
	// MakeNewBundle expects the DSSE envelope as JSON bytes
	dsseEnvelope := map[string]interface{}{
		"payload":     base64.StdEncoding.EncodeToString(req.Bundle.Content),
		"payloadType": "application/vnd.in-toto+json",
		"signatures": []map[string]interface{}{
			{
				"sig": base64.StdEncoding.EncodeToString(req.Bundle.Signature),
			},
		},
	}

	signedPayload, err := json.Marshal(dsseEnvelope)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling DSSE envelope for bundle")
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
		return nil, errors.Wrap(err, "creating protobuf bundle with cosign's MakeNewBundle")
	}

	// Store the bundle using WriteAttestationNewBundleFormat (same as cosign CLI)
	if err := ociremote.WriteAttestationNewBundleFormat(req.Artifact, bundleBytes, predicateType, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing Sigstore bundle with WriteAttestationNewBundleFormat")
	}

	logger.Infof("Successfully uploaded attestation using Sigstore bundle format for %s", req.Artifact.String())
	return &api.StoreResponse{}, nil
}
