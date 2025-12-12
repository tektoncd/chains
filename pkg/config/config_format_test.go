/*
Copyright 2025 The Tekton Authors
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

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateOCIFormat(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid legacy format",
			format:      OCIFormatLegacy,
			expectError: false,
		},
		{
			name:        "valid referrers-api format",
			format:      OCIFormatReferrersAPI,
			expectError: false,
		},
		{
			name:        "valid protobuf-bundle format",
			format:      OCIFormatProtobuf,
			expectError: false,
		},
		{
			name:        "empty format (should be valid - defaults to legacy)",
			format:      "",
			expectError: false,
		},
		{
			name:        "invalid format",
			format:      "invalid-format",
			expectError: true,
			errorMsg:    "invalid storage.oci.format: invalid-format, must be one of:",
		},
		{
			name:        "case sensitive validation",
			format:      "Legacy", // wrong case
			expectError: true,
			errorMsg:    "invalid storage.oci.format: Legacy, must be one of:",
		},
		{
			name:        "whitespace format",
			format:      " legacy ",
			expectError: true,
			errorMsg:    "invalid storage.oci.format:  legacy , must be one of:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOCIFormat(tt.format)

			if tt.expectError {
				if err == nil {
					t.Errorf("validateOCIFormat(%q) expected error but got none", tt.format)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("validateOCIFormat(%q) error = %v, expected to contain %q", tt.format, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateOCIFormat(%q) unexpected error = %v", tt.format, err)
				}
			}
		})
	}
}

func TestOCIFormatConstants(t *testing.T) {
	// Test that the constants have expected values
	expectedConstants := map[string]string{
		"OCIFormatLegacy":       "legacy",
		"OCIFormatReferrersAPI": "referrers-api",
		"OCIFormatProtobuf":     "protobuf-bundle",
	}

	actualConstants := map[string]string{
		"OCIFormatLegacy":       OCIFormatLegacy,
		"OCIFormatReferrersAPI": OCIFormatReferrersAPI,
		"OCIFormatProtobuf":     OCIFormatProtobuf,
	}

	if diff := cmp.Diff(expectedConstants, actualConstants); diff != "" {
		t.Errorf("OCI format constants mismatch (-expected +actual):\n%s", diff)
	}
}

func TestOCIStorageFormatConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		data           map[string]string
		expectedFormat string
		expectedError  bool
		errorContains  string
	}{
		{
			name:           "explicit legacy format",
			data:           map[string]string{ociStorageFormatKey: OCIFormatLegacy},
			expectedFormat: OCIFormatLegacy,
		},
		{
			name:           "explicit referrers-api format",
			data:           map[string]string{ociStorageFormatKey: OCIFormatReferrersAPI},
			expectedFormat: OCIFormatReferrersAPI,
		},
		{
			name:           "explicit protobuf-bundle format",
			data:           map[string]string{ociStorageFormatKey: OCIFormatProtobuf},
			expectedFormat: OCIFormatProtobuf,
		},
		{
			name:           "no format specified (should default to legacy)",
			data:           map[string]string{},
			expectedFormat: OCIFormatLegacy,
		},
		{
			name:           "empty format string (should default to legacy)",
			data:           map[string]string{ociStorageFormatKey: ""},
			expectedFormat: OCIFormatLegacy,
		},
		{
			name:          "invalid format",
			data:          map[string]string{ociStorageFormatKey: "invalid"},
			expectedError: true,
			errorContains: "invalid storage.oci.format: invalid",
		},
		{
			name: "format with other OCI config",
			data: map[string]string{
				ociStorageFormatKey:      OCIFormatReferrersAPI,
				ociRepositoryKey:         "example.com/repo",
				ociRepositoryInsecureKey: "true",
			},
			expectedFormat: OCIFormatReferrersAPI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfigFromMap(tt.data)

			if tt.expectedError {
				if err == nil {
					t.Fatalf("NewConfigFromMap() expected error but got none")
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("NewConfigFromMap() error = %v, expected to contain %q", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewConfigFromMap() unexpected error = %v", err)
			}

			if cfg.Storage.OCI.Format != tt.expectedFormat {
				t.Errorf("Storage.OCI.Format = %q, expected %q", cfg.Storage.OCI.Format, tt.expectedFormat)
			}
		})
	}
}

func TestMigrateOCIConfig(t *testing.T) {
	tests := []struct {
		name           string
		data           map[string]string
		initialConfig  *Config
		expectedFormat string
		description    string
	}{
		{
			name: "migrate from referrers-api true to protobuf format",
			data: map[string]string{
				ociReferrersAPIKey: "true",
			},
			initialConfig: &Config{
				Storage: StorageConfigs{
					OCI: OCIStorageConfig{}, // Empty format should trigger migration
				},
			},
			expectedFormat: OCIFormatProtobuf, // Legacy behavior was protobuf bundle
			description:    "When referrers-api=true and no format set, should migrate to protobuf-bundle format",
		},
		{
			name: "migrate from referrers-api false to legacy format",
			data: map[string]string{
				ociReferrersAPIKey: "false",
			},
			initialConfig: &Config{
				Storage: StorageConfigs{
					OCI: OCIStorageConfig{}, // Empty format should trigger migration
				},
			},
			expectedFormat: OCIFormatLegacy,
			description:    "When referrers-api=false and no format set, should migrate to legacy",
		},
		{
			name: "no format migration when format already set",
			data: map[string]string{
				ociReferrersAPIKey:  "true",
				ociStorageFormatKey: OCIFormatReferrersAPI,
			},
			initialConfig: &Config{
				Storage: StorageConfigs{
					OCI: OCIStorageConfig{
						Format: OCIFormatReferrersAPI, // Format already set
					},
				},
			},
			expectedFormat: OCIFormatReferrersAPI, // Should not be overridden
			description:    "When format already set, should not migrate format from referrers-api boolean",
		},
		{
			name: "no migration when neither setting present",
			data: map[string]string{
				// Neither format nor referrers-api set
			},
			initialConfig: &Config{
				Storage: StorageConfigs{
					OCI: OCIStorageConfig{}, // Empty
				},
			},
			expectedFormat: "", // Should remain empty before default assignment
			description:    "When neither format nor referrers-api set, no migration should occur",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the initial config to avoid modifying the test case
			cfg := &Config{
				Storage: StorageConfigs{
					OCI: OCIStorageConfig{
						Format: tt.initialConfig.Storage.OCI.Format,
					},
				},
			}

			// Apply migration
			migrateOCIConfig(tt.data, cfg)

			if cfg.Storage.OCI.Format != tt.expectedFormat {
				t.Errorf("%s: Format = %q, expected %q", tt.description, cfg.Storage.OCI.Format, tt.expectedFormat)
			}
		})
	}
}

func TestBackwardCompatibilityIntegration(t *testing.T) {
	// Integration tests that verify the complete migration flow through NewConfigFromMap
	tests := []struct {
		name           string
		data           map[string]string
		expectedFormat string
		description    string
	}{
		{
			name: "legacy config migration: referrers-api=true",
			data: map[string]string{
				ociReferrersAPIKey: "true",
			},
			expectedFormat: OCIFormatProtobuf,
			description:    "Legacy referrers-api=true should migrate to protobuf-bundle format",
		},
		{
			name: "legacy config migration: referrers-api=false",
			data: map[string]string{
				ociReferrersAPIKey: "false",
			},
			expectedFormat: OCIFormatLegacy,
			description:    "Legacy referrers-api=false should migrate to legacy format",
		},
		{
			name: "new format takes precedence over legacy setting",
			data: map[string]string{
				ociStorageFormatKey: OCIFormatReferrersAPI,
				ociReferrersAPIKey:  "false", // This should be ignored
			},
			expectedFormat: OCIFormatReferrersAPI,
			description:    "New format setting should take precedence over legacy boolean",
		},
		{
			name: "explicit format configuration",
			data: map[string]string{
				ociStorageFormatKey: OCIFormatProtobuf,
			},
			expectedFormat: OCIFormatProtobuf,
			description:    "Explicit format setting should be preserved",
		},
		{
			name:           "default configuration",
			data:           map[string]string{},
			expectedFormat: OCIFormatLegacy, // Should default to legacy
			description:    "Default configuration should use legacy format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfigFromMap(tt.data)
			if err != nil {
				t.Fatalf("NewConfigFromMap() unexpected error = %v", err)
			}

			if cfg.Storage.OCI.Format != tt.expectedFormat {
				t.Errorf("%s: Format = %q, expected %q", tt.description, cfg.Storage.OCI.Format, tt.expectedFormat)
			}
		})
	}
}

func TestOCIFormatConfigMap(t *testing.T) {
	// Test that configuration works through ConfigMap interface as well
	tests := []struct {
		name           string
		configMapData  map[string]string
		expectedFormat string
		expectedError  bool
	}{
		{
			name: "configmap with new format",
			configMapData: map[string]string{
				ociStorageFormatKey: OCIFormatReferrersAPI,
			},
			expectedFormat: OCIFormatReferrersAPI,
		},
		{
			name: "configmap with legacy boolean",
			configMapData: map[string]string{
				ociReferrersAPIKey: "true",
			},
			expectedFormat: OCIFormatProtobuf,
		},
		{
			name: "configmap with invalid format",
			configMapData: map[string]string{
				ociStorageFormatKey: "invalid-format",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "chains-config",
				},
				Data: tt.configMapData,
			}

			cfg, err := NewConfigFromConfigMap(cm)

			if tt.expectedError {
				if err == nil {
					t.Fatalf("NewConfigFromConfigMap() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewConfigFromConfigMap() unexpected error = %v", err)
			}

			if cfg.Storage.OCI.Format != tt.expectedFormat {
				t.Errorf("Storage.OCI.Format = %q, expected %q", cfg.Storage.OCI.Format, tt.expectedFormat)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && someContains(s, substr)))
}

func someContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
