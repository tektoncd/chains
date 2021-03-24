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
	"archive/tar"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"

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
	c, _, cleanup := setup(ctx, t)
	defer cleanup()
	dep, err := c.KubeClient.AppsV1().Deployments("tekton-pipelines").Get(ctx, "tekton-chains-controller", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Error getting chains deployment: %v", err)
	}
	if dep.Status.AvailableReplicas != 1 {
		t.Errorf("Chains installation not running correctly: %v", dep)
	}
}

func TestTektonStorage(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t)
	defer cleanup()

	// Setup the right config.
	resetConfig := setConfigMap(ctx, t, c, map[string]string{"artifacts.taskrun.storage": "tekton"})
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
	signature, body := tr.Annotations["chains.tekton.dev/signature-taskrun"], tr.Annotations["chains.tekton.dev/payload-taskrun"]
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
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t)
	defer cleanup()

	// Setup the right config.
	resetConfig := setConfigMap(ctx, t, c, map[string]string{"artifacts.oci.format": "simplesigning", "artifacts.oci.storage": "tekton"})
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

	checkPgpSignatures(t, sigBytes, bodyBytes)
}

func TestGCSStorage(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	if metadata.OnGCE() {
		t.Skip("Skipping, integration tests do not support GCS secrets yet.")
	}
	c, ns, cleanup := setup(ctx, t)
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
	if metadata.OnGCE() {
		t.Skip("Skipping, integration tests do not support GCS secrets yet.")
	}
	repo := os.Getenv("OCI_REPOSITORY")
	if repo == "" {
		t.Skipf("Skipping, %s requires OCI_REPOSITORY to be set.", t.Name())
	}
	c, ns, cleanup := setup(ctx, t)
	defer cleanup()

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.storage": "oci",
		"storage.oci.repository":    repo,
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

	imgName := fmt.Sprintf("%s/taskrun-%s-%s-%s/taskrun", repo, tr.Namespace, tr.Name, tr.UID)
	ref, err := name.ParseReference(imgName)
	if err != nil {
		t.Fatal(err)
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		t.Fatal(err)
	}
	layers, err := img.Layers()
	if err != nil {
		t.Fatal(err)
	}
	// There should only be one layer and one file in the layer.
	if len(layers) != 1 {
		t.Fatalf("expected only one layer, found %d", len(layers))
	}
	layer := layers[0]
	rc, err := layer.Uncompressed()
	defer rc.Close()
	tarfile := tar.NewReader(rc)

	numFiles := 0
	d, err := partial.Descriptor(layer)
	if err != nil {
		t.Error(err)
	}
	sigBytes := []byte(d.Annotations["signature"])
	var bodyBytes []byte
	for {
		hdr, err := tarfile.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Error(err)
		}
		numFiles += 1
		if hdr.Name != "signed" {
			t.Errorf("expected file name to be 'signed', got %s", hdr.Name)
		}

		bodyBytes, err = ioutil.ReadAll(tarfile)
		if err != nil {
			t.Error(err)
		}
	}
	if numFiles != 1 {
		t.Errorf("expected only one file in the tar, found %d", numFiles)
	}
	checkPgpSignatures(t, sigBytes, bodyBytes)
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
