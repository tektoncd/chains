# Tekton Chains
Supply Chain Security in Tekton Pipelines

## Getting Started

Tekton Chains is currently experimental, and does not have any published releases.
To use Tekton Chains, you'll have to build and deploy your own release from git.

### Installation

See [DEVELOPMENT.md](DEVELOPMENT.md) for a guide on how to build and deploy your own version.

### Configuration

#### Signing Secrets

To get started signing things with Chains, you first have to generate a GPG keypair
to be used by your Tekton Chains system.
There are many ways to go about this, but you can usually use something like this:

```shell
gpg --gen-key
```

Enter a passprase (make sure you remember it!) and a name for the key.

Next, you'll need to upload the private key as a Kubernetes `Secret` so Tekton can use it
to sign payloads.
To do that, export the public and private keys as files:

```shell
gpg --export-secret-key --armor $keyname > pgp.private-key
gpg --export --armor $keyname > pgp.public-key
```

And save the passphrase (if you set one) in it's own file:

```shell
echo -n $passphrase > pgp.passphrase
```

Then set all of these as fields in the Kubernetes secret `signing-secrets`:

```shell
kubectl create secret generic signing-secrets -n tekton-pipelines --from-file=pgp.passphrase --from-file=pgp.private-key --from-file=pgp.public-key
```

---
**NOTE**

If you're signing secrets is already populated, you may get the following error:

```shell
Error from server (AlreadyExists): secrets "signing-secrets" already exists
```

Simply prepend a delete:

```shell
kubectl delete secret signing-secrets -n tekton-pipelines && kubectl create secret generic signing-secrets -n tekton-pipelines --from-file=pgp.passphrase --from-file=pgp.private-key --from-file=pgp.public-key
```
---

## Usage

In its default mode of operation, Chains works by observing all `TaskRuns` in your cluster.
When `TaskRuns` complete, Chains takes a snapshot of them.
Chains then converts this snapshot to one or more standard payload formats, signs them and stores them somewhere.

To try it out, create a `TaskRun` of your choice.
For a sample one, try this:

```shell
$ kubectl create -f https://raw.githubusercontent.com/tektoncd/pipeline/master/examples/v1beta1/taskruns/home-is-set.yaml
taskrun.tekton.dev/home-is-set-rwhzs created
```

Then, take the name of the `TaskRun` you just created, and wait for it to finish (SUCCEEEDED should be True).

```shell
$ kubectl get taskrun.tekton.dev/taskrun home-is-set-rwhzs
NAME                SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
home-is-set-rwhzs   True        Succeeded   105s        100s
```

Next, retrieve the signature and payload from the object (they are stored as base64-encoded annotations):

```shell
$ kubectl get taskrun home-is-set-rwhzs -o=json | jq  -r '.metadata.annotations["chains.tekton.dev/payload"]' | base64 -D > payload
$ kubectl get taskrun home-is-set-rwhzs -o=json | jq  -r '.metadata.annotations["chains.tekton.dev/signature"]' | base64 -D > signature
```

---
**NOTE**
`base64` on Linux distributions is typically a lower -d flag, so `base64 -d`
---
Finally, we can check the signature:

```shell
 gpg --verify signature payload
gpg: Signature made Mon Jul 27 14:42:01 2020 CDT
gpg:                using RSA key 4FCFD41D993B806C
gpg: checking the trustdb
gpg: marginals needed: 3  completes needed: 1  trust model: pgp
gpg: depth: 0  valid:   4  signed:   0  trust: 0-, 0q, 0n, 0m, 0f, 4u
gpg: next trustdb check due at 2022-03-16
gpg: Good signature from "Tekton Unit Tests <testing@tekton.dev>" [ultimate]
```

Now we have a verifiable record of the `TaskRuns`!


## Want to contribute

We are so excited to have you!

See [CONTRIBUTING.md](CONTRIBUTING.md) for an overview of our processes
See [DEVELOPMENT.md](DEVELOPMENT.md) for how to get started
See [ROADMAP.md](ROADMAP.md) for the current roadmap
Look at our good first issues and our help wanted issues

