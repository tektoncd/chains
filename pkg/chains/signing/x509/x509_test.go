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
	"crypto/sha256"
	"io/ioutil"
	"path/filepath"
	"testing"

	logtesting "knative.dev/pkg/logging/testing"
)

// Generated with:
// openssl ecparam -genkey -name prime256v1 > ec_private.pem
// openssl pkcs8 -topk8 -in ec_private.pem  -nocrypt'
const ecdsaPriv = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg4Xc13PH8xxgNSSn1
N/UQvU4xt4czrBBwbGeOOAtYlbShRANCAAQCshJsXESGCznJs43YqPgqzGOaiYmk
/uimTe+vf9GtnN8Nek1d/sswubWomAGYWfhu4NrCHvK0/mq6hwwcYFZl
-----END PRIVATE KEY-----`

type testPayload struct {
	A int
	B string
}

// Generated with:
// openssl genpkey -algorithm ED25519
const ed25519Priv = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIGQn0bJwshjwuVdnd/FylMk3Gvb89aGgH49bQpgzCY0n
-----END PRIVATE KEY-----`

func TestSigner_SignECDSA(t *testing.T) {
	logger := logtesting.TestLogger(t)
	d := t.TempDir()
	p := filepath.Join(d, "x509.pem")
	if err := ioutil.WriteFile(p, []byte(ecdsaPriv), 0644); err != nil {
		t.Fatal(err)
	}

	// create a signer
	signer, err := NewSigner(d, logger)
	if err != nil {
		t.Fatal(err)
	}

	payload := testPayload{A: 4, B: "test"}
	signature, signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}

	privKey := signer.key.(*ecdsa.PrivateKey)
	h := sha256.Sum256(signed)

	if !ecdsa.VerifyASN1(&privKey.PublicKey, h[:], []byte(signature)) {
		t.Error("invalid signature")
	}
}

func TestSigner_SignED25519(t *testing.T) {
	logger := logtesting.TestLogger(t)
	d := t.TempDir()
	p := filepath.Join(d, "x509.pem")
	if err := ioutil.WriteFile(p, []byte(ed25519Priv), 0644); err != nil {
		t.Fatal(err)
	}

	// create a signer
	signer, err := NewSigner(d, logger)
	if err != nil {
		t.Fatal(err)
	}

	payload := testPayload{A: 4, B: "test"}
	signature, signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}

	privKey := signer.key.(ed25519.PrivateKey)

	pub := privKey.Public().(ed25519.PublicKey)
	if !ed25519.Verify(pub, signed, []byte(signature)) {
		t.Error("invalid signature")
	}
}
