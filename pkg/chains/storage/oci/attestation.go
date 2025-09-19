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
	"encoding/base64"
	"encoding/json"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/pkg/errors"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/cosign/v2/pkg/types"
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

// Protobuf bundle storage (current experimental implementation)
func (s *AttestationStorer) storeWithProtobufBundle(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement], repo name.Repository) (*api.StoreResponse, error) {
	_ = repo // repo parameter unused in protobuf bundle - uses req.Artifact directly
	logger := logging.FromContext(ctx)
	logger.Info("Using protobuf bundle format")

	// Create DSSE envelope
	payload, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling attestation")
	}

	envelope := dsse.Envelope{
		PayloadType: "application/vnd.in-toto+json",
		Payload:     base64.StdEncoding.EncodeToString(payload),
		Signatures: []dsse.Signature{
			{Sig: string(req.Bundle.Signature)},
		},
	}

	bundleBytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling DSSE envelope")
	}

	predicateType := req.Payload.PredicateType

	// Use WriteAttestationNewBundleFormat (current experimental implementation)
	if err := ociremote.WriteAttestationNewBundleFormat(req.Artifact, bundleBytes, predicateType, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing attestation with protobuf bundle")
	}

	logger.Infof("Successfully uploaded attestation using protobuf bundle for %s", req.Artifact.String())
	return &api.StoreResponse{}, nil
}
