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
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/provenance"
	"github.com/tektoncd/chains/pkg/test/tekton"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestInstall(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, _, cleanup := setup(ctx, t, setupOpts{})
	t.Cleanup(cleanup)
	dep, err := c.KubeClient.AppsV1().Deployments("tekton-chains").Get(ctx, "tekton-chains-controller", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Error getting chains deployment: %v", err)
	}
	if dep.Status.AvailableReplicas != 1 {
		t.Errorf("Chains installation not running correctly: %v", dep)
	}
}

func TestTektonStorage(t *testing.T) {
	tests := []struct {
		name      string
		cm        map[string]string
		getObject func(ns string) objects.TektonObject
	}{
		{
			name: "taskrun",
			cm: map[string]string{
				"artifacts.taskrun.format":  "in-toto",
				"artifacts.taskrun.signer":  "x509",
				"artifacts.taskrun.storage": "tekton",
				"artifacts.oci.format":      "simplesigning",
				"artifacts.oci.signer":      "x509",
				"artifacts.oci.storage":     "tekton",
			},
			getObject: getTaskRunObject,
		},
		{
			name: "pipelinerun",
			cm: map[string]string{
				"artifacts.pipelinerun.format":  "in-toto",
				"artifacts.pipelinerun.signer":  "x509",
				"artifacts.pipelinerun.storage": "tekton",
				"artifacts.oci.format":          "simplesigning",
				"artifacts.oci.signer":          "x509",
				"artifacts.oci.storage":         "tekton",
			},
			getObject: getPipelineRunObject,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, setupOpts{})
			t.Cleanup(cleanup)

			// Setup the right config.
			resetConfig := setConfigMap(ctx, t, c, test.cm)
			t.Cleanup(resetConfig)

			createdObj := tekton.CreateObject(t, ctx, c.PipelineClient, test.getObject(ns))

			// Give it a minute to complete.
			if o := waitForCondition(ctx, t, c.PipelineClient, createdObj, done, time.Minute); o == nil {
				t.Fatal("object never became `done`.")
			}

			// It can take up to a minute for the secret data to be updated!
			updatedObj := waitForCondition(ctx, t, c.PipelineClient, createdObj, signed, 2*time.Minute)
			if updatedObj == nil {
				t.Fatal("object never signed.")
			}

			// Verify the payload signature.
			verifySignature(ctx, t, c, updatedObj)
		})
	}
}

