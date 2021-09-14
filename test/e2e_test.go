//go:build e2e
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
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/pkg/ssl"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/dsse"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"github.com/tektoncd/chains/pkg/chains"
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
		"artifacts.taskrun.signer": "x509",
		"artifacts.oci.format":     "tekton",
		"artifacts.taskrun.format": "tekton",
		"artifacts.oci.storage":    "tekton",
		"artifacts.oci.signer":     "x509",
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

	if err := c.secret.x509priv.VerifySignature(bytes.NewReader(sigBytes), bytes.NewReader(bodyBytes)); err != nil {
		t.Fatal(err)
	}
}

func TestRekor(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{})
	defer cleanup()

	// Setup the right config.
	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.signer": "x509",
		"artifacts.oci.format":     "tekton",
		"artifacts.taskrun.format": "tekton",
		"artifacts.oci.storage":    "tekton",
		"artifacts.oci.signer":     "x509",
		"transparency.enabled":     "manual",
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

	if v, ok := tr.Annotations[chains.ChainsTransparencyAnnotation]; !ok || v == "" {
		t.Fatal("failed to upload to tlog")
	}

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

	if err := c.secret.x509priv.VerifySignature(bytes.NewReader(sigBytes), bytes.NewReader(bodyBytes)); err != nil {
		t.Fatal(err)
	}
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
			resetConfig := setConfigMap(ctx, t, c, map[string]string{"artifacts.oci.storage": "tekton", "artifacts.taskrun.format": "tekton"})

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

			sig, body := tr.Annotations["chains.tekton.dev/signature-05f95b26ed10"], tr.Annotations["chains.tekton.dev/payload-05f95b26ed10"]
			// base64 decode them
			sigBytes, err := base64.StdEncoding.DecodeString(sig)
			if err != nil {
				t.Error(err)
			}
			bodyBytes, err := base64.StdEncoding.DecodeString(body)
			if err != nil {
				t.Error(err)
			}

			var verifier *signature.ECDSASignerVerifier
			if test.name == "cosign" {
				verifier = c.secret.cosignPriv
			} else {
				verifier = c.secret.x509priv
			}

			if err := verifier.VerifySignature(bytes.NewReader(sigBytes), bytes.NewReader(bodyBytes)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGCSStorage(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	if metadata.OnGCE() {
		t.Skip("Skipping, integration tests do not support GCS secrets yet.")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Error(err)
	}
	bucketName, rmBucket := makeBucket(t, client)
	defer rmBucket()

	c, ns, cleanup := setup(ctx, t, setupOpts{})
	defer cleanup()

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.storage": "gcs",
		"storage.gcs.bucket":        bucketName,
		"artifacts.taskrun.signer":  "x509",
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

	root := fmt.Sprintf("taskrun-%s-%s", tr.Namespace, tr.Name)
	key := "taskrun-" + string(tr.UID)

	sigName := path.Join(root, fmt.Sprintf("%s.signature", key))
	payloadName := path.Join(root, fmt.Sprintf("%s.payload", key))

	t.Log(sigName)

	sigBytes := readObj(t, bucketName, sigName, client)
	bodyBytes := readObj(t, bucketName, payloadName, client)

	if err := c.secret.x509priv.VerifySignature(sigBytes, bodyBytes); err != nil {
		t.Fatal(err)
	}
}

func TestFulcio(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	if metadata.OnGCE() {
		t.Skip("Skipping, integration tests do not support workload identity yet.")
	}
	c, ns, cleanup := setup(ctx, t, setupOpts{ns: "default"})
	defer cleanup()
	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.signer":    "x509",
		"artifacts.taskrun.format":    "tekton-provenance",
		"artifacts.oci.signer":        "x509",
		"signers.x509.fulcio.enabled": "true",
		"transparency.enabled":        "false",
	})

	defer resetConfig()
	time.Sleep(3 * time.Second)

	tr, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(ctx, &simpleTaskRun, metav1.CreateOptions{})
	if err != nil {
		t.Errorf("error creating taskrun: %s", err)
	}

	// Give it a minute to complete.
	waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, done, 60*time.Second)
	tr = waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, signed, 120*time.Second)

	// verify the cert against the signature and payload

	sigKey := fmt.Sprintf("chains.tekton.dev/signature-taskrun-%s", tr.UID)
	payloadKey := fmt.Sprintf("chains.tekton.dev/payload-taskrun-%s", tr.UID)
	certKey := fmt.Sprintf("chains.tekton.dev/cert-taskrun-%s", tr.UID)

	envelopeBytes := base64Decode(t, tr.Annotations[sigKey])
	payload := base64Decode(t, tr.Annotations[payloadKey])
	certPEM := base64Decode(t, tr.Annotations[certKey])

	certs, err := cryptoutils.LoadCertificatesFromPEM(bytes.NewReader(certPEM))
	if err != nil {
		t.Fatal(err)
	}
	if len(certs) == 0 {
		t.Fatal("there are no certs -- make sure you have workload identity set up on the default service account in the default ns")
	}
	cert := certs[0]
	pubKey, err := signature.LoadECDSAVerifier(cert.PublicKey.(*ecdsa.PublicKey), crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}

	// verify the signature
	var envelope ssl.Envelope
	if err := json.Unmarshal(envelopeBytes, &envelope); err != nil {
		t.Fatal(err)
	}

	paeEnc := ssl.PAE(in_toto.PayloadType, string(payload))
	sigEncoded := envelope.Signatures[0].Sig
	sig := base64Decode(t, sigEncoded)

	if err := pubKey.VerifySignature(bytes.NewReader([]byte(sig)), bytes.NewReader(paeEnc)); err != nil {
		t.Fatal(err)
	}
}

