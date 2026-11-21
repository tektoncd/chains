/*
Copyright 2026 The Tekton Authors

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

	"github.com/tektoncd/chains/pkg/chains/annotations"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/test/tekton"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	faketaskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1/taskrun/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	rtesting "knative.dev/pkg/reconciler/testing"
)

// customRunTestSigner records the ordinary TaskRuns passed to the PipelineRun signer.
type customRunTestSigner struct {
	calls    int           // calls counts signing attempts, including repeated reconciliations.
	taskRuns []*v1.TaskRun // taskRuns contains the TaskRuns included in the signed object.
}

// Sign records a PipelineRun signing attempt without changing its annotations.
func (s *customRunTestSigner) Sign(_ context.Context, obj objects.TektonObject) error {
	s.calls++
	s.taskRuns = obj.(*objects.PipelineRunObjectV1).GetTaskRuns()
	return nil
}

// TestReconciler_CustomRunChildren verifies CustomRuns never weaken ordinary TaskRun gates.
func TestReconciler_CustomRunChildren(t *testing.T) {
	const (
		// namespaceName isolates the fixtures in the default test namespace.
		namespaceName = "default"
		// pipelineRunName identifies the PipelineRun fixture.
		pipelineRunName = "pipelinerun"
		// reconciledValue is the existing durable Chains completion marker.
		reconciledValue = "true"
		// signedTaskRun selects a completed, reconciled ordinary TaskRun.
		signedTaskRun = "signed"
		// pendingTaskRun selects an ordinary TaskRun without completionTime.
		pendingTaskRun = "pending"
	)
	// taskChild is an explicit ordinary TaskRun reference.
	taskChild := v1.ChildStatusReference{TypeMeta: runtime.TypeMeta{APIVersion: "tekton.dev/v1", Kind: "TaskRun"}, Name: "taskrun", PipelineTaskName: "build"}
	// customChild has no matching TaskRun; querying it would block PipelineRun signing.
	customChild := v1.ChildStatusReference{TypeMeta: runtime.TypeMeta{APIVersion: "tekton.dev/v1beta1", Kind: "CustomRun"}, Name: "customrun", PipelineTaskName: "approval"}
	// legacyChild preserves the historical empty-Kind TaskRun representation.
	legacyChild := v1.ChildStatusReference{Name: taskChild.Name, PipelineTaskName: taskChild.PipelineTaskName}
	// unknownChild preserves the existing lookup behavior for unrecognized kinds.
	unknownChild := taskChild
	unknownChild.Kind = "UnknownRun"
	// collidingChild makes an incorrect TaskRun lookup observable even when a same-name TaskRun exists.
	collidingChild := customChild
	collidingChild.Name = taskChild.Name

	// tests exercise both the restored signing path and the compatibility gates.
	tests := []struct {
		name          string                    // name identifies the regression scenario.
		children      []v1.ChildStatusReference // children are the PipelineRun's reported child resources.
		taskRunState  string                    // taskRunState selects absent, pending, unreconciled, or signed ordinary TaskRun data.
		running       bool                      // running leaves the PipelineRun's Succeeded condition unknown.
		failed        bool                      // failed exercises the existing terminal-failure signing policy.
		alreadySigned bool                      // alreadySigned exercises the existing idempotency gate.
		wantSign      bool                      // wantSign indicates whether the signer should be called.
		wantTaskRuns  int                       // wantTaskRuns is the ordinary TaskRun count in the signed object.
		wantTracked   int                       // wantTracked is the ordinary TaskRun count registered for follow-up events.
	}{
		{name: "only CustomRun", children: []v1.ChildStatusReference{customChild}, wantSign: true},
		{name: "TaskRun before CustomRun", children: []v1.ChildStatusReference{taskChild, customChild}, taskRunState: signedTaskRun, wantSign: true, wantTaskRuns: 1},
		{name: "CustomRun before TaskRun", children: []v1.ChildStatusReference{customChild, taskChild}, taskRunState: signedTaskRun, wantSign: true, wantTaskRuns: 1},
		{name: "failed PipelineRun with CustomRun", children: []v1.ChildStatusReference{customChild, taskChild}, taskRunState: signedTaskRun, failed: true, wantSign: true, wantTaskRuns: 1},
		{name: "running PipelineRun with CustomRun", children: []v1.ChildStatusReference{customChild}, running: true},
		{name: "CustomRun with pending TaskRun", children: []v1.ChildStatusReference{customChild, taskChild}, taskRunState: pendingTaskRun, wantTracked: 1},
		{name: "CustomRun with unreconciled TaskRun", children: []v1.ChildStatusReference{customChild, taskChild}, taskRunState: "unreconciled", wantTracked: 1},
		{name: "CustomRun with missing TaskRun", children: []v1.ChildStatusReference{customChild, taskChild}},
		{name: "empty Kind TaskRun", children: []v1.ChildStatusReference{customChild, legacyChild}, taskRunState: signedTaskRun, wantSign: true, wantTaskRuns: 1},
		{name: "empty Kind missing TaskRun", children: []v1.ChildStatusReference{customChild, legacyChild}},
		{name: "unknown Kind still queries TaskRun", children: []v1.ChildStatusReference{customChild, unknownChild}, taskRunState: pendingTaskRun, wantTracked: 1},
		{name: "CustomRun ignores same-name pending TaskRun", children: []v1.ChildStatusReference{collidingChild}, taskRunState: pendingTaskRun, wantSign: true},
		{name: "already signed with CustomRun", children: []v1.ChildStatusReference{customChild, taskChild}, alreadySigned: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ctx and client provide isolated PipelineRun annotations for this scenario.
			ctx, _ := rtesting.SetupFakeContext(t)
			client := fakepipelineclient.Get(ctx)
			// pr is terminal by default, so TaskRun reconciliation remains the only signing gate.
			pr := &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{Name: pipelineRunName, Namespace: namespaceName, Annotations: map[string]string{}},
				Status: v1.PipelineRunStatus{
					Status:                  duckv1.Status{Conditions: []apis.Condition{{Type: apis.ConditionSucceeded, Status: corev1.ConditionTrue}}},
					PipelineRunStatusFields: v1.PipelineRunStatusFields{ChildReferences: tt.children},
				},
			}
			if tt.running {
				pr.Status.Conditions[0].Status = corev1.ConditionUnknown
			}
			if tt.failed {
				pr.Status.Conditions[0].Status = corev1.ConditionFalse
			}
			if tt.alreadySigned {
				pr.Annotations[annotations.ChainsAnnotation] = reconciledValue
			}
			tekton.CreateObject(t, ctx, client, objects.NewPipelineRunObjectV1(pr))
			// informer contains only ordinary TaskRuns, never a CustomRun masquerading as one.
			informer := faketaskruninformer.Get(ctx)
			if tt.taskRunState != "" {
				// tr starts completed and signed; each gate scenario removes its required state.
				tr := &v1.TaskRun{
					TypeMeta:   metav1.TypeMeta{APIVersion: taskChild.APIVersion, Kind: taskChild.Kind},
					ObjectMeta: metav1.ObjectMeta{Name: taskChild.Name, Namespace: pr.Namespace, Annotations: map[string]string{annotations.ChainsAnnotation: reconciledValue}},
					Status:     v1.TaskRunStatus{TaskRunStatusFields: v1.TaskRunStatusFields{CompletionTime: &metav1.Time{}}},
				}
				if tt.taskRunState == pendingTaskRun {
					tr.Status.CompletionTime = nil
				}
				if tt.taskRunState == "unreconciled" {
					tr.Annotations = nil
				}
				if err := informer.Informer().GetIndexer().Add(tr); err != nil {
					t.Fatalf("Adding TaskRun: %v", err)
				}
				tekton.CreateObject(t, ctx, client, objects.NewTaskRunObjectV1(tr))
			}
			// signer captures provenance inputs while tracker captures ordinary TaskRun wait gates.
			signer := &customRunTestSigner{}
			tracker := &rtesting.FakeTracker{}
			reconciler := &Reconciler{PipelineRunSigner: signer, Pipelineclientset: client, TaskRunLister: informer.Lister(), Tracker: tracker}
			if err := reconciler.ReconcileKind(ctx, pr); err != nil {
				t.Fatalf("ReconcileKind() = %v", err)
			}
			if (signer.calls == 1) != tt.wantSign {
				t.Fatalf("signing calls = %d, wantSign = %v", signer.calls, tt.wantSign)
			}
			if len(signer.taskRuns) != tt.wantTaskRuns {
				t.Errorf("signed TaskRun count = %d, want %d", len(signer.taskRuns), tt.wantTaskRuns)
			}
			if len(tracker.References()) != tt.wantTracked {
				t.Errorf("tracked TaskRun count = %d, want %d", len(tracker.References()), tt.wantTracked)
			}
			if tt.wantSign {
				// Simulate successful signing's durable completion marker before another reconcile.
				pr.Annotations[annotations.ChainsAnnotation] = reconciledValue
				if _, err := client.TektonV1().PipelineRuns(pr.Namespace).Update(ctx, pr, metav1.UpdateOptions{}); err != nil {
					t.Fatalf("Marking PipelineRun reconciled: %v", err)
				}
				if err := reconciler.ReconcileKind(ctx, pr); err != nil {
					t.Fatalf("Repeated ReconcileKind() = %v", err)
				}
				if signer.calls != 1 {
					t.Errorf("already reconciled PipelineRun signed again: calls = %d", signer.calls)
				}
			}
		})
	}
}