func TestRekor(t *testing.T) {
	tests := []struct {
		name      string
		cm        map[string]string
		getObject func(ns string) objects.TektonObject
	}{
		{
			name: "taskrun",
			cm: map[string]string{
				"artifacts.taskrun.format":  "in-toto",
				"artifacts.taskrun.signer":  "x509",
				"artifacts.taskrun.storage": "tekton",
				"artifacts.oci.format":      "simplesigning",
				"artifacts.oci.signer":      "x509",
				"artifacts.oci.storage":     "tekton",
				"transparency.enabled":      "manual",
			},
			getObject: getTaskRunObject,
		},
		{
			name: "pipelinerun",
			cm: map[string]string{
				"artifacts.pipelinerun.format":  "in-toto",
				"artifacts.pipelinerun.signer":  "x509",
				"artifacts.pipelinerun.storage": "tekton",
				"artifacts.oci.format":          "simplesigning",
				"artifacts.oci.signer":          "x509",
				"artifacts.oci.storage":         "tekton",
				"transparency.enabled":          "manual",
			},
			getObject: getPipelineRunObject,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, setupOpts{})
			t.Cleanup(cleanup)

			// Setup the right config.
			resetConfig := setConfigMap(ctx, t, c, test.cm)
			t.Cleanup(resetConfig)

			createdObj := tekton.CreateObject(t, ctx, c.PipelineClient, test.getObject(ns))

			// Give it a minute to complete.
			if got := waitForCondition(ctx, t, c.PipelineClient, createdObj, done, time.Minute); got == nil {
				t.Fatal("object never done")
			}

			// It can take up to a minute for the secret data to be updated!
			obj := waitForCondition(ctx, t, c.PipelineClient, createdObj, signed, 2*time.Minute)
			if obj == nil {
				t.Fatal("object never signed")
			}

			if v, ok := obj.GetAnnotations()[chains.ChainsTransparencyAnnotation]; !ok || v == "" {
				t.Fatal("failed to upload to tlog")
			}

			// Verify the payload signature.
			verifySignature(ctx, t, c, obj)
		})
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
		},
		{
			name: "cosign",
			opts: setupOpts{useCosignSigner: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, test.opts)
			t.Cleanup(cleanup)

			// Setup the right config.
			resetConfig := setConfigMap(ctx, t, c, map[string]string{"artifacts.oci.storage": "tekton", "artifacts.taskrun.format": "in-toto"})
			t.Cleanup(resetConfig)

			tro := getTaskRunObject(ns)

			createdTro := tekton.CreateObject(t, ctx, c.PipelineClient, tro)

			// Give it a minute to complete.
			if got := waitForCondition(ctx, t, c.PipelineClient, createdTro, done, time.Minute); got == nil {
				t.Fatal("object never done")
			}

			// It can take up to a minute for the secret data to be updated!
			obj := waitForCondition(ctx, t, c.PipelineClient, createdTro, signed, 2*time.Minute)
			if obj == nil {
				t.Fatal("object never signed")
			}

			// Let's fetch the signature and body:
			t.Log(obj.GetAnnotations())

			sig, body := obj.GetAnnotations()["chains.tekton.dev/signature-05f95b26ed10"], obj.GetAnnotations()["chains.tekton.dev/payload-05f95b26ed10"]
			// base64 decode them
			sigBytes, err := base64.StdEncoding.DecodeString(sig)
			if err != nil {
				t.Error(err)
			}
			bodyBytes, err := base64.StdEncoding.DecodeString(body)
			if err != nil {
				t.Error(err)
			}

			var verifier signature.SignerVerifier
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
		t.Fatal(err)
	}
	bucketName, rmBucket := makeBucket(t, client)
	t.Cleanup(rmBucket)

	c, ns, cleanup := setup(ctx, t, setupOpts{})
	t.Cleanup(cleanup)

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.signer":  "x509",
		"artifacts.taskrun.storage": "gcs",
		"storage.gcs.bucket":        bucketName,
	})
	t.Cleanup(resetConfig)
	time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

	tro := getTaskRunObject(ns)

	createdTro := tekton.CreateObject(t, ctx, c.PipelineClient, tro)

	// Give it a minute to complete.
	if got := waitForCondition(ctx, t, c.PipelineClient, createdTro, done, time.Minute); got == nil {
		t.Fatal("object never done")
	}

	// It can take up to a minute for the secret data to be updated!
	obj := waitForCondition(ctx, t, c.PipelineClient, createdTro, signed, 2*time.Minute)
	if obj == nil {
		t.Fatal("object never signed")
	}

	// Verify the payload signature.
	verifySignature(ctx, t, c, obj)
}

