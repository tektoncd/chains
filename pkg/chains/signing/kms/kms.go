/*
Copyright 2020 The Tekton Authors
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

package kms

import (
	"context"
	"crypto"

	"github.com/tektoncd/chains/pkg/config"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/aws"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/azure"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/gcp"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/hashivault"
	"github.com/sigstore/sigstore/pkg/signature/options"

	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"go.uber.org/zap"
)

// Signer exposes methods to sign payloads using a KMS
type Signer struct {
	signature.SignerVerifier
	logger *zap.SugaredLogger
}

// NewSigner returns a configured Signer
func NewSigner(ctx context.Context, cfg config.KMSSigner, logger *zap.SugaredLogger) (*Signer, error) {
	kmsOpts := []signature.RPCOption{}
	rpcAuth := options.RPCAuth{
		Address: cfg.Auth.Address,
		OIDC: options.RPCAuthOIDC{
			Role: cfg.Auth.Role,
			Path: cfg.Auth.Path,
		},
	}
	// get token from spire socket
	if cfg.Auth.Spire.Sock != "" {
		jwtSource, err := workloadapi.NewJWTSource(
			ctx,
			workloadapi.WithClientOptions(workloadapi.WithAddr(cfg.Auth.Spire.Sock)),
		)
		if err != nil {
			return nil, err
		}
		svid, err := jwtSource.FetchJWTSVID(ctx, jwtsvid.Params{Audience: cfg.Auth.Spire.Audience})
		if err != nil {
			return nil, err
		}
		rpcAuth.OIDC.Token = svid.Marshal()
	}
	kmsOpts = append(kmsOpts, options.WithRPCAuthOpts(rpcAuth))
	k, err := kms.Get(ctx, cfg.KMSRef, crypto.SHA256, kmsOpts...)
	if err != nil {
		return nil, err
	}
	return &Signer{
		SignerVerifier: k,
		logger:         logger,
	}, nil
}

func (s *Signer) Type() string {
	return signing.TypeKMS
}

// there is no cert, return nothing
func (s *Signer) Cert() string {
	return ""
}

// there is no cert or chain, return nothing
func (s *Signer) Chain() string {
	return ""
}
