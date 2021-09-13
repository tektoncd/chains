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
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getTr(ctx context.Context, t *testing.T, c pipelineclientset.Interface, name, ns string) (tr *v1beta1.TaskRun) {
	t.Helper()
	tr, err := c.TektonV1beta1().TaskRuns(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	return tr
}

type conditionFn func(*v1beta1.TaskRun) bool

func waitForCondition(ctx context.Context, t *testing.T, c pipelineclientset.Interface, name, ns string, cond conditionFn, timeout time.Duration) *v1beta1.TaskRun {
	t.Helper()

	// Do a first quick check before setting the watch
	tr := getTr(ctx, t, c, name, ns)
	if cond(tr) {
		return tr
	}

	w, err := c.TektonV1beta1().TaskRuns(ns).Watch(ctx, metav1.SingleObject(metav1.ObjectMeta{
		Name:      name,
		Namespace: ns,
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
			tr := ev.Object.(*v1beta1.TaskRun)
			if cond(tr) {
				return tr
			}
		case <-timeoutChan:
			t.Fatal("time out")
		}
	}
}

func successful(tr *v1beta1.TaskRun) bool {
	return tr.IsSuccessful()
}

func failed(tr *v1beta1.TaskRun) bool {
	failed, ok := tr.Annotations["chains.tekton.dev/signed"]
	return ok && failed == "failed" && tr.Annotations["chains.tekton.dev/retries"] == "3"
}

func done(tr *v1beta1.TaskRun) bool {
	return tr.IsDone()
}

func signed(tr *v1beta1.TaskRun) bool {
	_, ok := tr.Annotations["chains.tekton.dev/signed"]
	return ok
}

var simpleTaskspec = v1beta1.TaskSpec{
	Steps: []v1beta1.Step{
		{
			Container: corev1.Container{
				Image: "busybox",
			},
			Script: "echo true",
		},
	},
}

var simpleTaskRun = v1beta1.TaskRun{
	ObjectMeta: metav1.ObjectMeta{
		GenerateName: "test-task",
	},
	Spec: v1beta1.TaskRunSpec{
		TaskSpec: &simpleTaskspec,
	},
}

func makeBucket(t *testing.T, client *storage.Client) (string, func()) {
	// Make a bucket
	rand.Seed(time.Now().UnixNano())
	testBucketName := fmt.Sprintf("tekton-chains-e2e-%d", rand.Intn(1000))

	ctx := context.Background()
	proj := os.Getenv("GCP_PROJECT_ID")
	if proj == "" {
		t.Skip("Skipping, no GCP_PROJECT_ID env var set")
	}
	if err := client.Bucket(testBucketName).Create(ctx, proj, nil); err != nil {
		t.Fatal(err)
	}
	return testBucketName, func() {
		// We have to remove the bucket using gsutil. The libraries/APIs require you to
		// first list the bucket then delete everything. List operations are eventually-consistent,
		// so this is non-trivial. gsutil takes care of this.
		cmd := exec.Command("gsutil", "ls", "gs://"+testBucketName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Log(err)
		}
		cmd = exec.Command("gsutil", "rm", "-rf", "gs://"+testBucketName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Log(err)
		}
	}
}

func readObj(t *testing.T, bucket, name string, client *storage.Client) io.Reader {
	ctx := context.Background()
	reader, err := client.Bucket(bucket).Object(name).NewReader(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return reader
}

func setConfigMap(ctx context.Context, t *testing.T, c *clients, data map[string]string) func() {
	// Change the config to be GCS storage with this bucket.
	cm, err := c.KubeClient.CoreV1().ConfigMaps("tekton-chains").Get(ctx, "chains-config", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Save the old data to reset it after.
	oldData := map[string]string{}
	for k, v := range cm.Data {
		oldData[k] = v
	}

	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	for k, v := range data {
		cm.Data[k] = v
	}
	cm, err = c.KubeClient.CoreV1().ConfigMaps("tekton-chains").Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Second)

	return func() {
		for k := range data {
			delete(cm.Data, k)
		}
		for k, v := range oldData {
			cm.Data[k] = v
		}
		if _, err := c.KubeClient.CoreV1().ConfigMaps("tekton-chains").Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
			t.Log(err)
		}
	}
}
