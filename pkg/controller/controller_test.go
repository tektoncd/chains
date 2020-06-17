package controller

import (
	"context"
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	informers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	faketaskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		if _, err := c.TektonV1beta1().TaskRuns(ta.Namespace).Create(ta); err != nil {
			t.Fatal(err)
		}
	}
	c.ClearActions()
	return tri
}
