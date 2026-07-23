<!--
---
linkTitle: "OCI Encoding Format"
weight: 35
---
-->

# OCI Storage Encoding Format

The `storage.oci.encoding-format` key controls how Chains serializes OCI
signatures and attestations when using the OCI storage backend. Setting this
key also determines the storage layout, because the two are tightly coupled —
see [Why they are coupled](#why-the-encoding-and-layout-are-coupled).

If you don't change anything, Chains keeps the existing `dsse` behavior and
everything just works. Read on if you want to opt into `sigstore-bundle`, or to
understand the difference.

## Encoding formats

The *encoding* is the serialization format of the attestation payload itself.

### DSSE (default)

[Dead Simple Signing Envelope](https://github.com/secure-systems-lab/dsse) is
the format cosign has used since its early versions. An attestation is a JSON
object with a base64-encoded payload and one or more signatures computed over
the DSSE Pre-Authentication Encoding (PAE) of the content.

DSSE is widely supported and is what all existing tooling — older
`cosign verify-attestation`, most policy engines — expects by default.

### Sigstore protobuf bundle

The [Sigstore bundle specification](https://github.com/sigstore/protobuf-specs)
defines a Protocol Buffer message that combines the payload, signature, signing
certificate, and optional transparency log entry into a single self-contained
artifact. This is the default format for cosign v3+ (it was available in
v2.4–v2.6 behind the `--new-bundle-format` flag).

Because the bundle includes the certificate and tlog entry inline, it supports
offline verification without out-of-band trust anchor lookups.

## How the encoding determines the storage layout

The encoding choice also determines *where* signatures and attestations are
stored in the registry. The supported combinations are:

| `encoding-format` | Attestation encoding | Storage layout |
|---|---|---|
| `dsse` (default) | DSSE JSON envelope | Tag-based: `.sig` / `.att` tags pushed alongside the image |
| `sigstore-bundle` | Sigstore protobuf bundle | OCI 1.1 Referrers API: referrer manifests with a `subject` field |

### Tag-based layout (`dsse`)

cosign stores signatures and attestations as extra tags next to the image:
`sha256-<digest>.sig` and `sha256-<digest>.att`. Every OCI-compliant registry
supports this, but the tags have drawbacks:

- Tags are meant to name top-level artifacts. Using them for signatures and
  attestations puts metadata in the same namespace as real images.
- The `.att` tag holds *all* attestations in a single manifest whose digest
  changes with each write, so there is no stable, individually addressable
  reference to any single attestation.
- No OCI standard describes this layout, so every tool must special-case it.

### OCI 1.1 Referrers API (`sigstore-bundle`)

The [OCI 1.1 distribution spec](https://github.com/opencontainers/distribution-spec/blob/v1.1.0/spec.md#listing-referrers)
added a standard way to attach one artifact to another. Each signature or
attestation is pushed as its own manifest with a `subject` field pointing to
the image it belongs to. No extra tags are created, and any OCI-compliant tool
can discover the relationship via the Referrers API.

> [!NOTE]
> On registries without native Referrers API support,
> `go-containerregistry` (the library Chains uses) automatically falls back to
> the spec's referrers tag schema — a single `sha256-<digest>` index tag
> listing all referrers. The protobuf bundle encoding is unchanged; only the
> discovery mechanism differs. No configuration is needed.

## Why the encoding and layout are coupled

The encoding and storage layout are intentionally coupled into a single knob.
`dsse` always uses tag-based storage; `sigstore-bundle` always uses the OCI 1.1
Referrers API. There is no mix-and-match.

Image **signatures** and attestations are both stored as Sigstore protobuf
bundle referrers in `sigstore-bundle` mode. Both signatures and attestations use
a `DsseEnvelope` bundle: for signatures the DSSE envelope wraps a
SimpleSigning payload (matching what `cosign sign` produces in v3.x), while for
attestations it wraps the in-toto statement. Using `DsseEnvelope` for signatures
is required because `WriteAttestationNewBundleFormat` always sets the
`dev.sigstore.bundle.content: dsse-envelope` annotation — a `MessageSignature`
bundle would make that annotation inconsistent and cause `cosign verify` to
fail. Both carry the
`dev.sigstore.bundle.predicateType` annotation to identify their content:
`"https://sigstore.dev/cosign/sign/v1"` for signatures and the in-toto
predicate type (e.g. `"https://slsa.dev/provenance/v0.2"`) for attestations.

## Configuring the encoding format

Chains exposes this through a single config flag in the `chains-config`
ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: chains-config
  namespace: tekton-chains
data:
  storage.oci.encoding-format: "sigstore-bundle"
```

| Value | Encoding | Storage layout |
|---|---|---|
| `dsse` (default) | DSSE JSON envelope | `.sig` / `.att` tags |
| `sigstore-bundle` | Sigstore protobuf bundle | OCI 1.1 Referrers API (with automatic fallback to referrers tag schema on older registries) |

> [!NOTE]
> This flag only takes effect when the OCI storage backend is in use — that is,
> when `artifacts.oci.storage`, `artifacts.taskrun.storage`, or
> `artifacts.pipelinerun.storage` includes `oci`. If you store signatures and
> attestations somewhere else (docstore, mongo, Grafeas, or other supported
> backends), `storage.oci.encoding-format` has no effect.

> [!TIP]
> Chains vendors the Sigstore libraries it needs, so `sigstore-bundle` mode
> works out of the box — there is nothing extra to install. You only need a
> separate `cosign` or `oras` CLI if you want to verify or inspect stored
> artifacts yourself, as shown below.

## Verifying

Verification is the same in both modes — point cosign at your key:

```shell
# Verify a signature
cosign verify \
  --key k8s://tekton-chains/signing-secrets \
  --insecure-ignore-tlog=true \
  <image>@sha256:<digest>

# Verify an attestation
cosign verify-attestation \
  --key k8s://tekton-chains/signing-secrets \
  --type slsaprovenance \
  --insecure-ignore-tlog=true \
  <image>@sha256:<digest>
```

> [!NOTE]
> In `sigstore-bundle` mode, `cosign verify --key` returns **both** the
> signature bundle and the SLSA attestation bundle. Check the `"type"` field in
> each result to distinguish them:
> - `"https://sigstore.dev/cosign/sign/v1"` → image signature
> - `"https://slsa.dev/provenance/v0.2"` → SLSA provenance attestation

> [!IMPORTANT]
> Starting with **cosign v2.0**, and continuing through the v3.x series,
> `cosign verify` and `cosign verify-attestation` check for transparency-log
> (Rekor) inclusion **by default**. Whether that check can pass depends on
> Chains' transparency setting:
>
> - **Transparency disabled** (`transparency.enabled: "false"`): Chains does not
>   record signatures in a transparency log, so there are no Rekor entries and
>   the default cosign check fails with an error such as:
>
>   ```text
>   Error: no matching signatures: ... not enough verified log entries from
>   transparency log: 0 < 1
>   ```
>
>   This is not a signature problem. Add `--insecure-ignore-tlog=true` to the
>   commands above to verify against the public key alone.
>
> - **Transparency enabled** (`transparency.enabled: "true"`): signatures are
>   recorded in Rekor, so the default tlog check passes and no extra flag is
>   needed.

To see what was stored, use [`oras`](https://oras.land/):

```shell
oras discover <image>@sha256:<digest>
```

In `sigstore-bundle` mode you will see two referrers, both with
`application/vnd.dev.sigstore.bundle.v0.3+json` as their `artifactType`.
`oras` displays the artifactType directly for both:

```text
<image>@sha256:<digest>
├── application/vnd.dev.sigstore.bundle.v0.3+json   ← signature
│   └── sha256:<sig-manifest-digest>
└── application/vnd.dev.sigstore.bundle.v0.3+json   ← SLSA attestation
    └── sha256:<att-manifest-digest>
```

To distinguish them, inspect the `dev.sigstore.bundle.predicateType` annotation
on each manifest:

```shell
oras manifest fetch <image>@sha256:<sig-manifest-digest> | jq '.annotations["dev.sigstore.bundle.predicateType"]'
# → "https://sigstore.dev/cosign/sign/v1"        (signature)

oras manifest fetch <image>@sha256:<att-manifest-digest> | jq '.annotations["dev.sigstore.bundle.predicateType"]'
# → "https://slsa.dev/provenance/v0.2"           (SLSA attestation)
```

In `dsse` mode, `oras discover` returns no referrers because the signature and
attestations are stored as ordinary tags (`.sig` / `.att`) rather than referrer
manifests.

## Things to keep in mind in `sigstore-bundle` mode

These are interoperability notes, not bugs in Chains.

1. **`storage.oci.repository` is ignored.** This setting normally redirects where
   OCI signatures and attestations are stored, letting you keep them in a
   different repository from the image. A referrer, by contrast, must live in the
   same repository as the image it points at, because the referrer manifest
   references its subject by digest within that repository. In `sigstore-bundle`
   mode Chains logs a warning and stores the referrer next to the image. The
   override still works in `dsse` mode.

2. **Older cosign discovery paths may not surface the attestation.** Chains
   stores attestations as a protobuf bundle, which is the default for current
   cosign versions. Older cosign releases that default to the tag-based layout
   discover attestations by a different type and may not list it. The attestation
   is still present — `oras discover` shows it and policy engines can consume it —
   and `cosign verify` of the signature is unaffected.

3. **Some registries accept a write but don't return it on read.** If a registry
   reports success but you can't read the referrer back, it isn't fully OCI 1.1
   compliant. Switch that registry to `dsse`.

4. **Both signature and attestation have the same `artifactType`.** Both are
   stored as `application/vnd.dev.sigstore.bundle.v0.3+json` referrers.
   Use the `dev.sigstore.bundle.predicateType` annotation to distinguish them:
   `"https://sigstore.dev/cosign/sign/v1"` for signatures and the in-toto
   predicate type for attestations. `cosign verify` and `cosign verify-attestation`
   do this automatically.

5. **Kyverno ClusterPolicy users must migrate to ImageValidatingPolicy.**
   `ClusterPolicy` discovers signatures via the `.sig` tag, which no longer
   exists in `sigstore-bundle` mode. Switch to `ImageValidatingPolicy`, which
   understands the OCI 1.1 Referrers API and the bundle predicateType annotation.

6. **Concurrent writes can race on the tag-schema fallback.** On registries
   without a native Referrers API, the index tag is updated with a
   read-append-write cycle, so simultaneous writes to the same image can drop an
   entry. Registries with the native Referrers API are not affected. This does
   not apply to `dsse` mode.

## Registry compatibility

cosign works with a wide range of registries, including Amazon ECR, Azure
Container Registry, Docker Hub, GitHub Container Registry, GitLab Container
Registry, Google Artifact Registry, Harbor, JFrog Artifactory, and Quay. See the
[cosign registry support page](https://docs.sigstore.dev/cosign/system_config/registry_support/)
for the current list.

Both `dsse` and `sigstore-bundle` work against any OCI-compliant registry.
In `sigstore-bundle` mode, registries with a native Referrers API use it directly;
the rest fall back automatically to the referrers tag schema, as described above.
You do not need to know a registry's level of OCI 1.1 support in advance.

## See also

- [Chains configuration reference](config.md) — all `storage.oci.*` keys.
- [Signing](signing.md) — how signing keys and secrets are configured.
- [cosign registry support](https://docs.sigstore.dev/cosign/system_config/registry_support/)
- [OCI distribution spec — Listing Referrers](https://github.com/opencontainers/distribution-spec/blob/v1.1.0/spec.md#listing-referrers)
