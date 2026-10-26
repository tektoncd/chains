/*
Copyright 2026 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package cosignsign creates the in-toto Statement that `cosign sign` (v3, bundle
// mode) signs for a container image.
package cosignsign

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/sigstore/cosign/v2/pkg/types"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"
	"google.golang.org/protobuf/types/known/structpb"
)

// PayloadTypeCosignSignV1 is the payload type registered by this package.
const PayloadTypeCosignSignV1 = formats.PayloadTypeCosignSignV1

func init() {
	formats.RegisterPayloader(PayloadTypeCosignSignV1, NewFormatter)
}

// CosignSign creates the cosign/sign/v1 in-toto Statement for an image digest.
type CosignSign struct{}

// CreatePayload implements the Payloader interface. It accepts a name.Digest.
func (c *CosignSign) CreatePayload(_ context.Context, obj interface{}) (interface{}, error) {
	digest, ok := obj.(name.Digest)
	if !ok {
		return nil, fmt.Errorf("unsupported type %T", obj)
	}

	algo, hex, found := strings.Cut(digest.DigestStr(), ":")
	if !found || algo != "sha256" {
		return nil, fmt.Errorf("unsupported digest %q: only sha256 is supported", digest.DigestStr())
	}

	return &intoto.Statement{
		Type: intoto.StatementTypeUri,
		Subject: []*intoto.ResourceDescriptor{{
			// The repository name is needed to rebuild the image reference when storing.
			Name:   digest.Repository.Name(),
			Digest: map[string]string{algo: hex},
		}},
		PredicateType: types.CosignSignPredicateType,
		Predicate:     &structpb.Struct{Fields: map[string]*structpb.Value{}},
	}, nil
}

// Wrap returns true: the Statement is signed as a DSSE envelope.
func (c *CosignSign) Wrap() bool {
	return true
}

// Type returns the payload type of this formatter.
func (c *CosignSign) Type() config.PayloadType {
	return PayloadTypeCosignSignV1
}

// RetrieveAllArtifactURIs always returns an error; it is not supported by the cosign-sign/v1 formatter.
func (c *CosignSign) RetrieveAllArtifactURIs(_ context.Context, _ interface{}) ([]string, error) {
	return nil, fmt.Errorf("RetrieveAllArtifactURIs not supported for cosign-sign/v1 formatter")
}

// NewFormatter returns a new cosign-sign/v1 payloader.
func NewFormatter(config.Config) (formats.Payloader, error) { //nolint:ireturn
	return &CosignSign{}, nil
}
