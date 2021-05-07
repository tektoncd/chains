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
	"os/exec"
	"path/filepath"
	"testing"
	"time"

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
	registry       string
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

	c.secret = setupSecret(ctx, t, c.KubeClient)
	c.registry = createRegistry(ctx, t, c.KubeClient)

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

type secret struct {
	x509Priv *ecdsa.PrivateKey
}

func createRegistry(ctx context.Context, t *testing.T, kubeClient kubernetes.Interface) string {
	t.Helper()
	namespace := "tekton-pipelines"
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
					Containers: []corev1.Container{
						{
							Name:  "registry",
							Image: "registry:2.7.1@sha256:d5459fcb27aecc752520df4b492b08358a1912fcdfa454f7d2101d4b09991daa",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5000,
								},
							},
						},
					},
				},
			},
		},
	}
	service := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Selector:              label,
			ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
			Type:                  corev1.ServiceTypeLoadBalancer,
			Ports:                 []corev1.ServicePort{{Port: int32(5000), Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{IntVal: int32(5000)}}},
		},
	}
	t.Logf("Creating insecure registry to deploy in ns %s", namespace)
	// first, check if the svc already exists
	if svc, err := kubeClient.CoreV1().Services(namespace).Get(ctx, service.Name, metav1.GetOptions{}); err == nil {
		if ingress := svc.Status.LoadBalancer.Ingress; ingress != nil {
			if ingress[0].IP != "" {
				return fmt.Sprintf("%s:5000", ingress[0].IP)
			}
		}
	}

	if _, err := kubeClient.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
		t.Fatalf("Failed to create deployment registry for tests: %s", err)
	}
	t.Logf("Exposing registry service")

	service, err := kubeClient.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create service for tests: %s", err)
	}

	t.Logf("Waiting for external service IP to be exposed...")
	return waitForExternalIP(ctx, t, service, 2*time.Minute, kubeClient)
}

func waitForExternalIP(ctx context.Context, t *testing.T, service *corev1.Service, timeout time.Duration, c kubernetes.Interface) string {
	t.Helper()
	w, err := c.CoreV1().Services(service.Namespace).Watch(ctx, metav1.SingleObject(metav1.ObjectMeta{
		Name:      service.Name,
		Namespace: service.Namespace,
	}))
	if err != nil {
		t.Errorf("error watching taskrun: %s", err)
	}
	// Setup a timeout channel
	timeoutChan := make(chan struct{})
	go func() {
		time.Sleep(timeout)
		timeoutChan <- struct{}{}
	}()

	// Wait for the condition to be true or a timeout
	for {
		select {
		case ev := <-w.ResultChan():
			tr := ev.Object.(*corev1.Service)
			if ingress := tr.Status.LoadBalancer.Ingress; ingress != nil {
				if ingress[0].IP != "" {
					return fmt.Sprintf("%s:5000", ingress[0].IP)
				}
			}
		case <-timeoutChan:
			output, err := exec.Command("kubectl", "get", "svc", "-A").CombinedOutput()
			t.Logf("ERROR creating registry, time out:%v\n%s", err, string(output))
			return ""
		}
	}
}

func setupSecret(ctx context.Context, t *testing.T, c kubernetes.Interface) secret {
	// Only overwrite the secret data if it isn't set.

	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "signing-secrets",
			Namespace: "tekton-pipelines",
		},
		StringData: map[string]string{},
	}
	// pgp
	paths := []string{"pgp.private-key", "pgp.passphrase", "pgp.public-key"}
	for _, p := range paths {
		b, err := ioutil.ReadFile(filepath.Join("./testdata", p))
		if err != nil {
			t.Error(err)
		}
		s.StringData[p] = string(b)
	}

	// x509
	_, priv := ecdsaKeyPair(t)

	s.StringData["x509.pem"] = toPem(t, priv)

	if _, err := c.CoreV1().Secrets("tekton-pipelines").Update(ctx, &s, metav1.UpdateOptions{}); err != nil {
		t.Error(err)
	}
	time.Sleep(60 * time.Second)
	return secret{
		x509Priv: priv,
	}
}

func ecdsaKeyPair(t *testing.T) (crypto.PublicKey, *ecdsa.PrivateKey) {
	kp, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	return kp.PublicKey, kp
}

func toPem(t *testing.T, priv *ecdsa.PrivateKey) string {
	b, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	p := pem.EncodeToMemory(&pem.Block{
		Bytes: b,
		Type:  "PRIVATE KEY",
	})
	return string(p)
}
