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
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstall(t *testing.T) {
	c, _, cleanup := setup(t)
	defer cleanup()
	dep, err := c.KubeClient.AppsV1().Deployments("tekton-pipelines").Get("tekton-chains-controller", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Error getting chains deployment: %v", err)
	}
	if dep.Status.AvailableReplicas != 1 {
		t.Errorf("Chains installation not running correctly: %v", dep)
	}
}

func TestTektonStorage(t *testing.T) {
	c, ns, cleanup := setup(t)
	defer cleanup()

	// Setup the right config.
	resetConfig := setConfigMap(t, c, map[string]string{"artifacts.taskrun.storage": "tekton"})
	defer resetConfig()

	tr, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(&simpleTaskRun)
	if err != nil {
		t.Errorf("error creating taskrun: %s", err)
	}

	// Give it a minute to complete.
	waitForCondition(t, c.PipelineClient, tr.Name, ns, done, 60*time.Second)

	signed := func(tr *v1beta1.TaskRun) bool {
		_, ok := tr.Annotations["chains.tekton.dev/signed"]
		return ok
	}

	// It can take up to a minute for the secret data to be updated!
	tr = waitForCondition(t, c.PipelineClient, tr.Name, ns, signed, 120*time.Second)

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

func TestGCSStorage(t *testing.T) {
	if metadata.OnGCE() {
		t.Skip("Skipping, integration tests do not support GCS secrets yet.")
	}
	c, ns, cleanup := setup(t)
	defer cleanup()

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Error(err)
	}
	bucketName, rmBucket := makeBucket(t, client)
	defer rmBucket()

	resetConfig := setConfigMap(t, c, map[string]string{
		"artifacts.taskrun.storage": "gcs",
		"storage.gcs.bucket":        bucketName,
	})
	defer resetConfig()
	time.Sleep(3 * time.Second)

	tr, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(&simpleTaskRun)
	if err != nil {
		t.Errorf("error creating taskrun: %s", err)
	}

	// Give it a minute to complete.
	waitForCondition(t, c.PipelineClient, tr.Name, ns, done, 60*time.Second)

	signed := func(tr *v1beta1.TaskRun) bool {
		_, ok := tr.Annotations["chains.tekton.dev/signed"]
		return ok
	}

	// It can take up to a minute for the secret data to be updated!
	tr = waitForCondition(t, c.PipelineClient, tr.Name, ns, signed, 120*time.Second)

	sigName := fmt.Sprintf("taskrun-%s-%s-%s/taskrun.signature", tr.Namespace, tr.Name, tr.UID)
	payloadName := fmt.Sprintf("taskrun-%s-%s-%s/taskrun.payload", tr.Namespace, tr.Name, tr.UID)

	sigBytes := readObj(t, bucketName, sigName, client)
	bodyBytes := readObj(t, bucketName, payloadName, client)

	checkPgpSignatures(t, sigBytes, bodyBytes)
}
