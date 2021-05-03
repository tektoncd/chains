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

package pgp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/openpgp"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestSigner_Sign(t *testing.T) {
	ctx := context.Background()
	logger := logtesting.TestLogger(t)
	// Parse out the public key before we get started.
	fp, err := os.Open("./testdata/pgp.public-key")
	if err != nil {
		t.Errorf("error reading public key: %v", err)
	}
	defer fp.Close()
	publicKey, err := openpgp.ReadArmoredKeyRing(fp)
	if err != nil {
		t.Errorf("error reading public key: %v", err)
	}

	// Create a signer with the passphrase/private key in the testdata directory
	signer, err := NewSigner("./testdata", logger)
	if err != nil {
		t.Errorf("error creating signer: %v", err)
	}

	// Sign a random payload
	payload := struct {
		A int
		B string
	}{
		A: 3,
		B: "test",
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("error marshaling payload: %v", err)
	}
	signature, _, err := signer.Sign(ctx, rawPayload)
	if err != nil {
		t.Errorf("Signer.Sign() error = %v", err)
		return
	}

	// Check that it is a valid signature
	if _, err := openpgp.CheckArmoredDetachedSignature(publicKey, bytes.NewReader(rawPayload), strings.NewReader(string(signature))); err != nil {
		t.Errorf("invalid signature: %v", err)
	}
}
