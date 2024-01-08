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

package externalparameters

import (
	"strings"
	"testing"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func TestBuildConfigSource(t *testing.T) {
	digest := map[string]string{"alg1": "hex1", "alg2": "hex2"}
	provenance := &v1.Provenance{
		RefSource: &v1.RefSource{
			Digest:     digest,
			URI:        "https://tekton.com",
			EntryPoint: "/path/to/entry",
		},
	}

	want := map[string]string{
		"repository": "https://tekton.com",
		"path":       "/path/to/entry",
	}

	got := BuildConfigSource(provenance)

	gotRef := strings.Split(got["ref"], ":")
	if len(gotRef) != 2 {
		t.Errorf("buildConfigSource() does not return the proper ref: want one of: %s got: %s", digest, got["ref"])
	}
	refValue, ok := digest[gotRef[0]]
	if !ok {
		t.Errorf("buildConfigSource() does not contain correct ref: want one of: %s got: %s:%s", digest, gotRef[0], gotRef[1])
	}

	if refValue != gotRef[1] {
		t.Errorf("buildConfigSource() does not contain correct ref: want one of: %s got: %s:%s", digest, gotRef[0], gotRef[1])
	}

	if got["repository"] != want["repository"] {
		t.Errorf("buildConfigSource() does not contain correct repository: want: %s got: %s", want["repository"], want["repository"])
	}

	if got["path"] != want["path"] {
		t.Errorf("buildConfigSource() does not contain correct path: want: %s got: %s", want["path"], got["path"])
	}
}
