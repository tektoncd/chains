<!--

---
linkTitle: "Sigstore"
weight: 50
---

-->

# Sigstore

## Transparency Log Support

Chains supports automatic binary uploads to a transparency log and defaults to
using [Rekor](https://github.com/sigstore/rekor). If enabled, all signatures and
attestations will be logged. The entry ID will be appended as an annotation on a
`TaskRun` or a `PipelineRun` once Chains has uploaded it:

```yaml
chains.tekton.dev/transparency: https://rekor.sigstore.dev/7599
```

### Enabling Transparency Log Support

To enable, run:

```shell
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"transparency.enabled": "true"}}'
```

Right now, Chains default to storing entries in the public Rekor instance
(<https://rekor.sigstore.dev>). To customize where entries are stored, run:

```shell
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"transparency.url": "<YOUR URL>"}}'
```

## Keyless Signing Mode

Chains also supports a keyless signing mode with
[Fulcio](https://github.com/sigstore/fulcio), Sigstore's free root certificate
authority.

In this mode, instead of setting up a signing key, Chains would request an
identity token from the cluster it is running in. This identity token will be
used to authorize a Fulcio certificate for a Tekton artifact (OCI image,
`TaskRun`, or `PipelineRun`). This feature has been tested on GKE,
[EKS](https://www.chainguard.dev/unchained/keyless-signing-with-tekton-on-amazon-eks-2),
and
[AKS](https://www.chainguard.dev/unchained/keyless-signing-with-tekton-on-aks),
but should work on any environment that supports
[Cosign OIDC signing](https://docs.sigstore.dev/cosign/openid_signing).

Once Chains has successfully requested a certificate, it will store the cert as
a base64 encoded annotation on the `TaskRun` or `PipelineRun` , along with the
payload and signature.

This can look like:

```yaml
Annotations:  chains.tekton.dev/cert-taskrun-57e7ef8e-13fb-4d27-af6e-dc4d68f73cc4:
              chains.tekton.dev/chain-taskrun-57e7ef8e-13fb-4d27-af6e-dc4d68f73cc4:
              chains.tekton.dev/payload-taskrun-57e7ef8e-13fb-4d27-af6e-dc4d68f73cc4:
                eyJfdHlwZSI6ImJ1aWxkLWNoYWlucy01dnhycyIsInByZWRpY2F0ZVR5cGUiOiJodHRwczovL3Rla3Rvbi5kZXYvY2hhaW5zL3Byb3ZlbmFuY2UiLCJzdWJqZWN0IjpbeyJuYW1lIj...
              chains.tekton.dev/signature-taskrun-57e7ef8e-13fb-4d27-af6e-dc4d68f73cc4:
                eyJwYXlsb2FkVHlwZSI6ImFwcGxpY2F0aW9uL3ZuZC5pbi10b3RvK2pzb24iLCJwYXlsb2FkIjoiZXlKZmRIbHdaU0k2SW1KMWFXeGtMV05vWVdsdWN5MDFkbmh5Y3lJc0luQnlaV1...
              chains.tekton.dev/signed: true
              chains.tekton.dev/transparency: https://rekor.sigstore.dev/7599
              pipeline.tekton.dev/release: v0.25.0
```

### Enabling Keyless Signing Mode

To enable singing with Fulcio, run:

```shell
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"signers.x509.fulcio.enabled": "true"}}'
```
