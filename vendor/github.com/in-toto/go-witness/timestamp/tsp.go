// Copyright 2022 The Witness Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package timestamp

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/digitorus/pkcs7"
	"github.com/digitorus/timestamp"
	"github.com/in-toto/go-witness/cryptoutil"
)

type TSPTimestamper struct {
	url                string
	hash               crypto.Hash
	requestCertificate bool
}

type TSPTimestamperOption func(*TSPTimestamper)

func TimestampWithUrl(url string) TSPTimestamperOption {
	return func(t *TSPTimestamper) {
		t.url = url
	}
}

func TimestampWithHash(h crypto.Hash) TSPTimestamperOption {
	return func(t *TSPTimestamper) {
		t.hash = h
	}
}

func TimestampWithRequestCertificate(requestCertificate bool) TSPTimestamperOption {
	return func(t *TSPTimestamper) {
		t.requestCertificate = requestCertificate
	}
}

func NewTimestamper(opts ...TSPTimestamperOption) TSPTimestamper {
	t := TSPTimestamper{
		hash:               crypto.SHA256,
		requestCertificate: true,
	}

	for _, opt := range opts {
		opt(&t)
	}

	return t
}

func (t TSPTimestamper) Timestamp(ctx context.Context, r io.Reader) ([]byte, error) {
	tsq, err := timestamp.CreateRequest(r, &timestamp.RequestOptions{
		Hash:         t.hash,
		Certificates: t.requestCertificate,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(tsq))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/timestamp-query")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
	default:
		return nil, fmt.Errorf("request to timestamp authority failed: %v", resp.Status)
	}

	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	timestamp, err := timestamp.ParseResponse(bodyBytes)
	if err != nil {
		return nil, err
	}

	return timestamp.RawToken, nil
}

type TSPVerifier struct {
	certChain *x509.CertPool
	hash      crypto.Hash
}

type TSPVerifierOption func(*TSPVerifier)

func VerifyWithCerts(certs []*x509.Certificate) TSPVerifierOption {
	return func(t *TSPVerifier) {
		t.certChain = x509.NewCertPool()
		for _, cert := range certs {
			t.certChain.AddCert(cert)
		}
	}
}

func VerifyWithHash(h crypto.Hash) TSPVerifierOption {
	return func(t *TSPVerifier) {
		t.hash = h
	}
}

func NewVerifier(opts ...TSPVerifierOption) TSPVerifier {
	v := TSPVerifier{
		hash: crypto.SHA256,
	}

	for _, opt := range opts {
		opt(&v)
	}

	return v
}

func (v TSPVerifier) Verify(ctx context.Context, tsrData, signedData io.Reader) (time.Time, error) {
	tsrBytes, err := io.ReadAll(tsrData)
	if err != nil {
		return time.Time{}, err
	}

	ts, err := timestamp.Parse(tsrBytes)
	if err != nil {
		return time.Time{}, err
	}

	hashedData, err := cryptoutil.Digest(signedData, v.hash)
	if err != nil {
		return time.Time{}, err
	}

	if !bytes.Equal(ts.HashedMessage, hashedData) {
		return time.Time{}, fmt.Errorf("signed payload does not match timestamped payload")
	}

	p7, err := pkcs7.Parse(tsrBytes)
	if err != nil {
		return time.Time{}, err
	}

	if err := p7.VerifyWithChain(v.certChain); err != nil {
		return time.Time{}, err
	}

	// RFC 3161 requires the TSA signing certificate to carry the id-kp-timeStamping extended key
	// usage (and only that EKU). VerifyWithChain does not enforce a specific EKU, so a certificate
	// issued for another purpose (for example TLS server authentication) under the same trust
	// anchor would otherwise be accepted as a timestamp signer.
	signer := p7.GetOnlySigner()
	if signer == nil {
		return time.Time{}, fmt.Errorf("timestamp token has no single signing certificate")
	}

	// RFC 3161 section 2.3 requires the timestamping EKU to be the sole extended key usage, so a
	// multi-purpose certificate (for example one that also bears ServerAuth) must be rejected.
	if len(signer.ExtKeyUsage) != 1 ||
		signer.ExtKeyUsage[0] != x509.ExtKeyUsageTimeStamping ||
		len(signer.UnknownExtKeyUsage) != 0 {
		return time.Time{}, fmt.Errorf("timestamp signing certificate must carry the id-kp-timeStamping extended key usage as its only EKU")
	}

	// RFC 3161 section 2.3 also requires that the extended key usage extension be marked critical.
	if !ekuExtensionIsCritical(signer) {
		return time.Time{}, fmt.Errorf("timestamp signing certificate extended key usage extension must be marked critical")
	}

	// p7.VerifyWithChain validates the chain with ExtKeyUsageAny, so an EKU-constrained intermediate
	// is not caught by the leaf-only check above. Re-verify the chain explicitly requiring the
	// timestamping EKU so the constraint is enforced through every certificate in the path.
	//
	// Only do this when a trust store was configured: with a nil cert pool p7.VerifyWithChain(nil)
	// intentionally skips chain validation (signature/hash-only mode), and passing Roots: nil to
	// signer.Verify would silently fall back to the host system roots, changing that behavior.
	if v.certChain != nil {
		intermediates := x509.NewCertPool()
		for _, cert := range p7.Certificates {
			if cert.Equal(signer) {
				continue
			}
			intermediates.AddCert(cert)
		}
		// Verify the chain at the timestamp's own time, not wall-clock time: a timestamp must remain
		// verifiable after the TSA certificate expires (that is the purpose of timestamping),
		// matching p7.VerifyWithChain's use of the signing time.
		if _, err := signer.Verify(x509.VerifyOptions{
			Roots:         v.certChain,
			Intermediates: intermediates,
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
			CurrentTime:   ts.Time,
		}); err != nil {
			return time.Time{}, fmt.Errorf("timestamp certificate chain is not valid for timestamping: %w", err)
		}
	}

	return ts.Time, nil
}

// oidExtensionExtKeyUsage is the ASN.1 object identifier of the X.509 extended key usage extension.
var oidExtensionExtKeyUsage = asn1.ObjectIdentifier{2, 5, 29, 37}

// ekuExtensionIsCritical reports whether the certificate's extended key usage extension is present
// and marked critical, as required of a TSA certificate by RFC 3161 section 2.3.
func ekuExtensionIsCritical(cert *x509.Certificate) bool {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(oidExtensionExtKeyUsage) {
			return ext.Critical
		}
	}

	return false
}
