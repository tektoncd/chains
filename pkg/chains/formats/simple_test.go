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

package formats

import (
	"reflect"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

func makeDigest(t *testing.T, dgst string) name.Digest {
	digest, err := name.NewDigest(dgst)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func TestSimpleSigning_CreatePayload(t *testing.T) {

	tests := []struct {
		name    string
		obj     interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name: "digest",
			obj:  makeDigest(t, "gcr.io/foo/bar@sha256:20ab676d319c93ef5b4bef9290ed913ed8feaa0c92c43a7cddc28a3697918b92"),
			want: simple{
				Critical: critical{
					Identity: map[string]string{
						"docker-reference": "gcr.io/foo/bar",
					},
					Image: map[string]string{
						"Docker-manifest-digest": "sha256:20ab676d319c93ef5b4bef9290ed913ed8feaa0c92c43a7cddc28a3697918b92",
					},
					Type: "Tekton container signature",
				},
				Optional: map[string]interface{}{},
			},
		},
		{
			name:    "not digest",
			obj:     struct{}{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &SimpleSigning{}
			got, err := i.CreatePayload(nil, tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("SimpleSigning.CreatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SimpleSigning.CreatePayload() = %v, want %v", got, tt.want)
			}
		})
	}
}
