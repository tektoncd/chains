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
	"encoding/json"

	"github.com/tektoncd/chains/pkg/config"

	"github.com/sigstore/sigstore/pkg/kms"

	"github.com/tektoncd/chains/pkg/chains/signing"
	"go.uber.org/zap"
)

// Signer exposes methods to sign payloads using a KMS
type Signer struct {
	k      kms.KMS
	logger *zap.SugaredLogger
}

// NewSigner returns a configured Signer
func NewSigner(cfg config.KMSSigner, logger *zap.SugaredLogger) (*Signer, error) {
	k, err := kms.Get(context.Background(), cfg.KMSRef)
	if err != nil {
		return nil, err
	}

	s := &Signer{
		k:      k,
		logger: logger,
	}

	return s, nil
}

// Sign signs an incoming payload.
// It returns the signature and the marshaled payload object.
func (s *Signer) Sign(i interface{}) (string, []byte, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return "", nil, err
	}

	sig, _, err := s.k.Sign(context.Background(), b)
	if err != nil {
		return "", nil, err
	}
	return string(sig), b, nil
}

func (s *Signer) Type() string {
	return signing.TypeKMS
}
