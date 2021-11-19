<!--
---
linkTitle: "Chains Configuration"
weight: 20
---
-->

# Chains Configuration

`Chains` works by observing `TaskRun` executions, capturing relevant information, and storing it in a cryptographically-signed format.

`TaskRuns` can indicate inputs and outputs which are then captured and surfaced in the `Chains` payload formats, where relevant.
`Chains` uses the standard mechanisms (`Results` and `PipelineResources`) where possible, and provides a few other mechanisms to *hint* at the correct inputs and outputs. These are outlined below:

## Chains Type Hinting

When outputting an OCI image without using a `PipelineResource`, `Chains` will look for the following Results:

* `*IMAGE_URL` - The URL to the built OCI image
* `*IMAGE_DIGEST` - The Digest of the built OCI image

where `*` indicates any expression.
For example, if **both** `MYIMAGE_IMAGE_URL` AND `MYIMAGE_IMAGE_DIGEST` are correctly formatted to point to an OCI image, then `chains` will pick up on it and try to sign the image.

Multiple images can be specified by using different prefixes in place of `*`.

Multiple images can also be specified by using the `IMAGES` Result.
The value of the `IMAGES` result is a list of comma-separates images, each qualified by digest.

```shell
- name: IMAGES
  value: img1@sha256:digest1, img2@sha256:digest2
```

Chains will parse through the list and sign each image.

For in-toto attestations, see [intoto.md](intoto.md) for description
of in-toto specific type hinting.

Note that these are provided automatically when using `PipelineResources`.

## Chains Configuration

Chains uses a `ConfigMap` called `chains-config` in the `tekton-chains` namespace for configuration.
Supported keys include:

### TaskRun Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `artifacts.taskrun.format` | The format to store `TaskRun` payloads in. | `tekton`, `in-toto`, `tekton-provenance` | `tekton` |
| `artifacts.taskrun.storage` | The storage backend to store `TaskRun` signatures in. | `tekton`, `oci`, `gcs`, `docdb` | `tekton` |
| `artifacts.taskrun.signer` | The signature backend to sign `Taskrun` payloads with. | `x509`, `kms` | `x509` |

### OCI Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `artifacts.oci.format` | The format to store `OCI` payloads in. | `tekton`, `simplesigning` | `simplesigning` |
| `artifacts.oci.storage` | The storage backend to store `OCI` signatures in. | `tekton`, `oci`, `gcs`, `docdb` | `oci` |
| `artifacts.oci.signer` | The signature backend to sign `OCI` payloads with. | `x509`, `kms` | `x509` |

### KMS Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `signers.kms.kmsref` | The URI reference to a KMS service to use in `KMS` signers. | `gcpkms://projects/[PROJECT]/locations/[LOCATION]>/keyRings/[KEYRING]/cryptoKeys/[KEY]`| |

### Storage Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `storage.gcs.bucket` | The GCS bucket for storage | | |
| `storage.oci.repository` | The OCI repo to store OCI signatures in  | | |
| `storage.docdb.url` | The go-cloud URI reference to a docstore collection | `firestore://projects/[PROJECT]/databases/(default)/documents/[COLLECTION]?name_field=name`| |

### In-toto Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `builder.id` | The builder ID to set for in-toto attestations | | `tekton-chains`|

### Experimental Features Configuration

#### Transparency Log

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `transparency.enabled` | EXPERIMENTAL. Whether to enable automatic binary transparency uploads. | `true`, `false`, `manual` | `false` |
| `transparency.url` | EXPERIMENTAL. The URL to upload binary transparency attestations to, if enabled. | |`https://rekor.sigstore.dev`|

**Note**: If `transparency.enabled` is set to `manual`, then only TaskRuns with the following annotation will be uploaded to the transparency log:

```yaml
chains.tekton.dev/transparency-upload: "true"
```

#### Keyless Signing with Fulcio

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `signers.x509.fulcio.enabled` | EXPERIMENTAL. Whether to enable automatic certificates from fulcio. | `true`, `false` | `false`|
| `signers.x509.fulcio.auth`    | EXPERIMENTAL. Auth mechanism for verifying identity for fulcio, if enabled  | `google`       | `google` |
| `signers.x509.fulcio.address` | EXPERIMENTAL. Fulcio address to request certificate from, if enabled | |`https://fulcio.sigstore.dev` |
