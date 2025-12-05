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

package oci

import (
	"os"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/chains/pkg/config"
)

func TestReferrersAPIOption(t *testing.T) {
	// Test that WithFormat option sets COSIGN_EXPERIMENTAL for referrers API format

	// Clear any existing env var
	originalValue := os.Getenv("COSIGN_EXPERIMENTAL")
	defer func() {
		if originalValue != "" {
			os.Setenv("COSIGN_EXPERIMENTAL", originalValue)
		} else {
			os.Unsetenv("COSIGN_EXPERIMENTAL")
		}
	}()
	os.Unsetenv("COSIGN_EXPERIMENTAL")

	// Create storer with referrers API format enabled
	repo, err := name.NewRepository("example.com/test")
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	opts := []AttestationStorerOption{
		WithTargetRepository(repo),
		WithFormat(config.OCIFormatReferrersAPI),
	}

	storer, err := NewAttestationStorer(opts...)
	if err != nil {
		t.Fatalf("Failed to create attestation storer: %v", err)
	}

	// Check that COSIGN_EXPERIMENTAL was set
	if os.Getenv("COSIGN_EXPERIMENTAL") != "1" {
		t.Errorf("Expected COSIGN_EXPERIMENTAL to be set to '1', got '%s'", os.Getenv("COSIGN_EXPERIMENTAL"))
	}

	// Check that the storer was configured correctly
	if storer.repo == nil {
		t.Errorf("Expected storer.repo to be set")
	}

	if storer.repo.Name() != "example.com/test" {
		t.Errorf("Expected repo name to be 'example.com/test', got '%s'", storer.repo.Name())
	}
}

func TestReferrersAPIDisabled(t *testing.T) {
	// Test that WithFormat(legacy) doesn't set COSIGN_EXPERIMENTAL

	// Clear any existing env var
	originalValue := os.Getenv("COSIGN_EXPERIMENTAL")
	defer func() {
		if originalValue != "" {
			os.Setenv("COSIGN_EXPERIMENTAL", originalValue)
		} else {
			os.Unsetenv("COSIGN_EXPERIMENTAL")
		}
	}()
	os.Unsetenv("COSIGN_EXPERIMENTAL")

	// Create storer with legacy format (referrers API disabled)
	repo, err := name.NewRepository("example.com/test")
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	opts := []SimpleStorerOption{
		WithTargetRepository(repo),
		WithFormat(config.OCIFormatLegacy),
	}

	storer, err := NewSimpleStorerFromConfig(opts...)
	if err != nil {
		t.Fatalf("Failed to create simple storer: %v", err)
	}

	// Check that COSIGN_EXPERIMENTAL was not set
	if os.Getenv("COSIGN_EXPERIMENTAL") != "" {
		t.Errorf("Expected COSIGN_EXPERIMENTAL to be unset, got '%s'", os.Getenv("COSIGN_EXPERIMENTAL"))
	}

	// Check that the storer was configured correctly
	if storer.repo == nil {
		t.Errorf("Expected storer.repo to be set")
	}
}

func TestOCIBackendFormatConfig(t *testing.T) {
	// Test that the OCI backend respects the format configuration
	cfg := config.Config{
		Storage: config.StorageConfigs{
			OCI: config.OCIStorageConfig{
				Repository: "example.com/repo",
				Format:     config.OCIFormatReferrersAPI,
			},
		},
	}

	backend := &Backend{
		cfg: cfg,
	}

	// Verify that the config is accessible
	if backend.cfg.Storage.OCI.Format != config.OCIFormatReferrersAPI {
		t.Errorf("Expected Format to be %s in backend config, got %s", config.OCIFormatReferrersAPI, backend.cfg.Storage.OCI.Format)
	}

	if backend.cfg.Storage.OCI.Repository != "example.com/repo" {
		t.Errorf("Expected repository to be 'example.com/repo', got '%s'", backend.cfg.Storage.OCI.Repository)
	}
}
