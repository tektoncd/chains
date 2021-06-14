# Chains Configuration

`Chains` works by observing `TaskRun` executions, capturing relevant information, and storing it in a cryptographically-signed format.

`TaskRuns` can indicate inputs and outputs which are then captured and surfaced in the `Chains` payload formats, where relevant.
`Chains` uses the standard mechanisms (`Results` and `PipelineResouces`) where possible, and provides a few other mechanisms to *hint* at the correct inputs and outputs. These are outlined below:

## Chains Type Hinting

When outputing an OCI image without using a `PipelineResource`, `Chains` will look for the following Results:

* `IMAGE_URL` - The URL to the built OCI image
* `IMAGE_DIGEST` - The Digest of the built OCI image

For in-toto attestations, see [INTOTO.md](INTOTO.md) for description
of in-toto specific type hinting.

Note that these are provided automatically when using `PipelineResources`.


## Chains Configuration

Chains uses a `ConfigMap` called `chains-config` in the `tekton-chains` namespace for configuration.
Supported keys include (scroll right for defaults):

| Key | Description | Supported Values | Default |
| --- | --- | --- | --- |
| `artifacts.taskrun.format` | The format to store `TaskRun` payloads in. | `tekton`, `in-toto` | `tekton` |
| `artifacts.taskrun.storage` | The storage backend to store `TaskRun` signatures in. | `tekton`, `oci`, `gcs`, `docdb` | `tekton` |
| `artifacts.taskrun.signer` | The signature backend to sign `Taskrun` payloads with. | `pgp`, `x509`, `kms` | `x509` |
| `artifacts.oci.format` | The format to store `OCI` payloads in. | `tekton`, `simplesigning` | `simplesigning` |
| `artifacts.oci.storage` | The storage backend to store `OCI` signatures in. | `tekton`, `oci`, `gcs`, `docdb` | `oci` |
| `artifacts.oci.signer` | The signature backend to sign `OCI` payloads with. | `pgp`, `x509`, `kms` | `x509` |
| `signers.kms.kmsref` | The URI reference to a KMS service to use in `KMS` signers. | `gcpkms://projects/<project>/locations/<location>/keyRings/<keyring>/cryptoKeys/<key>`| |
| `storage.docdb.url` | The go-cloud URI reference to a docstore collection | `firestore://projects/<project>/databases/(default)/documents/<collection>?name_field=name`| |
| `transparency.enabled` | EXPERIMENTAL. Whether to enable automatic binary transparency uploads. | `false` | |
| `transparency.url` | EXPERIMENTAL. The URL to upload binary transparency attestations to, if enabled. | `https://rekor.sigstore.dev`| |

