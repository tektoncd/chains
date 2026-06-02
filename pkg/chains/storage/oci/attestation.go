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
	// distributionMethod specifies how artifacts are attached ("legacy" tag-based or "referrers-api").
	distributionMethod string
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

// Store saves the given statement using the configured OCI storage format.
func (s *AttestationStorer) Store(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement]) (*api.StoreResponse, error) {
	repo := req.Artifact.Repository
	if s.repo != nil {
		repo = *s.repo
	}
	se, err := ociremote.SignedEntity(req.Artifact, ociremote.WithRemoteOptions(s.remoteOpts...))
	var entityNotFoundError *ociremote.EntityNotFoundError
	if errors.As(err, &entityNotFoundError) {
		se = ociremote.SignedUnknown(req.Artifact, ociremote.WithRemoteOptions(s.remoteOpts...))
	} else if err != nil {
		return nil, errors.Wrap(err, "getting signed entity")
	}

	switch s.distributionMethod {
	case config.OCIDistributionReferrersAPI:
		return s.storeReferrers(ctx, req, repo)
	default: // OCIDistributionLegacy or empty
		return s.storeLegacy(ctx, req, se, repo)
	}
}

// storeReferrers writes the attestation via the OCI 1.1 Referrers API using the
// Sigstore protobuf-bundle format. When the registry has no native Referrers API,
// cosign/go-containerregistry transparently uses the OCI referrers tag schema;
// either way no .att tags are created.
func (s *AttestationStorer) storeReferrers(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement], repo name.Repository) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)

	if referrersRepoOverrideIgnored(repo, req.Artifact.Repository) {
		logger.Warnf("storage.oci.repository override %q is ignored in referrers-api mode; OCI 1.1 referrers are stored alongside their subject image in %q", repo.String(), req.Artifact.Repository.String())
	}

	return s.storeWithProtobufBundle(ctx, req)
}

// storeLegacy is the default tag-based attestation upload path.
func (s *AttestationStorer) storeLegacy(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement], se oci.SignedEntity, repo name.Repository) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)

	attOpts := []static.Option{static.WithLayerMediaType(types.DssePayloadType)}
	if req.Bundle.Cert != nil {
		attOpts = append(attOpts, static.WithCertChain(req.Bundle.Cert, req.Bundle.Chain))
	}
	att, err := static.NewAttestation(req.Bundle.Signature, attOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "creating attestation")
	}

	// Skip upload if identical attestation already exists.
	newDigest, err := att.Digest()
	if err != nil {
		return nil, errors.Wrap(err, "getting new attestation digest")
	}
	if existingAtts, err := se.Attestations(); err != nil {
		logger.Debugf("Could not fetch existing attestations for %s, skipping dedup check: %v", req.Artifact.String(), err)
	} else if layers, err := existingAtts.Get(); err != nil {
		logger.Debugf("Could not get attestation layers for %s, skipping dedup check: %v", req.Artifact.String(), err)
	} else {
		for _, l := range layers {
			if d, err := l.Digest(); err == nil && d == newDigest {
				logger.Infof("Attestation with digest %s already exists for %s, skipping", newDigest, req.Artifact.String())
				return &api.StoreResponse{}, nil
			}
		}
	}

	newImage, err := mutate.AttachAttestationToEntity(se, att)
	if err != nil {
		return nil, errors.Wrap(err, "attaching attestation to entity")
	}
	if err := ociremote.WriteAttestations(repo, newImage, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing attestations")
	}
	logger.Infof("Successfully uploaded attestation using legacy format for %s", req.Artifact.String())
	return &api.StoreResponse{}, nil
}

// storeWithProtobufBundle uploads attestations using cosign's protobuf bundle
// format over the OCI 1.1 Referrers API.
func (s *AttestationStorer) storeWithProtobufBundle(ctx context.Context, req *api.StoreRequest[name.Digest, *intoto.Statement]) (*api.StoreResponse, error) {
	logger := logging.FromContext(ctx)
	logger.Infof("Using protobuf bundle format for attestation storage (%s)", req.Artifact.String())

	predicateType := req.Payload.PredicateType
	if predicateType == "" {
		return nil, errors.New("PredicateType is required for protobuf-bundle format")
	}

	pubKey, err := resolvePubKey(req.Bundle.PublicKey, req.Bundle.Cert)
	if err != nil {
		return nil, err
	}

	// req.Bundle.Signature is already a complete DSSE envelope (JSON) produced by
	// the wrapped signer: its signature is computed over the DSSE PAE. MakeNewBundle
	// expects exactly this envelope JSON as its `sig` argument - it extracts the
	// PayloadType and the raw signature from it. Re-wrapping it in another envelope
	// would place the whole envelope JSON into the inner sig field, producing a
	// bundle whose signature does not verify ("Found: 0").
	var rekorEntry *models.LogEntryAnon
	var timestampBytes []byte
	var signerBytes []byte
	if req.Bundle.Cert != nil {
		signerBytes = req.Bundle.Cert
	}

	bundleBytes, err := cbundle.MakeNewBundle(pubKey, rekorEntry, req.Bundle.Content, req.Bundle.Signature, signerBytes, timestampBytes)
	if err != nil {
		return nil, errors.Wrap(err, "creating protobuf bundle")
	}
	if err := ociremote.WriteAttestationNewBundleFormat(req.Artifact, bundleBytes, predicateType, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return nil, errors.Wrap(err, "writing protobuf bundle attestation")
	}
	logger.Infof("Successfully uploaded attestation using protobuf bundle format for %s", req.Artifact.String())
	return &api.StoreResponse{}, nil
}

// resolvePubKey returns the public key from the Bundle's explicit PublicKey field,
// or falls back to extracting it from the signer certificate bytes.
func resolvePubKey(explicit crypto.PublicKey, certPEM []byte) (crypto.PublicKey, error) {
	if explicit != nil {
		return explicit, nil
	}
	if len(certPEM) == 0 {
		return nil, errors.New("no public key available: neither from signer nor from certificate")
	}
	block, _ := pem.Decode(certPEM)
	var certBytes []byte
	if block != nil {
		certBytes = block.Bytes
	} else {
		certBytes = certPEM // assume DER
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, errors.Wrap(err, "parsing certificate for public key extraction")
	}
	return cert.PublicKey, nil
}
