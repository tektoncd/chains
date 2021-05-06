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

package pkcs9

import (
	"crypto"
	"crypto/hmac"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sassoftware/relic/lib/pkcs7"
	"github.com/sassoftware/relic/lib/x509tools"
)

// Verify that the digest (imprint) in a timestamp token matches the given data
func (i MessageImprint) Verify(data []byte) error {
	hash, err := x509tools.PkixDigestToHashE(i.HashAlgorithm)
	if err != nil {
		return errors.Wrap(err, "pkcs9")
	}
	w := hash.New()
	w.Write(data)
	digest := w.Sum(nil)
	if !hmac.Equal(digest, i.HashedMessage) {
		return errors.New("pkcs9: digest check failed")
	}
	return nil
}

// Verify a timestamp token using external data
func Verify(tst *pkcs7.ContentInfoSignedData, data []byte, certs []*x509.Certificate) (*CounterSignature, error) {
	if len(tst.Content.SignerInfos) != 1 {
		return nil, errors.New("timestamp should have exactly one SignerInfo")
	}
	tsi := tst.Content.SignerInfos[0]
	tsicerts, err := tst.Content.Certificates.Parse()
	if err != nil {
		return nil, err
	} else if len(tsicerts) != 0 {
		// keep both sets of certs just in case
		certs = append(certs, tsicerts...)
	}
	// verify the imprint in the TSTInfo
	tstinfo, err := unpackTokenInfo(tst)
	if err != nil {
		return nil, err
	}
	if err := tstinfo.MessageImprint.Verify(data); err != nil {
		return nil, fmt.Errorf("failed to verify timestamp imprint: %s", err)
	}
	imprintHash, _ := x509tools.PkixDigestToHash(tstinfo.MessageImprint.HashAlgorithm)
	// now the signature is over the TSTInfo blob
	verifyBlob, err := tst.Content.ContentInfo.Bytes()
	if err != nil {
		return nil, err
	}
	return finishVerify(&tsi, verifyBlob, certs, imprintHash)
}

func finishVerify(tsi *pkcs7.SignerInfo, blob []byte, certs []*x509.Certificate, hash crypto.Hash) (*CounterSignature, error) {
	cert, err := tsi.Verify(blob, false, certs)
	if err != nil {
		return nil, err
	}
	var signingTime time.Time
	if err := tsi.AuthenticatedAttributes.GetOne(pkcs7.OidAttributeSigningTime, &signingTime); err != nil {
		return nil, err
	}
	return &CounterSignature{
		Signature: pkcs7.Signature{
			SignerInfo:    tsi,
			Certificate:   cert,
			Intermediates: certs,
		},
		Hash:        hash,
		SigningTime: signingTime,
	}, nil
}