func TestFulcio(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	if metadata.OnGCE() {
		t.Skip("Skipping, integration tests do not support workload identity yet.")
	}
	c, ns, cleanup := setup(ctx, t, setupOpts{ns: "default"})
	t.Cleanup(cleanup)
	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.storage":   "tekton",
		"artifacts.taskrun.signer":    "x509",
		"artifacts.taskrun.format":    "slsa/v1",
		"artifacts.oci.signer":        "x509",
		"signers.x509.fulcio.enabled": "true",
		"transparency.enabled":        "false",
	})

	t.Cleanup(resetConfig)
	time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

	tro := getTaskRunObject(ns)

	createdTro := tekton.CreateObject(t, ctx, c.PipelineClient, tro)

	// Give it a minute to complete.
	if got := waitForCondition(ctx, t, c.PipelineClient, createdTro, done, time.Minute); got == nil {
		t.Fatal("object never done")
	}

	// It can take up to a minute for the secret data to be updated!
	obj := waitForCondition(ctx, t, c.PipelineClient, createdTro, signed, 2*time.Minute)
	if obj == nil {
		t.Fatal("object never signed")
	}

	// verify the cert against the signature and payload
	sigKey := fmt.Sprintf("chains.tekton.dev/signature-taskrun-%s", obj.GetUID())
	payloadKey := fmt.Sprintf("chains.tekton.dev/payload-taskrun-%s", obj.GetUID())
	certKey := fmt.Sprintf("chains.tekton.dev/cert-taskrun-%s", obj.GetUID())

	envelopeBytes := base64Decode(t, obj.GetAnnotations()[sigKey])
	payload := base64Decode(t, obj.GetAnnotations()[payloadKey])
	certPEM := base64Decode(t, obj.GetAnnotations()[certKey])

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
	var envelope dsse.Envelope
	if err := json.Unmarshal(envelopeBytes, &envelope); err != nil {
		t.Fatal(err)
	}

	paeEnc := dsse.PAE(in_toto.PayloadType, payload)
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
	t.Cleanup(cleanup)

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.oci.format":            "simplesigning",
		"artifacts.oci.storage":           "oci",
		"artifacts.oci.signer":            "x509",
		"artifacts.taskrun.format":        "slsa/v1",
		"artifacts.taskrun.signer":        "x509",
		"artifacts.taskrun.storage":       "oci",
		"storage.oci.repository.insecure": "true",
	})
	t.Cleanup(resetConfig)
	time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

	// create necessary resources
	imageName := "chains-test-oci-storage"
	image := fmt.Sprintf("%s/%s", c.internalRegistry, imageName)
	task := kanikoTask(t, ns, image)
	if _, err := c.PipelineClient.TektonV1beta1().Tasks(ns).Create(ctx, task, metav1.CreateOptions{}); err != nil {
		t.Fatalf("error creating task: %s", err)
	}

	tro := kanikoTaskRun(ns)

	createdTro := tekton.CreateObject(t, ctx, c.PipelineClient, tro)

	// Give it a minute to complete.
	if got := waitForCondition(ctx, t, c.PipelineClient, createdTro, done, time.Minute); got == nil {
		t.Fatal("object never done")
	}

	// It can take up to a minute for the secret data to be updated!
	if got := waitForCondition(ctx, t, c.PipelineClient, createdTro, signed, 2*time.Minute); got == nil {
		t.Fatal("object never signed")
	}

	publicKey, err := cryptoutils.MarshalPublicKeyToPEM(c.secret.x509priv.Public())
	if err != nil {
		t.Error(err)
	}
	verifyTrObj := verifyKanikoTaskRun(ns, image, string(publicKey))
	createdObj := tekton.CreateObject(t, ctx, c.PipelineClient, verifyTrObj)
	obj := waitForCondition(ctx, t, c.PipelineClient, createdObj, successful, 2*time.Minute)
	if obj == nil {
		t.Error("object never created successfully")
	}
	verifySignature(ctx, t, c, obj)
}

