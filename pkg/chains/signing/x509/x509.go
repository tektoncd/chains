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
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/cosign"
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
	if contents, err := ioutil.ReadFile(privateKeyPath); err == nil {
		return x509Signer(contents, logger)
	}

	privateKeyPath = filepath.Join(secretPath, "cosign.key")
	if contents, err := ioutil.ReadFile(privateKeyPath); err == nil {
		return cosignSigner(secretPath, contents, logger)
	}

	return nil, errors.New("no valid private key found, looked for: [x509.pem, cosign.key]")
}

func x509Signer(privateKey []byte, logger *zap.SugaredLogger) (*Signer, error) {
	logger.Info("Found x509 key...")
	p, _ := pem.Decode(privateKey)
	if p.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("expected private key, found object of type %s", p.Type)
	}
	pk, err := cx509.ParsePKCS8PrivateKey(p.Bytes)
	if err != nil {
		return nil, err
	}
	return signer(pk, logger)
}

func cosignSigner(secretPath string, privateKey []byte, logger *zap.SugaredLogger) (*Signer, error) {
	logger.Info("Found cosign key...")
	cosignPasswordPath := filepath.Join(secretPath, "cosign.password")
	password, err := ioutil.ReadFile(cosignPasswordPath)
	if err != nil {
		return nil, errors.Wrap(err, "reading cosign.password file")
	}
	pk, err := cosign.LoadECDSAPrivateKey(privateKey, password)
	if err != nil {
		return nil, errors.Wrap(err, "cosign private key")
	}
	return signer(pk.Key, logger)
}

func signer(pk crypto.PrivateKey, logger *zap.SugaredLogger) (*Signer, error) {
	var s signature.Signer
	switch k := pk.(type) {
	case *ecdsa.PrivateKey:
		s = signature.NewECDSASignerVerifier(k, crypto.SHA256)
	case ed25519.PrivateKey:
		return nil, errors.New("still need to implement ed25519")
	default:
		return nil, errors.New("unsupported key type")
	}

	return &Signer{
		Signer: s,
		logger: logger,
	}, nil
}

func (s *Signer) Type() string {
	return signing.TypeX509
}
