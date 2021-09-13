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
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"gocloud.dev/docstore"
	_ "gocloud.dev/docstore/memdocstore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestBackend_StorePayload(t *testing.T) {

	type args struct {
		tr        *v1beta1.TaskRun
		signed    interface{}
		signature string
		key       string
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
				signed:    &v1beta1.TaskRun{ObjectMeta: metav1.ObjectMeta{UID: "foo"}},
				signature: "signature",
				key:       "foo",
			},
		},
	}

	memUrl := "mem://chains/name"
	coll, err := docstore.OpenCollection(context.Background(), memUrl)
	if err != nil {
		t.Fatal(err)
	}
	defer coll.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			b := &Backend{
				logger: logtesting.TestLogger(t),
				tr:     tt.args.tr,
				coll:   coll,
			}
			sb, err := json.Marshal(tt.args.signed)
			if err != nil {
				t.Fatal(err)
			}
			if err := b.StorePayload(sb, tt.args.signature, config.StorageOpts{Key: tt.args.key}); (err != nil) != tt.wantErr {
				t.Fatalf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}

			obj := SignedDocument{
				Name: tt.args.key,
			}
			if err := coll.Get(ctx, &obj); err != nil {
				t.Fatal(err)
			}
			sig, err := base64.StdEncoding.DecodeString(obj.Signature)
			if err != nil {
				t.Fatal(err)
			}
			if string(sig) != tt.args.signature {
				t.Errorf("wrong signature, expected %s, got %s", tt.args.signature, string(sig))
			}
			if string(obj.Signed) != string(sb) {
				t.Errorf("wrong signature, expected %s, got %s", tt.args.signed, string(obj.Signed))
			}
		})
	}
}
