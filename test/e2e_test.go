// +build e2e

/*
Copyright 2019 The Tekton Authors

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

package test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/sigstore/pkg/signature"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestInstall(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, _, cleanup := setup(ctx, t, setupOpts{})
	defer cleanup()
	dep, err := c.KubeClient.AppsV1().Deployments("tekton-chains").Get(ctx, "tekton-chains-controller", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Error getting chains deployment: %v", err)
	}
	if dep.Status.AvailableReplicas != 1 {
		t.Errorf("Chains installation not running correctly: %v", dep)
	}
}

func TestTektonStorage(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{})
	defer cleanup()

	// Setup the right config.
	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.signer": "pgp",
		"artifacts.oci.format":     "tekton",
		"artifacts.oci.storage":    "tekton",
		"artifacts.oci.signer":     "pgp",
	})
	defer resetConfig()

	tr, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(ctx, &imageTaskRun, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("error creating taskrun: %s", err)
	}

	// Give it a minute to complete.
	waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, done, 60*time.Second)

	// It can take up to a minute for the secret data to be updated!
	tr = waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, signed, 120*time.Second)

	// Let's fetch the signature and body:

	sigKey := fmt.Sprintf("chains.tekton.dev/signature-taskrun-%s", tr.UID)
	payloadKey := fmt.Sprintf("chains.tekton.dev/payload-taskrun-%s", tr.UID)
	signature, body := tr.Annotations[sigKey], tr.Annotations[payloadKey]
	// base64 decode them
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		t.Error(err)
	}
	bodyBytes, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		t.Error(err)
	}

	checkPgpSignatures(t, sigBytes, bodyBytes)
}

func TestOCISigning(t *testing.T) {
	tests := []struct {
		name string
		opts setupOpts
	}{
		{
			name: "x509",
			opts: setupOpts{},
		}, {
			name: "cosign",
			opts: setupOpts{useCosignSigner: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, test.opts)
			defer cleanup()

			// Setup the right config.
			resetConfig := setConfigMap(ctx, t, c, map[string]string{"artifacts.oci.storage": "tekton"})

			defer resetConfig()

			tr, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(ctx, &imageTaskRun, metav1.CreateOptions{})
			if err != nil {
				t.Errorf("error creating taskrun: %s", err)
			}
			t.Logf("Created TaskRun: %s", tr.Name)

			// Give it a minute to complete.
			waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, done, 60*time.Second)

			// It can take up to a minute for the secret data to be updated!
			tr = waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, signed, 120*time.Second)

			// Let's fetch the signature and body:
			t.Log(tr.Annotations)

			signature, body := tr.Annotations["chains.tekton.dev/signature-05f95b26ed10"], tr.Annotations["chains.tekton.dev/payload-05f95b26ed10"]
			// base64 decode them
			sigBytes, err := base64.StdEncoding.DecodeString(signature)
			if err != nil {
				t.Error(err)
			}
			bodyBytes, err := base64.StdEncoding.DecodeString(body)
			if err != nil {
				t.Error(err)
			}

			pub := &c.secret.x509Priv.PublicKey
			if test.name == "cosign" {
				pub = &c.secret.cosignPriv.PublicKey
			}
			h := sha256.Sum256(bodyBytes)
			if !ecdsa.VerifyASN1(pub, h[:], sigBytes) {
				t.Error("invalid signature")
			}
		})
	}
}

func TestGCSStorage(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	if metadata.OnGCE() {
		t.Skip("Skipping, integration tests do not support GCS secrets yet.")
	}
	c, ns, cleanup := setup(ctx, t, setupOpts{})
	defer cleanup()

	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Error(err)
	}
	bucketName, rmBucket := makeBucket(t, client)
	defer rmBucket()

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.storage": "gcs",
		"storage.gcs.bucket":        bucketName,
		"artifacts.taskrun.signer":  "pgp",
	})
	defer resetConfig()
	time.Sleep(3 * time.Second)

	tr, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(ctx, &simpleTaskRun, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("error creating taskrun: %s", err)
	}

	// Give it a minute to complete.
	waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, done, 60*time.Second)

	// It can take up to a minute for the secret data to be updated!
	tr = waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, signed, 120*time.Second)

	sigName := fmt.Sprintf("taskrun-%s-%s-%s/taskrun.signature", tr.Namespace, tr.Name, tr.UID)
	payloadName := fmt.Sprintf("taskrun-%s-%s-%s/taskrun.payload", tr.Namespace, tr.Name, tr.UID)

	sigBytes := readObj(t, bucketName, sigName, client)
	bodyBytes := readObj(t, bucketName, payloadName, client)

	checkPgpSignatures(t, sigBytes, bodyBytes)
}

func TestOCIStorage(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{registry: true})
	defer cleanup()

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"storage.oci.repository.insecure": "true",
	})
	defer resetConfig()
	time.Sleep(3 * time.Second)

	// create necessary resources
	imageName := "chains-test-oci-storage"
	image := fmt.Sprintf("%s/%s", c.internalRegistry, imageName)
	task := kanikoTask(t, ns, image)

	if _, err := c.PipelineClient.TektonV1beta1().Tasks(ns).Create(ctx, task, metav1.CreateOptions{}); err != nil {
		t.Fatalf("error creating task: %s", err)
	}

	taskRun := kanikoTaskRun(ns)
	tr, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(ctx, taskRun, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("error creating taskrun: %s", err)
	}

	// Give it a minute to complete.
	waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, done, 60*time.Second)

	pubKey := signature.ECDSAVerifier{Key: &c.secret.x509Priv.PublicKey, HashAlg: crypto.SHA256}
	externalRef, err := name.ParseReference(fmt.Sprintf("%s/%s", c.externalRegistry, imageName), name.Insecure)
	if err != nil {
		t.Fatalf("parsing ref: %v", err)
	}
	t.Logf("Verifying %s...", externalRef.String())
	// wait two minutes for the controller to sign
	// setup a timeout channel
	timeoutChan := make(chan struct{})
	go func() {
		time.Sleep(2 * time.Minute)
		timeoutChan <- struct{}{}
	}()
	for {
		select {
		default:
			if _, err = cosign.Verify(ctx, externalRef, &cosign.CheckOpts{
				PubKey: pubKey,
			}, ""); err != nil {
				t.Log(err)
			} else {
				return
			}
		case <-timeoutChan:
			t.Fatal("time out")
		}
	}
}

var imageTaskSpec = v1beta1.TaskSpec{
	Steps: []v1beta1.Step{
		{
			Container: corev1.Container{
				Image: "busybox",
			},
			Script: `set -e
cat <<EOF > $(outputs.resources.image.path)/index.json
{
	"schemaVersion": 2,
	"manifests": [
	{
		"mediaType": "application/vnd.oci.image.index.v1+json",
		"size": 314,
		"digest": "sha256:05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b5"
	}
	]
}
`,
		},
	},
	Resources: &v1beta1.TaskResources{
		Outputs: []v1beta1.TaskResource{
			{
				ResourceDeclaration: v1alpha1.ResourceDeclaration{
					Name: "image",
					Type: "image",
				},
			},
		},
	},
}

var imageTaskRun = v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		GenerateName: "image-task",
	},
	Spec: v1beta1.TaskRunSpec{
		TaskSpec: &imageTaskSpec,
		Resources: &v1beta1.TaskRunResources{
			Outputs: []v1beta1.TaskResourceBinding{
				{
					PipelineResourceBinding: v1beta1.PipelineResourceBinding{
						Name: "image",
						ResourceSpec: &v1alpha1.PipelineResourceSpec{
							Type: "image",
							Params: []v1alpha1.ResourceParam{
								{
									Name:  "url",
									Value: "gcr.io/foo/bar",
								},
							},
						},
					},
				},
			},
		},
	},
}
