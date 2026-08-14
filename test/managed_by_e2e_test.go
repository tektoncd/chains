//go:build e2e
// +build e2e

/*
Copyright 2025 The Tekton Authors

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

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/test/tekton"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestManagedByTaskRunSigned(t *testing.T) {
	tests := []struct {
		name      string
		managedBy *string
	}{
		{
			name:      "no managedBy",
			managedBy: nil,
		},
		{
			name:      "tekton.dev/pipeline managedBy",
			managedBy: ptr.To("tekton.dev/pipeline"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, setupOpts{})
			t.Cleanup(cleanup)

			resetConfig := setConfigMap(ctx, t, c, map[string]string{
				"artifacts.taskrun.format":  "in-toto",
				"artifacts.taskrun.signer":  "x509",
				"artifacts.taskrun.storage": "tekton",
			})
			t.Cleanup(resetConfig)

			tr := &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "managed-by-test-"},
				Spec: v1.TaskRunSpec{
					TaskSpec:  &simpleTaskspec,
					ManagedBy: test.managedBy,
				},
			}
			obj := objects.NewTaskRunObjectV1(tr)
			obj.Namespace = ns
			createdObj := tekton.CreateObject(t, ctx, c.PipelineClient, obj)

			if o := waitForCondition(ctx, t, c.PipelineClient, createdObj, done, time.Minute); o == nil {
				t.Fatal("object never became done")
			}

			signedObj := waitForCondition(ctx, t, c.PipelineClient, createdObj, signed, 2*time.Minute)
			if signedObj == nil {
				t.Fatal("object never signed")
			}

			verifySignature(ctx, t, c, signedObj)
		})
	}
}

func TestManagedByCustomControllerNotProcessed(t *testing.T) {
	tests := []struct {
		name      string
		finalizer string
	}{
		{
			name:      "taskrun",
			finalizer: "chains.tekton.dev/taskrun",
		},
		{
			name:      "pipelinerun",
			finalizer: "chains.tekton.dev/pipelinerun",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, setupOpts{})
			t.Cleanup(cleanup)

			resetConfig := setConfigMap(ctx, t, c, map[string]string{
				"artifacts.taskrun.format":      "in-toto",
				"artifacts.taskrun.signer":      "x509",
				"artifacts.taskrun.storage":     "tekton",
				"artifacts.pipelinerun.format":  "in-toto",
				"artifacts.pipelinerun.signer":  "x509",
				"artifacts.pipelinerun.storage": "tekton",
			})
			t.Cleanup(resetConfig)

			var name string
			switch test.name {
			case "taskrun":
				tr := &v1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "managed-by-custom-",
						Namespace:    ns,
					},
					Spec: v1.TaskRunSpec{
						TaskSpec:  &simpleTaskspec,
						ManagedBy: ptr.To("custom-controller"),
					},
				}
				created, err := c.PipelineClient.TektonV1().TaskRuns(ns).Create(ctx, tr, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("Failed to create TaskRun: %v", err)
				}
				name = created.Name

			case "pipelinerun":
				pr := &v1.PipelineRun{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "managed-by-custom-pr-",
						Namespace:    ns,
					},
					Spec: v1.PipelineRunSpec{
						PipelineSpec: &v1.PipelineSpec{
							Tasks: []v1.PipelineTask{{
								Name: "echo",
								TaskSpec: &v1.EmbeddedTask{
									TaskSpec: v1.TaskSpec{
										Steps: []v1.Step{{
											Image:  "busybox",
											Script: "echo success",
										}},
									},
								},
							}},
						},
						ManagedBy: ptr.To("custom-controller"),
					},
				}
				created, err := c.PipelineClient.TektonV1().PipelineRuns(ns).Create(ctx, pr, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("Failed to create PipelineRun: %v", err)
				}
				name = created.Name
			}

			// The Pipeline controller skips this run (custom managedBy),
			// and Chains should also skip it (informer filter rejects it).
			time.Sleep(30 * time.Second)

			switch test.name {
			case "taskrun":
				got, err := c.PipelineClient.TektonV1().TaskRuns(ns).Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("Failed to get TaskRun: %v", err)
				}
				for _, f := range got.Finalizers {
					if f == test.finalizer {
						t.Errorf("Chains finalizer %q should not be added to externally-managed TaskRun", test.finalizer)
					}
				}
				if _, ok := got.Annotations["chains.tekton.dev/signed"]; ok {
					t.Error("chains.tekton.dev/signed annotation should not be set on externally-managed TaskRun")
				}

			case "pipelinerun":
				got, err := c.PipelineClient.TektonV1().PipelineRuns(ns).Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("Failed to get PipelineRun: %v", err)
				}
				for _, f := range got.Finalizers {
					if f == test.finalizer {
						t.Errorf("Chains finalizer %q should not be added to externally-managed PipelineRun", test.finalizer)
					}
				}
				if _, ok := got.Annotations["chains.tekton.dev/signed"]; ok {
					t.Error("chains.tekton.dev/signed annotation should not be set on externally-managed PipelineRun")
				}
			}
		})
	}
}
