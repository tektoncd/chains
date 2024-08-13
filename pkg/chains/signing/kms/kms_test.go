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
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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

func TestValidVaultAddressConnectionWithoutPortAndScheme(t *testing.T) {
	cfg := config.KMSSigner{}
	cfg.Auth.Address = "abc.com"

	_, err := NewSigner(context.Background(), cfg)
	expectedErrorMessage := "no kms provider found for key reference: "
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
	}
}

func TestValidVaultAddressConnectionWithoutScheme(t *testing.T) {
	cfg := config.KMSSigner{}
	cfg.Auth.Address = "abc.com:80"

	_, err := NewSigner(context.Background(), cfg)
	expectedErrorMessage := "no kms provider found for key reference: "
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
		expectedErrorMessage := "no kms provider found for key reference: "
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
		expectedErrorMessage := "no kms provider found for key reference: "
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
		expectedErrorMessage := "no kms provider found for key reference: "
		if err.Error() != expectedErrorMessage {
			t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
		}
	})
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
