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
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/chains/pkg/config"
)

func TestNewRepo(t *testing.T) {
	t.Run("Use any registry in storage oci repository", func(t *testing.T) {
		cfg := config.Config{}
		cfg.Storage.OCI.Repository = "example.com/foo"
		tests := []struct {
			imageName        string
			expectedRepoName string
		}{
			{
				imageName:        "ghcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:bc4f7468f87486e3835b09098c74cd7f54db2cf697cbb9b824271b95a2d0871e",
				expectedRepoName: "example.com/foo",
			},
			{
				imageName:        "foo.io/bar/kaniko-chains@sha256:bc4f7468f87486e3835b09098c74cd7f54db2cf697cbb9b824271b95a2d0871e",
				expectedRepoName: "example.com/foo",
			},
			{
				imageName:        "registry.com/spam/spam/spam/spam/spam/spam@sha256:bc4f7468f87486e3835b09098c74cd7f54db2cf697cbb9b824271b95a2d0871e",
				expectedRepoName: "example.com/foo",
			},
		}

		for _, test := range tests {
			ref, err := name.NewDigest(test.imageName)
			if err != nil {
				t.Error(err)
			}
			repo, err := newRepo(cfg, ref)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, repo.Name(), test.expectedRepoName)
		}
	})
}
