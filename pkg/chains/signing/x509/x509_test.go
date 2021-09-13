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
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/tektoncd/chains/pkg/config"
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
	signer, err := NewSigner(d, config.Config{}, logger)
	if err != nil {
		t.Fatal(err)
	}

	payload := testPayload{A: 4, B: "test"}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := signer.SignMessage(bytes.NewReader(rawPayload))
	if err != nil {
		t.Fatal(err)
	}

	pub, err := signer.PublicKey()
	if err != nil {
		t.Fatal(err)
	}

	pubKey, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("public key is not of type ecdsa.PublicKey")
	}

	h := sha256.Sum256(rawPayload)
	if !ecdsa.VerifyASN1(pubKey, h[:], []byte(signature)) {
		t.Error("invalid signature")
	}
}

func TestSigner_SignED25519(t *testing.T) {
	t.Skip("skip test until ed25519 signing is implemented")
	logger := logtesting.TestLogger(t)
	d := t.TempDir()
	p := filepath.Join(d, "x509.pem")
	if err := ioutil.WriteFile(p, []byte(ed25519Priv), 0644); err != nil {
		t.Fatal(err)
	}

	// create a signer
	signer, err := NewSigner(d, config.Config{}, logger)
	if err != nil {
		t.Fatal(err)
	}

	payload := testPayload{A: 4, B: "test"}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := signer.SignMessage(bytes.NewReader(rawPayload))
	if err != nil {
		t.Fatal(err)
	}

	pub, err := signer.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	pubKey, ok := pub.(ed25519.PublicKey)
	if !ok {
		t.Fatal("not an ed25519 pub key")
	}
	if !ed25519.Verify(pubKey, rawPayload, []byte(signature)) {
		t.Error("invalid signature")
	}
}
