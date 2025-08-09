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

func TestReferrersAPIConfig(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]string
		expected bool
	}{
		{
			name: "referrers api enabled",
			data: map[string]string{
				"storage.oci.referrers-api": "true",
			},
			expected: true,
		},
		{
			name: "referrers api disabled",
			data: map[string]string{
				"storage.oci.referrers-api": "false",
			},
			expected: false,
		},
		{
			name:     "referrers api default (disabled)",
			data:     map[string]string{},
			expected: false,
		},
		{
			name: "referrers api with other OCI config",
			data: map[string]string{
				"storage.oci.repository":    "example.com/repo",
				"storage.oci.referrers-api": "true",
			},
			expected: true,
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

			if cfg.Storage.OCI.ReferrersAPI != test.expected {
				t.Errorf("Expected ReferrersAPI to be %v, got %v", test.expected, cfg.Storage.OCI.ReferrersAPI)
			}
		})
	}
}

func TestReferrersAPIConfigFromMap(t *testing.T) {
	// Test that the config can be created from map data
	data := map[string]string{
		"storage.oci.referrers-api": "true",
		"storage.oci.repository":    "example.com/repo",
	}

	cfg, err := NewConfigFromMap(data)
	if err != nil {
		t.Fatalf("Failed to create config from map: %v", err)
	}

	if !cfg.Storage.OCI.ReferrersAPI {
		t.Errorf("Expected ReferrersAPI to be enabled")
	}

	if cfg.Storage.OCI.Repository != "example.com/repo" {
		t.Errorf("Expected repository to be 'example.com/repo', got %s", cfg.Storage.OCI.Repository)
	}
}
