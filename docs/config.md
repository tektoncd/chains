<!--
---
linkTitle: "Chains Configuration"
weight: 20
---
-->

# Chains Configuration

`Chains` works by observing `TaskRun` and `PipelineRun` executions, capturing relevant information, and storing it in a cryptographically-signed format.

`TaskRuns` and `PipelineRuns` can indicate inputs and outputs which are then captured and surfaced in the `Chains` payload formats, where relevant.
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

When processing a `TaskRun`, Chains will parse through the list, then sign and attest each image.
When processing a `PipelineRun`, Chains will only attest each image. Thus, if both `TaskRun` and
`PipelineRun` produce type hint results, each image will have one signature and two attestations.

For in-toto attestations, see [intoto.md](intoto.md) for description
of in-toto specific type hinting.

Note that these are provided automatically when using `PipelineResources`.

## Chains Configuration

Chains uses a `ConfigMap` called `chains-config` in the `tekton-chains` namespace for configuration.
Supported keys include:

### TaskRun Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `artifacts.taskrun.format` | The format to store `TaskRun` payloads in. | `in-toto`, `slsa/v1`, `slsa/v2alpha1`| `in-toto` |
| `artifacts.taskrun.storage` | The storage backend to store `TaskRun` signatures in. Multiple backends can be specified with comma-separated list ("tekton,oci"). To disable the `TaskRun` artifact input an empty string ("").  | `tekton`, `oci`, `gcs`, `docdb`, `grafeas` | `tekton` |
| `artifacts.taskrun.signer` | The signature backend to sign `TaskRun` payloads with. | `x509`, `kms` | `x509` |

> NOTE:, `slsa/v1` is an alias of `in-toto` for backwards compatibality.
> NOTE:, `slsa/v2alpha1` is WIP and is likely to undergo multiple changes. 

### PipelineRun Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `artifacts.pipelinerun.format` | The format to store `PipelineRun` payloads in. | `in-toto`, `slsa/v1`| `in-toto` |
| `artifacts.pipelinerun.storage` | The storage backend to store `PipelineRun` signatures in. Multiple backends can be specified with comma-separated list ("tekton,oci"). To disable the `PipelineRun` artifact input an empty string ("").  | `tekton`, `oci`, `gcs`, `docdb`, `grafeas` | `tekton` |
| `artifacts.pipelinerun.signer` | The signature backend to sign `PipelineRun` payloads with. | `x509`, `kms` | `x509` |

> NOTE: For grafeas storage backend, currently we only support Container Analysis. We will make grafeas server address configurabe within a short time.
> NOTE: `slsa/v1` is an alias of `in-toto` for backwards compatibality.
> NOTE:, `slsa/v2alpha1` is currently not supported for pipelinerun level provenance. 

### OCI Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `artifacts.oci.format` | The format to store `OCI` payloads in. | `simplesigning` | `simplesigning` |
| `artifacts.oci.storage` | The storage backend to store `OCI` signatures in. Multiple backends can be specified with comma-separated list ("oci,tekton"). To disable the `OCI` artifact input an empty string ("").| `tekton`, `oci`, `gcs`, `docdb`, `grafeas` | `oci` |
| `artifacts.oci.signer` | The signature backend to sign `OCI` payloads with. | `x509`, `kms` | `x509` |

### KMS Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `signers.kms.kmsref` | The URI reference to a KMS service to use in `KMS` signers. | One of the three formats: <br/>`gcpkms://projects/[PROJECT]/locations/[LOCATION]>/keyRings/[KEYRING]/cryptoKeys/[KEY]` <br/> `gcpkms://projects/[PROJECT]/locations/[LOCATION]>/keyRings/[KEYRING]/cryptoKeys/[KEY]/cryptoKeyVersions/[KEY_VERSION]` <br/> `gcpkms://projects/[PROJECT]/locations/[LOCATION]>/keyRings/[KEYRING]/cryptoKeys/[KEY]/versions/[KEY_VERSION]`| |

### Storage Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `storage.gcs.bucket` | The GCS bucket for storage | | |
| `storage.oci.repository` | The OCI repo to store OCI signatures and attestation in | If left undefined _and_ one of `artifacts.{oci,taskrun}.storage` includes `oci` storage, attestations will be stored alongside the stored OCI artifact itself. ([example on GCP](../images/attestations-in-artifact-registry.png)) Defining this value results in the OCI bundle stored in the designated location _instead of_ alongside the image. See [cosign documentation](https://github.com/sigstore/cosign#specifying-registry) for additional information. | |
| `storage.docdb.url` | The go-cloud URI reference to a docstore collection | `firestore://projects/[PROJECT]/databases/(default)/documents/[COLLECTION]?name_field=name`| |
|`storage.grafeas.projectid`|The project of where grafeas server is located for storing occurrences|||
|`storage.grafeas.noteid` (optional)|This field will be used as the prefix part of the note name that will be created. The value of this field must be a string without spaces. (See more details [below](#grafeas).) |||
|`storage.grafeas.notehint` (optional)|This field is used to set the [human_readable_name](https://github.com/grafeas/grafeas/blob/cd23d4dc1bef740d6d6d90d5007db5c9a2431c41/proto/v1/attestation.proto#L49) field in the Grafeas ATTESTATION note. If it is not provided, the default `This attestation note was generated by Tekton Chains` will be used.|||

#### docstore
You can read about the go-cloud docstore URI format [here](https://gocloud.dev/howto/docstore/). Tekton Chains supports the following docstore services:
  * `firestore`
  * `dynamodb`
  * `mongo`

#### MongoDB
With MongoDB you will need to add a `MONGO_SERVER_URL` env var with the MongoDB connection URI to the `tekton-chains-controller`, the go-cloud URI is just to point at the db and collection

#### Grafeas
You can read more about Grafeas notes and occurrences [here](https://github.com/grafeas/grafeas/blob/master/docs/grafeas_concepts.md). To create occurrences, we have to create notes first that are used to link occurrences. Two types of occurrences will be created: `ATTESTATION` Occurrence and `BUILD` Occrrence. The configurable `noteid` is used as the prefix of the note name. Under the hood, the suffix `-simplesigning` will be appended for the `ATTESTATION` note, and the suffix `-intoto` will be appended for the `BUILD` note. If the `noteid` field is not configured, `tekton-<NAMESPACE>` will be used as the prefix.

### In-toto Configuration

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `builder.id` | The builder ID to set for in-toto attestations | | `https://tekton.dev/chains/v2`|

### Experimental Features Configuration

#### Transparency Log

| Key | Description | Supported Values | Default |
| :--- | :--- | :--- | :--- |
| `transparency.enabled` | EXPERIMENTAL. Whether to enable automatic binary transparency uploads. | `true`, `false`, `manual` | `false` |
| `transparency.url` | EXPERIMENTAL. The URL to upload binary transparency attestations to, if enabled. | |`https://rekor.sigstore.dev`|

**Note**: If `transparency.enabled` is set to `manual`, then only `TaskRuns` and `PipelineRuns` with the following annotation will be uploaded to the transparency log:

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
