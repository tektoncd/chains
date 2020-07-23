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
	"testing"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
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

func TestSign(t *testing.T) {
	c, ns, cleanup := setup(t)
	defer cleanup()

	tr, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(&simpleTaskRun)
	if err != nil {
		t.Errorf("error creating taskrun: %s", err)
	}

	done := func(tr *v1beta1.TaskRun) bool {
		return tr.IsDone()
	}

	// Give it a minute to complete.
	waitForCondition(t, c.PipelineClient, tr.Name, ns, done, 60*time.Second)

	tr, err = c.PipelineClient.TektonV1beta1().TaskRuns(ns).Get(tr.Name, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	signed := func(tr *v1beta1.TaskRun) bool {
		_, ok := tr.Annotations["chains.tekton.dev/signed"]
		return ok
	}

	// Now it should be signed within 10s
	waitForCondition(t, c.PipelineClient, tr.Name, ns, signed, 10*time.Second)
}

type conditionFn func(*v1beta1.TaskRun) bool

func waitForCondition(t *testing.T, c pipelineclientset.Interface, name, ns string, cond conditionFn, timeout time.Duration) *v1beta1.TaskRun {
	t.Helper()

	// Do a first quick check before setting the watch
	tr, err := c.TektonV1beta1().TaskRuns(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}
	if cond(tr) {
		return tr
	}

	w, err := c.TektonV1beta1().TaskRuns(ns).Watch(metav1.SingleObject(metav1.ObjectMeta{
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
			t.Error("time out")
		}
	}
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
