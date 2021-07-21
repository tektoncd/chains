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

package gcs

import (
	"bytes"
	"io"
	"testing"

	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestBackend_StorePayload(t *testing.T) {

	type args struct {
		tr        *v1beta1.TaskRun
		signed    []byte
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
				signed:    []byte("signed"),
				signature: "signature",
				key:       "foo",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockGcs := &mockGcsWriter{objects: map[string]*bytes.Buffer{}}
			b := &Backend{
				logger: logtesting.TestLogger(t),
				tr:     tt.args.tr,
				writer: mockGcs,
				cfg:    config.Config{Storage: config.StorageConfigs{GCS: config.GCSStorageConfig{Bucket: "foo"}}},
			}
			if err := b.StorePayload(tt.args.signed, tt.args.signature, config.StorageOpts{Key: tt.args.key}); (err != nil) != tt.wantErr {
				t.Errorf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
			got := mockGcs.objects["taskrun-foo-bar-uid/foo.signature"].String()
			if got != tt.args.signature {
				t.Errorf("wrong signature, expected %s, got %s", tt.args.signature, got)
			}
			got = mockGcs.objects["taskrun-foo-bar-uid/foo.payload"].String()
			if got != string(tt.args.signed) {
				t.Errorf("wrong signature, expected %s, got %s", tt.args.signed, got)
			}
		})
	}
}

type mockGcsWriter struct {
	objects map[string]*bytes.Buffer
}

func (m *mockGcsWriter) GetWriter(object string) io.WriteCloser {
	buf := bytes.NewBuffer([]byte{})
	m.objects[object] = buf
	return &writeCloser{buf}
}

type writeCloser struct {
	*bytes.Buffer
}

func (wc *writeCloser) Close() error {
	// Noop
	return nil
}
