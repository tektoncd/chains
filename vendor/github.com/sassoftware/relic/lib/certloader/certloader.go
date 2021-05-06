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
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"

	"github.com/sassoftware/relic/lib/pkcs7"
	"github.com/sassoftware/relic/lib/pkcs9"
	"github.com/sassoftware/relic/lib/x509tools"
)

const asn1Magic = 0x30 // weak but good enough?
var pkcs7SignedData = []byte{0x06, 0x09, 0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x07, 0x02}

// A bundle of X509 certificate chain and/or PGP certificate, with optional private key
type Certificate struct {
	Leaf         *x509.Certificate
	Certificates []*x509.Certificate
	PgpKey       *openpgp.Entity
	PrivateKey   crypto.PrivateKey
	Timestamper  pkcs9.Timestamper
	KeyName      string
}

// Return the X509 certificates in the chain up to, but not including, the root CA certificate
func (s *Certificate) Chain() []*x509.Certificate {
	var chain []*x509.Certificate
	if s.Leaf != nil {
		// ensure leaf comes first
		chain = append(chain, s.Leaf)
	}
	for i, cert := range s.Certificates {
		if i > 0 && bytes.Equal(cert.RawIssuer, cert.RawSubject) {
			// omit root CA
			continue
		} else if cert == s.Leaf {
			// already in list
			continue
		}
		chain = append(chain, cert)
	}
	return chain
}

// Return the certificate that issued the leaf certificate
func (s *Certificate) Issuer() *x509.Certificate {
	if s.Leaf == nil {
		return nil
	}
	for _, cert := range s.Certificates {
		if bytes.Equal(cert.RawSubject, s.Leaf.RawIssuer) {
			return cert
		}
	}
	return nil
}

// Return the private key in the form of a crypto.Signer
func (s *Certificate) Signer() crypto.Signer {
	if s.PrivateKey == nil {
		return nil
	}
	return s.PrivateKey.(crypto.Signer)
}

// Return a tls.Certificate structure containing the X509 certificate chain and
// private key
func (s *Certificate) TLS() tls.Certificate {
	var raw [][]byte
	for _, cert := range s.Certificates {
		raw = append(raw, cert.Raw)
	}
	return tls.Certificate{Leaf: s.Leaf, Certificate: raw, PrivateKey: s.PrivateKey}
}

// Parse a private key from a DER block
// See crypto/tls.parsePrivateKey
func parsePrivateKey(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("tls: found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.New("tls: failed to parse private key")
}

// Parse a list of certificates, PEM or DER, X509 or PKCS#7
func parseCertificates(pemData []byte) (*Certificate, error) {
	if len(pemData) >= 1 && pemData[0] == asn1Magic {
		// already in DER form
		return parseCertificatesDer(pemData)
	}
	var certs []*x509.Certificate
	for {
		var block *pem.Block
		block, pemData = pem.Decode(pemData)
		if block == nil {
			break
		} else if block.Type == "CERTIFICATE" || block.Type == "PKCS7" {
			newcerts, err := parseCertificatesDer(block.Bytes)
			if err != nil {
				return nil, err
			}
			certs = append(certs, newcerts.Certificates...)
		}
	}
	if len(certs) == 0 {
		return nil, ErrNoCerts
	}
	return &Certificate{Leaf: certs[0], Certificates: certs}, nil
}

// Parse certificates from DER
func parseCertificatesDer(der []byte) (*Certificate, error) {
	var certs []*x509.Certificate
	var err error
	if bytes.Contains(der[:32], pkcs7SignedData) {
		psd, err := pkcs7.Unmarshal(der)
		if err != nil {
			return nil, err
		}
		certs, err = psd.Content.Certificates.Parse()
	} else {
		certs, err = x509.ParseCertificates(der)
	}
	if err != nil {
		return nil, err
	} else if len(certs) == 0 {
		return nil, ErrNoCerts
	} else {
		return &Certificate{Leaf: certs[0], Certificates: certs}, nil
	}
}

// ParseX509Certificates parses a blob in PEM or DER, X509 or PKCS#7 format and returns a list of certificates
func ParseX509Certificates(blob []byte) ([]*x509.Certificate, error) {
	cert, err := parseCertificates(blob)
	if err != nil {
		return nil, err
	}
	return cert.Certificates, nil
}

// Load a X509 private key and certificate
func LoadX509KeyPair(certFile, keyFile string) (*Certificate, error) {
	keyblob, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	certblob, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	key, err := ParseAnyPrivateKey(keyblob, nil)
	if err != nil {
		return nil, err
	}
	cert, err := parseCertificates(certblob)
	if err != nil {
		return nil, err
	}
	if !x509tools.SameKey(cert.Leaf.PublicKey, key) {
		return nil, errors.New("Private key does not match certificate")
	}
	cert.PrivateKey = key
	return cert, nil
}

// Load X509 and/or PGP certificates from named paths and return a Certificate
// structure together with the given private key
func LoadTokenCertificates(key crypto.PrivateKey, x509cert, pgpcert string) (*Certificate, error) {
	var cert *Certificate
	if x509cert != "" {
		blob, err := ioutil.ReadFile(x509cert)
		if err != nil {
			return nil, err
		}
		cert, err = parseCertificates(blob)
		if err != nil {
			return nil, err
		}
		if !x509tools.SameKey(key, cert.Leaf.PublicKey) {
			return nil, errors.New("certificate does not match key in token")
		}
		cert.PrivateKey = key
	} else {
		cert = &Certificate{PrivateKey: key}
	}
	if pgpcert != "" {
		blob, err := ioutil.ReadFile(pgpcert)
		if err != nil {
			return nil, err
		}
		keyring, err := parsePGP(blob)
		if err != nil {
			return nil, err
		}
		if len(keyring) != 1 {
			return nil, fmt.Errorf("expected exactly 1 entity in pgp certificate %s", pgpcert)
		}
		entity := keyring[0]
		priv := &packet.PrivateKey{
			PublicKey:  *entity.PrimaryKey,
			Encrypted:  false,
			PrivateKey: key,
		}
		if !x509tools.SameKey(key, priv.PublicKey.PublicKey) {
			return nil, errors.New("certificate does not match key in token")
		}
		entity.PrivateKey = priv
		cert.PgpKey = entity
	}
	return cert, nil
}

type errNoCerts struct{}

func (errNoCerts) Error() string {
	return "failed to find any certificates in PEM file"
}

var ErrNoCerts = errNoCerts{}
