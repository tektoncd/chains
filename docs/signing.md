# Signing Secrets

To get started signing things in Chains, you will need to generate a keypair and instruct Chains to sign with it via a Kubernetes secret.
Chains expects a private key, and password if the key is encrypted, to exist in a Kubernetes secret `signing-secrets` in the `tekon-chains` namespace. 

Chains supports a few different signature schemes, including x509 and KMS systems.

This doc explains how to generate keys and configure Chains for each type.
Note, **only one** of the following keys needs to be set up for Chains to work:
- [x509](#x509)
- [Cosign](#cosign)
- [KMS](#KMS)

# x509
For x509, Chains expects the private key to be stored in a secret called `signing-secrets` with the following structure:

* x509.pem (the private key)

Chains also has the following requirements:
* The private key to be stored as an unencrpyted PKCS8 PEM file (`BEGIN PRIVATE KEY`)
* The key is of type `ed25519` or `ecdsa`


## Generate x509 Keypair

To create a valid x509 keypair, you can run [gen-keys.sh](../gen-keys.sh).
This script will take care of creating the keypair (`x509.pem` and `x509.pub`), and storing the private key correctly in the `signing-secrets` Kubernetes Secret.

_Note: requires `openssl` is installed_


# Cosign
For cosign, Chains expects the encrypted private key to be stored in a secret called `signing-secrets` with the following structure:

* `cosign.key` (the cosign-generated private key)
* `cosign.password` (the password to decrypt the private key)

Chains also has the following requirements:
* The private key must be stored as an encrypted PEM file of type `ENCRYPTED COSIGN PRIVATE KEY`

## Generate cosign Keypair

To create a cosign keypair, `cosign.key` and `cosign.pub`, install [cosign](https://github.com/sigstore/cosign) and run the following:
```shell
cosign generate-key-pair -k8s tekton-chains/signing-secrets
```

Cosign will prompt you for a password, and create the Kubernetes secret for you.

# KMS
Chains uses a ["go-cloud"](https://github.com/google/go-cloud) URI like scheme for KMS references.
Chains supports GCP KMS and Hashicorp Vault today, but we would love to add support for more.

You can configure Chains to use a specific KMS key using the `signers.kms.kmsref` config key in `chains-config`.

For GCP, this should have the structure of `gcpkms://projects/<project>/locations/<location>/keyRings/<keyring>/cryptoKeys/<key>`
where `location`, `keyring`, and `key` are filled in appropriately.

For Vault, this should have the structure of `hashivault://<keyname>`, where the `keyname` is filled out appropriately.

For AWS, this should have the structure of `awskms://[ENDPOINT]/[ID/ALIAS/ARN]` (endpoint optional).

For Azure, this should have the structure of `azurekms://[VAULT_NAME][VAULT_URL]/[KEY_NAME]`.

## Authentication

Most likely, you will need to set up some additional authentication so that the `chains-controller` deployment has access to your KMS service for signing.
For GCP/GKE, we suggest enabling [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity), and giving your service account `Cloud KMS Admin` permissions.
Other Service Account techniques would work as well.

# Troubleshooting

If your signing secrets is already populated, you may get the following error:

```shell
Error from server (AlreadyExists): secrets "signing-secrets" already exists
```

Simply delete the secret and then recreate as described above:

```shell
kubectl delete secret signing-secrets -n tekton-chains 
```
