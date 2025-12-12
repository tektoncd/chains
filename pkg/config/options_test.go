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
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"reflect"
	"testing"
)

func TestStorageOpts_PublicKey(t *testing.T) {
	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	tests := []struct {
		name string
		opts StorageOpts
		want crypto.PublicKey
	}{
		{
			name: "StorageOpts with PublicKey set",
			opts: StorageOpts{
				FullKey:       "test-full-key",
				ShortKey:      "test-short",
				Cert:          "test-cert",
				Chain:         "test-chain",
				PublicKey:     publicKey,
				PayloadFormat: "tekton",
			},
			want: publicKey,
		},
		{
			name: "StorageOpts with nil PublicKey",
			opts: StorageOpts{
				FullKey:       "test-full-key",
				ShortKey:      "test-short",
				Cert:          "test-cert",
				Chain:         "test-chain",
				PublicKey:     nil,
				PayloadFormat: "tekton",
			},
			want: nil,
		},
		{
			name: "StorageOpts without PublicKey field populated",
			opts: StorageOpts{
				FullKey:       "test-full-key",
				ShortKey:      "test-short",
				Cert:          "test-cert",
				Chain:         "test-chain",
				PayloadFormat: "tekton",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.PublicKey != tt.want {
				t.Errorf("StorageOpts.PublicKey = %v, want %v", tt.opts.PublicKey, tt.want)
			}

			// Test that PublicKey field can be set and retrieved
			if tt.want != nil {
				// Verify the key is the expected type
				if _, ok := tt.opts.PublicKey.(*rsa.PublicKey); !ok {
					t.Errorf("Expected PublicKey to be *rsa.PublicKey, got %T", tt.opts.PublicKey)
				}

				// Verify the key matches our test key
				rsaKey, ok := tt.opts.PublicKey.(*rsa.PublicKey)
				if !ok {
					t.Fatalf("PublicKey is not *rsa.PublicKey")
				}
				expectedRSAKey, ok := tt.want.(*rsa.PublicKey)
				if !ok {
					t.Fatalf("Expected key is not *rsa.PublicKey")
				}

				if rsaKey.N.Cmp(expectedRSAKey.N) != 0 || rsaKey.E != expectedRSAKey.E {
					t.Errorf("PublicKey does not match expected key")
				}
			}
		})
	}
}

func TestStorageOpts_JSON_Serialization(t *testing.T) {
	// Test that StorageOpts can handle JSON serialization/deserialization
	// Note: crypto.PublicKey cannot be directly JSON marshaled, but we test
	// that the struct can be marshaled with nil PublicKey field
	tests := []struct {
		name    string
		opts    StorageOpts
		wantErr bool
	}{
		{
			name: "StorageOpts with nil PublicKey",
			opts: StorageOpts{
				FullKey:       "test-full-key",
				ShortKey:      "test-short",
				Cert:          "test-cert",
				Chain:         "test-chain",
				PublicKey:     nil,
				PayloadFormat: "tekton",
			},
			wantErr: false,
		},
		{
			name: "Empty StorageOpts",
			opts: StorageOpts{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Unmarshal from JSON
				var decoded StorageOpts
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Errorf("json.Unmarshal() error = %v", err)
					return
				}

				// Compare non-PublicKey fields (PublicKey will be nil after JSON round-trip)
				if decoded.FullKey != tt.opts.FullKey ||
					decoded.ShortKey != tt.opts.ShortKey ||
					decoded.Cert != tt.opts.Cert ||
					decoded.Chain != tt.opts.Chain ||
					decoded.PayloadFormat != tt.opts.PayloadFormat {
					t.Errorf("JSON round-trip failed, decoded = %+v, want %+v", decoded, tt.opts)
				}

				// PublicKey should be nil after JSON round-trip since crypto.PublicKey
				// cannot be JSON marshaled directly
				if decoded.PublicKey != nil {
					t.Errorf("Expected PublicKey to be nil after JSON round-trip, got %v", decoded.PublicKey)
				}
			}
		})
	}
}

func TestStorageOpts_FieldTypes(t *testing.T) {
	// Test that all fields have the expected types
	opts := StorageOpts{}

	// Get the type reflection
	optsType := reflect.TypeOf(opts)

	expectedFields := map[string]string{
		"FullKey":       "string",
		"ShortKey":      "string",
		"Cert":          "string",
		"Chain":         "string",
		"PublicKey":     "crypto.PublicKey",
		"PayloadFormat": "config.PayloadType",
	}

	for fieldName, expectedType := range expectedFields {
		field, found := optsType.FieldByName(fieldName)
		if !found {
			t.Errorf("Field %s not found in StorageOpts", fieldName)
			continue
		}

		actualType := field.Type.String()
		if actualType != expectedType {
			t.Errorf("Field %s has type %s, expected %s", fieldName, actualType, expectedType)
		}
	}
}

func TestStorageOpts_Copy(t *testing.T) {
	// Generate a test key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	publicKey := &privateKey.PublicKey

	original := StorageOpts{
		FullKey:       "test-full-key",
		ShortKey:      "test-short",
		Cert:          "test-cert",
		Chain:         "test-chain",
		PublicKey:     publicKey,
		PayloadFormat: "tekton",
	}

	// Test shallow copy behavior
	copy := original

	// Modify the copy's string fields
	copy.FullKey = "modified-full-key"
	copy.ShortKey = "modified-short"

	// Original should be unchanged for string fields
	if original.FullKey != "test-full-key" {
		t.Errorf("Original FullKey changed after copy modification")
	}
	if original.ShortKey != "test-short" {
		t.Errorf("Original ShortKey changed after copy modification")
	}

	// PublicKey should point to the same object (reference copy)
	if original.PublicKey != copy.PublicKey {
		t.Errorf("PublicKey references should be the same after copy")
	}
}