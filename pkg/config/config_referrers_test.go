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

package config

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReferrersAPIMigrationConfig(t *testing.T) {
	tests := []struct {
		name           string
		data           map[string]string
		expectedFormat string
	}{
		{
			name: "referrers api enabled migrates to protobuf",
			data: map[string]string{
				"storage.oci.referrers-api": "true",
			},
			expectedFormat: OCIFormatProtobuf,
		},
		{
			name: "referrers api disabled migrates to legacy",
			data: map[string]string{
				"storage.oci.referrers-api": "false",
			},
			expectedFormat: OCIFormatLegacy,
		},
		{
			name:           "no referrers api setting defaults to legacy",
			data:           map[string]string{},
			expectedFormat: OCIFormatLegacy,
		},
		{
			name: "referrers api with other OCI config",
			data: map[string]string{
				"storage.oci.repository":    "example.com/repo",
				"storage.oci.referrers-api": "true",
			},
			expectedFormat: OCIFormatProtobuf,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "chains-config",
				},
				Data: test.data,
			}

			cfg, err := NewConfigFromConfigMap(cm)
			if err != nil {
				t.Fatalf("Failed to create config: %v", err)
			}

			if cfg.Storage.OCI.Format != test.expectedFormat {
				t.Errorf("Expected Format to be %v, got %v", test.expectedFormat, cfg.Storage.OCI.Format)
			}
		})
	}
}

func TestReferrersAPIMigrationFromMap(t *testing.T) {
	// Test that the config can be created from map data and migration works
	data := map[string]string{
		"storage.oci.referrers-api": "true",
		"storage.oci.repository":    "example.com/repo",
	}

	cfg, err := NewConfigFromMap(data)
	if err != nil {
		t.Fatalf("Failed to create config from map: %v", err)
	}

	if cfg.Storage.OCI.Format != OCIFormatProtobuf {
		t.Errorf("Expected Format to be %s, got %s", OCIFormatProtobuf, cfg.Storage.OCI.Format)
	}

	if cfg.Storage.OCI.Repository != "example.com/repo" {
		t.Errorf("Expected repository to be 'example.com/repo', got %s", cfg.Storage.OCI.Repository)
	}
}
