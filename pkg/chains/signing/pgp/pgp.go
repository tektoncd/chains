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

package pgp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/tektoncd/chains/pkg/chains/signing"
	"go.uber.org/zap"
	"golang.org/x/crypto/openpgp"
)

// Signer exposes methods to sign payloads using PGP
type Signer struct {
	key    *openpgp.Entity
	logger *zap.SugaredLogger
}

// NewSigner returns a configured Signer
func NewSigner(secretPath string, logger *zap.SugaredLogger) (*Signer, error) {
	privateKeyPath := filepath.Join(secretPath, "pgp.private-key")
	passphrasePath := filepath.Join(secretPath, "pgp.passphrase")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no private key at %s", privateKeyPath)
	}

	logger.Debugf("Reading private key from path %s", privateKeyPath)
	privateKey, err := os.Open(privateKeyPath)
	if err != nil {
		return nil, err
	}

	el, err := openpgp.ReadArmoredKeyRing(privateKey)
	if err != nil {
		return nil, err
	}
	s := &Signer{
		key:    el[0],
		logger: logger,
	}

	var passphrase []byte
	if _, err := os.Stat(passphrasePath); err == nil {
		logger.Debugf("Found passphrase for private key at %s. Decrypting...", passphrasePath)
		passphrase, err = ioutil.ReadFile(passphrasePath)
		if err != nil {
			return nil, err
		}
		if err := s.key.PrivateKey.Decrypt(passphrase); err != nil {
			return nil, err
		}
	} else {
		logger.Debugf("No passphrase found for private key at  %s", passphrasePath)
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
	signature := bytes.Buffer{}
	if err := openpgp.ArmoredDetachSignText(&signature, s.key, bytes.NewReader(b), nil); err != nil {
		return "", nil, err
	}
	return signature.String(), b, nil
}

func (s *Signer) Type() string {
	return signing.TypePgp
}
