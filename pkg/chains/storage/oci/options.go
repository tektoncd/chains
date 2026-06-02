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

// WithDistributionMethod configures where and how artifacts are attached to images in the registry.
//
// Supported values are the OCIDistribution* constants in pkg/config:
//   - OCIDistributionLegacy      (default) – tag-based storage
//   - OCIDistributionReferrersAPI          – OCI 1.1 Referrers API
//
//nolint:ireturn // returning interface is the intended pattern here
func WithDistributionMethod(method string) Option {
	return &distributionMethodOption{method: method}
}

type distributionMethodOption struct {
	method string
}

func (o *distributionMethodOption) applyAttestationStorer(s *AttestationStorer) error {
	s.distributionMethod = o.method
	return nil
}

func (o *distributionMethodOption) applySimpleStorer(s *SimpleStorer) error {
	s.distributionMethod = o.method
	return nil
}

// referrersRepoOverrideIgnored reports whether a configured repository override
// would be silently dropped for an OCI 1.1 referrer write. Referrers must be
// colocated with their subject image (the referrer manifest references the
// subject by digest within the same repository), so a storage.oci.repository
// override cannot redirect them to a different repository. The override only
// applies to the legacy tag-based storage path.
func referrersRepoOverrideIgnored(override, artifactRepo name.Repository) bool {
	return override.String() != artifactRepo.String()
}
