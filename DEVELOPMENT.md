# Developing

## Getting started

1. Create [a GitHub account](https://github.com/join)
1. Setup
   [GitHub access via SSH](https://help.github.com/articles/connecting-to-github-with-ssh/)
1. [Create and checkout a repo fork](#checkout-your-fork)
1. Set up your [shell environment](#environment-setup)
1. Install [requirements](#requirements)
1. [Set up a Kubernetes cluster](#kubernetes-cluster)
1. [Running Tests](#Running-Tests)

Then you can [iterate](#iterating).

### Checkout your fork

The Go tools require that you clone the repository to the
`src/github.com/tektoncd/chains` directory in your
[`GOPATH`](https://github.com/golang/go/wiki/SettingGOPATH).

To check out this repository:

1. Create your own
   [fork of this repo](https://help.github.com/articles/fork-a-repo/)
1. Clone it to your machine:

```shell
mkdir -p ${GOPATH}/src/github.com/tektoncd
cd ${GOPATH}/src/github.com/tektoncd
git clone git@github.com:${YOUR_GITHUB_USERNAME}/chains.git
cd chains
git remote add upstream git@github.com:tektoncd/chains.git
git remote set-url --push upstream no_push
```

_Adding the `upstream` remote sets you up nicely for regularly
[syncing your fork](https://help.github.com/articles/syncing-a-fork/)._

### Requirements

You must install these tools:

1. [`go`](https://golang.org/doc/install): The language Tekton
   Chains is built in
1. [`git`](https://help.github.com/articles/set-up-git/): For source control
1. [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
   (optional): For interacting with your kube cluster

## Kubernetes cluster

To setup a Kubernetes cluster for development, see the Tekton Pipelines [documentation](https://github.com/tektoncd/pipeline/blob/master/DEVELOPMENT.md#kubernetes-cluster).

## Environment Setup

To build the Tekton Chains project, you'll need to set `GO111MODULE=on`
environment variable to force `go` to use [go
modules](https://github.com/golang/go/wiki/Modules#quick-start).

## Iterating

## Install Chains

You can stand up a version of this controller on-cluster (to your `kubectl
config current-context`):

```shell
ko apply -f config/
```

### Redeploy controller

As you make changes to the code, you can redeploy your controller with:

```shell
ko apply -f config/100-deployment.yaml
```

### Tear it down

You can clean up everything with:

```shell
ko delete -f config/
```

## Accessing logs

To look at the controller logs, run:

```shell
kubectl -n tekton-pipelines logs deployment/tekton-chains-controller
```

## Running Tests

Chains uses the standard go testing framework.
Unit tests can be run with:

```shell
go test ./...
```

Integration tests require a running cluster and Chains to be installed.
These are protected by a build tag "e2e".
To run integration tests:

```shell
go test -v -count=1 -tags=e2e -timeout=20m  ./test
```

Some of these tests require access to GCP resources.
To run those, set the name of the GCP project you are running in:

```shell
GCP_PROJECT_ID=dlorenc-vmtest2 go test -v -count=1 -tags=e2e -timeout=20m  ./test
```
