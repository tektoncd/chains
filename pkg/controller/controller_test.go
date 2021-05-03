package controller

import (
	"context"
	"testing"

	signing "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	informers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	faketaskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		taskRuns []*v1beta1.TaskRun
	}{
		{
			name:     "no taskruns",
			key:      "foo/bar",
			taskRuns: []*v1beta1.TaskRun{},
		},
		{
			name: "found taskrun",
			key:  "foo/bar",
			taskRuns: []*v1beta1.TaskRun{
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
			tri := setupData(ctx, t, tt.taskRuns)
			r := &Reconciler{
				Logger:        logtesting.TestLogger(t),
				TaskRunLister: tri.Lister(),
			}
			if err := r.Reconcile(ctx, tt.key); err != nil {
				t.Errorf("Reconciler.Reconcile() error = %v", err)
			}
		})
	}
}

func setupData(ctx context.Context, t *testing.T, trs []*v1beta1.TaskRun) informers.TaskRunInformer {
	tri := faketaskruninformer.Get(ctx)
	c := fakepipelineclient.Get(ctx)

	for _, ta := range trs {
		ta := ta.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := c.TektonV1beta1().TaskRuns(ta.Namespace).Create(ctx, ta, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	c.ClearActions()
	return tri
}

func TestReconciler_handleTaskRun(t *testing.T) {

	tests := []struct {
		name       string
		tr         *v1beta1.TaskRun
		shouldSign bool
	}{
		{
			name: "complete, already signed",
			tr: &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{signing.ChainsAnnotation: "true"},
				},
				Status: v1beta1.TaskRunStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					}},
			},
			shouldSign: false,
		},
		{
			name: "complete, not already signed",
			tr: &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Status: v1beta1.TaskRunStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					}},
			},
			shouldSign: true,
		},
		{
			name: "not complete, not already signed",
			tr: &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Status: v1beta1.TaskRunStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{},
					}},
			},
			shouldSign: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &mockSigner{}
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)

			r := &Reconciler{
				PipelineClientSet: c,
				Logger:            logtesting.TestLogger(t),
				TaskRunSigner:     signer,
			}
			if err := r.handleTaskRun(ctx, tt.tr); err != nil {
				t.Errorf("Reconciler.handleTaskRun() error = %v", err)
			}
			if signer.signed != tt.shouldSign {
				t.Errorf("Reconciler.handleTaskRun() signed = %v, wanted %v", signer.signed, tt.shouldSign)
			}
		})
	}
}

type mockSigner struct {
	signed bool
}

func (m *mockSigner) SignTaskRun(ctx context.Context, tr *v1beta1.TaskRun) error {
	m.signed = true
	return nil
}
