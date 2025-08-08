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
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

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

// WithReferrersAPI configures the storers to use OCI 1.1 referrers API.
func WithReferrersAPI(enabled bool) Option {
	return &referrersAPIOption{
		enabled: enabled,
	}
}

type referrersAPIOption struct {
	enabled bool
}

func (o *referrersAPIOption) applyAttestationStorer(s *AttestationStorer) error {
	if o.enabled {
		// Enable cosign experimental mode to activate OCI 1.1 referrers API
		os.Setenv("COSIGN_EXPERIMENTAL", "1")
		s.remoteOpts = append(s.remoteOpts, remote.WithUserAgent("chains/referrers-api"))
	}
	return nil
}

func (o *referrersAPIOption) applySimpleStorer(s *SimpleStorer) error {
	if o.enabled {
		// Enable cosign experimental mode to activate OCI 1.1 referrers API
		os.Setenv("COSIGN_EXPERIMENTAL", "1")
		s.remoteOpts = append(s.remoteOpts, remote.WithUserAgent("chains/referrers-api"))
	}
	return nil
}
