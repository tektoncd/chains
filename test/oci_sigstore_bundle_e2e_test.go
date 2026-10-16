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

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/test/tekton"
)

// TestOCIStorageSigstoreBundle_TaskRun verifies that when encoding-format is
// set to sigstore-bundle, Chains stores OCI artifact signatures and attestations
// as OCI 1.1 referrers instead of legacy digest-derived .sig/.att tags.
func TestOCIStorageSigstoreBundle_TaskRun(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{registry: true})
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

	// Verify that no legacy .sig or .att tags were written. In sigstore-bundle
	// mode, signatures and attestations must be stored as referrers, not tags.
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
		registry:        true,
		kanikoTaskImage: imageName,
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

// TestOCIStorageSigstoreBundle_Transparency_TaskRun verifies that with
// transparency enabled, the signature bundle Chains writes is consistent: its
// content is a MessageSignature and it carries a tlog entry. Chains uploads a
// hashedrekord entry for simplesigning, which only pairs with a MessageSignature
// bundle; a DsseEnvelope would be unverifiable. Exercised against the configured
// public Rekor.
func TestOCIStorageSigstoreBundle_Transparency_TaskRun(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{registry: true})
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
		"transparency.enabled":            "true", //nolint:goconst
	})
	t.Cleanup(resetConfig)
	time.Sleep(3 * time.Second) // https://github.com/tektoncd/chains/issues/664

	imageName := "chains-test-referrers-transparency"
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

	// Wait for Chains to sign the image and upload to the transparency log.
	obj := waitForCondition(ctx, t, c.PipelineClient, createdTro, signed, 2*time.Minute)
	if obj == nil {
		t.Fatal("kaniko TaskRun was never signed by Chains")
	}

	// Assert the signature bundle is a message-signature with a non-empty tlog
	// entry. Run in-cluster so the internal registry is reachable.
	verifyTro := verifyBundleConsistencyTaskRun(ns, image)
	createdVerify := tekton.CreateObject(t, ctx, c.PipelineClient, verifyTro)
	if got := waitForCondition(ctx, t, c.PipelineClient, createdVerify, successful, 2*time.Minute); got == nil {
		t.Error("bundle-consistency check TaskRun never succeeded; signature bundle may be a DsseEnvelope or missing its tlog entry")
	}
}

