/*
Copyright 2021 The Tekton Authors

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

package config

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestArtifact_Enabled(t *testing.T) {
	tests := []struct {
		name           string
		signer         string
		storageBackend sets.Set[string]
		want           bool
	}{
		{
			name:           "enabled by default with valid storage and x509 signer",
			signer:         "x509",
			storageBackend: sets.New[string]("oci"),
			want:           true,
		},
		{
			name:           "enabled with kms signer",
			signer:         "kms",
			storageBackend: sets.New[string]("oci"),
			want:           true,
		},
		{
			name:           "disabled when signer is none",
			signer:         "none",
			storageBackend: sets.New[string]("oci"),
			want:           false,
		},
		{
			name:           "disabled when no storage backend",
			signer:         "x509",
			storageBackend: sets.New[string](""),
			want:           false,
		},
		{
			name:           "disabled when signer none and no storage",
			signer:         "none",
			storageBackend: sets.New[string](""),
			want:           false,
		},
		{
			name:           "enabled with multiple storage backends",
			signer:         "x509",
			storageBackend: sets.New[string]("oci", "gcs"),
			want:           true,
		},
		{
			name:           "disabled with multiple storage backends but signer none",
			signer:         "none",
			storageBackend: sets.New[string]("oci", "gcs"),
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := &Artifact{
				Signer:         tt.signer,
				StorageBackend: tt.storageBackend,
			}
			if got := artifact.Enabled(); got != tt.want {
				t.Errorf("Artifact.Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOCIDistributionMethodDefault(t *testing.T) {
	cfg, err := NewConfigFromMap(map[string]string{})
	if err != nil {
		t.Fatalf("NewConfigFromMap() error: %v", err)
	}
	if cfg.Storage.OCI.DistributionMethod != OCIDistributionLegacy {
		t.Errorf("default DistributionMethod = %q, want %q", cfg.Storage.OCI.DistributionMethod, OCIDistributionLegacy)
	}
}

func TestOCIDistributionMethodExplicit(t *testing.T) {
	for _, method := range []string{OCIDistributionLegacy, OCIDistributionReferrersAPI} {
		t.Run(method, func(t *testing.T) {
			cfg, err := NewConfigFromMap(map[string]string{ociDistributionMethodKey: method})
			if err != nil {
				t.Fatalf("NewConfigFromMap() error: %v", err)
			}
			if cfg.Storage.OCI.DistributionMethod != method {
				t.Errorf("DistributionMethod = %q, want %q", cfg.Storage.OCI.DistributionMethod, method)
			}
		})
	}
}

func TestOCIDistributionMethodInvalid(t *testing.T) {
	_, err := NewConfigFromMap(map[string]string{ociDistributionMethodKey: "unknown-method"})
	if err == nil {
		t.Error("expected error for invalid distribution method, got nil")
	}
}

func TestValidateOCIDistributionMethod(t *testing.T) {
	if err := validateOCIDistributionMethod(""); err != nil {
		t.Errorf("empty string: unexpected error: %v", err)
	}
	for _, valid := range []string{OCIDistributionLegacy, OCIDistributionReferrersAPI} {
		if err := validateOCIDistributionMethod(valid); err != nil {
			t.Errorf("valid %q: unexpected error: %v", valid, err)
		}
	}
	if err := validateOCIDistributionMethod("bad-method"); err == nil {
		t.Error("expected error for invalid method, got nil")
	}
}
