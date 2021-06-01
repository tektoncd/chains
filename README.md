# Tekton Chains
Supply Chain Security in Tekton Pipelines

<p align="center">
<img src="tekton_chains-color.png" alt="Tekton Chains logo"></img>
</p>


## Getting Started

Tekton Chains is a Kubernetes Custom Resource Definition (CRD) controller that allows you to manage your supply chain security in Tekton.

In its default mode of operation, Chains works by observing all `TaskRuns` executions in your cluster.
When `TaskRuns` complete, Chains takes a snapshot of them.
Chains then converts this snapshot to one or more standard payload formats, signs them and stores them somewhere.

Current features include:
* Signing `TaskRun` results with user provided cryptographic keys, including `TaskRun`s themselves and OCI Images
* Attestation formats like [intoto](docs/intoto.md)
* Signing with a variety of cryptograhic key types and services (PGP, x509, KMS)
* Support for multiple storage backends for signatures

### Installation
To install the latest version of Chains to your Kubernetes cluster, run:
```
kubectl apply --filename https://storage.googleapis.com/tekton-releases/chains/latest/release.yaml
```

To install a specific version of Chains, run:
```
kubectl apply -f https://storage.googleapis.com/tekton-releases/chains/previous/${VERSION}/release.yaml
```

To verify that installation was successful, wait until all Pods have Status `Running`:
```shell
$ kubectl get po -n tekton-chains --watch
NAME                                       READY   STATUS      RESTARTS   AGE
tekton-chains-controller-c4f7c57c4-nrjb2   1/1     Running     0          160m
```

### Setup
To finish setting up Chains, please complete the following steps:
* [Add authentication to the Chains controller](docs/authentication.md)
* [Generate a cryptographic key and configure Chains to use it for signing](docs/signing.md)
* [Set up any additional configuration](docs/config.md)


## Tutorial
To get started with Chains, try out our getting started [tutorial](docs/tutorial.md)!


## Want to contribute

We are so excited to have you!

* See [CONTRIBUTING.md](CONTRIBUTING.md) for an overview of our processes
* See [DEVELOPMENT.md](DEVELOPMENT.md) for how to get started
* See [ROADMAP.md](ROADMAP.md) for the current roadmap
Check out our good first issues and our help wanted issues to get started!
