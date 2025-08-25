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
		disableSigning bool
		storageBackend sets.Set[string]
		want           bool
	}{
		{
			name:           "enabled by default with valid storage",
			disableSigning: false,
			storageBackend: sets.New[string]("oci"),
			want:           true,
		},
		{
			name:           "disabled when DisableSigning is true",
			disableSigning: true,
			storageBackend: sets.New[string]("oci"),
			want:           false,
		},
		{
			name:           "disabled when no storage backend",
			disableSigning: false,
			storageBackend: sets.New[string](""),
			want:           false,
		},
		{
			name:           "disabled when DisableSigning true and no storage",
			disableSigning: true,
			storageBackend: sets.New[string](""),
			want:           false,
		},
		{
			name:           "enabled with multiple storage backends",
			disableSigning: false,
			storageBackend: sets.New[string]("oci", "gcs"),
			want:           true,
		},
		{
			name:           "disabled with multiple storage backends but DisableSigning true",
			disableSigning: true,
			storageBackend: sets.New[string]("oci", "gcs"),
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact := &Artifact{
				DisableSigning: tt.disableSigning,
				StorageBackend: tt.storageBackend,
			}
			if got := artifact.Enabled(); got != tt.want {
				t.Errorf("Artifact.Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
