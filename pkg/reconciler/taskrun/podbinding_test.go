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

package taskrun

import (
	"testing"

	"github.com/tektoncd/chains/pkg/chains/annotations"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/internal/mocksigner"
	"github.com/tektoncd/chains/pkg/test/tekton"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestReconciler_VerifyPodBinding(t *testing.T) {
	const (
		ns      = "default"
		trName  = "build"
		trUID   = types.UID("taskrun-uid")
		podName = "build-pod"
	)
	pod := func(owner *metav1.OwnerReference) *corev1.Pod {
		p := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: ns}}
		if owner != nil {
			p.OwnerReferences = []metav1.OwnerReference{*owner}
		}
		return p
	}
	isController := true
	boundPod := pod(&metav1.OwnerReference{
		APIVersion: "tekton.dev/v1", Kind: "TaskRun", Name: trName, UID: trUID, Controller: &isController,
	})
	otherPod := pod(&metav1.OwnerReference{
		APIVersion: "tekton.dev/v1", Kind: "TaskRun", Name: "other", UID: "other-uid", Controller: &isController,
	})

	tests := []struct {
		name         string
		mode         string
		pods         []runtime.Object
		wantSigned   bool
		wantVerified string // value of ExecutionVerifiedAnnotation, "" when unset
		wantChains   string // value of ChainsAnnotation, "" when unset
	}{
		{
			name:       "off ignores a foreign pod",
			mode:       config.VerifyPodBindingOff,
			pods:       []runtime.Object{otherPod},
			wantSigned: true,
		},
		{
			name:       "unset mode skips the check",
			mode:       "",
			pods:       []runtime.Object{otherPod},
			wantSigned: true,
		},
		{
			name:       "warn with the taskrun's pod",
			mode:       config.VerifyPodBindingWarn,
			pods:       []runtime.Object{boundPod},
			wantSigned: true,
		},
		{
			name:         "warn with a missing pod",
			mode:         config.VerifyPodBindingWarn,
			wantSigned:   true,
			wantVerified: "false",
		},
		{
			name:         "warn with a foreign pod",
			mode:         config.VerifyPodBindingWarn,
			pods:         []runtime.Object{otherPod},
			wantSigned:   true,
			wantVerified: "false",
		},
		{
			name:         "warn with an unowned pod",
			mode:         config.VerifyPodBindingWarn,
			pods:         []runtime.Object{pod(nil)},
			wantSigned:   true,
			wantVerified: "false",
		},
		{
			name:         "block still signs when the pod is gone",
			mode:         config.VerifyPodBindingBlock,
			wantSigned:   true,
			wantVerified: "false",
		},
		{
			name:         "block refuses a foreign pod",
			mode:         config.VerifyPodBindingBlock,
			pods:         []runtime.Object{otherPod},
			wantSigned:   false,
			wantVerified: "false",
			wantChains:   "failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			ps := fakepipelineclient.Get(ctx)
			tr := &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Name: trName, Namespace: ns, UID: trUID, Annotations: map[string]string{}},
				Status: v1.TaskRunStatus{
					Status: duckv1.Status{
						Conditions: []apis.Condition{{Type: apis.ConditionSucceeded}},
					},
					TaskRunStatusFields: v1.TaskRunStatusFields{PodName: podName},
				},
			}
			tekton.CreateObject(t, ctx, ps, objects.NewTaskRunObjectV1(tr))

			signer := &mocksigner.Signer{}
			r := &Reconciler{
				TaskRunSigner:     signer,
				Pipelineclientset: ps,
				KubeClientSet:     fakek8s.NewSimpleClientset(tt.pods...),
			}
			ctx = config.ToContext(ctx, &config.Config{Security: config.SecurityConfig{VerifyPodBinding: tt.mode}})
			if err := r.ReconcileKind(ctx, tr); err != nil {
				t.Fatalf("ReconcileKind() error = %v", err)
			}
			if signer.Signed != tt.wantSigned {
				t.Errorf("signed = %v, want %v", signer.Signed, tt.wantSigned)
			}

			got, err := ps.TektonV1().TaskRuns(ns).Get(ctx, trName, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if v := got.Annotations[ExecutionVerifiedAnnotation]; v != tt.wantVerified {
				t.Errorf("%s = %q, want %q", ExecutionVerifiedAnnotation, v, tt.wantVerified)
			}
			if v := got.Annotations[annotations.ChainsAnnotation]; v != tt.wantChains {
				t.Errorf("%s = %q, want %q", annotations.ChainsAnnotation, v, tt.wantChains)
			}
		})
	}
}
