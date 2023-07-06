package oci

import "github.com/google/go-containerregistry/pkg/name"

// Option provides a config option compatible with all OCI storers.
type Option interface {
	AttestationStorerOption
	SimpleStorerOption
}

// AttestationStorerOption provides a config option compatible with AttestationStorer.
type AttestationStorerOption interface {
	applyAttestationStorer(s *AttestationStorer) error
}

// SimpleStorerOption provides a config option compatible with SimpleStorer.
type SimpleStorerOption interface {
	applySimpleStorer(s *SimpleStorer) error
}

// WithTargetRepository configures the target repository where objects will be stored.
func WithTargetRepository(repo name.Repository) Option {
	return &targetRepoOption{
		repo: repo,
	}
}

type targetRepoOption struct {
	repo name.Repository
}

func (o *targetRepoOption) applyAttestationStorer(s *AttestationStorer) error {
	s.repo = &o.repo
	return nil
}

func (o *targetRepoOption) applySimpleStorer(s *SimpleStorer) error {
	s.repo = &o.repo
	return nil
}
