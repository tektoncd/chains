<!--
---
linkTitle: "OCI Artifact Distribution (Referrers)"
weight: 35
---
-->

# OCI Artifact Distribution: the Referrers Schema

When Chains signs an image, it has to put the signature and the attestation
somewhere. Both are stored in the same registry as the image. This page explains
the two ways Chains can do that, and how to turn on the newer one.

If you don't change anything, Chains uses the older tag-based layout and just
works. Read on if you want to use the OCI 1.1 Referrers API instead.

## The problem with the old layout

cosign, which Chains uses under the hood, has always stored a signature by
pushing an extra tag next to the image. For an image at digest `sha256:abc...`,
it creates tags like `sha256-abc....sig` and `sha256-abc....att`.

This works on every registry, but it isn't ideal:

- The registry fills up with extra tags that aren't real images.
- These tags are easy to confuse with actual image tags.
- No OCI standard describes this layout, so every tool has to special-case it.

## What the Referrers schema is

The [OCI 1.1 distribution spec](https://github.com/opencontainers/distribution-spec/blob/v1.1.0/spec.md#listing-referrers)
added a standard way to attach one artifact to another. Instead of inventing a
tag, you push the signature or attestation as its own manifest that records the
image it belongs to in a `subject` field. The registry can then answer a simple
question: "what artifacts refer to this image?"

This is the **Referrers schema**. The payoff is no extra tags, a clean registry,
and a standard that registries and policy tools already understand.

## How cosign enables it

cosign (and the `go-containerregistry` library it uses) already speaks the
Referrers schema. If the registry supports the Referrers API natively, cosign
uses it. If it doesn't, cosign falls back to the spec's **referrers tag schema**:
it keeps a single `sha256-<digest>` index tag and uses it to track referrers.

Either way it is still "referrers mode" — the content is the same, no `.sig` or
`.att` tags are created, and `cosign verify` and `oras discover` both work. This
fallback is automatic and needs no configuration, so it covers registries that
don't yet have native support.

## How Chains enables it

Chains exposes this through a single config flag in the `chains-config`
ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: chains-config
  namespace: tekton-chains
data:
  storage.oci.distribution-method: "referrers-api"
```

| Value | What Chains does |
|---|---|
| `legacy` (default) | Old tag-based layout (`.sig` / `.att` tags), DSSE-encoded. Works everywhere. |
| `referrers-api` | OCI 1.1 Referrers schema. No extra tags, with automatic fallback on registries that lack native support. |

That's the only knob. You pick *where* artifacts go, and Chains picks the right
encoding to match (explained next).

> [!NOTE]
> This flag only takes effect when the OCI storage backend is in use — that is,
> when `artifacts.oci.storage`, `artifacts.taskrun.storage`, or
> `artifacts.pipelinerun.storage` includes `oci`. If you store signatures and
> attestations somewhere else (for example Tekton results, a docstore, or
> Grafeas), `storage.oci.distribution-method` has no effect.

> [!TIP]
> The cosign bundled with Chains is new enough for everything described here, so
> you don't need to install anything or pass extra flags.

## Why referrers + protobuf, and not referrers + DSSE

There is one subtlety worth understanding: the *encoding* of the payload.

- In **legacy** mode, the attestation is a **DSSE** envelope. This is what cosign
  has always written and what existing tooling verifies.
- In **referrers** mode, Chains writes the attestation as a **Sigstore protobuf
  bundle**.

You might expect a third option — referrers with DSSE — but it's a dead end in
practice. cosign's verification for referrer-stored attestations expects the
protobuf bundle, not a DSSE envelope. A DSSE-over-referrers attestation can be
written, but `cosign verify-attestation` won't reliably verify it. The industry
is also moving toward the protobuf bundle as the standard format.

So Chains keeps it simple: legacy means DSSE, referrers means protobuf. The two
are tied together on purpose — it avoids a confusing matrix of combinations,
some of which no tool can actually verify. If a genuine need for a separate
encoding option ever appears, it can be added later without changing this flag.

Image **signatures** are not affected by this. In referrers mode they use
cosign's native signature manifest — the exact thing `cosign verify` looks for —
so signature verification works with no extra flags.

## Verifying

Verification is the same in both modes — point cosign at your key:

```shell
# Verify a signature
cosign verify \
  --key k8s://tekton-chains/signing-secrets \
  <image>@sha256:<digest>

# Verify an attestation
cosign verify-attestation \
  --key k8s://tekton-chains/signing-secrets \
  --type slsaprovenance \
  <image>@sha256:<digest>
```

To see what was stored, use [`oras`](https://oras.land/):

```shell
oras discover <image>@sha256:<digest>
```

## Things to keep in mind in referrers mode

These are interoperability notes, not bugs in Chains.

1. **`storage.oci.repository` is ignored.** A referrer has to live next to the
   image it points at, so the override that redirects storage elsewhere doesn't
   apply. Chains logs a warning and stores the referrer next to the image. The
   override still works in `legacy` mode.

2. **cosign's `--experimental-oci11` discovery may not find the attestation.**
   Chains stores it as a protobuf bundle; that older discovery path filters on a
   different type. The attestation is still there — `oras discover` shows it and
   policy engines can use it — and `cosign verify` of the signature is
   unaffected.

3. **Some registries accept a write but don't return it on read.** If a registry
   reports success but you can't read the referrer back, it isn't fully OCI 1.1
   compliant. Switch that registry to `legacy`.

4. **`oras discover` may show a different `artifactType` per registry.** This is
   display only; the stored content and verification don't change.

5. **Concurrent writes can race on fallback registries.** Without the native API,
   the index tag is updated with read-append-write, so simultaneous writes to the
   same image can drop an entry. A registry with the native Referrers API avoids
   this.

## Registry compatibility

cosign works with a wide range of registries, including AWS ECR, GCP Artifact
Registry, Docker Hub, Azure Container Registry, JFrog Artifactory, GitLab and
GitHub Container Registries, Harbor, and Quay. See the
[cosign registry support page](https://docs.sigstore.dev/cosign/system_config/registry_support/)
for the current list.

| Registry | `legacy` | `referrers-api` |
|---|---|---|
| GCR, ECR, Artifact Registry, quay.io | ✓ | ✓ (native Referrers API) |
| GHCR (`ghcr.io`) | ✓ | ✓ (native API not exposed; cosign uses the tag-schema fallback) |
| Any other OCI registry | ✓ | ✓ (native API if available, otherwise tag-schema fallback) |

## See also

- [Chains configuration reference](config.md) — all `storage.oci.*` keys.
- [Signing](signing.md) — how signing keys and secrets are configured.
- [cosign registry support](https://docs.sigstore.dev/cosign/system_config/registry_support/)
- [OCI distribution spec — Listing Referrers](https://github.com/opencontainers/distribution-spec/blob/v1.1.0/spec.md#listing-referrers)
