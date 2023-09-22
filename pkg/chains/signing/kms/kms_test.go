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
	"testing"

	"github.com/tektoncd/chains/pkg/config"
)

func TestInValidVaultAddressTimeout(t *testing.T) {
	cfg := config.KMSSigner{}
	cfg.Auth.Address = "http://8.8.8.8:8200"

	_, err := NewSigner(context.TODO(), cfg)
	expectedErrorMessage := "dial tcp 8.8.8.8:8200: i/o timeout"
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
	}
}

func TestInValidVaultAddressConnectionRefused(t *testing.T) {
	cfg := config.KMSSigner{}
	cfg.Auth.Address = "http://127.0.0.1:8200"

	_, err := NewSigner(context.TODO(), cfg)
	expectedErrorMessage := "dial tcp 127.0.0.1:8200: connect: connection refused"
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrorMessage, err.Error())
	}
}
