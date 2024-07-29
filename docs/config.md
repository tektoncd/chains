<!--
---
linkTitle: "Chains Configuration"
weight: 20
---
-->
# Chains Configuration
`Chains` works by observing `TaskRun` and `PipelineRun` executions, capturing relevant information, and storing it in a cryptographically-signed format.

`TaskRuns` and `PipelineRuns` can indicate inputs and outputs which are then captured and surfaced in the `Chains` payload formats, where relevant.
`Chains` uses the `Results` to _hint_ at the correct inputs and outputs. Check out [slsa-provenance.md](slsa-provenance.md) for more details.

## Chains Configuration

Chains uses a `ConfigMap` called `chains-config` in the `tekton-chains` namespace for configuration.
Supported keys include:

### TaskRun Configuration

| Key                         | Description                                                                                                                                                                                      | Supported Values                           | Default   |
| :-------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------- | :-------- |
| `artifacts.taskrun.format`  | The format to store `TaskRun` payloads in.                                                                                                                                                       | `in-toto`, `slsa/v1`, `slsa/v2alpha3`, `slsa/v2alpha4`      | `in-toto` |
| `artifacts.taskrun.storage` | The storage backend to store `TaskRun` signatures in. Multiple backends can be specified with comma-separated list ("tekton,oci"). To disable the `TaskRun` artifact input an empty string (""). | `tekton`, `oci`, `gcs`, `docdb`, `grafeas` | `tekton`  |
| `artifacts.taskrun.signer`  | The signature backend to sign `TaskRun` payloads with.                                                                                                                                           | `x509`, `kms`                              | `x509`    |

