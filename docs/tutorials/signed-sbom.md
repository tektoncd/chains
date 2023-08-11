<!--
---
linkTitle: "Tutorial: Signed SBOMs"
weight: 300
---
-->

# Chains Signed SBOM Tutorial

This tutorial details the steps required to use Tekton Chains to sign an SBOM for an image. It adds
SBOM generation to a minimal clone-build-push Pipeline.

At the end of this tutorial, the built image will contain a signed SBOM attestation. NOTE: This is
different than an SBOM attachment. SBOM attestations have a securer link to the described image as
well as the flexibility to provide multiple SBOMs for the same image.

## Prerequisites

A Kubernetes cluster with the following installed:

* Tekton Chains
* Tekton Pipelines

## Generate a Key Pair

First, we'll generate an encrypted x509 keypair and save it as a Kubernetes secret. Install
[cosign](https://github.com/sigstore/cosign) and run the following:

```shell
cosign generate-key-pair k8s://tekton-chains/signing-secrets
```

`cosign` will prompt you for a password, which will be stored in a Kubernetes secret named
`signing-secrets` in the `tekton-chains` namespace.

The public key will be written to a local file called `cosign.pub`.

## Set up Authentication

There are two forms of authentication that need to be set up:

1. The Chains controller will push signatures and attestations to an OCI registry using the
   credentials linked to your `TaskRun`'s service account. See our [authentication
   doc](../authentication.md)
2. The build and sbom Tasks will build and push content to the OCI registry.

Both of those can be setup by creating a `docker-registry` secret and linking it to the
ServiceAccount used by the TaskRuns. This tutorial assumes the ServiceAccount is `default`.

```shell
# Create a secret based on your local docker config.
kubectl create secret docker-registry tutorial-secret \
  --from-file=.dockerconfigjson=$HOME/.docker/config.json

# Link secret to service account
kubectl patch serviceaccount default -p '{"imagePullSecrets": [{"name": "tutorial-secret"}]}'
kubectl patch serviceaccount default -p '{"secrets": [{"name": "tutorial-secret"}]}'
```

## Configure Tekton Chains

You'll need to make these changes to the Tekton Chains Config:

* `artifacts.sbom.format=in-toto`
* `artifacts.sbom.storage=oci`
* `transparency.enabled=true`

You can set these fields by running:

```shell
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"artifacts.sbom.format": "in-toto"}}'
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"artifacts.sbom.storage": "oci"}}'
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"transparency.enabled": "true"}}'
```

This tells Chains to generate an in-toto attestation for the SBOM and store it in the image's OCI
registry. The SBOM signature will also be stored in [rekor](https://github.com/sigstore/rekor) since
transparency is enabled.

## Create SBOM Task

Similar to image signatures, Tekton Chains requires that the task producing the SBOM emits results
[named in a certain way](https://tekton.dev/docs/chains/config/#chains-type-hinting). This section
creates a sample Tekton Task that meets these requirements.

The [sample sbom Task](../../examples/sbom/sbom-task.yaml) uses
[syft](https://github.com/anchore/syft) to generate an SBOM in the CycloneDX format for a given
image. It then stores the generated SBOM as blob in the OCI registry. When the Task completes,
Tekton Chains downloads the SBOM blob and uses it as the payload of a new signed attestation that is
then attached to the image. To create it:

```shell
kubectl apply -f examples/sbom/sbom-task.yaml
```

It is possible to run this Task to see SBOM signing in action. But let's use it in a Pipeline for a
more realistic example.

## Build Pipeline

The [sample build pipeline](../../examples/sbom/sbom-pipeline.yaml) uses git to clone an application
repository, buildah to build an application container image, and syft to create an SBOM for the
image. To create it:

```shell
kubectl apply -f examples/sbom/sbom-pipeline.yaml
```

## Running

Let's use the tkn CLI to run the build pipeline. To do so, you'll need an OCI registry to push the
image to and a git repository that contains a valid Dockerfile in its root. For example:

```shell
# Set these accordingly
GIT_REPO=https://github.com/user/example
GIT_REVISION=main
OCI_REPO=quay.io/user/example

tkn -n minimal-container pipeline start simple-build \
  --param git-repo=${GIT_REPO} --param git-revision=${GIT_REVISION} \
  --param output-image=${OCI_REPO}:latest --param sbom-repo=${OCI_REPO} \
  --workspace name=shared,pvc,claimName="tekton-build" \
  --showlog
```

NOTE: The above assumes you have an existing PVC named `tekton-build`.

Towards the end of the log, there should a line like this:

```text
[sbom : store-sbom-blob] Digest: sha256:bb19013e908abf6d0d024d82c8990c30e84bea29796a361d828f78499b7ddf12
```

Make a note of the digest. This is the expected digest of the contents of the SBOM.

The digest can also be retrieved from the TaskRun's result:

```shell
kubectl get taskrun $TASK_RUN_NAME -o yaml | \
  yq '.status.results[] | select(.name == "IMAGE_SBOM_URL") | .value'
```

## Verification

At this point, the image is built and should have an SBOM attestation attached to it. Use cosign to
inspect it:

```shell
cosign download attestation ${OCI_REPO}:latest | \
  jq '.payload | @base64d | fromjson | select(.predicateType == "https://cyclonedx.org/schema") | .predicate'
```

Pipe the command above through `sha256sum`. The digest should match the digest of the SBOM retrieved
in the previous section.

You can also use cosign to verify that the SBOM is in fact signed:

```shell
cosign verify-attestation --key cosign.pub \
  --type 'https://cyclonedx.org/schema' ${OCI_REPO}:latest | \
  jq '.payload | @base64d | fromjson | .predicate' | sha256sum
```

NOTE: The above assumes you still have the `cosign.pub` file from the previous step. If you don't,
you can load the public key directly from Kubernetes by using `k8s://tekton-chains/signing-secrets`
as the value for the `--key` flag.

NOTE: If you do not set `transparency.enabled` to `true` in `chains-config`, you must use the flag
`--insecure-ignore-tlog` in the command above.

## Large SBOMs

Because Tekton Chains has to read the SBOM in order to sign it, it has a default maxium size of
10MB. This can be adjusted via the `artifacts.sbom.maxbytes` property in `chains-config`.
