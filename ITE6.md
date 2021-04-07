# ITE6 Attestator PoC

## Overview

The ite6 format can be enabled bo configuring Chains to use format
`in-toto-ite6`.

To provide a Git url/commit as material, add a parameter named
`CHAINS-GIT_COMMIT` and `CHAINS-GIT_URL`. The value of these
parameters should be fed by the clone task. Like this (from a `PipeLine`):

```
    - name: build
      params:
        - name: CHAINS-GIT_COMMIT
          value: "$(tasks.checkout.results.commit)"
        - name: CHAINS-GIT_URL
          value: "$(tasks.checkout.results.url)"
```

To capture arifacts created by a task, Chains will scan the TaskRun
result for a result name `*-DIGEST`. The result shall be a string on
the format `alg:digest`, alg is common `sha256`. If the result is named
`Foo-DIGEST`, Chains will try to find a parameter or result (in that
order) with the name `Foo`. The parameter or result named `Foo` shall
contain the name (reference) to the built artifact.

An example (a Task):
```
  params:
  - name: IMAGE
    description: Name (reference) of the image to build.
...
  results:
  - name: IMAGE-DIGEST
    description: Digest of the image just built.
```

So if the `IMAGE` param is
`863522791199.dkr.ecr.us-east-1.amazonaws.com/trebuchet/grafeas` and
the result `IMAGE-DIGEST` is `sha256:abcd` then an attestation for the
subject
`863522791199.dkr.ecr.us-east-1.amazonaws.com/trebuchet/grafeas@sha256:abcd`
is created.

## Limitations
This is far from complete, only a poc. To capture multiple subjects,
each subject must be defined as a separate result, no arrays are
currently supported.

## Fetching attestations
To fetch the attestations, a utility script `getatt.sh` can be found
in this repo. To run it, provide a TaskRun identifer, like this:

```
$ ./getatt.sh go-build-pipeline-edge-run-4gwnb-docker-zpbcr
{
  "attestation_type": "https://in-toto.io/Provenance/v1",
  "subject": {
    "863522791199.dkr.ecr.us-east-1.amazonaws.com/trebuchet/grafeas": {
      "sha256": "4d6a9dd4f98b7078864d4543fffd705a13b3432d9a30763e9fada05926d7eb4a"
    }
  },
  "materials": {
    "gcr.io/kaniko-project/executor": {
      "sha256": "68bb272f681f691254acfbdcef00962f22efe2f0c1e287e6a837b0abe07fb94b"
    },
    "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/imagedigestexporter": {
      "sha256": "c756458c29ef62cc1d660e6118057f2dae33926eefb6950b6a6a5455cdf35250"
    },
    "git+https://github.com/solarwinds/grafeasmirror": {
      "git_commit": "21c9f3b94b385a96f83ed9a5e2ef617aa9e1fbe8"
    },
    "stedolan/jq": {
      "sha256": "a61ed0bca213081b64be94c5e1b402ea58bc549f457c2682a86704dd55231e09"
    }
  },
  "builder": {
    "id": "go-build-pipeline-edge-run-4gwnb-docker-zpbcr-pod-7q5hj"
  },
  "recipe": {
    "type": "https://tekton.dev/attestations/chains@v1",
    "material": "git+https://github.com/solarwinds/grafeasmirror",
    "entry_point": "kaniko-edge"
  },
  "metadata": {
    "build_timestamp": "2021-04-01T06:52:04Z"
  }
}
```
