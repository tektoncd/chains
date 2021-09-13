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

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestReconciled(t *testing.T) {
	tests := []struct {
		name       string
		annotation string
		want       bool
	}{
		{
			name:       "signed",
			want:       true,
			annotation: "true",
		},
		{
			name:       "signed",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						ChainsAnnotation: tt.annotation,
					},
				},
			}
			got := Reconciled(tr)
			if got != tt.want {
				t.Errorf("Reconciled() got = %v, want %v", got, tt.want)
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
			tr := &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: test.annotations,
				},
			}
			got := RetryAvailable(tr)
			if got != test.expected {
				t.Fatalf("RetryAvailble() got %v expected %v", got, test.expected)
			}
		})
	}
}

func TestAddRetry(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	c := fakepipelineclient.Get(ctx)

	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{Name: "mytaskrun"},
	}
	if _, err := c.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	// run it through AddRetry, make sure annotation is added
	if err := AddRetry(tr, c, nil); err != nil {
		t.Fatal(err)
	}

	signed, err := c.TektonV1beta1().TaskRuns(tr.Namespace).Get(ctx, tr.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}

	if val, ok := signed.Annotations[RetryAnnotation]; !ok || val != "0" {
		t.Fatalf("annotation isn't correct: %v %v", ok, val)
	}

	// run it again, make sure we see an increase
	if err := AddRetry(signed, c, nil); err != nil {
		t.Fatal(err)
	}
	signed, err = c.TektonV1beta1().TaskRuns(tr.Namespace).Get(ctx, tr.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if val, ok := signed.Annotations[RetryAnnotation]; val != "1" {
		t.Fatalf("annotation isn't correct: %v %v", ok, val)
	}
}
