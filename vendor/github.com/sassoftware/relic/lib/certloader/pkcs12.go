//
// Copyright (c) SAS Institute Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package certloader

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/sassoftware/relic/lib/passprompt"
	"github.com/sassoftware/relic/lib/x509tools"
	"golang.org/x/crypto/pkcs12"
)

func ParsePKCS12(blob []byte, prompt passprompt.PasswordGetter) (*Certificate, error) {
	var password string
	var blocks []*pem.Block
	var triedEmpty bool
	for {
		var err error
		password, err = prompt.GetPasswd("Password for PKCS12: ")
		if err != nil {
			return nil, err
		} else if password == "" {
			if triedEmpty {
				return nil, errors.New("aborted")
			}
			triedEmpty = true
		}
		blocks, err = pkcs12.ToPEM(blob, password)
		if err == nil {
			break
		} else if err != pkcs12.ErrIncorrectPassword {
			return nil, err
		}
	}
	var certs []*x509.Certificate
	var privKey crypto.PrivateKey
	for _, block := range blocks {
		switch block.Type {
		case "CERTIFICATE":
			newcerts, err := parseCertificatesDer(block.Bytes)
			if err != nil {
				return nil, err
			}
			certs = append(certs, newcerts.Certificates...)
		case "PRIVATE KEY":
			if privKey != nil {
				return nil, errors.New("multiple private keys")
			}
			var err error
			privKey, err = parsePrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
		}
	}
	if privKey == nil {
		return nil, errors.New("incorrect password or no private key")
	}
	ret := &Certificate{PrivateKey: privKey, Certificates: certs}
	for _, cert := range certs {
		if x509tools.SameKey(cert.PublicKey, privKey) {
			ret.Leaf = cert
		}
	}
	if ret.Leaf == nil {
		return nil, errors.New("leaf certificate not found")
	}
	return ret, nil
}