func TestMultiBackendStorage(t *testing.T) {
	tests := []struct {
		name      string
		cm        map[string]string
		getObject func(ns string) objects.TektonObject
	}{
		{
			name: "taskrun",
			cm: map[string]string{
				"artifacts.oci.format":            "simplesigning",
				"artifacts.oci.storage":           "tekton,oci",
				"artifacts.oci.signer":            "x509",
				"artifacts.taskrun.format":        "slsa/v1",
				"artifacts.taskrun.signer":        "x509",
				"artifacts.taskrun.storage":       "tekton,oci",
				"storage.oci.repository.insecure": "true",
			},
			getObject: kanikoTaskRun,
		},
		{
			name: "pipelinerun",
			cm: map[string]string{
				"artifacts.pipelinerun.format":    "slsa/v1",
				"artifacts.pipelinerun.signer":    "x509",
				"artifacts.pipelinerun.storage":   "tekton,oci",
				"storage.oci.repository.insecure": "true",
			},
			getObject: kanikoPipelineRun,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			image := "chains-test-multibackendstorage"
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, setupOpts{
				registry:        true,
				kanikoTaskImage: image,
			})
			t.Cleanup(cleanup)

			resetConfig := setConfigMap(ctx, t, c, test.cm)
			t.Cleanup(resetConfig)
			time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

			createdObj := tekton.CreateObject(t, ctx, c.PipelineClient, test.getObject(ns))

			publicKey, err := cryptoutils.MarshalPublicKeyToPEM(c.secret.x509priv.Public())
			if err != nil {
				t.Error(err)
			}

			// Verify object has been signed
			obj := waitForCondition(ctx, t, c.PipelineClient, createdObj, signed, 2*time.Minute)
			if obj == nil {
				t.Fatal("object never signed")
			}

			// Verify the payload signature.
			verifySignature(ctx, t, c, obj)

			verifyTro := verifyKanikoTaskRun(ns, fmt.Sprintf("%s/%s", c.internalRegistry, image), string(publicKey))
			createdVerifyTro := tekton.CreateObject(t, ctx, c.PipelineClient, verifyTro)
			if got := waitForCondition(ctx, t, c.PipelineClient, createdVerifyTro, successful, time.Minute); got == nil {
				t.Fatal("object never successful")
			}
		})
	}
}

func TestRetryFailed(t *testing.T) {
	tests := []struct {
		name      string
		cm        map[string]string
		opts      setupOpts
		getObject func(ns string) objects.TektonObject
	}{
		{
			name: "taskrun",
			cm: map[string]string{
				// don't set insecure repository, forcing signature upload to fail
				"artifacts.oci.storage":     "oci",
				"artifacts.taskrun.storage": "tekton",
				"storage.oci.repository":    "gcr.io/not-real",
			},
			opts: setupOpts{
				registry:        true,
				kanikoTaskImage: "chains-test-tr-retryfailed",
			},
			getObject: getTaskRunObject,
		},
		{
			name: "pipelinerun",
			cm: map[string]string{
				// force failure by trying to push transperency log to url that
				// does not exist
				"transparency.url":              "doesnotexist",
				"transparency.enabled":          "true",
				"artifacts.pipelinerun.storage": "tekton",
				"artifacts.pipelinerun.format":  "slsa/v1",
			},
			opts: setupOpts{
				registry:        true,
				kanikoTaskImage: "chains-test-pr-retryfailed",
			},
			getObject: getPipelineRunObject,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, test.opts)
			t.Cleanup(cleanup)

			resetConfig := setConfigMap(ctx, t, c, test.cm)
			t.Cleanup(resetConfig)
			time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

			obj := tekton.CreateObject(t, ctx, c.PipelineClient, test.getObject(ns))

			// Give it a minute to complete.
			if got := waitForCondition(ctx, t, c.PipelineClient, obj, failed, time.Minute); got == nil {
				t.Fatal("expected failure; object never failed")
			}
		})
	}
}

var imageTaskSpec = v1beta1.TaskSpec{
	Steps: []v1beta1.Step{
		{
			Image: "busybox",
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
			Outputs: []v1beta1.TaskResourceBinding{{
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
			}},
		},
	},
}

func getTaskRunObject(ns string) objects.TektonObject {
	o := objects.NewTaskRunObject(&imageTaskRun)
	o.Namespace = ns
	return o
}

func getTaskRunObjectWithParams(ns string, params []v1beta1.Param) objects.TektonObject {
	o := objects.NewTaskRunObject(&imageTaskRun)
	o.Namespace = ns
	o.Spec.Params = params
	return o
}

