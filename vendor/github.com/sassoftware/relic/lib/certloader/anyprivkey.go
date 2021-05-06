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
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sassoftware/relic/lib/passprompt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

// Parse and decrypt a private key. It can be a RSA or ECDA key in PKCS#1 or
// PKCS#8 format and DER or PEM encoding, or it can be a PGP private key. If
// the private key is encrypted then the given prompter will be invoked to ask
// for the passphrase, if provided.
func ParseAnyPrivateKey(blob []byte, prompt passprompt.PasswordGetter) (crypto.PrivateKey, error) {
	if bytes.HasPrefix(blob, []byte("-----BEGIN PGP")) {
		return parsePgpPrivateKey(blob, prompt)
	} else if bytes.HasPrefix(blob, []byte("-----BEGIN")) {
		var block *pem.Block
		for {
			block, blob = pem.Decode(blob)
			if block == nil {
				break
			} else if block.Type == "PRIVATE KEY" || strings.HasSuffix(block.Type, " PRIVATE KEY") {
				return parsePemPrivateKey(block, prompt)
			}
		}
		return nil, errors.New("failed to find any private keys in PEM data")
	} else if blob[0] == asn1Magic {
		return parsePrivateKey(blob)
	} else if blob[0]&0x80 != 0 {
		return parsePgpPrivateKey(blob, prompt)
	} else {
		return nil, errors.New("unrecognized private key format")
	}
}

func parsePemPrivateKey(block *pem.Block, prompt passprompt.PasswordGetter) (crypto.PrivateKey, error) {
	if !x509.IsEncryptedPEMBlock(block) {
		return parsePrivateKey(block.Bytes)
	}
	if prompt == nil {
		return nil, errors.New("private key is encrypted and no password was provided")
	}
	for {
		password, err := prompt.GetPasswd("Password for private key: ")
		if err != nil {
			return nil, err
		} else if password == "" {
			return nil, errors.New("aborted")
		}
		keyblob, err := x509.DecryptPEMBlock(block, []byte(password))
		if err == x509.IncorrectPasswordError {
			continue
		} else if err != nil {
			return nil, err
		} else {
			return parsePrivateKey(keyblob)
		}
	}
}

func parsePgpPrivateKey(blob []byte, prompt passprompt.PasswordGetter) (crypto.PrivateKey, error) {
	var reader io.Reader = bytes.NewReader(blob)
	if blob[0] == '-' {
		block, err := armor.Decode(reader)
		if err != nil {
			return nil, err
		}
		reader = block.Body
	}
	entity, err := openpgp.ReadEntity(packet.NewReader(reader))
	if err != nil {
		return nil, err
	}
	if entity.PrivateKey == nil {
		return nil, errors.New("file does not contain a private key")
	}
	if entity.PrivateKey.Encrypted {
		fmt.Fprintln(os.Stderr, "Key fingerprint:", entity.PrimaryKey.KeyIdString())
		for name := range entity.Identities {
			fmt.Fprintln(os.Stderr, "UID:", name)
		}
		fmt.Fprintln(os.Stderr)
		for {
			password, err := prompt.GetPasswd("Passphrase for key: ")
			if err != nil {
				return nil, err
			} else if password == "" {
				return nil, errors.New("Aborted")
			}
			err = entity.PrivateKey.Decrypt([]byte(password))
			if err == nil {
				break
			}
		}
	}
	return entity.PrivateKey.PrivateKey, nil
}
