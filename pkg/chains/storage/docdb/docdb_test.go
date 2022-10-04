/*
Copyright 2021 The Tekton Authors
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

package docdb

import (
	"encoding/json"
	"testing"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"gocloud.dev/docstore"
	_ "gocloud.dev/docstore/memdocstore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestBackend_StorePayload(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	type args struct {
		tr         *v1beta1.TaskRun
		rawPayload interface{}
		signature  string
		key        string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "no error",
			args: args{
				tr: &v1beta1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "bar",
						UID:       types.UID("uid"),
					},
				},
				rawPayload: &v1beta1.TaskRun{ObjectMeta: metav1.ObjectMeta{UID: "foo"}},
				signature:  "signature",
				key:        "foo",
			},
		},
	}

	memURL := "mem://chains/name"
	coll, err := docstore.OpenCollection(ctx, memURL)
	if err != nil {
		t.Fatal(err)
	}
	defer coll.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare the document.
			b := &Backend{
				logger: logtesting.TestLogger(t),
				coll:   coll,
			}
			sb, err := json.Marshal(tt.args.rawPayload)
			if err != nil {
				t.Fatal(err)
			}

			// Store the document.
			opts := config.StorageOpts{ShortKey: tt.args.key}
			trObj := objects.NewTaskRunObject(tt.args.tr)
			if err := b.StorePayload(ctx, trObj, sb, tt.args.signature, opts); (err != nil) != tt.wantErr {
				t.Fatalf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
			obj := SignedDocument{
				Name: tt.args.key,
			}
			if err := coll.Get(ctx, &obj); err != nil {
				t.Fatal(err)
			}

			// Check the signature.
			signatures, err := b.RetrieveSignatures(ctx, trObj, opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(signatures[obj.Name]) != 1 {
				t.Fatalf("unexpected number of signatures: %d", len(signatures[obj.Name]))
			}

			if signatures[obj.Name][0] != tt.args.signature {
				t.Errorf("wrong signature, expected %s, got %s", tt.args.signature, signatures[obj.Name][0])
			}

			// Check the payload.
			payloads, err := b.RetrievePayloads(ctx, trObj, opts)
			if err != nil {
				t.Fatal(err)
			}
			if payloads[obj.Name] != string(sb) {
				t.Errorf("wrong payload, expected %s, got %s", tt.args.rawPayload, payloads[obj.Name])
			}
		})
	}
}
