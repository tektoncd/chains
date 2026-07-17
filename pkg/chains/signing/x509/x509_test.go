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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sigstore/cosign/v2/pkg/providers"
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

// Generated with RS256 algorithm (required for cosign v2.6.0+)
// openssl genrsa -out private.pem 2048
// python3 -c "import jwt; import time; private_key = open('/tmp/private.pem').read(); payload = {'iat': int(time.time()), 'exp': int(time.time()) + 3600 * 24 * 365 * 10, 'iss': 'user123'}; print(jwt.encode(payload, private_key, algorithm='RS256'))"
const token = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE3NjIzMTIzOTQsImV4cCI6MjA3NzY3MjM5NCwiaXNzIjoidXNlcjEyMyJ9.Adm27mf955gZA2pcWLqF4LLrqzFbXYsdYNg1sScF9MbyeuE-4eVpqV91Rk-iRwwIrtKuOVkEDdulrAqeuIhMxGB7jNXWXxf6sVEHV57_QgB0KR_z-JVxEbTZBu6nIVBwDxmVFGQFVMtZbqsyX8J4F_jp0pSInFPqYQbS9xAGhvOnni_owp325Siev2Z-kWsnTTFOTi0C9g9BApPxXQEE17COYdXjxsBCJQQttb1Ww7IQLCf59wU5ZpNM7npzxvKuOBT1kmHPp1ZDCNxfA_a6JMIB4NQAzYV0ULRbXNftxwglFoyitWge-SyxohnTVfV1gplE8qi6kR2CQJORBMvx6w`

func TestCreateSignerFulcioEnabledDefaultTokenFileMissing(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	d := t.TempDir()

	data := make(map[string]string)
	data["signers.x509.fulcio.enabled"] = "true"
	cfg, err := config.NewConfigFromMap(data)
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.Signers.X509.FulcioEnabled {
		t.Fatal("fulcio is not enabled, expected to be enabled")
	}
	_, _ = NewSigner(ctx, d, *cfg)
	//  With default file not present, expect the list of providers to be empty
	if providers.Enabled(ctx) {
		t.Fatal("Expected providers to be false")
	}
}

func TestCreateSignerFulcioEnabled(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	d := t.TempDir()
	tk := filepath.Join(d, "token")
	if err := os.WriteFile(tk, []byte(token), 0644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "x509.pem")
	if err := os.WriteFile(p, []byte(ecdsaPriv), 0644); err != nil {
		t.Fatal(err)
	}

	data := make(map[string]string)
	data["signers.x509.fulcio.enabled"] = "true"
	data["signers.x509.identity.token.file"] = tk
	cfg, err := config.NewConfigFromMap(data)
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.Signers.X509.FulcioEnabled {
		t.Fatal("fulcio is not enabled, expected to be enabled")
	}
	_, err = NewSigner(ctx, d, *cfg)
	if err != nil {
		if !providers.Enabled(ctx) {
			t.Fatal("fulcio provider not configured")
		}
		// at this point the signer is configured, but would need a valid id token to proceed
		if !strings.Contains(err.Error(), "no subject found in claims") {
			t.Fatal(err)
		}
	}
}

func TestCreateSignerFulcioEnabledFilesystemProvider(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	d := t.TempDir()
	tk := filepath.Join(d, "token")
	if err := os.WriteFile(tk, []byte(token), 0644); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(d, "x509.pem")
	if err := os.WriteFile(p, []byte(ecdsaPriv), 0644); err != nil {
		t.Fatal(err)
	}

	data := make(map[string]string)
	data["signers.x509.fulcio.enabled"] = "true"
	data["signers.x509.identity.token.file"] = tk
	data["signers.x509.fulcio.provider"] = "filesystem"
	cfg, err := config.NewConfigFromMap(data)
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.Signers.X509.FulcioEnabled {
		t.Fatal("fulcio is not enabled, expected to be enabled")
	}
	_, err = NewSigner(ctx, d, *cfg)
	if err != nil {
		if !providers.Enabled(ctx) {
			t.Fatal("fulcio provider not configured")
		}
		// at this point the signer is configured, but would need a valid id token to proceed
		if !strings.Contains(err.Error(), "no subject found in claims") {
			t.Fatal(err)
		}
	}
}

// TestInitializeTUFWithWritableCacheDir verifies that initializeTUF() succeeds
// past the filesystem step when TUF_ROOT points to a writable directory. This
// exercises the real code path that fails in the Chains distroless container
// when readOnlyRootFilesystem is true and no writable cache location is
// provided (see SRVKP-9439). The mock server serves invalid TUF metadata, so
// we expect a TUF validation error — but NOT a filesystem error.
func TestInitializeTUFWithWritableCacheDir(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"signed":{"_type":"root","version":1},"signatures":[]}`))
	}))
	defer ts.Close()

	writableDir := t.TempDir()
	t.Setenv("TUF_ROOT", filepath.Join(writableDir, ".sigstore", "root"))

	ctx := logtesting.TestContextWithLogger(t)
	err := initializeTUF(ctx, ts.URL)
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "read-only file system") ||
		strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "creating cached local store") {
		t.Fatalf("Got filesystem error with writable TUF_ROOT: %v", err)
	}
	t.Logf("TUF init got past filesystem step (expected metadata error): %v", err)
}

func TestSigner_SignECDSA(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	d := t.TempDir()
	p := filepath.Join(d, "x509.pem")
	if err := os.WriteFile(p, []byte(ecdsaPriv), 0644); err != nil {
		t.Fatal(err)
	}

	// create a signer
	signer, err := NewSigner(ctx, d, config.Config{})
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
	ctx := logtesting.TestContextWithLogger(t)
	d := t.TempDir()
	p := filepath.Join(d, "x509.pem")
	if err := os.WriteFile(p, []byte(ed25519Priv), 0644); err != nil {
		t.Fatal(err)
	}

	// create a signer
	signer, err := NewSigner(ctx, d, config.Config{})
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

func TestNewSignerMalformedX509Key(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	d := t.TempDir()
	p := filepath.Join(d, "x509.pem")
	if err := os.WriteFile(p, []byte("this is not a PEM block"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NewSigner(ctx, d, config.Config{})
	if err == nil {
		t.Fatal("expected an error for a key file without a PEM block, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode PEM block") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewSignerUnsupportedX509KeyType(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	d := t.TempDir()
	p := filepath.Join(d, "x509.pem")
	// ed25519 keys are a valid PKCS8 key but not supported by the x509 signer.
	if err := os.WriteFile(p, []byte(ed25519Priv), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := NewSigner(ctx, d, config.Config{})
	if err == nil {
		t.Fatal("expected an error for an unsupported key type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported private key type") {
		t.Fatalf("unexpected error: %v", err)
	}
}