// verifyNoLegacyTagsTaskRun returns a TaskRun that fails if any legacy .sig or
// .att tags exist in the given OCI image repository, confirming Chains stored
// signatures and attestations as OCI referrers. Runs in-cluster so the internal
// registry is reachable.
func verifyNoLegacyTagsTaskRun(ns, image string) *objects.TaskRunObjectV1 {
	// Split "host:port/repo/name" into registry host and repository path.
	parts := strings.SplitN(image, "/", 2)
	if len(parts) != 2 {
		panic(fmt.Sprintf("verifyNoLegacyTagsTaskRun: image %q has no '/' separator", image))
	}
	registryHost := parts[0]
	imageRepo := parts[1]

	// Fail if the registry has any tags ending in .sig or .att (legacy cosign
	// tag-based storage), which must not appear in sigstore-bundle mode.
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

// verifyBundleConsistencyTaskRun returns a TaskRun that asserts the signature
// bundle stored as an OCI referrer is a MessageSignature
// (dev.sigstore.bundle.content == "message-signature") and carries a non-empty
// tlog entry. Runs in-cluster so the internal registry is reachable.
func verifyBundleConsistencyTaskRun(ns, image string) *objects.TaskRunObjectV1 {
	// Split "host:port/repo/name" into registry host and repository path.
	parts := strings.SplitN(image, "/", 2)
	if len(parts) != 2 {
		panic(fmt.Sprintf("verifyBundleConsistencyTaskRun: image %q has no '/' separator", image))
	}
	registryHost := parts[0]
	imageRepo := parts[1]

	// Resolve the image digest, list its referrers, locate the message-signature
	// bundle, pull the bundle blob, and assert it carries a tlog entry.
	script := fmt.Sprintf(`#!/bin/sh
set -e
apk add --no-cache curl jq >/dev/null

REG=%s
REPO=%s
ACCEPT='application/vnd.oci.image.index.v1+json,application/vnd.oci.image.manifest.v1+json,application/vnd.docker.distribution.manifest.v2+json,application/vnd.docker.distribution.manifest.list.v2+json'

# Resolve the digest of the built image (pushed to the "latest" tag).
DIGEST=$(curl -sfI -H "Accept: ${ACCEPT}" "http://${REG}/v2/${REPO}/manifests/latest" \
  | tr -d '\r' | awk -F': ' 'tolower($1)=="docker-content-digest"{print $2}')
if [ -z "${DIGEST}" ]; then
  echo "FAIL: could not resolve image digest for ${REPO}:latest"
  exit 1
fi
echo "Image digest: ${DIGEST}"

# List referrers of the image and select the signature bundle by its content annotation.
IDX=$(curl -sf -H "Accept: application/vnd.oci.image.index.v1+json" \
  "http://${REG}/v2/${REPO}/referrers/${DIGEST}")
echo "Referrers: ${IDX}"

CONTENT=$(printf '%%s' "${IDX}" \
  | jq -r '[.manifests[].annotations["dev.sigstore.bundle.content"]] | map(select(. != null)) | .[0] // ""')
if [ "${CONTENT}" != "message-signature" ]; then
  echo "FAIL: signature bundle content is \"${CONTENT}\"; want \"message-signature\""
  exit 1
fi

BUNDLE_MANIFEST=$(printf '%%s' "${IDX}" \
  | jq -r '.manifests[] | select(.annotations["dev.sigstore.bundle.content"]=="message-signature") | .digest' \
  | head -1)
if [ -z "${BUNDLE_MANIFEST}" ]; then
  echo "FAIL: no message-signature bundle referrer found"
  exit 1
fi

# Fetch the bundle manifest, read its single layer, and pull the bundle blob.
MANIFEST=$(curl -sf -H "Accept: application/vnd.oci.image.manifest.v1+json" \
  "http://${REG}/v2/${REPO}/manifests/${BUNDLE_MANIFEST}")
LAYER=$(printf '%%s' "${MANIFEST}" | jq -r '.layers[0].digest')
if [ -z "${LAYER}" ] || [ "${LAYER}" = "null" ]; then
  echo "FAIL: bundle manifest has no layer"
  exit 1
fi
BUNDLE=$(curl -sf "http://${REG}/v2/${REPO}/blobs/${LAYER}")

# The bundle must carry at least one transparency-log entry.
TLOG_COUNT=$(printf '%%s' "${BUNDLE}" | jq '(.verificationMaterial.tlogEntries // []) | length')
if [ "${TLOG_COUNT}" -lt 1 ]; then
  echo "FAIL: signature bundle has no tlog entries; transparency upload missing"
  exit 1
fi

# The bundle content itself must be a messageSignature, not a dsseEnvelope.
if printf '%%s' "${BUNDLE}" | jq -e '.dsseEnvelope != null' >/dev/null; then
  echo "FAIL: signature bundle is a dsseEnvelope; want messageSignature"
  exit 1
fi
if printf '%%s' "${BUNDLE}" | jq -e '.messageSignature == null' >/dev/null; then
  echo "FAIL: signature bundle has no messageSignature content"
  exit 1
fi

echo "PASS: message-signature bundle with ${TLOG_COUNT} tlog entrie(s)"
`, registryHost, imageRepo)

	return objects.NewTaskRunObjectV1(&v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "verify-bundle-consistency-",
			Namespace:    ns,
		},
		Spec: v1.TaskRunSpec{
			TaskSpec: &v1.TaskSpec{
				Steps: []v1.Step{{
					Name:   "check-bundle-consistency",
					Image:  "alpine:3.19",
					Script: script,
				}},
			},
		},
	})
}
