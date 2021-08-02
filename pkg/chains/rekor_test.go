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

package chains

import (
	"testing"

	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldUploadTlog(t *testing.T) {
	tests := []struct {
		description string
		cfg         config.TransparencyConfig
		annotations map[string]string
		expected    bool
	}{
		{
			description: "transparency disabled",
			cfg: config.TransparencyConfig{
				Enabled:          false,
				VerifyAnnotation: true,
			},
			annotations: map[string]string{RekorAnnotation: "true"},
			expected:    false,
		},
		{
			description: "transparency enabled, verify annotation disabled",
			cfg: config.TransparencyConfig{
				Enabled:          true,
				VerifyAnnotation: false,
			},
			expected: true,
		},
		{
			description: "annotation doesn't exist",
			cfg: config.TransparencyConfig{
				Enabled:          true,
				VerifyAnnotation: true,
			},
			annotations: map[string]string{RekorAnnotation: "f"},
			expected:    false,
		},
		{
			description: "annotation exists",
			cfg: config.TransparencyConfig{
				Enabled:          true,
				VerifyAnnotation: true,
			},
			annotations: map[string]string{RekorAnnotation: "true"},
			expected:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tr := &v1beta1.TaskRun{
				ObjectMeta: v1.ObjectMeta{
					Annotations: test.annotations,
				},
			}
			cfg := config.Config{Transparency: test.cfg}
			got := shouldUploadTlog(cfg, tr)
			if got != test.expected {
				t.Fatalf("got (%v) doesn't match expected (%v)", got, test.expected)
			}
		})
	}
}
