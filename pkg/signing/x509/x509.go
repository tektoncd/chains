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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	cx509 "crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// Signer exposes methods to sign payloads using x509
type Signer struct {
	key    interface{}
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

	s := &Signer{
		key:    pk,
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

	h := sha256.Sum256(b)
	var signature []byte
	switch k := s.key.(type) {
	case *ecdsa.PrivateKey:
		signature, err = ecdsa.SignASN1(rand.Reader, k, h[:])
	case ed25519.PrivateKey:
		signature = ed25519.Sign(k, b)
	}
	if err != nil {
		return "", nil, err
	}

	return string(signature), b, nil
}
