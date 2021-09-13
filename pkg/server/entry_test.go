/*
Copyright 2021 The Tekton Authors
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

package server

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/chains/proto"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestAddGetEntry(t *testing.T) {
	storagePath = t.TempDir()
	t.Cleanup(func() { os.Remove(storagePath) })

	e := &proto.Entry{
		Uid:  "my-uid",
		Svid: "svid",
		Signatures: map[string]string{
			"sig1": "yay",
		},
	}
	addEntry(e, logtesting.TestLogger(t))
	expected := `{"svid":"svid","signatures":{"sig1":"yay"},"uid":"my-uid"}`
	file := filepath.Join(storagePath, "my-uid")
	got, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(string(got), expected); d != "" {
		t.Fatalf("got: %v, expected: %v, diff: %v", got, expected, d)
	}

	gotEntry, err := getEntry("my-uid", logtesting.TestLogger(t))
	if err != nil {
		t.Fatal(err)
	}
	if d := cmp.Diff(gotEntry, e); d != "" {
		t.Fatalf("got: %v, expected: %v, diff: %v", gotEntry, e, d)
	}
}
