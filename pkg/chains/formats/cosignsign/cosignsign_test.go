/*
Copyright 2026 The Tekton Authors
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

package cosignsign

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"
	"google.golang.org/protobuf/encoding/protojson"
)

const testHex = "0a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f9"

func TestCreatePayload(t *testing.T) {
	ref, err := name.NewDigest("registry.example.com/team/app@sha256:" + testHex)
	if err != nil {
		t.Fatal(err)
	}

	p, err := NewFormatter(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.CreatePayload(context.Background(), ref)
	if err != nil {
		t.Fatalf("CreatePayload() error: %v", err)
	}
	stmt, ok := got.(*intoto.Statement)
	if !ok {
		t.Fatalf("CreatePayload() returned %T, want *intoto.Statement", got)
	}

	if stmt.Type != "https://in-toto.io/Statement/v1" {
		t.Errorf("_type = %q, want https://in-toto.io/Statement/v1", stmt.Type)
	}
	if len(stmt.Subject) != 1 {
		t.Fatalf("got %d subjects, want 1", len(stmt.Subject))
	}
	if stmt.Subject[0].Name != "registry.example.com/team/app" {
		t.Errorf("subject name = %q, want registry.example.com/team/app", stmt.Subject[0].Name)
	}
	if len(stmt.Subject[0].Digest) != 1 || stmt.Subject[0].Digest["sha256"] != testHex {
		t.Errorf("subject digest = %v, want sha256:%s", stmt.Subject[0].Digest, testHex)
	}
	if stmt.PredicateType != "https://sigstore.dev/cosign/sign/v1" {
		t.Errorf("predicateType = %q, want https://sigstore.dev/cosign/sign/v1", stmt.PredicateType)
	}
	if stmt.Predicate == nil || len(stmt.Predicate.Fields) != 0 {
		t.Errorf("predicate = %v, want empty object", stmt.Predicate)
	}

	raw, err := protojson.Marshal(stmt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"predicate":{}`) && !strings.Contains(string(raw), `"predicate": {}`) {
		t.Errorf("marshalled statement must contain an empty predicate object, got %s", raw)
	}
}

func TestCreatePayload_Errors(t *testing.T) {
	p := &CosignSign{}
	if _, err := p.CreatePayload(context.Background(), "not-a-digest"); err == nil {
		t.Error("expected error for unsupported input type")
	}

	ref, err := name.NewDigest("registry.example.com/team/app@sha512:" + strings.Repeat("a", 128))
	if err != nil {
		t.Skipf("sha512 digests not supported by name package: %v", err)
	}
	if _, err := p.CreatePayload(context.Background(), ref); err == nil {
		t.Error("expected error for non-sha256 digest")
	}
}

func TestPayloaderMetadata(t *testing.T) {
	p, err := formats.GetPayloader(formats.PayloadTypeCosignSignV1, config.Config{})
	if err != nil {
		t.Fatalf("GetPayloader() error: %v", err)
	}
	if !p.Wrap() {
		t.Error("Wrap() = false, want true")
	}
	if p.Type() != "cosign-sign/v1" {
		t.Errorf("Type() = %q, want cosign-sign/v1", p.Type())
	}
}
