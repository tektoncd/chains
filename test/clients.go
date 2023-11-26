//go:build e2e
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
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/pipeline/pkg/names"

	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	knativetest "knative.dev/pkg/test"
)

// clients holds instances of interfaces for making requests to the Pipeline controllers.
type clients struct {
	KubeClient     kubernetes.Interface
	PipelineClient pipelineclientset.Interface
	secret         secret
	// insecure registry available from within the cluster
	internalRegistry string
}

// newClients instantiates and returns several clientsets required for making requests to the
// cluster specified by the combination of clusterName and configPath.
func newClients(t *testing.T, configPath, clusterName string) *clients {
	t.Helper()
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

type setupOpts struct {
	useCosignSigner bool
	registry        bool
	kanikoTaskImage string
	ns              string
}

func setup(ctx context.Context, t *testing.T, opts setupOpts) (*clients, string, func()) {
	t.Helper()

	c := newClients(t, knativetest.Flags.Kubeconfig, knativetest.Flags.Cluster)
	namespace := "default"
	if opts.ns == "" {
		namespace = names.SimpleNameGenerator.RestrictLengthWithRandomSuffix("earth")
		createNamespace(ctx, t, namespace, c.KubeClient)
	}

	c.secret = setupSecret(ctx, t, c.KubeClient, opts)
	if opts.registry {
		c.internalRegistry = createRegistry(ctx, t, namespace, c.KubeClient)
	}

	if opts.kanikoTaskImage != "" {
		imageDest := fmt.Sprintf("%s/%s", c.internalRegistry, opts.kanikoTaskImage)
		t.Logf("Creating Kaniko task referencing image %s", imageDest)
		task := kanikoTask(t, namespace, imageDest)
		if _, err := c.PipelineClient.TektonV1().Tasks(namespace).Create(ctx, task, metav1.CreateOptions{}); err != nil {
			t.Fatalf("error creating task: %s", err)
		}
	}

	var cleanup = func() {
		if namespace == "default" {
			return
		}
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
			Name:   namespace,
			Labels: map[string]string{"chains": "integration-testing"},
		},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create namespace %s for tests: %s", namespace, err)
	}
}

func createRegistry(ctx context.Context, t *testing.T, namespace string, kubeClient kubernetes.Interface) string {
	t.Helper()
	replicas := int32(1)
	label := map[string]string{"app": "registry"}
	meta := metav1.ObjectMeta{
		Name:      "registry",
		Namespace: namespace,
		Labels:    label,
	}
	deployment := &v1.Deployment{
		ObjectMeta: meta,
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: label},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: meta,
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "registry",
						Image: "registry:2.7.1@sha256:d5459fcb27aecc752520df4b492b08358a1912fcdfa454f7d2101d4b09991daa",
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 5000,
							},
						},
					}},
				},
			},
		},
	}
	service := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Selector: label,
			Ports:    []corev1.ServicePort{{Port: int32(5000), Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{IntVal: int32(5000)}}},
		},
	}
	// first, check if the svc already exists
	if svc, err := kubeClient.CoreV1().Services(namespace).Get(ctx, service.Name, metav1.GetOptions{}); err == nil {
		return fmt.Sprintf("%s.%s.svc.cluster.local:5000", svc.Name, svc.Namespace)
	}
	t.Logf("Creating insecure registry to deploy in ns %s", namespace)
	if _, err := kubeClient.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create deployment registry for tests: %s", err)
	}
	t.Logf("Exposing registry service")

	service, err := kubeClient.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create service for tests: %s", err)
	}

	return fmt.Sprintf("%s.%s.svc.cluster.local:5000", service.Name, service.Namespace)
}

type secret struct {
	x509priv   *signature.ECDSASignerVerifier
	cosignPriv signature.SignerVerifier
}

func setupSecret(ctx context.Context, t *testing.T, c kubernetes.Interface, opts setupOpts) secret {
	// Only overwrite the secret data if it isn't set.
	namespace := "tekton-chains"
	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "signing-secrets",
			Namespace: namespace,
		},
		StringData: map[string]string{},
	}

	// x509
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	b, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	s.StringData["x509.pem"] = string(pem.EncodeToMemory(&pem.Block{Bytes: b, Type: "PRIVATE KEY"}))

	x509Priv, err := signature.LoadECDSASignerVerifier(priv, crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}

	// cosign
	paths := []string{"cosign.key", "cosign.pub", "cosign.password"}
	for _, p := range paths {
		b, err := ioutil.ReadFile(filepath.Join("./testdata", p))
		if err != nil {
			t.Fatal(err)
		}
		s.StringData[p] = string(b)
	}
	cosignPriv, err := cosign.LoadPrivateKey([]byte(s.StringData["cosign.key"]), []byte(s.StringData["cosign.password"]))
	if err != nil {
		t.Fatal(err)
	}

	if opts.useCosignSigner {
		delete(s.StringData, "x509.pem")
	}
	if _, err := c.CoreV1().Secrets(namespace).Update(ctx, &s, metav1.UpdateOptions{}); err != nil {
		t.Fatal(err)
	}

	return secret{
		cosignPriv: cosignPriv,
		x509priv:   x509Priv,
	}
}
