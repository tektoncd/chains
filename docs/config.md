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
The value of the `IMAGES` result is a list of images, each qualified by digest.
The list of images can be separated by commas or by newlines.

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
| `artifacts.taskrun.format` | The format to store `TaskRun` payloads in. | `tekton`, `in-toto`| `tekton` |
| `artifacts.taskrun.storage` | The storage backend to store `TaskRun` signatures in. Multiple backends can be specified with comma-separated list ("tekton,oci"). To disable the `TaskRun` artifact input an empty string ("").  | `tekton`, `oci`, `gcs`, `docdb`, `grafeas` | `tekton` |
| `artifacts.taskrun.signer` | The signature backend to sign `Taskrun` payloads with. | `x509`, `kms` | `x509` |

> NOTE: For grafeas storage backend, currently we only support Container Analysis. We will make grafeas server address configurabe within a short time.

### OCI Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `artifacts.oci.format` | The format to store `OCI` payloads in. | `simplesigning` | `simplesigning` |
| `artifacts.oci.storage` | The storage backend to store `OCI` signatures in. Multiple backends can be specified with comma-separated list ("oci,tekton"). To disable the `OCI` artifact input an empty string ("").| `tekton`, `oci`, `gcs`, `docdb`, `grafeas` | `oci` |
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
|`storage.grafeas.projectid`|The project ID to store occurrences|||
|`storage.grafeas.noteid` (optional)|The note ID to link occurrences. If noteid is not provided, a name in the format of `tekton-<NAMESPACE>` will be used.|||

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
| `signers.x509.fulcio.address` | EXPERIMENTAL. Fulcio address to request certificate from, if enabled | |`https://v1.fulcio.sigstore.dev` |
| `signers.x509.fulcio.issuer` | EXPERIMENTAL. Expected OIDC issuer. | |`https://oauth2.sigstore.dev/auth` |
| `signers.x509.fulcio.provider` | EXPERIMENTAL. Provider to request ID Token from | `google`, `spiffe`, `github`, `filesystem` | Unset, each provider will be attempted. |

#### KMS OIDC and Spire Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `signers.kms.auth.address` | URI of KMS server (e.g. the value of `VAULT_ADDR`) | |
| `signers.kms.auth.token` | Auth token KMS server (e.g. the value of `VAULT_TOKEN`) | |
| `signers.kms.auth.oidc.path` | Path used for OIDC authentication (e.g. `jwt` for Vault) | |
| `signers.kms.auth.oidc.role` | Role used for OIDC authentication | |
| `signers.kms.auth.spire.sock` | URI of the Spire socket used for KMS token (e.g. `unix:///tmp/spire-agent/public/api.sock`) | |
| `signers.kms.auth.spire.audience` | Audience for requesting a SVID from Spire | |