var imagePipelineRun = v1beta1.PipelineRun{
	ObjectMeta: metav1.ObjectMeta{
		GenerateName: "image-pipelinerun",
		Annotations:  map[string]string{chains.RekorAnnotation: "true"},
	},
	Spec: v1beta1.PipelineRunSpec{
		PipelineSpec: &v1beta1.PipelineSpec{
			Tasks: []v1beta1.PipelineTask{{
				Name: "echo",
				TaskSpec: &v1beta1.EmbeddedTask{
					TaskSpec: v1beta1.TaskSpec{
						Steps: []v1beta1.Step{
							{
								Image:  "busybox",
								Script: "echo success",
							},
						},
					},
				},
			}},
		},
	},
}

func getPipelineRunObject(ns string) objects.TektonObject {
	o := objects.NewPipelineRunObject(&imagePipelineRun)
	o.Namespace = ns
	return o
}

func getPipelineRunObjectWithParams(ns string, params []v1beta1.Param) objects.TektonObject {
	o := objects.NewPipelineRunObject(&imagePipelineRun)
	o.Namespace = ns
	o.Spec.Params = params
	return o
}

func TestProvenanceMaterials(t *testing.T) {
	tests := []struct {
		name                string
		cm                  map[string]string
		getObjectWithParams func(ns string, params []v1beta1.Param) objects.TektonObject
		payloadKey          string
	}{
		{
			name: "taskrun",
			cm: map[string]string{
				"artifacts.taskrun.format":  "slsa/v1",
				"artifacts.taskrun.signer":  "x509",
				"artifacts.taskrun.storage": "tekton",
			},
			payloadKey:          "chains.tekton.dev/payload-taskrun-%s",
			getObjectWithParams: getTaskRunObjectWithParams,
		},
		{
			name: "pipelinerun",
			cm: map[string]string{
				"artifacts.pipelinerun.format":  "slsa/v1",
				"artifacts.pipelinerun.signer":  "x509",
				"artifacts.pipelinerun.storage": "tekton",
			},
			payloadKey:          "chains.tekton.dev/payload-pipelinerun-%s",
			getObjectWithParams: getPipelineRunObjectWithParams,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, setupOpts{})
			t.Cleanup(cleanup)

			// Setup the right config.
			resetConfig := setConfigMap(ctx, t, c, test.cm)
			t.Cleanup(resetConfig)

			commit := "my-git-commit"
			url := "https://my-git-url"
			params := []v1beta1.Param{{
				Name: "CHAINS-GIT_COMMIT", Value: *v1beta1.NewArrayOrString(commit),
			}, {
				Name: "CHAINS-GIT_URL", Value: *v1beta1.NewArrayOrString(url),
			}}
			obj := test.getObjectWithParams(ns, params)

			createdObj := tekton.CreateObject(t, ctx, c.PipelineClient, obj)

			// Give it a minute to complete.
			if got := waitForCondition(ctx, t, c.PipelineClient, createdObj, done, time.Minute); got == nil {
				t.Fatal("object never done")
			}

			// It can take up to a minute for the secret data to be updated!
			signedObj := waitForCondition(ctx, t, c.PipelineClient, createdObj, signed, 2*time.Minute)
			if signedObj == nil {
				t.Fatal("object never signed")
			}

			// get provenance, and make sure it has a materials section
			payloadKey := fmt.Sprintf(test.payloadKey, signedObj.GetUID())
			body := signedObj.GetAnnotations()[payloadKey]
			bodyBytes, err := base64.StdEncoding.DecodeString(body)
			if err != nil {
				t.Error(err)
			}
			var actualProvenance in_toto.Statement
			if err := json.Unmarshal(bodyBytes, &actualProvenance); err != nil {
				t.Error(err)
			}
			predicateBytes, err := json.Marshal(actualProvenance.Predicate)
			if err := json.Unmarshal(bodyBytes, &actualProvenance); err != nil {
				t.Error(err)
			}
			var predicate provenance.ProvenancePredicate
			if err := json.Unmarshal(predicateBytes, &predicate); err != nil {
				t.Fatal(err)
			}
			want := []provenance.ProvenanceMaterial{
				{
					URI: "git+" + url + ".git",
					Digest: provenance.DigestSet{
						"sha1": commit,
					},
				},
			}
			if test.name == "pipelinerun" {
				pr := signedObj.GetObject().(*v1beta1.PipelineRun)
				for _, trStatus := range pr.Status.TaskRuns {
					for _, step := range trStatus.Status.Steps {
						want = append(want, provenance.ProvenanceMaterial{
							URI: strings.Split(step.ImageID, "@")[0],
							Digest: provenance.DigestSet{
								"sha256": strings.Split(step.ImageID, ":")[1],
							},
						})
					}
				}
			}
			got := predicate.Materials

			sortMaterials := cmpopts.SortSlices(func(i, j provenance.ProvenanceMaterial) bool {
				return i.URI < j.URI
			})
			if d := cmp.Diff(want, got, sortMaterials); d != "" {
				t.Fatal(string(d))
			}
		})
	}
}

