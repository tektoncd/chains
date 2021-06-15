# Chains Tutorial
This tutorial will guide you through:
* Generating your own keypair and storing it as a Kubernetes Secret
* Creating a sample TaskRun
* Retrieving the signature and payload from the signed TaskRun
* Verifying the signature

We'll be creating a `TaskRun`, signing it, and storing the signature and the payload as annotations on the `TaskRun` itself.
So, no additional authentication should be required!

You can opt to try the tutorial with either of the following key types:
* [x509 (Default)](#x509)
* [PGP/GPG](#PGP/GPG)

## x509
To generate your own encrypted x509 keypair and save it as a Kubernetes secret, install [cosign](https://github.com/sigstore/cosign) and run the following:

```shell
cosign generate-key-pair -k8s tekton-chains/signing-secrets
```
cosign will prompt you for a password, which will be stored in the Kubernetes secret.

To create a simple `TaskRun`, run:

```shell
$ kubectl create -f examples/taskruns/task-output-image.yaml
taskrun.tekton.dev/build-push-run-output-image-qbjvh created
```

Save the name of your `TaskRun` as an environment variable:
```shell
$ export TASKRUN=<Name of your TaskRun> # Replace with your taskrun name
```

Then, take the name of the `TaskRun` you just created, and wait for it to finish (SUCCEEEDED should be True).

```shell
$ kubectl get taskrun $TASKRUN
NAME                                SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
build-push-run-output-image-qbjvh   True        Succeeded   36m         36m    
```

Next, retrieve the signature and payload from the object (they are stored as base64-encoded annotations):
```shell
$ export TASKRUN_UID=$(kubectl get taskrun $TASKRUN -o=json | jq -r '.metadata.uid')
$ kubectl get taskrun $TASKRUN -o=json | jq  -r ".metadata.annotations[\"chains.tekton.dev/payload-taskrun-$TASKRUN_UID\"]" | base64 --decode > payload
$ kubectl get taskrun $TASKRUN -o=json | jq  -r ".metadata.annotations[\"chains.tekton.dev/signature-taskrun-$TASKRUN_UID\"]" | base64 --decode > signature
```

Finally, we can check the signature with [cosign](https://github.com/sigstore/cosign):

```shell
$ cosign verify-blob -key cosign.pub -signature ./signature ./payload 
Verified OK
```

Now we have a verifiable record of the `TaskRun`!

## PGP/GPG
First, generate a keypair and store it as a Kubernetes secret, as described in the [GPG/PGP signing doc](signing.md#GPG/PGP).

Then, configure Chains to sign with `gpg`:
```
$ kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"artifacts.taskrun.signer": "pgp"}}'
configmap/chains-config patched
```

To create a simple `TaskRun`, run:

```shell
$ kubectl create -f examples/taskruns/task-output-image.yaml
taskrun.tekton.dev/build-push-run-output-image-qbjvh created
```

Save the name of your `TaskRun` as an environment variable:
```shell
$ export TASKRUN=<Name of your TaskRun> # Replace with your taskrun name
```

Then, take the name of the `TaskRun` you just created, and wait for it to finish (SUCCEEEDED should be True).

```shell
$ kubectl get taskrun $TASKRUN
NAME                                SUCCEEDED   REASON      STARTTIME   COMPLETIONTIME
build-push-run-output-image-qbjvh   True        Succeeded   36m         36m    
```

Next, retrieve the signature and payload from the object (they are stored as base64-encoded annotations):
```shell
$ export TASKRUN_UID=$(kubectl get taskrun $TASKRUN -o=json | jq -r '.metadata.uid')
$ kubectl get taskrun $TASKRUN -o=json | jq  -r ".metadata.annotations[\"chains.tekton.dev/payload-taskrun-$TASKRUN_UID\"]" | base64 --decode > payload
$ kubectl get taskrun $TASKRUN -o=json | jq  -r ".metadata.annotations[\"chains.tekton.dev/signature-taskrun-$TASKRUN_UID\"]" | base64 --decode > signature
```

Finally, we can check the signature with [gpg](https://gnupg.org/):

```shell
$ gpg --verify signature payload
gpg: Signature made Mon Jul 27 14:42:01 2020 CDT
gpg:                using RSA key 4FCFD41D993B806C
gpg: checking the trustdb
gpg: marginals needed: 3  completes needed: 1  trust model: pgp
gpg: depth: 0  valid:   4  signed:   0  trust: 0-, 0q, 0n, 0m, 0f, 4u
gpg: next trustdb check due at 2022-03-16
gpg: Good signature from "Tekton Unit Tests <testing@tekton.dev>" [ultimate]
```

Now we have a verifiable record of the `TaskRun`!
