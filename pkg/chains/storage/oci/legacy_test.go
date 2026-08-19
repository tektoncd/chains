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
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				imageName:        "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:bc4f7468f87486e3835b09098c74cd7f54db2cf697cbb9b824271b95a2d0871e",
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

// TestK8schainOptions_IgnoresImagePullSecrets guards against regressing to the
// bug reported in https://github.com/tektoncd/chains/issues/1336: if the
// ServiceAccount running the TaskRun/PipelineRun has both imagePullSecrets and
// mounted secrets for the same registry, Chains must not let the read-only
// imagePullSecret shadow the push-capable mounted secret.
func TestK8schainOptions_IgnoresImagePullSecrets(t *testing.T) {
	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: "my-taskrun", Namespace: "my-namespace"},
		Spec:       v1.TaskRunSpec{ServiceAccountName: "my-sa"},
	}
	obj := objects.NewTaskRunObjectV1(tr)

	opts := k8schainOptions(obj)

	assert.Equal(t, "my-namespace", opts.Namespace)
	assert.Equal(t, "my-sa", opts.ServiceAccountName)
	assert.True(t, opts.UseMountSecrets)
	assert.True(t, opts.IgnorePullSecrets, "chains only pushes artifacts, so imagePullSecrets must be ignored in favor of the ServiceAccount's mounted (push-capable) secrets")
}
