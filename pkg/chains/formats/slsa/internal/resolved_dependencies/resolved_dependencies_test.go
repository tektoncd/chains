/*
Copyright 2023 The Tekton Authors

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

package resolveddependencies

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	v1slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
)

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name string
		rds  []v1slsa.ResourceDescriptor
		want []v1slsa.ResourceDescriptor
	}{{
		name: "no duplicate resolvedDependencies",
		rds: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
		want: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "a1234f6e7a69617db57b685893256f978436277094c21d43b153994acd8a09567",
				},
			},
		},
	}, {
		name: "same uri and digest",
		rds: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "same uri but different digest",
		rds: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
		want: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			},
		},
	}, {
		name: "same uri but different digest, swap order",
		rds: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "task config must be present",
		rds: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "task",
				URI:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "task",
				URI:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}, {
		name: "pipeline config must be present",
		rds: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "pipeline",
				URI:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
		want: []v1slsa.ResourceDescriptor{
			{
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01248",
				},
			}, {
				URI: "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			}, {
				Name: "pipeline",
				URI:  "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247",
				},
			},
		},
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rds, err := RemoveDuplicateResolvedDependencies(tc.rds)
			if err != nil {
				t.Fatalf("Did not expect an error but got %v", err)
			}
			if diff := cmp.Diff(tc.want, rds); diff != "" {
				t.Errorf("resolvedDependencies(): -want +got: %s", diff)
			}
		})
	}
}
