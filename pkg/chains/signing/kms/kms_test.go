/*
Copyright 2023 The Tekton Authors
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

// Package kms creates a signer using a key management server

package kms

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/chains/pkg/config"
)

func TestInValidVaultAddressTimeout(t *testing.T) {
	cfg := config.KMSSigner{}
	cfg.Auth.Address = "http://8.8.8.8:8200"

	_, err := NewSigner(context.Background(), cfg)
	expectedErrorMessage := "dial tcp 8.8.8.8:8200: i/o timeout"
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
	}
}

func TestInValidVaultAddressConnectionRefused(t *testing.T) {
	cfg := config.KMSSigner{}
	cfg.Auth.Address = "http://127.0.0.1:8200"

	_, err := NewSigner(context.Background(), cfg)
	expectedErrorMessage := "dial tcp 127.0.0.1:8200: connect: connection refused"
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
	}
}

var expectedErrorMessage = "no kms provider found for key reference: : parsing input key resource id: expected format: [plugin name]://[key ref], got: "

func TestValidVaultAddressConnectionWithoutPortAndScheme(t *testing.T) {
	cfg := config.KMSSigner{}
	cfg.Auth.Address = "abc.com"

	_, err := NewSigner(context.Background(), cfg)
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
	}
}

func TestValidVaultAddressConnectionWithoutScheme(t *testing.T) {
	cfg := config.KMSSigner{}
	cfg.Auth.Address = "abc.com:80"

	_, err := NewSigner(context.Background(), cfg)
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
	}
}

func TestValidVaultAddressConnection(t *testing.T) {
	t.Run("Validation for Vault Address with HTTP Url", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := config.KMSSigner{}
		cfg.Auth.Address = server.URL

		_, err := NewSigner(context.Background(), cfg)
		if err.Error() != expectedErrorMessage {
			t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
		}
	})

	t.Run("Validation for Vault Address with HTTPS URL", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := config.KMSSigner{}
		cfg.Auth.Address = server.URL

		_, err := NewSigner(context.Background(), cfg)
		if err.Error() != expectedErrorMessage {
			t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
		}
	})

	t.Run("Validation for Vault Address with Custom Port URL", func(t *testing.T) {
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		listener, err := net.Listen("tcp", "127.0.0.1:41227")
		if err != nil {
			t.Fatalf("Failed to create listener: %v", err)
		}

		server.Listener = listener
		server.Start()

		cfg := config.KMSSigner{}
		cfg.Auth.Address = "http://127.0.0.1:41227"

		_, err = NewSigner(context.Background(), cfg)
		if err.Error() != expectedErrorMessage {
			t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
		}
	})
}

// TestOIDCTokenEndToEnd proves the full flow: JWT file → rpcAuth.OIDC.Token
// → ApplyRPCAuthOpts → oidcLogin → Vault HTTP request.
// A mock Vault server captures the login request and verifies the JWT, role,
// and auth path arrive exactly as configured.
func TestOIDCTokenEndToEnd(t *testing.T) {
	var mu sync.Mutex
	var loginCalled bool
	var receivedJWT, receivedRole, receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		switch {
		case r.URL.Path == "/v1/auth/jwt/login" && r.Method == http.MethodPut:
			loginCalled = true
			receivedPath = "jwt"

			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				if v, ok := body["jwt"].(string); ok {
					receivedJWT = v
				}
				if v, ok := body["role"].(string); ok {
					receivedRole = v
				}
			}

			w.Header().Set("Content-Type", "application/json")
			resp := `{"auth":{"client_token":"hvs.mock-vault-token","policies":["default"],"lease_duration":3600,"renewable":true}}`
			w.Write([]byte(resp))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tokenFile, err := os.CreateTemp("", "jwt-token")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer os.Remove(tokenFile.Name())

	err = os.WriteFile(tokenFile.Name(), []byte("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test-payload\n"), 0644)
	if err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	cfg := config.KMSSigner{
		KMSRef: "hashivault://supply-chain",
		Auth: config.KMSAuth{
			Address: server.URL,
			OIDC: config.KMSAuthOIDC{
				Path:      "jwt",
				Role:      "tekton-chains",
				TokenPath: tokenFile.Name(),
			},
		},
	}

	signer, err := NewSigner(context.Background(), cfg)

	// The signer should be created successfully — oidcLogin exchanges the
	// JWT for a Vault token, newHashivaultClient stores it, no transit API
	// call happens during construction.
	if err != nil {
		t.Fatalf("NewSigner should succeed when OIDC login returns a valid token, got: %v", err)
	}
	if signer == nil {
		t.Fatal("signer must not be nil")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, loginCalled, "Vault auth/jwt/login endpoint must have been called")
	assert.Equal(t, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test-payload", receivedJWT,
		"JWT sent to Vault must match the trimmed file contents")
	assert.Equal(t, "tekton-chains", receivedRole,
		"role sent to Vault must match the configured OIDC role")
	assert.Equal(t, "jwt", receivedPath,
		"auth path must match the configured OIDC path")
}

// TestOIDCTokenFallbackToDefaultPath proves that when oidc.path/role are set
// but no token-path is given, the code tries the default K8s SA token path.
func TestOIDCTokenFallbackToDefaultPath(t *testing.T) {
	cfg := config.KMSSigner{
		Auth: config.KMSAuth{
			OIDC: config.KMSAuthOIDC{
				Path: "jwt",
				Role: "tekton-chains",
			},
		},
	}

	_, err := NewSigner(context.Background(), cfg)

	if err == nil {
		t.Fatal("expected error when default SA token path does not exist")
	}
	assert.Contains(t, err.Error(), "reading OIDC token")
	assert.Contains(t, err.Error(), defaultOIDCTokenPath,
		"error must reference the default K8s SA token path")
}

// TestOIDCTokenSkippedWhenNotConfigured proves the OIDC block is not entered
// when neither oidc.path nor oidc.role are set.
func TestOIDCTokenSkippedWhenNotConfigured(t *testing.T) {
	cfg := config.KMSSigner{}

	_, err := NewSigner(context.Background(), cfg)

	if err == nil {
		t.Fatal("expected error when no KMS config is set")
	}
	assert.NotContains(t, err.Error(), "reading OIDC token",
		"OIDC reading must not be attempted when OIDC is not configured")
	assert.NotContains(t, err.Error(), "OIDC token file",
		"OIDC empty-file check must not be reached when OIDC is not configured")
}

// TestOIDCTokenSkippedWhenStaticTokenSet proves that the file-based OIDC
// token reading is skipped when a static Vault token is already set, even if
// oidc.path/oidc.role are configured. This prevents breaking existing users
// who have both a static token and leftover OIDC config.
func TestOIDCTokenSkippedWhenStaticTokenSet(t *testing.T) {
	cfg := config.KMSSigner{
		Auth: config.KMSAuth{
			Token: "my-static-vault-token",
			OIDC: config.KMSAuthOIDC{
				Path: "jwt",
				Role: "tekton-chains",
			},
		},
	}

	_, err := NewSigner(context.Background(), cfg)

	if err == nil {
		t.Fatal("expected error (no KMSRef), but should NOT be an OIDC reading error")
	}
	assert.NotContains(t, err.Error(), "reading OIDC token",
		"OIDC file reading must be skipped when a static token is set")
	assert.NotContains(t, err.Error(), "OIDC token file",
		"OIDC empty check must be skipped when a static token is set")
}

// TestOIDCTokenEmptyFileErrors proves that an empty token file produces a
// clear error rather than silently breaking OIDC.
func TestOIDCTokenEmptyFileErrors(t *testing.T) {
	tokenFile, err := os.CreateTemp("", "empty-jwt")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer os.Remove(tokenFile.Name())

	cfg := config.KMSSigner{
		Auth: config.KMSAuth{
			OIDC: config.KMSAuthOIDC{
				Path:      "jwt",
				Role:      "tekton-chains",
				TokenPath: tokenFile.Name(),
			},
		},
	}

	_, err = NewSigner(context.Background(), cfg)

	if err == nil {
		t.Fatal("expected error for empty OIDC token file")
	}
	assert.Contains(t, err.Error(), "OIDC token file")
	assert.Contains(t, err.Error(), "is empty")
}

// TestOIDCTokenNotReadWhenSpireConfigured proves that the file-based OIDC
// token reading is skipped when Spire is configured (Spire takes precedence).
// The Spire gRPC client retries indefinitely, so we use a short-lived context
// to make it fail quickly and verify the error is about Spire, not file-based
// OIDC.
func TestOIDCTokenNotReadWhenSpireConfigured(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg := config.KMSSigner{
		Auth: config.KMSAuth{
			OIDC: config.KMSAuthOIDC{
				Path: "jwt",
				Role: "tekton-chains",
			},
			Spire: config.KMSAuthSpire{
				Sock:     "unix:///tmp/nonexistent-spire.sock",
				Audience: "test",
			},
		},
	}

	_, err := NewSigner(ctx, cfg)

	if err == nil {
		t.Fatal("expected error when Spire socket does not exist")
	}
	// The error should be about Spire/context, NOT about reading an OIDC
	// token file — proving the Spire block runs and the file block is skipped.
	assert.NotContains(t, err.Error(), "reading OIDC token",
		"file-based OIDC reading must be skipped when Spire is configured")
	assert.NotContains(t, err.Error(), "OIDC token file",
		"file-based OIDC empty check must be skipped when Spire is configured")
}

// Test for getKMSAuthToken with non-directory path
func TestGetKMSAuthToken_NotADirectory(t *testing.T) {
	tempFile, err := os.CreateTemp("", "not-a-dir")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	token, err := getKMSAuthToken(tempFile.Name())
	assert.Equal(t, err, nil)
	assert.Equal(t, "", token)
}

// Test for getKMSAuthToken with missing token file
func TestGetKMSAuthToken_FileNotFound(t *testing.T) {
	tempDir := t.TempDir() // Creates a temporary directory
	token, err := getKMSAuthToken(tempDir)
	assert.Error(t, err)
	assert.Equal(t, "", token)
}

// Test for verifying return value of getKMSAuthToken
func TestGetKMSAuthToken_ValidToken(t *testing.T) {
	tempFile, err := os.CreateTemp("", "vault-token")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	err = os.WriteFile(tempFile.Name(), []byte("test-token"), 0644) // write a sample token "test-token"
	assert.NoError(t, err)

	token, err := getKMSAuthToken(tempFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "test-token", token) // verify the value returned by getKMSAuthToken matches "test-token"
}
