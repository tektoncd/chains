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
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestBackend_StorePayload(t *testing.T) {
	// The Tekton storage backend JSON serializes, base64 encodes, and attaches the result as an annotation

	tests := []struct {
		name    string
		payload interface{}
		wantErr bool
	}{
		{
			name: "simple",
			payload: mockPayload{
				A: "foo",
				B: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			c := fakepipelineclient.Get(ctx)
			tr := &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Status: v1beta1.TaskRunStatus{
					TaskRunStatusFields: v1beta1.TaskRunStatusFields{
						TaskRunResults: []v1beta1.TaskRunResult{
							{Name: "IMAGE_URL", Value: "mockImage"},
						},
					},
				},
			}
			if _, err := c.TektonV1beta1().TaskRuns(tr.Namespace).Create(ctx, tr, metav1.CreateOptions{}); err != nil {
				t.Errorf("error setting up fake taskrun: %v", err)
			}

			b := &Backend{
				pipelienclientset: c,
				logger:            logtesting.TestLogger(t),
				tr:                tr,
			}
			payload, err := json.Marshal(tt.payload)
			if err != nil {
				t.Errorf("error marshaling json: %v", err)
			}
			opts := config.StorageOpts{Key: "mockpayload"}
			mockSignature := "mocksignature"
			if err := b.StorePayload(payload, mockSignature, opts); (err != nil) != tt.wantErr {
				t.Errorf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}

			// The rest is invalid if we wanted an error.
			if tt.wantErr {
				return
			}

			payloadAnnotation := b.PayloadName(opts)
			payloads, err := b.RetrievePayloads(opts)
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
			signatureAnnotation := b.SigName(opts)
			sigs, err := b.RetrieveSignatures(opts)
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
