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

	"github.com/tektoncd/chains/pkg/chains/annotations"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/mocksigner"
	_ "github.com/tektoncd/chains/pkg/pipelinerunmetrics/fake"
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
			namespacedScopedController := NewNamespacesScopedController(nil)
			ctl := namespacedScopedController(ctx, configMapWatcher)

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
					Annotations: map[string]string{annotations.ChainsAnnotation: "true"},
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
				Tracker:           &rtesting.FakeTracker{},
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

			if err := r.ReconcileKind(ctx, tt.pr); err != nil && !tt.wantErr {
				t.Errorf("Reconciler.handlePipelineRun() error = %v", err)
			}
			if signer.Signed != tt.shouldSign {
				t.Errorf("Reconciler.handlePipelineRun() signed = %v, wanted %v", signer.Signed, tt.shouldSign)
			}
		})
	}
}

func TestFinalizeKind_SSAMigration(t *testing.T) {
	now := metav1.Now()
	mergePatchFieldsV1 := metav1.NewFieldsV1(
		`{"f:metadata":{"f:finalizers":{".":{},"v:\"chains.tekton.dev/pipelinerun\"":{}}}}`,
	)
	ssaFieldsV1 := metav1.NewFieldsV1(
		`{"f:metadata":{"f:finalizers":{".":{},"v:\"chains.tekton.dev/pipelinerun\"":{}}}}`,
	)

	tests := []struct {
		name                   string
		pr                     *v1.PipelineRun
		expectFinalizerRemoved bool
	}{
		{
			name: "migration removes merge-patch finalizer on deletion",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pipelinerun",
					Namespace:         "default",
					DeletionTimestamp: &now,
					Finalizers:        []string{pipelineRunFinalizerName, "other.finalizer"},
					Annotations:       map[string]string{annotations.ChainsAnnotation: "true"},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Operation: metav1.ManagedFieldsOperationUpdate,
							FieldsV1:  mergePatchFieldsV1,
						},
					},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					},
				},
			},
			expectFinalizerRemoved: true,
		},
		{
			name: "migration skips when no DeletionTimestamp",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pipelinerun",
					Namespace:   "default",
					Finalizers:  []string{pipelineRunFinalizerName},
					Annotations: map[string]string{annotations.ChainsAnnotation: "true"},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Operation: metav1.ManagedFieldsOperationUpdate,
							FieldsV1:  mergePatchFieldsV1,
						},
					},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					},
				},
			},
			expectFinalizerRemoved: false,
		},
		{
			name: "migration skips when finalizer is SSA-owned",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pipelinerun",
					Namespace:         "default",
					DeletionTimestamp: &now,
					Finalizers:        []string{pipelineRunFinalizerName},
					Annotations:       map[string]string{annotations.ChainsAnnotation: "true"},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Operation: metav1.ManagedFieldsOperationApply,
							FieldsV1:  ssaFieldsV1,
						},
					},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					},
				},
			},
			expectFinalizerRemoved: false,
		},
		{
			name: "migration skips when no managed fields match",
			pr: &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pipelinerun",
					Namespace:         "default",
					DeletionTimestamp: &now,
					Finalizers:        []string{pipelineRunFinalizerName},
					Annotations:       map[string]string{annotations.ChainsAnnotation: "true"},
				},
				Status: v1.PipelineRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					},
				},
			},
			expectFinalizerRemoved: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)
			tri := faketaskruninformer.Get(ctx)

			// Create the object in the fake client (without DeletionTimestamp since
			// the fake client does not allow creating objects with it set).
			createObj := tt.pr.DeepCopy()
			createObj.DeletionTimestamp = nil
			if _, err := c.TektonV1().PipelineRuns(createObj.Namespace).Create(ctx, createObj, metav1.CreateOptions{}); err != nil {
				t.Fatal(err)
			}

			r := &Reconciler{
				PipelineRunSigner: &mocksigner.Signer{},
				Pipelineclientset: c,
				TaskRunLister:     tri.Lister(),
				Tracker:           &rtesting.FakeTracker{},
			}
			ctx = config.ToContext(ctx, &config.Config{})

			if err := r.FinalizeKind(ctx, tt.pr); err != nil {
				t.Fatalf("FinalizeKind() error = %v", err)
			}

			got, err := c.TektonV1().PipelineRuns(tt.pr.Namespace).Get(ctx, tt.pr.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			hasChainsFinalizer := false
			for _, f := range got.Finalizers {
				if f == pipelineRunFinalizerName {
					hasChainsFinalizer = true
				}
			}

			if tt.expectFinalizerRemoved && hasChainsFinalizer {
				t.Errorf("expected chains finalizer to be removed, but it is still present: %v", got.Finalizers)
			}
			if !tt.expectFinalizerRemoved && !hasChainsFinalizer && len(tt.pr.Finalizers) > 0 {
				t.Errorf("expected chains finalizer to be preserved, but it was removed: %v", got.Finalizers)
			}

			// For the positive case, verify other finalizers are preserved.
			if tt.expectFinalizerRemoved {
				hasOther := false
				for _, f := range got.Finalizers {
					if f == "other.finalizer" {
						hasOther = true
					}
				}
				if !hasOther {
					t.Errorf("expected other.finalizer to be preserved, but got: %v", got.Finalizers)
				}
			}
		})
	}
}
