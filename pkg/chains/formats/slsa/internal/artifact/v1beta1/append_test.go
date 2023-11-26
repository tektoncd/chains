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
package artifactv1beta1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
)

func TestAppendSubjects(t *testing.T) {
	tests := []struct {
		name     string
		original []intoto.Subject
		toAdd    []intoto.Subject
		want     []intoto.Subject
	}{{
		name: "add a completely new subject",
		original: []intoto.Subject{
			{
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			}, {
				Name: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "b",
				},
			},
		},
		toAdd: []intoto.Subject{
			{
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "c",
				},
			},
		},
		want: []intoto.Subject{
			{
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			}, {
				Name: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "b",
				},
			}, {
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "c",
				},
			},
		},
	}, {
		name: "add a subject with same uri and digest",
		original: []intoto.Subject{
			{
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			},
		},
		toAdd: []intoto.Subject{
			{
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			},
		},
		want: []intoto.Subject{
			{
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			},
		},
	}, {
		name: "add a subject with same uri but different digest",
		original: []intoto.Subject{
			{
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			},
		},
		toAdd: []intoto.Subject{
			{
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b",
				},
			},
		},
		want: []intoto.Subject{
			{
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			}, {
				Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b",
				},
			},
		},
	},
		{
			name: "add a subject with same uri, one common digest and one different digest",
			original: []intoto.Subject{
				{
					Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
					Digest: common.DigestSet{
						"sha256": "a",
						"sha224": "b",
					},
				},
			},
			toAdd: []intoto.Subject{
				{
					Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
					Digest: common.DigestSet{
						"sha256": "a",
						"sha512": "c",
					},
				},
			},
			want: []intoto.Subject{
				{
					Name: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
					Digest: common.DigestSet{
						"sha256": "a",
						"sha224": "b",
						"sha512": "c",
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AppendSubjects(tc.original, tc.toAdd...)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("materials(): -want +got: %s", diff)
			}
		})
	}
}

func TestAppendMaterials(t *testing.T) {
	tests := []struct {
		name     string
		original []common.ProvenanceMaterial
		toAdd    []common.ProvenanceMaterial
		want     []common.ProvenanceMaterial
	}{{
		name: "add a completely new material",
		original: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			}, {
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "b",
				},
			},
		},
		toAdd: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "c",
				},
			},
		},
		want: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			}, {
				URI: "gcr.io/cloud-marketplace-containers/google/bazel",
				Digest: common.DigestSet{
					"sha256": "b",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/sidecar-git-init",
				Digest: common.DigestSet{
					"sha256": "c",
				},
			},
		},
	}, {
		name: "add a material with same uri and digest",
		original: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			},
		},
		toAdd: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			},
		},
		want: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			},
		},
	}, {
		name: "add a material with same uri but different digest",
		original: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			},
		},
		toAdd: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b",
				},
			},
		},
		want: []common.ProvenanceMaterial{
			{
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "a",
				},
			}, {
				URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
				Digest: common.DigestSet{
					"sha256": "b",
				},
			},
		},
	},
		{
			name: "add a material with same uri, one common digest and one different digest",
			original: []common.ProvenanceMaterial{
				{
					URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
					Digest: common.DigestSet{
						"sha256": "a",
						"sha224": "b",
					},
				},
			},
			toAdd: []common.ProvenanceMaterial{
				{
					URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
					Digest: common.DigestSet{
						"sha256": "a",
						"sha512": "c",
					},
				},
			},
			want: []common.ProvenanceMaterial{
				{
					URI: "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
					Digest: common.DigestSet{
						"sha256": "a",
						"sha224": "b",
						"sha512": "c",
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AppendMaterials(tc.original, tc.toAdd...)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("materials(): -want +got: %s", diff)
			}
		})
	}
}
