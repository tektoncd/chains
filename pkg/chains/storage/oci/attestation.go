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

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/oci"
	"github.com/sigstore/cosign/v2/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	"github.com/sigstore/cosign/v2/pkg/oci/static"
	"github.com/sigstore/cosign/v2/pkg/types"
	"github.com/tektoncd/chains/pkg/artifacts"
	v1 "github.com/tektoncd/chains/pkg/chains/formats/slsa/v1"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"knative.dev/pkg/logging"
)

var (
	_ api.Storer[objects.TektonObject, v1.ProvenanceStatement] = &AttestationStorer[v1.ProvenanceStatement]{}
)

// AttestationStorer stores in-toto Attestation payloads in OCI registries.
type AttestationStorer[T any] struct {
	// repo configures the repo where data should be stored.
	// If empty, the repo is inferred from the Artifact.
	repo *name.Repository
	// remoteOpts are additional remote options (i.e. auth) to use for client operations.
	remoteOpts []remote.Option
}

func NewAttestationStorer[T any](opts ...Option) (*AttestationStorer[T], error) {
	o := &ociOption{}
	for _, f := range opts {
		f(o)
	}
	return &AttestationStorer[T]{
		repo:       o.repo,
		remoteOpts: o.remote,
	}, nil
}

func (s *AttestationStorer[T]) Store(ctx context.Context, req *api.StoreRequest[objects.TektonObject, T]) (*api.StoreResponse, error) {
	log := logging.FromContext(ctx)

	// Create the new attestation for this entity.
	attOpts := []static.Option{static.WithLayerMediaType(types.DssePayloadType)}
	if req.Bundle.Cert != nil {
		attOpts = append(attOpts, static.WithCertChain(req.Bundle.Cert, req.Bundle.Chain))
	}
	att, err := static.NewAttestation(req.Bundle.Signature, attOpts...)
	if err != nil {
		return nil, err
	}

	// Store attestation to all images present in object.
	images, err := artifacts.ExtractOCI(ctx, req.Object)
	if err != nil {
		return nil, err
	}

	var merr error
	for _, img := range images {
		log.Infof("storing attestation in %s", img)
		if err := s.storeImage(ctx, img, att); err != nil {
			merr = multierror.Append(merr, err)
		}
	}
	if merr != nil {
		return nil, merr
	}

	return &api.StoreResponse{}, nil
}

func (s *AttestationStorer[T]) storeImage(ctx context.Context, img name.Digest, att oci.Signature) error {
	logger := logging.FromContext(ctx)
	repo := img.Repository
	if s.repo != nil {
		repo = *s.repo
	}
	se, err := ociremote.SignedEntity(img, ociremote.WithRemoteOptions(s.remoteOpts...))
	if err != nil {
		return errors.Wrap(err, "getting signed image")
	}

	newImage, err := mutate.AttachAttestationToEntity(se, att)
	if err != nil {
		return err
	}

	// Publish the signatures associated with this entity
	if err := ociremote.WriteAttestations(repo, newImage, ociremote.WithRemoteOptions(s.remoteOpts...)); err != nil {
		return err
	}
	logger.Infof("Successfully uploaded attestation for %s", img.String())
	return nil
}
