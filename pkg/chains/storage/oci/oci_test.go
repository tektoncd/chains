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

package oci

import (
	"encoding/json"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestBackend_StorePayload(t *testing.T) {

	// pretty much anything that has no Subject
	sampleIntotoStatementBytes, _ := json.Marshal(in_toto.Statement{})
	logger := logtesting.TestLogger(t)

	type fields struct {
		logger *zap.SugaredLogger
		tr     *v1beta1.TaskRun
		cfg    config.Config
		kc     authn.Keychain
		auth   remote.Option
	}
	type args struct {
		rawPayload  []byte
		signature   string
		storageOpts config.StorageOpts
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "no subject",
			fields: fields{
				tr:     &v1beta1.TaskRun{ObjectMeta: v1.ObjectMeta{Name: "foo", Namespace: "bar"}},
				logger: logger,
			},
			args: args{
				rawPayload: sampleIntotoStatementBytes,
				signature:  "",
				storageOpts: config.StorageOpts{
					PayloadFormat: "tekton-provenance",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Backend{
				logger: tt.fields.logger,
				tr:     tt.fields.tr,
				cfg:    tt.fields.cfg,
				kc:     tt.fields.kc,
				auth:   tt.fields.auth,
			}
			if err := b.StorePayload(tt.args.rawPayload, tt.args.signature, tt.args.storageOpts); (err != nil) != tt.wantErr {
				t.Errorf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
