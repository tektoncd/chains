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

	x509PrivateKeyPath := filepath.Join(secretPath, "x509.pem")
	cosignPrivateKeypath := filepath.Join(secretPath, "cosign.key")

	var signer *signature.ECDSASignerVerifier
	if contents, err := ioutil.ReadFile(x509PrivateKeyPath); err == nil {
		signer, err = x509Signer(contents, logger)
		if err != nil {
			return nil, err
		}
	} else if contents, err := ioutil.ReadFile(cosignPrivateKeypath); err == nil {
		signer, err = cosignSigner(secretPath, contents, logger)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("no valid private key found, looked for: [x509.pem, cosign.key]")
	}
	return &Signer{
		Signer: signer,
		logger: logger,
	}, nil
}

func x509Signer(privateKey []byte, logger *zap.SugaredLogger) (*signature.ECDSASignerVerifier, error) {
	logger.Info("Found x509 key...")

	p, _ := pem.Decode(privateKey)
	if p.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("expected private key, found object of type %s", p.Type)
	}
	pk, err := cx509.ParsePKCS8PrivateKey(p.Bytes)
	if err != nil {
		return nil, err
	}
	return signature.LoadECDSASignerVerifier(pk.(*ecdsa.PrivateKey), crypto.SHA256)
}

func cosignSigner(secretPath string, privateKey []byte, logger *zap.SugaredLogger) (*signature.ECDSASignerVerifier, error) {
	logger.Info("Found cosign key...")
	cosignPasswordPath := filepath.Join(secretPath, "cosign.password")
	password, err := ioutil.ReadFile(cosignPasswordPath)
	if err != nil {
		return nil, errors.Wrap(err, "reading cosign.password file")
	}
	return cosign.LoadECDSAPrivateKey(privateKey, password)
}

func (s *Signer) Type() string {
	return signing.TypeX509
}
