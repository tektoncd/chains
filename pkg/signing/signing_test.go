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

package signing

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/chains/pkg/signing/formats"
	"github.com/tektoncd/chains/pkg/signing/storage"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestMarkSigned(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	// Create a TR for testing
	c := fakepipelineclient.Get(ctx)
	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-taskrun",
		},
		Spec: v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name: "foo",
			},
		},
	}
	if _, err := c.TektonV1beta1().TaskRuns(tr.Namespace).Create(tr); err != nil {
		t.Fatal(err)
	}

	// Now mark it as signed.
	if err := MarkSigned(tr, c); err != nil {
		t.Errorf("MarkSigned() error = %v", err)
	}

	// Now check the signature.
	signed, err := c.TektonV1beta1().TaskRuns(tr.Namespace).Get(tr.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if _, ok := signed.Annotations[ChainsAnnotation]; !ok {
		t.Error("Taskrun not signed.")
	}
}

func TestIsSigned(t *testing.T) {
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
			got := IsSigned(tr)
			if got != tt.want {
				t.Errorf("IsSigned() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskRunSigner_SignTaskRun(t *testing.T) {
	// SignTaskRun does three main things:
	// - generates payloads
	// - stores them in the configured systems
	// - marks the taskrun as signed
	tests := []struct {
		name     string
		backends []*mockBackend
	}{
		{
			name: "single system",
			backends: []*mockBackend{
				&mockBackend{},
			},
		},
		{
			name: "multiple systems",
			backends: []*mockBackend{
				&mockBackend{},
				&mockBackend{},
			},
		},
		{
			name: "multiple systems, multiple errors",
			backends: []*mockBackend{
				&mockBackend{},
				&mockBackend{},
				&mockBackend{shouldErr: true},
				&mockBackend{shouldErr: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupMocks(tt.backends)
			defer cleanup()

			ctx, _ := rtesting.SetupFakeContext(t)
			logger := logtesting.TestLogger(t)
			ps := fakepipelineclient.Get(ctx)
			ts := &TaskRunSigner{
				Logger:            logger,
				Pipelineclientset: ps,
			}

			tr := &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			}
			if _, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Create(tr); err != nil {
				t.Errorf("error creating fake taskrun: %v", err)
			}
			if err := ts.SignTaskRun(tr); err != nil {
				t.Errorf("TaskRunSigner.SignTaskRun() error = %v", err)
			}

			// Fetch a new TR!
			tr, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Get(tr.Name, metav1.GetOptions{})
			if err != nil {
				t.Errorf("error fetching fake taskrun: %v", err)
			}
			// Check it is marked as signed
			if !IsSigned(tr) {
				t.Errorf("TaskRun %s/%s should be marked as signed, was not", tr.Namespace, tr.Name)
			}
			// Check the payloads were stored in all the backends.
			payloads := generatePayloads(logger, tr)
			for _, b := range tt.backends {
				if b.shouldErr {
					continue
				}
				if diff := cmp.Diff(payloads, b.storedPayloads); diff != "" {
					t.Errorf("unexpected payload: (-want, +got): %s", diff)
				}
			}

		})
	}
}

func setupMocks(backends []*mockBackend) func() {
	oldGet := getBackends
	getBackends = func(ps versioned.Interface, logger *zap.SugaredLogger) []storage.Backend {
		newBackends := []storage.Backend{}
		for _, m := range backends {
			newBackends = append(newBackends, m)
		}
		return newBackends
	}
	return func() {
		getBackends = oldGet
	}
}

type mockBackend struct {
	storedPayloads map[formats.PayloadType]interface{}
	shouldErr      bool
}

// StorePayload implements the Payloader interface.
func (b *mockBackend) StorePayload(payload interface{}, payloadType formats.PayloadType, tr *v1beta1.TaskRun) error {
	if b.shouldErr {
		return errors.New("mock error storing")
	}
	if b.storedPayloads == nil {
		b.storedPayloads = map[formats.PayloadType]interface{}{}
	}
	b.storedPayloads[payloadType] = payload
	return nil
}

func (b *mockBackend) Type() string {
	return "mock"
}
