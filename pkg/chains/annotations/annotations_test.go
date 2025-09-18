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

package annotations

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
				"chains.tekton.dev/extra": "bar",
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
			if signed.GetAnnotations()["chains.tekton.dev/extra"] != "bar" {
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

// TestAddAnnotationsValidation tests the new validation logic for annotations
func TestAddAnnotationsValidation(t *testing.T) {
	tests := []struct {
		name            string
		annotations     map[string]string
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "valid chains annotation",
			annotations: map[string]string{
				"chains.tekton.dev/test": "value",
			},
			wantErr: false,
		},
		{
			name: "invalid annotation without prefix",
			annotations: map[string]string{
				"invalid-key": "value",
			},
			wantErr:         true,
			wantErrContains: "invalid annotation key",
		},
		{
			name: "valid annotations with prefix",
			annotations: map[string]string{
				"chains.tekton.dev/test":   "value",
				"chains.tekton.dev/extra1": "value1",
				"chains.tekton.dev/extra2": "value2",
			},
			wantErr: false,
		},
		{
			name: "mixed valid and invalid annotations",
			annotations: map[string]string{
				"chains.tekton.dev/extra1": "value1",
				"invalid":                  "value2",
			},
			wantErr:         true,
			wantErrContains: "invalid annotation key",
		},
		{
			name:        "empty annotations map",
			annotations: map[string]string{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)

			obj := objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-taskrun",
					Namespace: "default",
				},
			})

			tekton.CreateObject(t, ctx, c, obj)

			err := AddAnnotations(ctx, obj, c, tt.annotations)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AddAnnotations() expected error but got none")
				} else if tt.wantErrContains != "" && !contains(err.Error(), tt.wantErrContains) {
					t.Errorf("AddAnnotations() error = %v, wantErrContains %v", err, tt.wantErrContains)
				}
			} else {
				if err != nil {
					t.Errorf("AddAnnotations() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestAnnotationPreservation tests that only Chains annotations are preserved
func TestAnnotationPreservation(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	c := fakepipelineclient.Get(ctx)

	// Create object with mixed annotations
	obj := objects.NewTaskRunObjectV1(&v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-taskrun",
			Namespace: "default",
			Annotations: map[string]string{
				"chains.tekton.dev/existing": "keep-me",
				"tekton.dev/other":           "ignore-me",
				"kubernetes.io/annotation":   "ignore-me",
				"chains.tekton.dev/another":  "keep-me-too",
			},
		},
	})

	tekton.CreateObject(t, ctx, c, obj)

	// Add a new annotation
	err := AddAnnotations(ctx, obj, c, map[string]string{"chains.tekton.dev/new": "new-value"})
	if err != nil {
		t.Fatalf("AddAnnotations() error = %v", err)
	}

	// Verify the result
	updated, err := tekton.GetObject(t, ctx, c, obj)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	annotations := updated.GetAnnotations()

	// Should have all chains annotations
	expectedChains := map[string]string{
		"chains.tekton.dev/existing": "keep-me",
		"chains.tekton.dev/another":  "keep-me-too",
		"chains.tekton.dev/new":      "new-value",
	}

	for k, v := range expectedChains {
		if got := annotations[k]; got != v {
			t.Errorf("Expected annotation %s=%s, got %s", k, v, got)
		}
	}

	// Should still have non-chains annotations (they weren't removed, just not included in patch)
	expectedNonChains := map[string]string{
		"tekton.dev/other":         "ignore-me",
		"kubernetes.io/annotation": "ignore-me",
	}

	for k, v := range expectedNonChains {
		if got := annotations[k]; got != v {
			t.Errorf("Expected non-chains annotation %s=%s to be preserved, got %s", k, v, got)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCreateAnnotationsPatch(t *testing.T) {
	// Create a real TaskRun object to test with
	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-taskrun",
			Namespace: "test-namespace",
		},
	}

	// Convert to TektonObject
	obj := objects.NewTaskRunObjectV1(tr)

	tests := []struct {
		name           string
		newAnnotations map[string]string
		want           string
		wantErr        bool
	}{
		{
			name:           "empty",
			newAnnotations: map[string]string{},
			want:           `{"apiVersion":"tekton.dev/v1","kind":"TaskRun","metadata":{"name":"test-taskrun","namespace":"test-namespace"}}`,
		},
		{
			name: "one",
			newAnnotations: map[string]string{
				"foo": "bar",
			},
			want: `{"apiVersion":"tekton.dev/v1","kind":"TaskRun","metadata":{"name":"test-taskrun","namespace":"test-namespace","annotations":{"foo":"bar"}}}`,
		},
		{
			name: "many",
			newAnnotations: map[string]string{
				"foo": "bar",
				"baz": "bat",
			},
			want: `{"apiVersion":"tekton.dev/v1","kind":"TaskRun","metadata":{"name":"test-taskrun","namespace":"test-namespace","annotations":{"baz":"bat","foo":"bar"}}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CreateAnnotationsPatch(tt.newAnnotations, obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAnnotationsPatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotStr := string(got)
			if gotStr != tt.want {
				t.Errorf("CreateAnnotationsPatch() = %v, want %v", gotStr, tt.want)
			}
		})
	}
}

func TestTransparencyAnnotation(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	c := fakepipelineclient.Get(ctx)

	// Create a TaskRun with transparency annotation
	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-taskrun",
			Namespace: "default",
		},
	}

	obj := objects.NewTaskRunObjectV1(tr)
	tekton.CreateObject(t, ctx, c, obj)

	// Test adding transparency annotation
	transparencyAnnotations := map[string]string{
		ChainsTransparencyAnnotation: "https://rekor.sigstore.dev/api/v1/log/entries?logIndex=123",
	}

	err := AddAnnotations(ctx, obj, c, transparencyAnnotations)
	if err != nil {
		t.Fatalf("AddAnnotations() error = %v", err)
	}

	// Verify the annotation was added
	updated, err := tekton.GetObject(t, ctx, c, obj)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	annotations := updated.GetAnnotations()
	if v, ok := annotations[ChainsTransparencyAnnotation]; !ok || v == "" {
		t.Errorf("Expected transparency annotation to be set, got %v", annotations)
	}

	expectedValue := "https://rekor.sigstore.dev/api/v1/log/entries?logIndex=123"
	if annotations[ChainsTransparencyAnnotation] != expectedValue {
		t.Errorf("Expected transparency annotation value %q, got %q", expectedValue, annotations[ChainsTransparencyAnnotation])
	}
}
