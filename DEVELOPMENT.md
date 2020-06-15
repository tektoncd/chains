# Developing

## Getting started

1. Create [a GitHub account](https://github.com/join)
1. Setup
   [GitHub access via SSH](https://help.github.com/articles/connecting-to-github-with-ssh/)
1. [Create and checkout a repo fork](#checkout-your-fork)
1. Set up your [shell environment](#environment-setup)
1. Install [requirements](#requirements)
1. [Set up a Kubernetes cluster](#kubernetes-cluster)

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
cd cli
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

Docker for Desktop using an edge version has been proven to work for both
developing and running Pipelines. Your Kubernetes version must be 1.11 or later.

To setup a cluster with GKE:

1. [Install required tools and setup GCP project](https://github.com/knative/docs/blob/master/docs/install/Knative-with-GKE.md#before-you-begin)
   (You may find it useful to save the ID of the project in an environment
   variable (e.g. `PROJECT_ID`).
1. [Create a GKE cluster](https://github.com/knative/docs/blob/master/docs/install/Knative-with-GKE.md#creating-a-kubernetes-cluster)

Note that
[the `--scopes` argument to `gcloud container cluster create`](https://cloud.google.com/sdk/gcloud/reference/container/clusters/create#--scopes)
controls what GCP resources the cluster's default service account has access to;
for example to give the default service account full access to your GCR
registry, you can add `storage-full` to your `--scopes` arg.

## Environment Setup

To build the Tekton Chains project, you'll need to set `GO111MODULE=on`
environment variable to force `go` to use [go
modules](https://github.com/golang/go/wiki/Modules#quick-start).

## Iterating

Coming soon!