> NOTE:
>
> - `slsa/v1` is an alias of `in-toto` for backwards compatibility.
> - `slsa/v2alpha3` corresponds to the slsav1.0 spec. and uses latest [`v1` Tekton Objects](https://tekton.dev/docs/pipelines/pipeline-api/#tekton.dev/v1).  Recommended format for new chains users who want the slsav1.0 spec.
> - `slsa/v2alpha4` corresponds to the slsav1.0 spec. and uses latest [`v1` Tekton Objects](https://tekton.dev/docs/pipelines/pipeline-api/#tekton.dev/v1). It reads type-hinted results from [StepActions](https://tekton.dev/docs/pipelines/pipeline-api/#tekton.dev/v1alpha1.StepAction). Recommended format for new chains users who want the slsav1.0 spec.

### PipelineRun Configuration

| Key                                            | Description                                                                                                                                                                                                                                                                                 | Supported Values                           | Default   |
| :--------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | :----------------------------------------- | :-------- |
| `artifacts.pipelinerun.format`                 | The format to store `PipelineRun` payloads in.                                                                                                                                                                                                                                              | `in-toto`, `slsa/v1`, `slsa/v2alpha3`, `slsa/v2alpha4`      | `in-toto` |
| `artifacts.pipelinerun.storage`                | The storage backend to store `PipelineRun` signatures in. Multiple backends can be specified with comma-separated list ("tekton,oci"). To disable the `PipelineRun` artifact input an empty string ("").                                                                                    | `tekton`, `oci`, `gcs`, `docdb`, `grafeas` | `tekton`  |
| `artifacts.pipelinerun.signer`                 | The signature backend to sign `PipelineRun` payloads with.                                                                                                                                                                                                                                  | `x509`, `kms`                              | `x509`    |
| `artifacts.pipelinerun.enable-deep-inspection` | This boolean option will configure whether Chains should inspect child taskruns in order to capture inputs/outputs within a pipelinerun. `"false"` means that Chains only checks pipeline level results, whereas `"true"` means Chains inspects both pipeline level and task level results. | `"true"`, `"false"`                        | `"false"` |

> NOTE:
>
> - For grafeas storage backend, currently we only support Container Analysis. We will make grafeas server address configurabe within a short time.
> - `slsa/v1` is an alias of `in-toto` for backwards compatibility.
> - `slsa/v2alpha3` corresponds to the slsav1.0 spec. and uses latest [`v1` Tekton Objects](https://tekton.dev/docs/pipelines/pipeline-api/#tekton.dev/v1). Recommended format for new chains users who want the slsav1.0 spec.
> - `slsa/v2alpha4` corresponds to the slsav1.0 spec. and uses latest [`v1` Tekton Objects](https://tekton.dev/docs/pipelines/pipeline-api/#tekton.dev/v1). It reads type-hinted results from [StepActions](https://tekton.dev/docs/pipelines/pipeline-api/#tekton.dev/v1alpha1.StepAction) when `artifacts.pipelinerun.enable-deep-inspection` is set to `true`. Recommended format for new chains users who want the slsav1.0 spec.


### OCI Configuration

| Key                     | Description                                                                                                                                                                              | Supported Values                           | Default         |
| :---------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------- | :-------------- |
| `artifacts.oci.format`  | The format to store `OCI` payloads in.                                                                                                                                                   | `simplesigning`                            | `simplesigning` |
| `artifacts.oci.storage` | The storage backend to store `OCI` signatures in. Multiple backends can be specified with comma-separated list ("oci,tekton"). To disable the `OCI` artifact input an empty string (""). | `tekton`, `oci`, `gcs`, `docdb`, `grafeas` | `oci`           |
| `artifacts.oci.signer`  | The signature backend to sign `OCI` payloads with.                                                                                                                                       | `x509`, `kms`                              | `x509`          |

### KMS Configuration

| Key                  | Description                                                 | Supported Values                                                                                                                                | Default |
| :------------------- | :---------------------------------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------- | :------ |
| `signers.kms.kmsref` | The URI reference to a KMS service to use in `KMS` signers. | Supported schemes: `gcpkms://`, `awskms://`, `azurekms://`, `hashivault://`. See https://docs.sigstore.dev/cosign/kms_support for more details. |         |

### Storage Configuration

| Key                                             | Description                                                                                                                                                                                                                                                                                                         | Supported Values                                                                                                                                                                                                                                                                                                                                                                                                                                                    | Default |
|:------------------------------------------------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------|
| `storage.gcs.bucket`                            | The GCS bucket for storage                                                                                                                                                                                                                                                                                          |                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |         |
| `storage.oci.repository`                        | The OCI repo to store OCI signatures and attestation in                                                                                                                                                                                                                                                             | If left undefined _and_ one of `artifacts.{oci,taskrun}.storage` includes `oci` storage, attestations will be stored alongside the stored OCI artifact itself. ([example on GCP](../images/attestations-in-artifact-registry.png)) Defining this value results in the OCI bundle stored in the designated location _instead of_ alongside the image. See [cosign documentation](https://github.com/sigstore/cosign#specifying-registry) for additional information. |         |
| `storage.docdb.url`                             | The go-cloud URI reference to a docstore collection                                                                                                                                                                                                                                                                 | `firestore://projects/[PROJECT]/databases/(default)/documents/[COLLECTION]?name_field=name`                                                                                                                                                                                                                                                                                                                                                                         |         |
| `storage.docdb.mongo-server-url` (optional)     | The value of MONGO_SERVER_URL env var with the MongoDB connection URI                                                                                                                                                                                                                                               | Example: `mongodb://[USER]:[PASSWORD]@[HOST]:[PORT]/[DATABASE]`                                                                                                                                                                                                                                                                                                                                                                                                     |         |
| `storage.docdb.mongo-server-url-dir` (optional) | The path of the directory that contains the file named MONGO_SERVER_URL that stores the value of MONGO_SERVER_URL env var                                                                                                                                                                                           | If the file `/mnt/mongo-creds-secret/MONGO_SERVER_URL` has the value of MONGO_SERVER_URL, then set `storage.docdb.mongo-server-url-dir: /mnt/mongo-creds-secret`                                                                                                                                                                                                                                                                                                    |         |
| `storage.grafeas.projectid`                     | The project of where grafeas server is located for storing occurrences                                                                                                                                                                                                                                              |                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |         |
| `storage.grafeas.noteid` (optional)             | This field will be used as the prefix part of the note name that will be created. The value of this field must be a string without spaces. (See more details [below](#grafeas).)                                                                                                                                    |                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |         |
| `storage.grafeas.notehint` (optional)           | This field is used to set the [human_readable_name](https://github.com/grafeas/grafeas/blob/cd23d4dc1bef740d6d6d90d5007db5c9a2431c41/proto/v1/attestation.proto#L49) field in the Grafeas ATTESTATION note. If it is not provided, the default `This attestation note was generated by Tekton Chains` will be used. |                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |         |

#### docstore

You can read about the go-cloud docstore URI format [here](https://gocloud.dev/howto/docstore/). Tekton Chains supports the following docstore services:

- `firestore`
- `dynamodb`
- `mongo`

#### MongoDB

With MongoDB you will need to supply the value of `MONGO_SERVER_URL` env var with the MongoDB connection URI to the Tekton Chains, the go-cloud URI is just to point at the db and collection.
This can be achieved in a few ways:

- Setting the `MONGO_SERVER_URL` env var in the `tekton-chains-controller` deployment.
- Setting the value of `storage.docdb.mongo-server-url` field.
  - This field takes precedence over the `MONGO_SERVER_URL` env var.
- Setting the value of `storage.docdb.mongo-server-url-dir` field.
  - This field takes precedence over `storage.docdb.mongo-server-url` and `MONGO_SERVER_URL` env var.
  - The value should point to a directory that has a file named `MONGO_SERVER_URL` that contains the env var. Each time the file is updated, the new value will be read.
  - One common use case is to store the value of `MONGO_SERVER_URL` in a secret with the key `MONGO_SERVER_URL` and mount the secret at the path specified in this field. When the secret is updated, the new value will be fetched by Tekton Chains.

#### Grafeas

You can read more about Grafeas notes and occurrences [here](https://github.com/grafeas/grafeas/blob/master/docs/grafeas_concepts.md). To create occurrences, we have to create notes first that are used to link occurrences. Two types of occurrences will be created: `ATTESTATION` Occurrence and `BUILD` Occrrence. The configurable `noteid` is used as the prefix of the note name. Under the hood, the suffix `-simplesigning` will be appended for the `ATTESTATION` note, and the suffix `-intoto` will be appended for the `BUILD` note. If the `noteid` field is not configured, `tekton-<NAMESPACE>` will be used as the prefix.

### In-toto Configuration

| Key                         | Description                                    | Supported Values                                                                | Default                             |
| :-------------------------- | :--------------------------------------------- | :------------------------------------------------------------------------------ | :---------------------------------- |
| `builder.id`                | The builder ID to set for in-toto attestations |                                                                                 | `https://tekton.dev/chains/v2`      |
| `builddefinition.buildtype` | The buildType for in-toto attestations         | `https://tekton.dev/chains/v2/slsa`, `https://tekton.dev/chains/v2/slsa-tekton` | `https://tekton.dev/chains/v2/slsa` |

> NOTE:
> Considerations for the builddefinition.buildtype parameter:
>
> - It is only valid for `slsa/v2alpha3` configurations (see TaskRun or PipelineRun configuration).
> - The parameter can take one of two values:
>   - `https://tekton.dev/chains/v2/slsa`: This buildType strictly conforms to the slsav1.0 spec.
>   - `https://tekton.dev/chains/v2/slsa-tekton`: This buildType also conforms to the slsav1.0 spec, but adds additional informaton specific to Tekton. This information includes the PipelinRun/TaskRun labels and annotations as internalParameters. It also includes capturing each pipeline task in a PipelinRun under resolvedDependencies.

### Sigstore Features Configuration

#### Transparency Log

| Key                    | Description                                                        | Supported Values          | Default                      |
| :--------------------- | :----------------------------------------------------------------- | :------------------------ | :--------------------------- |
| `transparency.enabled` | Whether to enable automatic binary transparency uploads.           | `true`, `false`, `manual` | `false`                      |
| `transparency.url`     | The URL to upload binary transparency attestations to, if enabled. |                           | `https://rekor.sigstore.dev` |

**Note**: If `transparency.enabled` is set to `manual`, then only `TaskRuns` and `PipelineRuns` with the following annotation will be uploaded to the transparency log:

```yaml
chains.tekton.dev/transparency-upload: "true"
```

#### Keyless Signing with Fulcio

| Key                                | Description                                                   | Supported Values                           | Default                                            |
| :--------------------------------- | :------------------------------------------------------------ | :----------------------------------------- | :------------------------------------------------- |
| `signers.x509.fulcio.enabled`      | Whether to enable automatic certificates from fulcio.         | `true`, `false`                            | `false`                                            |
| `signers.x509.fulcio.address`      | Fulcio address to request certificate from, if enabled        |                                            | `https://v1.fulcio.sigstore.dev`                   |
| `signers.x509.fulcio.issuer`       | Expected OIDC issuer.                                         |                                            | `https://oauth2.sigstore.dev/auth`                 |
| `signers.x509.fulcio.provider`     | Provider to request ID Token from                             | `google`, `spiffe`, `github`, `filesystem` | Unset, each provider will be attempted.            |
| `signers.x509.identity.token.file` | Path to file containing ID Token.                             |                                            |
| `signers.x509.tuf.mirror.url`      | TUF server URL. $TUF_URL/root.json is expected to be present. |                                            | `https://sigstore-tuf-root.storage.googleapis.com` |

#### KMS OIDC and Spire Configuration

| Key                               | Description                                                                                 | Supported Values | Default |
| :-------------------------------- | :------------------------------------------------------------------------------------------ | :--------------- | :------ |
| `signers.kms.auth.address`        | URI of KMS server (e.g. the value of `VAULT_ADDR`)                                          |                  |
| `signers.kms.auth.token`          | Auth token KMS server (e.g. the value of `VAULT_TOKEN`)                                     |                  |
| `signers.kms.auth.token-dir`      | Path to store KMS server Auth token (e.g. `/etc/kms-secrets`)                               |                  |
| `signers.kms.auth.oidc.path`      | Path used for OIDC authentication (e.g. `jwt` for Vault)                                    |                  |
| `signers.kms.auth.oidc.role`      | Role used for OIDC authentication                                                           |                  |
| `signers.kms.auth.spire.sock`     | URI of the Spire socket used for KMS token (e.g. `unix:///tmp/spire-agent/public/api.sock`) |                  |
| `signers.kms.auth.spire.audience` | Audience for requesting a SVID from Spire                                                   |                  |
> NOTE:
>
> If `signers.kms.auth.token-dir` is set, create a secret with the key `KMS_AUTH_TOKEN` and ensure the Chains deployment mounts this secret to 
> the path specified by `signers.kms.auth.token-dir`. 

> [!IMPORTANT]
> To project the latest token values without needing to recreate the pod, avoid using `subPath` in volume mount.

## Namespaces Restrictions in Chains Controller
This feature allows you to specify a list of namespaces for the controller to monitor, providing granular control over its operation. If no namespaces are specified, the controller defaults to monitoring all namespaces.

### Usage
To restrict the Chains Controller to specific namespaces, pass a comma-separated list of namespaces as an argument to the controller using the --namespace flag.

### Example
To restrict the controller to the dev and test namespaces, you would start the controller with the following argument:
```shell
--namespace=dev,test
```
In this example, the controller will only monitor resources (pipelinesruns and taskruns) within the dev and test namespaces.
