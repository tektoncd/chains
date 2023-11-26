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
	"context"
	"io"
	"testing"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/objects"

	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	rtesting "knative.dev/pkg/reconciler/testing"
)

//nolint:staticcheck
func TestBackend_StorePayload(t *testing.T) {
	type args struct {
		tr        *v1.TaskRun
		pr        *v1.PipelineRun
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
				tr: &v1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "bar",
						UID:       types.UID("uid"),
					},
				},
				pr: &v1.PipelineRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "bar",
						UID:       types.UID("uid"),
					},
				},
				signed:    []byte("signed"),
				signature: "signature",
				opts:      config.StorageOpts{ShortKey: "foo.uuid", PayloadFormat: formats.PayloadTypeSlsav1},
			},
		},
		{
			name: "no error, tekton",
			args: args{
				tr: &v1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "bar",
						UID:       types.UID("uid"),
					},
				},
				pr: &v1.PipelineRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "foo",
						Name:      "bar",
						UID:       types.UID("uid"),
					},
				},
				signed:    []byte("signed"),
				signature: "signature",
				opts:      config.StorageOpts{ShortKey: "foo.uuid", PayloadFormat: formats.PayloadTypeTekton},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			mockGcsWrite := &mockGcsWriter{objects: map[string]*bytes.Buffer{}}
			mockGcsRead := &mockGcsReader{objects: mockGcsWrite.objects}
			b := &Backend{
				writer: mockGcsWrite,
				reader: mockGcsRead,
				cfg:    config.Config{Storage: config.StorageConfigs{GCS: config.GCSStorageConfig{Bucket: "foo"}}},
			}
			trObj := objects.NewTaskRunObjectV1(tt.args.tr)
			if err := b.StorePayload(ctx, trObj, tt.args.signed, tt.args.signature, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}

			objectSig := taskRunSigName(tt.args.tr, tt.args.opts)
			objectPayload := taskRunPayloadName(tt.args.tr, tt.args.opts)
			got, err := b.RetrieveSignatures(ctx, trObj, tt.args.opts)
			if err != nil {
				t.Fatal(err)
			}
			if got[objectSig][0] != tt.args.signature {
				t.Errorf("wrong signature, expected %q, got %q", tt.args.signature, got[objectSig][0])
			}
			var gotPayload map[string]string
			gotPayload, err = b.RetrievePayloads(ctx, trObj, tt.args.opts)
			if err != nil {
				t.Fatal(err)
			}
			if gotPayload[objectPayload] != string(tt.args.signed) {
				t.Errorf("wrong signature, expected %s, got %s", tt.args.signed, gotPayload[objectPayload])
			}

			prObj := objects.NewPipelineRunObjectV1(tt.args.pr)
			if err := b.StorePayload(ctx, prObj, tt.args.signed, tt.args.signature, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}

			objectSig = pipelineRunSigname(tt.args.pr, tt.args.opts)
			objectPayload = pipelineRunPayloadName(tt.args.pr, tt.args.opts)
			got, err = b.RetrieveSignatures(ctx, prObj, tt.args.opts)
			if err != nil {
				t.Fatal(err)
			}

			if got[objectSig][0] != tt.args.signature {
				t.Errorf("wrong signature, expected %q, got %q", tt.args.signature, got[objectSig][0])
			}

			gotPayload, err = b.RetrievePayloads(ctx, prObj, tt.args.opts)
			if err != nil {
				t.Fatal(err)
			}
			if gotPayload[objectPayload] != string(tt.args.signed) {
				t.Errorf("wrong signature, expected %s, got %s", tt.args.signed, gotPayload[objectPayload])
			}
		})
	}
}

type mockGcsWriter struct {
	objects map[string]*bytes.Buffer
}

func (m *mockGcsWriter) GetWriter(ctx context.Context, object string) io.WriteCloser {
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

func (m *mockGcsReader) GetReader(ctx context.Context, object string) (io.ReadCloser, error) {
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
