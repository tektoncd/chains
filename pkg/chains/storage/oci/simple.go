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

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
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
	logger.Info("Using OCI 1.1 referrers API for signature storage")

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

	// Use WriteSignaturesExperimentalOCI (referrers API)
	if err := ociremote.WriteSignaturesExperimentalOCI(req.Artifact, newSE, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing signatures with referrers API")
	}

	logger.Info("Successfully uploaded signature using referrers API")
	return &api.StoreResponse{}, nil
}

func (s *SimpleStorer) storeWithProtobufBundle(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage], se oci.SignedEntity, repo name.Repository) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)
	logger.Info("Using protobuf bundle format for signature storage")

	// Create signature bundle in protobuf format
	// Note: This uses referrers API as the storage mechanism but with protobuf serialization
	// TODO: Implement proper protobuf bundle serialization for signatures
	// This should serialize the signature as a protobuf bundle similar to how
	// attestations are handled in storeWithProtobufBundle, then store via referrers API
	// For now, use referrers API with standard signature format
	return s.storeWithReferrersAPI(ctx, req, se, repo)
}
