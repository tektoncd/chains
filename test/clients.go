// +build e2e

/*
Copyright 2020 Tekton Authors LLC
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

/*
Get access to client objects
To initialize client objects you can use the newClients function. It returns a clients struct
that contains initialized clients for accessing:
	- Kubernetes resources
	- Pipelines resources (https://github.com/tektoncd/pipeline)
	- Triggers resources (https://github.com/tektoncd/triggers)
*/

package test

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/tektoncd/pipeline/pkg/names"

	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	knativetest "knative.dev/pkg/test"
)

// clients holds instances of interfaces for making requests to the Pipeline controllers.
type clients struct {
	KubeClient kubernetes.Interface

	PipelineClient pipelineclientset.Interface
}

// newClients instantiates and returns several clientsets required for making requests to the
// cluster specified by the combination of clusterName and configPath.
func newClients(t *testing.T, configPath, clusterName string) *clients {
	t.Helper()
	var err error
	c := &clients{}

	cfg, err := knativetest.BuildClientConfig(configPath, clusterName)
	if err != nil {
		t.Fatalf("Failed to create configuration obj from %s for cluster %s: %s", configPath, clusterName, err)
	}

	c.KubeClient, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create kubernetes clientset from config file at %s: %s", configPath, err)
	}

	c.PipelineClient, err = pipelineclientset.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create pipeline clientset from config file at %s: %s", configPath, err)
	}
	return c
}

func setup(ctx context.Context, t *testing.T) (*clients, string, func()) {
	t.Helper()
	namespace := names.SimpleNameGenerator.RestrictLengthWithRandomSuffix("earth")

	c := newClients(t, knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster)
	createNamespace(ctx, t, namespace, c.KubeClient)

	setupSecret(ctx, t, c.KubeClient)

	var cleanup = func() {
		t.Logf("Deleting namespace %s", namespace)
		if err := c.KubeClient.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
			t.Fatalf("Failed to delete namespace %s for tests: %s", namespace, err)
		}
	}
	return c, namespace, cleanup
}

func createNamespace(ctx context.Context, t *testing.T, namespace string, kubeClient kubernetes.Interface) {
	t.Helper()
	t.Logf("Create namespace %s to deploy to", namespace)
	if _, err := kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create namespace %s for tests: %s", namespace, err)
	}
}

func setupSecret(ctx context.Context, t *testing.T, c kubernetes.Interface) {
	// Only overwrite the secret data if it isn't set.

	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "signing-secrets",
			Namespace: "tekton-pipelines",
		},
		StringData: map[string]string{},
	}
	paths := []string{"pgp.private-key", "pgp.passphrase", "pgp.public-key"}
	for _, p := range paths {
		b, err := ioutil.ReadFile(filepath.Join("./testdata", p))
		if err != nil {
			t.Error(err)
		}
		s.StringData[p] = string(b)
	}
	if _, err := c.CoreV1().Secrets("tekton-pipelines").Update(ctx, &s, metav1.UpdateOptions{}); err != nil {
		t.Error(err)
	}
	time.Sleep(60 * time.Second)
}
