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

package x509

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	cx509 "crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"go.uber.org/zap"
)

// Signer exposes methods to sign payloads using PGP
type Signer struct {
	signature.Signer
	logger *zap.SugaredLogger
}

// NewSigner returns a configured Signer
func NewSigner(secretPath string, logger *zap.SugaredLogger) (*Signer, error) {
	privateKeyPath := filepath.Join(secretPath, "x509.pem")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no private key at %s", privateKeyPath)
	}

	logger.Debugf("Reading private key from path %s", privateKeyPath)

	b, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode(b)
	if p.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("expected private key, found object of type %s", p.Type)
	}
	pk, err := cx509.ParsePKCS8PrivateKey(p.Bytes)
	if err != nil {
		return nil, err
	}

	var s signature.Signer
	switch k := pk.(type) {
	case *ecdsa.PrivateKey:
		s = signature.NewECDSASignerVerifier(k, crypto.SHA256)
	case ed25519.PrivateKey:
		return nil, errors.New("still need to implement ed25519")
	}

	return &Signer{
		Signer: s,
		logger: logger,
	}, nil
}

func (s *Signer) Type() string {
	return signing.TypeX509
}
