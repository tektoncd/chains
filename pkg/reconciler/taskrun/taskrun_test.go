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

package taskrun

import (
	"context"
	"testing"

	signing "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/mocksigner"
	"github.com/tektoncd/chains/pkg/test/tekton"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	informers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
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
		name     string
		key      string
		taskRuns []*v1.TaskRun
	}{
		{
			name:     "no taskruns",
			key:      "foo/bar",
			taskRuns: []*v1.TaskRun{},
		},
		{
			name: "found taskrun",
			key:  "foo/bar",
			taskRuns: []*v1.TaskRun{
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
			setupData(ctx, t, tt.taskRuns)

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

func setupData(ctx context.Context, t *testing.T, trs []*v1.TaskRun) informers.TaskRunInformer {
	tri := faketaskruninformer.Get(ctx)
	c := fakepipelineclient.Get(ctx)

	for _, ta := range trs {
		ta := ta.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.TektonV1().TaskRuns(ta.Namespace).Create(ctx, ta, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	c.ClearActions()
	return tri
}

func TestReconciler_handleTaskRun(t *testing.T) {

	tests := []struct {
		name       string
		tr         *v1.TaskRun
		shouldSign bool
	}{
		{
			name: "complete, already signed",
			tr: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{signing.ChainsAnnotation: "true"},
				},
				Status: v1.TaskRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					}},
			},
			shouldSign: false,
		},
		{
			name: "complete, not already signed",
			tr: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Status: v1.TaskRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					}},
			},
			shouldSign: true,
		},
		{
			name: "not complete, not already signed",
			tr: &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Status: v1.TaskRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{},
					}},
			},
			shouldSign: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &mocksigner.Signer{}
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)
			tekton.CreateObject(t, ctx, c, objects.NewTaskRunObjectV1(tt.tr))

			r := &ReconcilerV1{
				TaskRunSigner:     signer,
				Pipelineclientset: c,
			}
			ctx = config.ToContext(ctx, &config.Config{})
			if err := r.ReconcileKind(ctx, tt.tr); err != nil {
				t.Errorf("Reconciler.handleTaskRun() error = %v", err)
			}
			if signer.Signed != tt.shouldSign {
				t.Errorf("Reconciler.handleTaskRun() signed = %v, wanted %v", signer.Signed, tt.shouldSign)
			}
		})
	}
}
