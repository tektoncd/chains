//go:build e2e
// +build e2e

/*
Copyright 2024 The Tekton Authors

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

package test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logtesting "knative.dev/pkg/logging/testing"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/test/tekton"
)

const (
	// cosignV3Image must contain a shell; the -dev variants do.
	cosignV3Image           = "ghcr.io/sigstore/cosign/cosign:v3.1.4-dev"
	cosignSignPredicateType = "https://sigstore.dev/cosign/sign/v1"

	storageTekton          = "tekton"
	formatInToto           = "in-toto"
	transparencyEnabledKey = "transparency.enabled"
)

// TestOCIStorageSigstoreBundle_TaskRun verifies that when encoding-format is
// set to sigstore-bundle, Chains stores OCI artifact signatures and attestations
// as OCI 1.1 referrers instead of legacy digest-derived .sig/.att tags.
func TestOCIStorageSigstoreBundle_TaskRun(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{registry: true, registryByClusterIP: true})
	t.Cleanup(cleanup)

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.oci.format":            "simplesigning",
		"artifacts.oci.storage":           "oci",
		"artifacts.oci.signer":            "x509",
		"artifacts.taskrun.format":        "slsa/v1",
		"artifacts.taskrun.signer":        "x509",
		"artifacts.taskrun.storage":       "oci",
		"storage.oci.repository.insecure": "true",
		"storage.oci.encoding-format":     "sigstore-bundle",
	})
	t.Cleanup(resetConfig)
	time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

	imageName := "chains-test-referrers-taskrun"
	image := fmt.Sprintf("%s/%s", c.internalRegistry, imageName)

	if os.Getenv("OPENSHIFT") == localhost {
		if err := assignSCC(ns); err != nil {
			t.Fatalf("error creating scc: %s", err)
		}
	}

	task := kanikoTask(t, ns, image)
	if _, err := c.PipelineClient.TektonV1().Tasks(ns).Create(ctx, task, metav1.CreateOptions{}); err != nil {
		t.Fatalf("error creating kaniko task: %s", err)
	}

	createdTro := tekton.CreateObject(t, ctx, c.PipelineClient, kanikoTaskRun(ns))

	// Wait for the image build to complete.
	if got := waitForCondition(ctx, t, c.PipelineClient, createdTro, done, 2*time.Minute); got == nil {
		t.Fatal("kaniko TaskRun never finished")
	}

	// Wait for Chains to sign the image and TaskRun.
	obj := waitForCondition(ctx, t, c.PipelineClient, createdTro, signed, 2*time.Minute)
	if obj == nil {
		t.Fatal("kaniko TaskRun was never signed by Chains")
	}

	// Verify that no legacy .sig or .att tags were written to the OCI image
	// repository. In sigstore-bundle mode, signatures and attestations must be
	// stored as referrer manifests discoverable via the OCI referrers API, not
	// as digest-derived tags (e.g. sha256-<digest>.sig / sha256-<digest>.att).
	verifyTro := verifyNoLegacyTagsTaskRun(ns, image)
	createdVerify := tekton.CreateObject(t, ctx, c.PipelineClient, verifyTro)
	if got := waitForCondition(ctx, t, c.PipelineClient, createdVerify, successful, time.Minute); got == nil {
		t.Error("no-legacy-tags check TaskRun never succeeded; unexpected .sig/.att tags may exist")
	}
}

// TestOCIStorageSigstoreBundle_PipelineRun verifies that sigstore-bundle mode works
// correctly for OCI artifact signatures produced during a PipelineRun image build.
func TestOCIStorageSigstoreBundle_PipelineRun(t *testing.T) {
	const imageName = "chains-test-referrers-pipelinerun"
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{
		registry:            true,
		registryByClusterIP: true,
		kanikoTaskImage:     imageName,
	})
	t.Cleanup(cleanup)

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.oci.format":            "simplesigning",
		"artifacts.oci.storage":           "oci",
		"artifacts.oci.signer":            "x509",
		"artifacts.pipelinerun.format":    "slsa/v1",
		"artifacts.pipelinerun.signer":    "x509",
		"artifacts.pipelinerun.storage":   "oci",
		"storage.oci.repository.insecure": "true",
		"storage.oci.encoding-format":     "sigstore-bundle",
	})
	t.Cleanup(resetConfig)
	time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

	if os.Getenv("OPENSHIFT") == localhost {
		if err := assignSCC(ns); err != nil {
			t.Fatalf("error creating scc: %s", err)
		}
	}

	createdObj := tekton.CreateObject(t, ctx, c.PipelineClient, kanikoPipelineRun(ns))

	// Wait for the pipeline (image build) to complete.
	if got := waitForCondition(ctx, t, c.PipelineClient, createdObj, done, 2*time.Minute); got == nil {
		t.Fatal("PipelineRun never finished")
	}

	// Wait for Chains to sign the built OCI image.
	obj := waitForCondition(ctx, t, c.PipelineClient, createdObj, signed, 2*time.Minute)
	if obj == nil {
		t.Fatal("PipelineRun image was never signed by Chains")
	}
	_ = obj

	// Verify no legacy .sig or .att tags on the built OCI image.
	// The image name is known from the kaniko task configuration created by setup().
	image := fmt.Sprintf("%s/%s", c.internalRegistry, imageName)
	verifyTro := verifyNoLegacyTagsTaskRun(ns, image)
	createdVerify := tekton.CreateObject(t, ctx, c.PipelineClient, verifyTro)
	if got := waitForCondition(ctx, t, c.PipelineClient, createdVerify, successful, time.Minute); got == nil {
		t.Error("no-legacy-tags check TaskRun never succeeded; unexpected .sig/.att tags may exist")
	}
}

// TestOCIStorageSigstoreBundle_CosignVerify checks that the image signature written in
// sigstore-bundle mode is accepted by cosign v3, the same way cosign's own e2e tests
// verify bundles: against a registry that go-containerregistry reaches over plain HTTP.
// `cosign verify` succeeds if any bundle verifies, so with an attestation present the
// output is also checked for a cosign/sign/v1 entry.
func TestOCIStorageSigstoreBundle_CosignVerify(t *testing.T) {
	tests := []struct {
		name            string
		taskrunStorage  string
		transparency    bool
		wantAttestation bool
	}{
		{name: "signature only, tlog disabled", taskrunStorage: storageTekton, transparency: false},
		{name: "signature only, tlog enabled", taskrunStorage: storageTekton, transparency: true},
		{name: "signature and attestation, tlog disabled", taskrunStorage: "oci", transparency: false, wantAttestation: true},
		{name: "signature and attestation, tlog enabled", taskrunStorage: "oci", transparency: true, wantAttestation: true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageName := fmt.Sprintf("chains-test-bundle-cosign-verify-%d", i)
			ctx := logtesting.TestContextWithLogger(t)
			c, ns, cleanup := setup(ctx, t, setupOpts{
				registry:            true,
				registryByClusterIP: true,
				kanikoTaskImage:     imageName,
			})
			t.Cleanup(cleanup)

			resetConfig := setConfigMap(ctx, t, c, map[string]string{
				"artifacts.oci.format":            "simplesigning",
				"artifacts.oci.storage":           "oci",
				"artifacts.oci.signer":            "x509",
				"artifacts.taskrun.format":        formatInToto,
				"artifacts.taskrun.signer":        "x509",
				"artifacts.taskrun.storage":       tt.taskrunStorage,
				"storage.oci.repository.insecure": "true",
				"storage.oci.encoding-format":     "sigstore-bundle",
				transparencyEnabledKey:            fmt.Sprintf("%t", tt.transparency),
			})
			t.Cleanup(resetConfig)
			time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

			if os.Getenv("OPENSHIFT") == localhost {
				if err := assignSCC(ns); err != nil {
					t.Fatalf("error creating scc: %s", err)
				}
			}

			createdTro := tekton.CreateObject(t, ctx, c.PipelineClient, kanikoTaskRun(ns))
			if got := waitForCondition(ctx, t, c.PipelineClient, createdTro, done, 2*time.Minute); got == nil {
				t.Fatal("kaniko TaskRun never finished")
			}
			obj := waitForCondition(ctx, t, c.PipelineClient, createdTro, signed, 2*time.Minute)
			if obj == nil {
				t.Fatal("kaniko TaskRun was never signed by Chains")
			}
			if got := obj.GetAnnotations()["chains.tekton.dev/signed"]; got != "true" {
				t.Fatalf("chains.tekton.dev/signed = %q, want true", got)
			}

			imageRef := fmt.Sprintf("%s/%s@%s", c.internalRegistry, imageName, imageDigestResult(t, obj))
			publicKey, err := cryptoutils.MarshalPublicKeyToPEM(c.secret.x509priv.Public())
			if err != nil {
				t.Fatal(err)
			}

			verifyTro := verifyCosignBundleTaskRun(ns, imageRef, string(publicKey), !tt.transparency, tt.wantAttestation)
			createdVerify := tekton.CreateObject(t, ctx, c.PipelineClient, verifyTro)
			if got := waitForCondition(ctx, t, c.PipelineClient, createdVerify, successful, 3*time.Minute); got == nil {
				t.Error("cosign verification TaskRun never succeeded")
			}
		})
	}
}

// imageDigestResult returns the IMAGE_DIGEST result of a finished kaniko TaskRun.
func imageDigestResult(t *testing.T, obj objects.TektonObject) string {
	t.Helper()
	tr, ok := obj.GetObject().(*v1.TaskRun)
	if !ok {
		t.Fatalf("object is %T, want *v1.TaskRun", obj.GetObject())
	}
	for _, r := range tr.Status.Results {
		if r.Name == "IMAGE_DIGEST" {
			return strings.TrimSpace(r.Value.StringVal)
		}
	}
	t.Fatal("TaskRun has no IMAGE_DIGEST result")
	return ""
}

// verifyCosignBundleTaskRun returns a TaskRun that runs cosign v3 against the image. It asserts
// the output lists a cosign/sign/v1 entry, because exit code 0 alone is satisfied by any
// verified bundle, including the SLSA attestation.
func verifyCosignBundleTaskRun(ns, imageRef, publicKey string, ignoreTlog, withAttestation bool) *objects.TaskRunObjectV1 {
	tlogFlag := ""
	if ignoreTlog {
		tlogFlag = "--insecure-ignore-tlog=true"
	}
	script := fmt.Sprintf(`#!/busybox/sh
set -e
export PATH="$PATH:/ko-app"

echo "%s" > cosign.pub

cosign verify --allow-insecure-registry --key cosign.pub %s %s > verify.json
cat verify.json
grep -Eq '"type": *"%s"' verify.json
`, publicKey, tlogFlag, imageRef, cosignSignPredicateType)
	if withAttestation {
		script += fmt.Sprintf("\ncosign verify-attestation --allow-insecure-registry --key cosign.pub --type slsaprovenance %s %s\n", tlogFlag, imageRef)
	}

	return objects.NewTaskRunObjectV1(&v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "verify-cosign-bundle-",
			Namespace:    ns,
		},
		Spec: v1.TaskRunSpec{
			TaskSpec: &v1.TaskSpec{
				Steps: []v1.Step{{
					Name:   "verify-bundle",
					Image:  cosignV3Image,
					Script: script,
				}},
			},
		},
	})
}

// verifyNoLegacyTagsTaskRun returns a TaskRun that fails if any legacy .sig or
// .att tags exist in the given OCI image repository. This confirms that Chains
// stored signatures and attestations as OCI referrers rather than as tags.
//
// The check is performed from inside the cluster so that the internal registry
// (accessible only from within the cluster network) is reachable.
func verifyNoLegacyTagsTaskRun(ns, image string) *objects.TaskRunObjectV1 {
	// Split "host:port/repo/name" into registry host and repository path.
	parts := strings.SplitN(image, "/", 2)
	if len(parts) != 2 {
		panic(fmt.Sprintf("verifyNoLegacyTagsTaskRun: image %q has no '/' separator", image))
	}
	registryHost := parts[0]
	imageRepo := parts[1]

	// Query the registry's v2 tags/list API and assert that no tags ending in
	// .sig or .att are present. Such tags are the hallmark of legacy cosign
	// tag-based storage; they must NOT appear in sigstore-bundle mode.
	script := fmt.Sprintf(`#!/bin/sh
set -e
# Fetch the tag list; exit immediately if wget fails so a registry error
# does not silently let the test pass with an empty/error response.
if ! TAGS=$(wget -qO- "http://%s/v2/%s/tags/list"); then
  echo "FAIL: could not reach registry tags/list endpoint"
  exit 1
fi
echo "Tags response: ${TAGS}"
if printf '%%s' "${TAGS}" | grep -qE '"[^"]*\.(sig|att)"'; then
  echo "FAIL: found legacy .sig or .att tags; sigstore-bundle mode must not create these"
  exit 1
fi
echo "PASS: no legacy signature or attestation tags found"
`, registryHost, imageRepo)

	return objects.NewTaskRunObjectV1(&v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "verify-no-legacy-tags-",
			Namespace:    ns,
		},
		Spec: v1.TaskRunSpec{
			TaskSpec: &v1.TaskSpec{
				Steps: []v1.Step{{
					Name:   "check-no-legacy-tags",
					Image:  "alpine:3.19",
					Script: script,
				}},
			},
		},
	})
}
