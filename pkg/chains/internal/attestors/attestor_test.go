package attestors

import (
	"crypto"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	v1 "github.com/tektoncd/chains/pkg/chains/formats/slsa/v1"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/signing/x509"
	"github.com/tektoncd/chains/pkg/chains/storage/oci"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	logtest "knative.dev/pkg/logging/testing"
)

func TestOCIAttestor(t *testing.T) {
	digest := setupRegistry(t)

	// Create local signer using randomly generated key.
	sv := newSignerVerifier(t)
	signer := &x509.Signer{SignerVerifier: sv}

	storer, err := oci.NewSimpleStorer()
	if err != nil {
		t.Fatal(err)
	}

	att := &Attestor[name.Digest, simple.SimpleContainerImage]{
		payloader: simple.SimpleSigningPayloader{},
		signer:    signer,
		storer:    storer,
	}
	ctx := logtest.TestContextWithLogger(t)
	if _, err := att.Attest(ctx, nil, digest); err != nil {
		t.Error(err)
	}

	// Verify signature to make sure it was pushed properly.
	if _, _, err := cosign.VerifyImageSignatures(ctx, digest, &cosign.CheckOpts{
		SigVerifier: sv,
		IgnoreTlog:  true,
	}); err != nil {
		t.Error(err)
	}
}

func setupRegistry(t *testing.T) name.Digest {
	t.Helper()

	reg := httptest.NewServer(registry.New())
	t.Cleanup(reg.Close)

	// Push an image to the local registry.
	ref, err := name.ParseReference(fmt.Sprintf("%s/foo", strings.TrimPrefix(reg.URL, "http://")))
	if err != nil {
		t.Fatal(err)
	}
	if err := remote.Put(ref, empty.Image); err != nil {
		t.Fatal(err)
	}
	h, err := empty.Image.Digest()
	if err != nil {
		t.Fatal(err)
	}
	return ref.Context().Digest(h.String())
}

func newSignerVerifier(t *testing.T) signature.SignerVerifier {
	t.Helper()

	priv, err := cosign.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("error generating keypair: %v", err)
	}
	sv, err := signature.LoadECDSASignerVerifier(priv, crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	return sv
}

func TestSLSAAttestor(t *testing.T) {
	digest := setupRegistry(t)

	// Create local signer using randomly generated key.
	sv := newSignerVerifier(t)
	signer := &x509.Signer{SignerVerifier: sv}
	wrapped, err := signing.Wrap(signer)
	if err != nil {
		t.Fatal(err)
	}

	storer, err := oci.NewAttestationStorer[*v1.ProvenanceStatement]()
	if err != nil {
		t.Fatal(err)
	}

	tr := &v1beta1.TaskRun{
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				TaskRunResults: []v1beta1.TaskRunResult{{
					Name:  "IMAGES",
					Value: *v1beta1.NewArrayOrString(digest.String()),
				}},
			},
		},
	}
	obj := objects.NewTaskRunObject(tr)

	att := &Attestor[objects.TektonObject, *v1.ProvenanceStatement]{
		payloader: v1.NewFormatter(),
		signer:    wrapped,
		storer:    storer,
	}
	ctx := logtest.TestContextWithLogger(t)
	if _, err := att.Attest(ctx, obj, obj); err != nil {
		t.Error(err)
	}

	// Verify attestation to make sure it was stored properly.
	if _, _, err := cosign.VerifyImageAttestations(ctx, digest, &cosign.CheckOpts{
		SigVerifier: sv,
		IgnoreTlog:  true,
	}); err != nil {
		t.Error(err)
	}
}
