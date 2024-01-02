package attestors

import (
	"bytes"
	"context"
	"encoding"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	v1 "github.com/tektoncd/chains/pkg/chains/formats/slsa/v1"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/chains/storage/oci"
	"github.com/tektoncd/chains/pkg/config"
)

type AttestorHandler[Input any] interface {
	Attest(context.Context, objects.TektonObject, Input) error
}

type Attestor[Input any, Output encoding.BinaryMarshaler] struct {
	payloader formats.Formatter[Input, Output]
	signer    signing.Signer
	storer    api.Storer[Input, Output]
}

// Handler takes an input object -> creates, signs, and stores its attestation.
func (a *Attestor[Input, Output]) Attest(ctx context.Context, obj objects.TektonObject, in Input) (*api.StoreResponse, error) {
	out, err := a.payloader.FormatPayload(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("error creating attestation payload: %w", err)
	}

	b, err := out.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %w", err)
	}

	sig, err := a.signer.SignMessage(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("error signing payload: %w", err)
	}
	req := &api.StoreRequest[Input, Output]{
		Object:   obj,
		Artifact: in,
		Payload:  out,
		Bundle: &signing.Bundle{
			Content:   b,
			Signature: sig,
			Cert:      []byte(a.signer.Cert()),
			Chain:     []byte(a.signer.Chain()),
		},
	}
	return a.storer.Store(ctx, req)
}

func NewContainerSigner(ctx context.Context, cfg config.Config) (*Attestor[name.Digest, simple.SimpleContainerImage], error) {
	signer, err := chains.NewSignerFromConfig(ctx, "", cfg)
	if err != nil {
		return nil, err
	}

	var opts []oci.Option
	if repo := cfg.Storage.OCI.Repository; repo != "" {
		r, err := name.NewRepository(repo)
		if err != nil {
			return nil, fmt.Errorf("error parsing OCI repo name: %w", err)
		}
		opts = append(opts, oci.WithTargetRepository(r))
	}

	storer, err := oci.NewSimpleStorer(opts...)
	if err != nil {
		return nil, err
	}

	return &Attestor[name.Digest, simple.SimpleContainerImage]{
		payloader: simple.SimpleSigningPayloader{},
		signer:    signer,
		storer:    storer,
	}, nil
}

func NewProvenanceSigner(ctx context.Context, cfg config.Config) (*Attestor[objects.TektonObject, *v1.ProvenanceStatement], error) {
	signer, err := chains.NewSignerFromConfig(ctx, "", cfg)
	if err != nil {
		return nil, err
	}
	wrapped, err := signing.Wrap(signer)
	if err != nil {
		return nil, err
	}

	var opts []oci.Option
	if repo := cfg.Storage.OCI.Repository; repo != "" {
		r, err := name.NewRepository(repo)
		if err != nil {
			return nil, fmt.Errorf("error parsing OCI repo name: %w", err)
		}
		opts = append(opts, oci.WithTargetRepository(r))
	}
	storer, err := oci.NewAttestationStorer[*v1.ProvenanceStatement](opts...)
	if err != nil {
		return nil, err
	}

	return &Attestor[objects.TektonObject, *v1.ProvenanceStatement]{
		payloader: v1.NewPayloaderFromConfig(cfg),
		signer:    wrapped,
		// TODO: add support for other storage options.
		storer: storer,
	}, nil
}
