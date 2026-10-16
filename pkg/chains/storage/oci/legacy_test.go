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
	"context"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8s "k8s.io/client-go/kubernetes/fake"
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

// TestGetAuthenticator_ToleratesMalformedSecret guards against SRVKP-9199: a
// single malformed secret linked to the PipelineRun/TaskRun's ServiceAccount
// must not abort keychain construction for the remaining valid secrets, which
// silently blocked all signing/attestation uploads. This relies on the
// go-containerregistry keychain skipping (rather than failing on) secrets whose
// docker-config data cannot be decoded.
func TestGetAuthenticator_ToleratesMalformedSecret(t *testing.T) {
	ns := "my-namespace"
	saName := "my-sa"

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns},
		Secrets: []corev1.ObjectReference{
			{Name: "good-secret"},
			{Name: "bad-secret"},
		},
	}
	goodSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "good-secret", Namespace: ns},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{"example.com":{"auth":"dXNlcjpwYXNz"}}}`),
		},
	}
	badSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "bad-secret", Namespace: ns},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{ this is not valid docker config json `),
		},
	}

	client := fakek8s.NewSimpleClientset(sa, goodSecret, badSecret)

	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: "my-taskrun", Namespace: ns},
		Spec:       v1.TaskRunSpec{ServiceAccountName: saName},
	}
	obj := objects.NewTaskRunObjectV1(tr)

	ctx := context.Background()
	b := NewStorageBackend(ctx, client, config.Config{})

	_, err := b.getAuthenticator(ctx, obj, client)
	assert.NoError(t, err, "a malformed secret linked to the ServiceAccount must not block keychain construction from the valid secrets")
}

// TestGetAuthenticator_FallsBackToAmbientWhenNoValidSecret guards the
// SRVKP-9199 fix against a regression introduced by the conforma keychain:
// when a ServiceAccount has mounted secrets but none carry usable registry
// credentials (e.g. only an opaque/token secret), keychain construction must
// NOT hard-fail and block signing. Chains falls back to the ambient/cloud
// credential providers, matching the behavior before the dependency swap.
func TestGetAuthenticator_FallsBackToAmbientWhenNoValidSecret(t *testing.T) {
	ns := "my-namespace"
	saName := "my-sa"

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: ns},
		Secrets:    []corev1.ObjectReference{{Name: "token-secret"}},
	}
	tokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "token-secret", Namespace: ns},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"token": []byte("not-a-docker-config")},
	}

	client := fakek8s.NewSimpleClientset(sa, tokenSecret)

	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: "my-taskrun", Namespace: ns},
		Spec:       v1.TaskRunSpec{ServiceAccountName: saName},
	}
	obj := objects.NewTaskRunObjectV1(tr)

	ctx := context.Background()
	b := NewStorageBackend(ctx, client, config.Config{})

	_, err := b.getAuthenticator(ctx, obj, client)
	assert.NoError(t, err, "a ServiceAccount with no usable registry secret must fall back to ambient credentials, not block signing")
}

// scenarioNS is the namespace shared by the secret helpers and scenario matrix.
const scenarioNS = "ns"

func dockerCfgJSONSecret(name, data string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: scenarioNS},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{corev1.DockerConfigJsonKey: []byte(data)},
	}
}

func legacyDockerCfgSecret(name, data string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: scenarioNS},
		Type:       corev1.SecretTypeDockercfg,
		Data:       map[string][]byte{corev1.DockerConfigKey: []byte(data)},
	}
}

func opaqueSecret(name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: scenarioNS},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"token": []byte("not-a-docker-config")},
	}
}

// TestGetAuthenticator_SecretScenarios exercises the keychain construction path
// across a matrix of ServiceAccount mounted-secret configurations. It is
// intentionally observational: it never fails, it just logs the outcome
// (OK/ERROR) per scenario so the same test can be run with and without the fix
// (conforma fork) to compare behavior. Run with: go test -v -run SecretScenarios
func TestGetAuthenticator_SecretScenarios(t *testing.T) {
	const validJSON = `{"auths":{"example.com":{"auth":"dXNlcjpwYXNz"}}}`
	const badJSON = `{ not valid json `
	const validLegacy = `{"example.com":{"auth":"dXNlcjpwYXNz"}}`
	const badLegacy = `{ not valid `
	const badURLEntry = `{"auths":{"htt p://bad url":{"auth":"dXNlcjpwYXNz"}}}`

	tests := []struct {
		name    string
		secrets []*corev1.Secret
	}{
		{"good_only", []*corev1.Secret{dockerCfgJSONSecret("good", validJSON)}},
		{"good_plus_malformed_json", []*corev1.Secret{dockerCfgJSONSecret("good", validJSON), dockerCfgJSONSecret("bad", badJSON)}},
		{"malformed_json_only", []*corev1.Secret{dockerCfgJSONSecret("bad", badJSON)}},
		{"good_plus_bad_url_entry", []*corev1.Secret{dockerCfgJSONSecret("good", validJSON), dockerCfgJSONSecret("badurl", badURLEntry)}},
		{"good_plus_malformed_legacy_dockercfg", []*corev1.Secret{dockerCfgJSONSecret("good", validJSON), legacyDockerCfgSecret("badlegacy", badLegacy)}},
		{"valid_legacy_dockercfg_only", []*corev1.Secret{legacyDockerCfgSecret("legacy", validLegacy)}},
		{"opaque_secret_only", []*corev1.Secret{opaqueSecret("token")}},
		{"opaque_plus_good", []*corev1.Secret{opaqueSecret("token"), dockerCfgJSONSecret("good", validJSON)}},
		{"no_secrets", nil},
		{"two_malformed_plus_one_good", []*corev1.Secret{dockerCfgJSONSecret("bad1", badJSON), dockerCfgJSONSecret("bad2", badJSON), dockerCfgJSONSecret("good", validJSON)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa", Namespace: scenarioNS}}
			runtimeObjs := []runtime.Object{sa}
			for _, s := range tc.secrets {
				sa.Secrets = append(sa.Secrets, corev1.ObjectReference{Name: s.Name})
				runtimeObjs = append(runtimeObjs, s)
			}
			client := fakek8s.NewSimpleClientset(runtimeObjs...)

			tr := &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Name: "tr", Namespace: scenarioNS},
				Spec:       v1.TaskRunSpec{ServiceAccountName: "sa"},
			}
			obj := objects.NewTaskRunObjectV1(tr)

			ctx := context.Background()
			b := NewStorageBackend(ctx, client, config.Config{})

			_, err := b.getAuthenticator(ctx, obj, client)
			if err != nil {
				t.Logf("SCENARIO %-40s => ERROR: %v", tc.name, err)
			} else {
				t.Logf("SCENARIO %-40s => OK (keychain built)", tc.name)
			}
		})
	}
}