func base64Decode(t *testing.T, s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		b, err = base64.URLEncoding.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	return b
}

// DSSE messages contain the signature and payload in one object, but our interface expects a signature and payload
// This means we need to use one field and ignore the other. The DSSE verifier upstream uses the signature field and ignores
// The message field, but we want the reverse here.
type reverseDSSEVerifier struct {
	signature.Verifier
}

func (w *reverseDSSEVerifier) VerifySignature(s io.Reader, m io.Reader, opts ...signature.VerifyOption) error {
	return w.Verifier.VerifySignature(m, nil, opts...)
}

func TestOCIStorage(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{registry: true})
	defer cleanup()

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"storage.oci.repository.insecure": "true",
		"artifacts.oci.storage":           "oci",
		"artifacts.oci.signer":            "x509",
		"artifacts.taskrun.storage":       "oci",
		"artifacts.taskrun.format":        "tekton-provenance",
		"artifacts.taskrun.signer":        "x509",
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
			// verify the image
			if _, err = cosign.Verify(ctx, externalRef, &cosign.CheckOpts{
				SignatureRepo: externalRef.Context(),
				SigVerifier:   c.secret.x509priv,
			}); err != nil {
				t.Log(err)
				continue
			}
			// verify the attestation
			if _, err = cosign.Verify(ctx, externalRef, &cosign.CheckOpts{
				SigTagSuffixOverride: cosign.AttestationTagSuffix,
				SignatureRepo:        externalRef.Context(),
				SigVerifier:          &reverseDSSEVerifier{dsse.WrapVerifier(c.secret.x509priv)},
			}); err != nil {
				t.Log(err)
				continue
			}
			return
		case <-timeoutChan:
			t.Fatal("time out")
		}
	}
}

func TestRetryFailed(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{registry: true})
	defer cleanup()

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		// don't set insecure repository, forcing signature upload to fail
		"artifacts.oci.storage":     "oci",
		"artifacts.taskrun.storage": "tekton",
		"storage.oci.repository":    "gcr.io/not-real",
	})
	defer resetConfig()
	time.Sleep(3 * time.Second)

	// create necessary resources
	imageName := "chains-test-retry"
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
	waitForCondition(ctx, t, c.PipelineClient, tr.Name, ns, failed, 60*time.Second)
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
		Annotations:  map[string]string{chains.RekorAnnotation: "true"},
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
