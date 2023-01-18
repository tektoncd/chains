/*
Copyright 2022 The Tekton Authors

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

package material

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/test"
)

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name string
		mats []slsa.ProvenanceMaterial
		want []slsa.ProvenanceMaterial
	}{{
		name: "no duplicate materials",
		mats: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: slsa.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: slsa.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
		want: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: slsa.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: slsa.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
	}, {
		name: "same uri and digest",
		mats: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "same uri but different digest",
		mats: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
		want: []slsa.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: slsa.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
	}}
	for _, tc := range tests {
		mat, err := RemoveDuplicateMaterials(tc.mats)
		if err != nil {
			t.Fatalf("Did not expect an error but got %v", err)
		}
		if diff := cmp.Diff(tc.want, mat, test.OptSortMaterial); diff != "" {
			t.Errorf("materials(): -want +got: %s", diff)
		}
	}
}
