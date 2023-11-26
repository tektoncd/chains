/*
Copyright 2020 The Tekton Authors
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

package pipelinerun

import (
	"context"
	"testing"
	"time"

	signing "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/mocksigner"
	"github.com/tektoncd/chains/pkg/test/tekton"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	informers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	fakepipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1/pipelinerun/fake"
	faketaskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1/taskrun/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	_ "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/configmap"
	pkgreconciler "knative.dev/pkg/reconciler"
	reconcilertesting "knative.dev/pkg/reconciler/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
	"knative.dev/pkg/system"
)

func TestReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		pipelineRuns []*v1.PipelineRun
	}{
		{
			name:         "no pipelineRuns",
			key:          "foo/bar",
			pipelineRuns: []*v1.PipelineRun{},
		},
		{
			name: "found PipelineRun",
			key:  "foo/bar",
			pipelineRuns: []*v1.PipelineRun{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bar",
						Namespace: "foo",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx, _ := rtesting.SetupFakeContext(t)
			setupData(ctx, t, tt.pipelineRuns)

			configMapWatcher := configmap.NewStaticWatcher(&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: system.Namespace(),
					Name:      config.ChainsConfig,
				},
			})
			ctl := NewControllerV1(ctx, configMapWatcher)

			if la, ok := ctl.Reconciler.(pkgreconciler.LeaderAware); ok {
				if err := la.Promote(pkgreconciler.UniversalBucket(), func(pkgreconciler.Bucket, types.NamespacedName) {}); err != nil {
					t.Fatalf("Promote() = %v", err)
				}
			}

			if err := ctl.Reconciler.Reconcile(ctx, tt.key); err != nil {
				t.Errorf("Reconciler.Reconcile() error = %v", err)
			}
		})
	}
}

func setupData(ctx context.Context, t *testing.T, prs []*v1.PipelineRun) informers.PipelineRunInformer {
	pri := fakepipelineruninformer.Get(ctx)
	c := fakepipelineclient.Get(ctx)

	for _, pa := range prs {
		pa := pa.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.TektonV1().PipelineRuns(pa.Namespace).Create(ctx, pa, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	c.ClearActions()
	return pri
}

func TestReconciler_handlePipelineRun(t *testing.T) {

	tests := []struct {
		name       string
		pr         *v1.PipelineRun
		taskruns   []*v1.TaskRun
		shouldSign bool
		wantErr    bool
	}{
		{
			name: "complete, already signed",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pipelinerun",
					Namespace:   "default",
					Annotations: map[string]string{signing.ChainsAnnotation: "true"},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					}},
			},
			shouldSign: false,
		},
		{
			name: "complete, not already signed",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pipelinerun",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					}},
			},
			shouldSign: true,
		},
		{
			name: "not complete, not already signed",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pipelinerun",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{},
					}},
			},
			shouldSign: false,
		},
		{
			name: "taskruns completed with full taskrun status",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pipelinerun",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					},
				},
			},
			taskruns: []*v1.TaskRun{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "taskrun1",
						Namespace: "default",
						Annotations: map[string]string{
							"chains.tekton.dev/signed": "true",
						},
					},
					Status: v1.TaskRunStatus{
						TaskRunStatusFields: v1.TaskRunStatusFields{
							CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
						},
					},
				},
			},
			shouldSign: true,
			wantErr:    false,
		},
		{
			name: "taskruns completed with child references",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pipelinerun",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					},
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						ChildReferences: []v1.ChildStatusReference{
							{
								Name:             "taskrun1",
								PipelineTaskName: "task1",
							},
						},
					},
				},
			},
			taskruns: []*v1.TaskRun{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "taskrun1",
						Namespace: "default",
						Annotations: map[string]string{
							"chains.tekton.dev/signed": "true",
						},
					},
					Status: v1.TaskRunStatus{
						TaskRunStatusFields: v1.TaskRunStatusFields{
							CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
						},
					},
				},
			},
			shouldSign: true,
			wantErr:    false,
		},
		{
			name: "taskruns not yet completed with child references",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pipelinerun",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					},
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						ChildReferences: []v1.ChildStatusReference{
							{
								Name:             "taskrun1",
								PipelineTaskName: "task1",
							},
						},
					},
				},
			},
			taskruns: []*v1.TaskRun{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "taskrun1",
						Namespace: "default",
					},
				},
			},
			shouldSign: false,
			wantErr:    true,
		},
		{
			name: "missing taskrun with child references",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pipelinerun",
					Namespace:   "default",
					Annotations: map[string]string{},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					},
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						ChildReferences: []v1.ChildStatusReference{
							{
								Name:             "taskrun1",
								PipelineTaskName: "task1",
							},
						},
					},
				},
			},
			shouldSign: false,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &mocksigner.Signer{}
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)
			tekton.CreateObject(t, ctx, c, objects.NewPipelineRunObjectV1(tt.pr))
			tri := faketaskruninformer.Get(ctx)

			r := &Reconciler{
				PipelineRunSigner: signer,
				Pipelineclientset: c,
				TaskRunLister:     tri.Lister(),
				Tracker:           &reconcilertesting.FakeTracker{},
			}

			// Create mock taskruns
			for _, tr := range tt.taskruns {
				if err := tri.Informer().GetIndexer().Add(tr); err != nil {
					t.Fatalf("Adding TaskRun to informer: %v", err)
				}
				// Ensure the TaskRun was indeed added successfully
				if _, err := tri.Lister().TaskRuns(tt.pr.Namespace).Get(tr.Name); err != nil {
					t.Fatalf("TaskRun not added to informer: %v, namespace: %v", err, tt.pr.Namespace)
				}
			}
			ctx = config.ToContext(ctx, &config.Config{})
			if err := r.ReconcileKind(ctx, tt.pr); err != nil && !tt.wantErr {
				t.Errorf("Reconciler.handlePipelineRun() error = %v", err)
			}
			if signer.Signed != tt.shouldSign {
				t.Errorf("Reconciler.handlePipelineRun() signed = %v, wanted %v", signer.Signed, tt.shouldSign)
			}
		})
	}
}