func TestVaultKMSSpire(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{})
	t.Cleanup(cleanup)

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.oci.signer":            "kms",
		"artifacts.taskrun.signer":        "kms",
		"signers.kms.kmsref":              "hashivault://e2e",
		"signers.kms.auth.address":        "http://vault.vault:8200",
		"signers.kms.auth.oidc.path":      "jwt",
		"signers.kms.auth.oidc.role":      "spire-chains-controller",
		"signers.kms.auth.spire.sock":     "unix:///tmp/spire-agent/public/api.sock",
		"signers.kms.auth.spire.audience": "e2e",
	})

	t.Cleanup(resetConfig)
	time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

	tro := getTaskRunObject(ns)
	createdTro := tekton.CreateObject(t, ctx, c.PipelineClient, tro)

	// Give it a minute to complete.
	waitForCondition(ctx, t, c.PipelineClient, createdTro, done, time.Minute)

	// It can take up to a minute for the secret data to be updated!
	obj := waitForCondition(ctx, t, c.PipelineClient, createdTro, signed, 2*time.Minute)
	t.Log(obj.GetAnnotations())

	// Verify the payload signature.
	verifySignature(ctx, t, c, obj) // TODO: consider removing

	// verify the cert against the signature and payload
	sigKey := fmt.Sprintf("chains.tekton.dev/signature-taskrun-%s", obj.GetUID())
	payloadKey := fmt.Sprintf("chains.tekton.dev/payload-taskrun-%s", obj.GetUID())
	envelopeBytes := base64Decode(t, obj.GetAnnotations()[sigKey])
	payloadBytes := base64Decode(t, obj.GetAnnotations()[payloadKey])

	certPEM, err := os.ReadFile("testdata/vault.pub")
	if err != nil {
		t.Fatalf("failed reading vault.pub key: %v", err)
	}
	cert, err := cryptoutils.UnmarshalPEMToPublicKey(certPEM)
	if err != nil {
		t.Fatalf("failed unmarshaling vault.pub key: %v", err)
	}
	pubKey, err := signature.LoadECDSAVerifier(cert.(*ecdsa.PublicKey), crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}

	// verify the signature
	var envelope dsse.Envelope
	if err := json.Unmarshal(envelopeBytes, &envelope); err != nil {
		t.Fatal(err)
	}

	paeEnc := dsse.PAE(in_toto.PayloadType, payloadBytes)
	sigEncoded := envelope.Signatures[0].Sig
	sig := base64Decode(t, sigEncoded)

	if err := pubKey.VerifySignature(bytes.NewReader([]byte(sig)), bytes.NewReader(paeEnc)); err != nil {
		t.Fatal(err)
	}
}
