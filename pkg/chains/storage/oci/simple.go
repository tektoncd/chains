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
	// distributionMethod specifies how artifacts are attached ("legacy" tag-based or "referrers-api").
	distributionMethod string
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
	se, err := ociremote.SignedEntity(req.Artifact, ociremote.WithRemoteOptions(s.remoteOpts...))
	var entityNotFoundError *ociremote.EntityNotFoundError
	if errors.As(err, &entityNotFoundError) {
		se = ociremote.SignedUnknown(req.Artifact, ociremote.WithRemoteOptions(s.remoteOpts...))
	} else if err != nil {
		return nil, errors.Wrap(err, "getting signed entity")
	}

	repo := req.Artifact.Repository
	if s.repo != nil {
		repo = *s.repo
	}

	if s.distributionMethod == config.OCIDistributionReferrersAPI {
		return s.storeReferrers(ctx, req, se, repo)
	}
	return s.storeLegacy(ctx, req, se, repo)
}

// storeReferrers writes the signature via the OCI 1.1 Referrers API. When the
// registry has no native Referrers API, cosign/go-containerregistry transparently
// uses the OCI referrers tag schema; either way no .sig tags are created.
func (s *SimpleStorer) storeReferrers(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage], se oci.SignedEntity, repo name.Repository) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx).With("image", req.Artifact.String())

	if referrersRepoOverrideIgnored(repo, req.Artifact.Repository) {
		logger.Warnf("storage.oci.repository override %q is ignored in referrers-api mode; OCI 1.1 referrers are stored alongside their subject image in %q", repo.String(), req.Artifact.Repository.String())
	}

	// Image signatures are always stored via cosign's native signature referrer.
	// A cosign image signature is a plain signature over the simplesigning payload,
	// not a DSSE envelope (whose signature must be over the DSSE PAE), so it cannot
	// be represented as a verifiable DSSE-envelope bundle. This mirrors upstream
	// cosign, where image signatures use the standard signature format while
	// attestations may use the protobuf bundle format.
	return s.storeWithReferrersAPI(ctx, req, se)
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

// storeWithReferrersAPI uploads signatures using the OCI 1.1 Referrers API.
func (s *SimpleStorer) storeWithReferrersAPI(ctx context.Context, req *api.StoreRequest[name.Digest, simple.SimpleContainerImage], se oci.SignedEntity) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx).With("image", req.Artifact.String())
	logger.Info("Using OCI 1.1 referrers API for signature storage")

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
	// WriteSignaturesExperimentalOCI publishes the signature as an OCI 1.1 referrer
	// using cosign's signature-native writer. It sets config.mediaType to
	// application/vnd.dev.cosign.artifact.sig.v1+json, populates the subject, and
	// writes SimpleSigning layers — exactly the manifest shape that `cosign verify`
	// reverse-discovers in referrers mode.
	if err := ociremote.WriteSignaturesExperimentalOCI(req.Artifact, newSE, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing signature referrer")
	}
	logger.Info("Successfully uploaded signature using referrers API")
	return &api.StoreResponse{}, nil
}
