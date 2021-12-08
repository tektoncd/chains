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
	"github.com/tektoncd/chains/pkg/chains/formats"
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
		opts      config.StorageOpts
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "no error, intoto",
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
				opts:      config.StorageOpts{Key: "foo.uuid", PayloadFormat: formats.PayloadTypeInTotoIte6},
			},
		},
		{
			name: "no error, tekton",
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
				opts:      config.StorageOpts{Key: "foo.uuid", PayloadFormat: formats.PayloadTypeTekton},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockGcsWrite := &mockGcsWriter{objects: map[string]*bytes.Buffer{}}
			mockGcsRead := &mockGcsReader{objects: mockGcsWrite.objects}
			b := &Backend{
				logger: logtesting.TestLogger(t),
				tr:     tt.args.tr,
				writer: mockGcsWrite,
				reader: mockGcsRead,
				cfg:    config.Config{Storage: config.StorageConfigs{GCS: config.GCSStorageConfig{Bucket: "foo"}}},
			}
			if err := b.StorePayload(tt.args.signed, tt.args.signature, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
			got, err := b.RetrieveSignature(tt.args.opts)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.args.signature {
				t.Errorf("wrong signature, expected %q, got %q", tt.args.signature, got)
			}
			got, err = b.RetrievePayload(tt.args.opts)
			if err != nil {
				t.Fatal(err)
			}
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

type mockGcsReader struct {
	objects map[string]*bytes.Buffer
}

func (m *mockGcsReader) GetReader(object string) (io.ReadCloser, error) {
	buf := m.objects[object]
	return &ReaderCloser{buf}, nil
}

type ReaderCloser struct {
	*bytes.Buffer
}

func (rc *ReaderCloser) Close() error {
	// Noop
	return nil
}
