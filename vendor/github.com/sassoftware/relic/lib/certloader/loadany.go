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
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

type AnyCerts struct {
	X509Certs []*x509.Certificate
	PGPCerts  openpgp.EntityList
}

// Load X509 and/or PGP certificates from the named file paths
func LoadAnyCerts(paths []string) (any AnyCerts, err error) {
	for _, path := range paths {
		blob, err := ioutil.ReadFile(path)
		if err != nil {
			return any, err
		}
		x509certs, err := parseCertificates(blob)
		if err == nil {
			any.X509Certs = append(any.X509Certs, x509certs.Certificates...)
			continue
		} else if err != ErrNoCerts {
			return any, fmt.Errorf("%s: %s", path, err)
		}
		pgpcerts, err := parsePGP(blob)
		if err == nil {
			any.PGPCerts = append(any.PGPCerts, pgpcerts...)
		} else {
			return any, fmt.Errorf("%s: %s", path, err)
		}
	}
	return any, nil
}

// Parse one or more PGP certificates from the given possibly-armored blob
func parsePGP(blob []byte) (openpgp.EntityList, error) {
	reader := io.Reader(bytes.NewReader(blob))
	if blob[0] == '-' {
		block, err := armor.Decode(reader)
		if err != nil {
			return nil, err
		}
		reader = block.Body
	}
	return openpgp.ReadKeyRing(reader)
}
