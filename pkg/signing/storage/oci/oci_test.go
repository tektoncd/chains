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
	"archive/tar"
	"bytes"
	"io"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestBackend_StorePayload(t *testing.T) {
	type args struct {
		signed    []byte
		signature string
		key       string
		tr        *v1beta1.TaskRun
	}
	tests := []struct {
		name      string
		args      args
		imageName string
	}{
		{
			name:      "one image",
			imageName: "gcr.io/foo/taskrun-ns-name-123/key",
			args: args{
				signed:    []byte("foo"),
				signature: "bar",
				key:       "key",
				tr: &v1beta1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns",
						Name:      "name",
						UID:       types.UID("123"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockRepo{}
			defer setupMocks(m)()

			cfg := config.Config{Storage: config.Storage{OCI: config.OCI{Repository: "gcr.io/foo"}}}
			b := &Backend{
				logger: logtesting.TestLogger(t),
				tr:     tt.args.tr,
				cfg:    cfg,
			}
			if err := b.StorePayload(tt.args.signed, tt.args.signature, tt.args.key); err != nil {
				t.Errorf("Backend.StorePayload() error = %v", err)
			}

			img, ok := m.images[tt.imageName]
			if !ok {
				t.Fatalf("image %s not pushed, got %v", tt.imageName, m.images)
			}

			// There should only be one layer
			mfst, err := img.Manifest()
			if err != nil {
				t.Fatal(err)
			}
			if len(mfst.Layers) != 1 {
				t.Fatalf("Expected only one layer. Got %d", len(mfst.Layers))
			}
			// It should have the signed payload in it.
			layer, err := img.LayerByDigest(mfst.Layers[0].Digest)
			if err != nil {
				t.Fatal(err)
			}
			rc, err := layer.Uncompressed()
			if err != nil {
				t.Fatal(err)
			}
			defer rc.Close()
			tr := tar.NewReader(rc)
			// There should be one file, with the name "signed" containing the payload.
			hdr, err := tr.Next()
			if err != nil {
				t.Fatal(err)
			}
			if hdr.Name != "signed" {
				t.Errorf("tar name was %s, expected 'signed'", hdr.Name)
			}
			buf := bytes.Buffer{}
			if _, err := io.Copy(&buf, tr); err != nil {
				t.Error(err)
			}

			if buf.String() != string(tt.args.signed) {
				t.Errorf("tar contents were %s, expected %s", buf.String(), string(tt.args.signed))
			}

			// The layer annotation should have the signature
			if mfst.Layers[0].Annotations["signature"] != tt.args.signature {
				t.Errorf("signature was %s, expected %s", mfst.Layers[0].Annotations["signature"], tt.args.signature)
			}

			// There should be no other files in the layer.
			if _, err = tr.Next(); err != io.EOF {
				t.Error("Expected EOF reading tar. didn't get one.")
			}
		})
	}
}

func setupMocks(m *mockRepo) func() {
	oldWrite := writeImage
	writeImage = m.writeImage
	return func() {
		writeImage = oldWrite
	}
}

type mockRepo struct {
	images map[string]v1.Image
}

func (m *mockRepo) writeImage(ref name.Reference, img v1.Image, options ...remote.Option) error {
	if m.images == nil {
		m.images = map[string]v1.Image{}
	}
	m.images[ref.String()] = img
	return nil
}
