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

package tekton

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/storage/api"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/test/tekton"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestBackend_StorePayload(t *testing.T) {
	// The Tekton storage backend JSON serializes, base64 encodes, and attaches the result as an annotation

	tests := []struct {
		name    string
		payload interface{}
		wantErr bool
		object  objects.TektonObject
	}{
		{
			name: "simple taskrun",
			payload: mockPayload{
				A: "foo",
				B: 3,
			},
			object: objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						Results: []v1.TaskRunResult{
							{Name: "IMAGE_URL", Value: *v1.NewStructuredValues("mockImage")},
						},
					},
				},
			}),
		},
		{
			name: "simple pipelinerun",
			payload: mockPayload{
				A: "foo",
				B: 3,
			},
			object: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Status: v1.PipelineRunStatus{
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						Results: []v1.PipelineRunResult{
							{Name: "IMAGE_URL", Value: *v1.NewStructuredValues("mockImage")},
						},
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

			b := &Backend{
				pipelineclientset: c,
			}
			payload, err := json.Marshal(tt.payload)
			if err != nil {
				t.Errorf("error marshaling json: %v", err)
			}
			opts := config.StorageOpts{ShortKey: "mockpayload"}
			mockSignature := "mocksignature"
			if err := b.StorePayload(ctx, tt.object, payload, mockSignature, opts); (err != nil) != tt.wantErr {
				t.Errorf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}

			// The rest is invalid if we wanted an error.
			if tt.wantErr {
				return
			}

			payloadAnnotation := payloadName(opts)
			payloads, err := b.RetrievePayloads(ctx, tt.object, opts)
			if err != nil {
				t.Errorf("error base64 decoding: %v", err)
			}

			mp := mockPayload{}
			if err := json.Unmarshal([]byte(payloads[payloadAnnotation]), &mp); err != nil {
				t.Errorf("error json decoding: %v", err)
			}

			// Compare to the input.
			if diff := cmp.Diff(tt.payload, mp); diff != "" {
				t.Errorf("unexpected payload: (-want, +got): %s", diff)
			}

			// Compare the signature.
			signatureAnnotation := sigName(opts)
			sigs, err := b.RetrieveSignatures(ctx, tt.object, opts)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(mockSignature, sigs[signatureAnnotation][0]); diff != "" {
				t.Errorf("unexpected signature: (-want, +got): %s", diff)
			}

		})
	}
}

// Just a simple struct to serialize
type mockPayload struct {
	A string
	B int
}

// TestStorerAnnotationPreservation tests that the Storer preserves existing Chains annotations
func TestStorerAnnotationPreservation(t *testing.T) {
	tests := []struct {
		name                string
		existingAnnotations map[string]string
		expectedPreserved   []string
		expectedIgnored     []string
	}{
		{
			name: "preserve existing chains annotations",
			existingAnnotations: map[string]string{
				"chains.tekton.dev/existing1": "value1",
				"chains.tekton.dev/existing2": "value2",
				"tekton.dev/other":            "ignore",
				"kubernetes.io/annotation":    "ignore",
				"chains.tekton.dev/preserve":  "keep-me",
			},
			expectedPreserved: []string{
				"chains.tekton.dev/existing1",
				"chains.tekton.dev/existing2",
				"chains.tekton.dev/preserve",
			},
			expectedIgnored: []string{
				"tekton.dev/other",
				"kubernetes.io/annotation",
			},
		},
		{
			name: "no existing chains annotations",
			existingAnnotations: map[string]string{
				"tekton.dev/other":         "ignore",
				"kubernetes.io/annotation": "ignore",
			},
			expectedPreserved: []string{},
			expectedIgnored: []string{
				"tekton.dev/other",
				"kubernetes.io/annotation",
			},
		},
		{
			name: "only chains annotations",
			existingAnnotations: map[string]string{
				"chains.tekton.dev/existing1": "value1",
				"chains.tekton.dev/existing2": "value2",
			},
			expectedPreserved: []string{
				"chains.tekton.dev/existing1",
				"chains.tekton.dev/existing2",
			},
			expectedIgnored: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)

			obj := objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-taskrun",
					Namespace:   "default",
					Annotations: tt.existingAnnotations,
				},
			})

			tekton.CreateObject(t, ctx, c, obj)

			storer := &Storer{
				client: c,
				key:    "test-key",
			}

			mockPayload := []byte(`{"test": "payload"}`)
			mockSignature := []byte("mock-signature")
			mockCert := []byte("mock-cert")
			mockChain := []byte("mock-chain")

			req := &api.StoreRequest[objects.TektonObject, *intoto.Statement]{
				Object:   obj,
				Artifact: obj,
				Payload:  nil,
				Bundle: &signing.Bundle{
					Content:   mockPayload,
					Signature: mockSignature,
					Cert:      mockCert,
					Chain:     mockChain,
				},
			}

			_, err := storer.Store(ctx, req)
			if err != nil {
				t.Fatalf("Store() error = %v", err)
			}

			// Verify preserved annotations still exist
			updated, err := tekton.GetObject(t, ctx, c, obj)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			annotations := updated.GetAnnotations()

			// Check preserved annotations
			for _, key := range tt.expectedPreserved {
				if expectedValue := tt.existingAnnotations[key]; annotations[key] != expectedValue {
					t.Errorf("Expected preserved annotation %s=%s, got %s", key, expectedValue, annotations[key])
				}
			}

			// Check ignored annotations (should still be there, just not in patch)
			for _, key := range tt.expectedIgnored {
				if expectedValue := tt.existingAnnotations[key]; annotations[key] != expectedValue {
					t.Errorf("Expected ignored annotation %s=%s to remain, got %s", key, expectedValue, annotations[key])
				}
			}

			// Check that new storage annotations were added
			expectedStorageAnnotations := []string{
				"chains.tekton.dev/payload-test-key",
				"chains.tekton.dev/signature-test-key",
				"chains.tekton.dev/cert-test-key",
				"chains.tekton.dev/chain-test-key",
			}

			for _, key := range expectedStorageAnnotations {
				if _, exists := annotations[key]; !exists {
					t.Errorf("Expected storage annotation %s to be added", key)
				}
			}
		})
	}
}
