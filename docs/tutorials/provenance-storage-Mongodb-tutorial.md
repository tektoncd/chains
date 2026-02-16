<!--
---
linkTitle: "Tutorial: Store Signed Provenance artifacts in DocDB (MongoDB)"
weight: 200
---
-->

# Using DocDB (MongoDB) as storage for Tekton Chains signed artifacts

This tutorial will guide you through setting up **MongoDB** as the storage backend for Tekton Chains. We will cover deploying a MongoDB instance, securely configuring authentication using Kubernetes Secrets, and understanding the different configuration options available.

## Prerequisites

Before starting, ensure you have the following installed:

*   **Kubernetes Cluster**: A local cluster like [kind](https://kind.sigs.k8s.io/) is sufficient.
    ```shell
    kind create cluster --name chains-mongo-test
    ```
*   **kubectl**: The Kubernetes command-line tool.
*   **Cosign**: For generating signing keys. [Installation Guide](https://docs.sigstore.dev/cosign/system_config/installation/)

## Step 1: Install Tekton Pipelines and Chains

```shell
# Install Tekton Pipelines
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml

# Install Tekton Chains
kubectl apply -f https://storage.googleapis.com/tekton-releases/chains/latest/release.yaml
```

Wait for the components to be ready:

```shell
kubectl wait --for=condition=Ready pods --all -n tekton-chains --timeout=60s
```

## Step 2: Set up Signing Keys

Generate a key pair for signing artifacts and store it as a Kubernetes Secret.

```shell
# Generate key pair
cosign generate-key-pair

# Create the secret in the tekton-chains namespace
kubectl create secret generic signing-secrets \
  -n tekton-chains \
  --from-file=cosign.key \
  --dry-run=client -o yaml | kubectl apply -f -
```

## Step 3: Deploy a MongoDB Instance

For this tutorial, we will deploy a simple MongoDB instance inside your cluster and expose it via a Service on port 27017. (27017 is the standard port for MongoDB servers.)

```shell
# Run a MongoDB pod using the standard port 27017
kubectl run mongo1 --image=mongo:6 --port=27017

# Expose the pod as a Service
kubectl expose pod mongo1 --name=mongo1-svc --port=27017 --target-port=27017
```

Your MongoDB connection string is now: `mongodb://mongo1-svc.default:27017`

## Step 4: Securely Configure MongoDB Authentication

Instead of hardcoding the connection string in the Chains configuration, we will store it in a Kubernetes Secret and mount it into the Chains controller.

1.  **Create the Secret**

    First, write the connection string to a file:
    ```shell
    echo -n "mongodb://mongo1-svc.default:27017" > mongo-url.txt
    ```

    Create a secret named `mongo-secret` containing this file:
    ```shell
    kubectl create secret generic mongo-secret \
      -n tekton-chains \
      --from-file=mongo-url=mongo-url.txt
    ```

2.  **Mount the Secret to the Chains Controller**

    We need to update the Chains controller deployment to mount this secret so the application can read the connection string from a file.

    We will patch the deployment to:
    *   Add a volume for the `mongo-secret`.
    *   Mount that volume to `/etc/secrets/mongo`.
    (This patch assumes the controller is the first controller)

    ```shell
    kubectl patch deployment tekton-chains-controller \
      -n tekton-chains \
      --type='json' \
      -p='[
        {
          "op": "add",
          "path": "/spec/template/spec/volumes/-",
          "value": {
            "name": "mongo-secret",
            "secret": {
              "secretName": "mongo-secret"
            }
          }
        },
        {
          "op": "add",
          "path": "/spec/template/spec/containers/0/volumeMounts/-",
          "value": {
            "name": "mongo-secret",
            "mountPath": "/etc/secrets/mongo",
            "readOnly": true
          }
        }
      ]'
    ```

    Restart the controller to apply the changes:
    ```shell
    kubectl rollout restart deploy/tekton-chains-controller -n tekton-chains
    ```

## Step 5: Configure Chains to use DocDB

Now we configure Chains to use the `docdb` backend and point it to the mounted secret file.

```shell
kubectl patch configmap chains-config -n tekton-chains --type merge -p '{
  "data": {
    "artifacts.taskrun.storage": "docdb",
    "artifacts.pipelinerun.storage": "docdb",
    "storage.docdb.url": "mongo://tekton-chains/attestations?id_field=_id",
    "storage.docdb.mongo-server-url-path": "/etc/secrets/mongo/mongo-url"
  }
}'
```

**Explanation of Configuration:**
*   `artifacts.taskrun.storage`: Tells Chains to store TaskRun artifacts in `docdb`.
*   `storage.docdb.url`: The Go Cloud URL for the collection. `mongo://` indicates MongoDB, `tekton-chains` is the database, `attestations` is the collection, and `id_field=_id` sets the primary key.
*   `storage.docdb.mongo-server-url-path`: The absolute path to the file containing the connection string (which we mounted in Step 4).


## Configuration Priority for MongoDB URL

Chains offers multiple ways to provide the MongoDB connection string. They are checked in the following order of priority (highest to lowest):

1.  **`storage.docdb.mongo-server-url-path`** (Recommended)
    *   **Description**: Path to a specific file containing the connection string.
    *   **Use Case**: When mounting a specific secret key as a file (as done in this tutorial).
    *   **Example**: `/etc/secrets/mongo/mongo-url`

2.  **`storage.docdb.mongo-server-url-dir`**
    *   **Description**: Path to a directory. Chains looks for a file named `MONGO_SERVER_URL` inside this directory.
    *   **Use Case**: When mounting a whole secret or directory where the filename is implicitly expected to be `MONGO_SERVER_URL`.

3.  **`storage.docdb.mongo-server-url`**
    *   **Description**: The connection string value directly.
    *   **Use Case**: Quick testing or non-sensitive environments. **Not recommended** for production as it exposes credentials in the ConfigMap.

4.  **`MONGO_SERVER_URL` Environment Variable**
    *   **Description**: An environment variable set on the controller pod.
    *   **Use Case**: Legacy configuration or 12-factor app patterns where config maps aren't used.

## Step 6: Verify the Setup

1.  **Create a TaskRun**
    Create a simple TaskRun to generate an artifact.
    ```shell
    kubectl create -f - <<EOF
    apiVersion: tekton.dev/v1beta1
    kind: TaskRun
    metadata:
      generateName: simple-task-
    spec:
      taskSpec:
        steps:
        - name: echo
          image: alpine
          script: |
            echo "Hello World"
    EOF
    ```

2.  **Check Logs**
    Watch the Chains controller logs to see the artifact being stored.
    ```shell
    kubectl logs -f deploy/tekton-chains-controller -n tekton-chains
    ```
    You should see messages indicating successful storage to DocDB.

3.  **Query MongoDB**
    Connect to the MongoDB pod to verify the data exists.
    ```shell
    kubectl exec -it mongo1 -- mongosh --eval "use tekton-chains; db.attestations.find()"
    ```
    You should see the stored provenance documents.

## Understanding Stored Artifacts

When Tekton Chains processes a TaskRun or PipelineRun, it generates provenance (attestation) and signs it. When using the `docdb` backend, these artifacts are stored as documents in your MongoDB collection.

### What Artifacts are Stored?

1.  **TaskRun / PipelineRun Provenance**:
    *   By enabling `artifacts.taskrun.storage: docdb` (or `artifacts.pipelinerun.storage: docdb`), Chains stores the signed provenance of the execution itself.
    *   The payload is typically in **SLSA** (Supply-chain Levels for Software Artifacts) format.

2.  **OCI Image Signatures**:
    *   If you enable `artifacts.oci.storage: docdb`, Chains can also store signatures for built container images in the database, rather than attaching them to the container registry.

### Document Schema

Each artifact is stored as a single document with the following fields:

| Field | Type | Description |
| :--- | :--- | :--- |
| `_id` | String | The unique identifier for the document (e.g., `taskrun-uid`). |
| `name` | String | The name/key of the artifact. |
| `signed` | Binary | The raw payload that was signed (e.g., the SLSA provenance JSON). |
| `signature` | String | The base64-encoded signature of the payload. |
| `cert` | String | The certificate used for signing (if using X.509). |
| `chain` | String | The certificate chain (if applicable). |
| `object` | Object | The unmarshaled JSON object of the payload. This allows you to query fields directly (e.g., `db.attestations.find({"object.predicate.buildType": "..."})`). |

**Example Document (JSON representation):**

```json
{
  "_id": "taskrun-simple-task-abcde",
  "name": "taskrun-simple-task-abcde",
  "signed": <Binary Data>,
  "signature": "MEUCIQD...",
  "object": {
    "_type": "https://in-toto.io/Statement/v0.1",
    "subject": [...],
    "predicateType": "https://slsa.dev/provenance/v0.2",
    "predicate": { ... }
  }
}
```

## Advanced: Hot Reloading Secrets

One advantage of using `storage.docdb.mongo-server-url-path` with Kubernetes Secrets is that Chains watches the file for changes. If you rotate your database credentials and update the Secret, Kubernetes updates the mounted file, and Chains automatically reloads the connection string without needing a restart.

To test this:
1.  Spin up a second MongoDB instance (`mongo2`).
2.  Update the `mongo-secret` with the new URL (`mongodb://mongo2-svc.default:27017`).
3.  Chains will detect the change and switch to the new database automatically.
