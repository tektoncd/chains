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

package chains

import (
	"testing"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/test/tekton"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestReconciled(t *testing.T) {
	tests := []struct {
		name             string
		annotation       string
		latestAnnotation string
		want             bool
	}{
		{
			name:       "signed success",
			want:       true,
			annotation: "true",
		},
		{
			name:       "signed failed",
			want:       true,
			annotation: "failed",
		},
		{
			name:       "signed with other string",
			want:       false,
			annotation: "baz",
		},
		{
			name:       "not signed",
			want:       false,
			annotation: "",
		},
		{
			name:             "latest signed success",
			want:             true,
			latestAnnotation: "true",
		},
		{
			name:             "latest signed failed",
			want:             true,
			latestAnnotation: "failed",
		},
		{
			name:             "latest signed with other string",
			want:             false,
			latestAnnotation: "baz",
		},
		{
			name:             "latest not signed",
			want:             false,
			latestAnnotation: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)

			// Test TaskRun
			taskRun := objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ChainsAnnotation: tt.annotation,
					},
				},
			})
			tekton.CreateObject(t, ctx, c, taskRun)

			cachedTaskRun := objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ChainsAnnotation: tt.latestAnnotation,
					},
				},
			})

			got := Reconciled(ctx, c, cachedTaskRun)
			if got != tt.want {
				t.Errorf("Reconciled() got = %v, want %v", got, tt.want)
			}

			// Test PipelineRun
			pipelineRun := objects.NewPipelineRunObjectV1(&v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ChainsAnnotation: tt.annotation,
					},
				},
			})
			tekton.CreateObject(t, ctx, c, pipelineRun)

			cachedPipelineRun := objects.NewPipelineRunObjectV1(&v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ChainsAnnotation: tt.latestAnnotation,
					},
				},
			})

			got = Reconciled(ctx, c, cachedPipelineRun)
			if got != tt.want {
				t.Errorf("Reconciled() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarkSigned(t *testing.T) {
	tests := []struct {
		name   string
		object objects.TektonObject
	}{
		{
			name: "mark taskrun",
			object: objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-taskrun",
				},
				Spec: v1.TaskRunSpec{
					TaskRef: &v1.TaskRef{
						Name: "foo",
					},
				},
			}),
		},
		{
			name: "mark pipelinerun",
			object: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-pipelinerun",
				},
				Spec: v1.PipelineRunSpec{
					PipelineRef: &v1.PipelineRef{
						Name: "foo",
					},
				},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)

			tekton.CreateObject(t, ctx, c, tt.object)

			// Now mark it as signed.
			if err := MarkSigned(ctx, tt.object, c, nil); err != nil {
				t.Errorf("MarkSigned() error = %v", err)
			}

			// Now check the signature.
			signed, err := tekton.GetObject(t, ctx, c, tt.object)
			if err != nil {
				t.Errorf("Get() error = %v", err)
			}
			if _, ok := signed.GetAnnotations()[ChainsAnnotation]; !ok {
				t.Error("Object not signed.")
			}

			// Try some extra annotations

			// Now mark it as signed.
			extra := map[string]string{
				"foo": "bar",
			}

			if err := MarkSigned(ctx, tt.object, c, extra); err != nil {
				t.Errorf("MarkSigned() error = %v", err)
			}

			// Now check the signature.
			signed, err = tekton.GetObject(t, ctx, c, tt.object)
			if err != nil {
				t.Errorf("Get() error = %v", err)
			}
			if _, ok := signed.GetAnnotations()[ChainsAnnotation]; !ok {
				t.Error("Object not signed.")
			}
			if signed.GetAnnotations()["foo"] != "bar" {
				t.Error("Extra annotations not applied")
			}
		})
	}
}

func TestMarkFailed(t *testing.T) {
	tests := []struct {
		name   string
		object objects.TektonObject
	}{
		{
			name: "mark taskrun failed",
			object: objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-taskrun",
					Annotations: map[string]string{RetryAnnotation: "3"},
				},
				Spec: v1.TaskRunSpec{
					TaskRef: &v1.TaskRef{
						Name: "foo",
					},
				},
			}),
		},
		{
			name: "mark pipelinerun failed",
			object: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-pipelinerun",
					Annotations: map[string]string{RetryAnnotation: "3"},
				},
				Spec: v1.PipelineRunSpec{
					PipelineRef: &v1.PipelineRef{
						Name: "foo",
					},
				},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			// Create a TR for testing
			c := fakepipelineclient.Get(ctx)
			tekton.CreateObject(t, ctx, c, tt.object)

			// Test HandleRetry, should mark it as failed
			if err := HandleRetry(ctx, tt.object, c, nil); err != nil {
				t.Errorf("HandleRetry() error = %v", err)
			}

			failed, err := tekton.GetObject(t, ctx, c, tt.object)
			if err != nil {
				t.Errorf("Get() error = %v", err)
			}

			if failed.GetAnnotations()[ChainsAnnotation] != "failed" {
				t.Errorf("Object not marked as 'failed', was: '%s'", failed.GetAnnotations()[ChainsAnnotation])
			}
		})
	}
}

func TestRetryAvailble(t *testing.T) {

	tests := []struct {
		description string
		annotations map[string]string
		expected    bool
	}{
		{
			description: "no annotation set",
			expected:    true,
		}, {
			description: "annotation < 3",
			annotations: map[string]string{
				RetryAnnotation: "2",
			},
			expected: true,
		}, {
			description: "annotation not a number",
			annotations: map[string]string{
				RetryAnnotation: "sfd",
			},
		}, {
			description: "annotation is 3",
			annotations: map[string]string{
				RetryAnnotation: "3",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// test taskrun
			tr := &v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: test.annotations,
				},
			}
			trObj := objects.NewTaskRunObjectV1(tr)
			got := RetryAvailable(trObj)
			if got != test.expected {
				t.Fatalf("RetryAvailble() got %v expected %v", got, test.expected)
			}
			// test pipelinerun
			pr := &v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: test.annotations,
				},
			}
			prObj := objects.NewPipelineRunObjectV1(pr)
			got = RetryAvailable(prObj)
			if got != test.expected {
				t.Fatalf("RetryAvailble() got %v expected %v", got, test.expected)
			}
		})
	}
}

func TestAddRetry(t *testing.T) {
	tests := []struct {
		name   string
		object objects.TektonObject
	}{
		{
			name: "add retry to taskrun",
			object: objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{Name: "mytaskrun"},
			}),
		},
		{
			name: "add retry to pipelinerun",
			object: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{Name: "mypipelinerun"},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)

			tekton.CreateObject(t, ctx, c, tt.object)

			// run it through AddRetry, make sure annotation is added
			if err := AddRetry(ctx, tt.object, c, nil); err != nil {
				t.Fatal(err)
			}

			signed, err := tekton.GetObject(t, ctx, c, tt.object)
			if err != nil {
				t.Errorf("Get() error = %v", err)
			}

			if val, ok := signed.GetAnnotations()[RetryAnnotation]; !ok || val != "0" {
				t.Fatalf("annotation isn't correct: %v %v", ok, val)
			}

			// run it again, make sure we see an increase
			if err := AddRetry(ctx, signed, c, nil); err != nil {
				t.Fatal(err)
			}
			signed, err = tekton.GetObject(t, ctx, c, tt.object)
			if err != nil {
				t.Errorf("Get() error = %v", err)
			}
			if val, ok := signed.GetAnnotations()[RetryAnnotation]; val != "1" {
				t.Fatalf("annotation isn't correct: %v %v", ok, val)
			}
		})
	}
}
