# In-toto attestation formater

## Overview

In-toto attestation spec is defined
[here](https://github.com/in-toto/attestation/tree/v0.1.0/spec).

The in-toto format can be enabled by configuring Chains to use format
`in-toto-ite6`.

To provide a git URL/commit as material, add a parameter named
`CHAINS-GIT_COMMIT` and `CHAINS-GIT_URL`. The value of these
parameters should be fed by some VCS task (e.g like this
[task](https://github.com/tektoncd/catalog/blob/main/task/git-clone/0.3/git-clone.yaml#L81).
A `PipeLine` example where another task `checkout` have URL/commit as
task results:

```
    - name: build
      params:
        - name: CHAINS-GIT_COMMIT
          value: "$(tasks.checkout.results.commit)"
        - name: CHAINS-GIT_URL
          value: "$(tasks.checkout.results.url)"
```

### Type Hinting

To capture arifacts created by a task, Chains will scan the TaskRun
result for a result name `*_DIGEST`. The result shall be a string on
the format `alg:digest`, alg is common `sha256`. If the result is named
`Foo_DIGEST`, Chains will try to find a parameter or result (in that
order) with the name `Foo`. The parameter or result named `Foo` shall
contain the name (reference) to the built artifact.

An example (a Task):
```
  params:
  - name: IMAGE
    description: Name (reference) of the image to build.
...
  results:
  - name: IMAGE_DIGEST
    description: Digest of the image just built.
```

So if the `IMAGE` parameter have the value `gcr.io/test/foo` and the
result `IMAGE_DIGEST` is `sha256:abcd` then an attestation for the
subject `pkg:/docker/test/foo@sha256:abcd?repository_url=gcr.io`
is created. Note that image references are represented using [Package
URL](https://github.com/package-url/purl-spec) format.

## Limitations
This is an MVP implementation of the the in-toto attestation
format. More work would be required to properly capture the
`Entrypoint` field in the provenance predicate, now the `TaskRef`'s name
is used. Also metadata related to hermeticity/reproducible are
currently not populated.

## Examples

Example attestation:

```
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "https://in-toto.io/Provenance/v0.1",
  "subject": [
    {
      "name": "file-SNAPSHOT.jar",
      "digest": {
        "sha256": "3f1017b520fe358d7b3796879232cd36259066ccd5bab5466cbedb444064dfed"
      }
    }
  ],
  "predicate": {
    "builder": {
      "id": "https://configured.builder@v1"
    },
    "recipe": {
      "type": "https://tekton.dev/attestations/chains@v1",
      "definedInMaterial": 0,
      "entryPoint": "maven"
    },
    "metadata": {
      "buildStartedOn": "2021-05-11T11:05:50Z",
      "buildFinishedOn": "2021-05-11T11:15:42Z",
      "completeness": {
        "arguments": false,
        "environment": false,
        "materials": false
      },
      "reproducible": false
    },
    "materials": [
      {
        "uri": "git+https://github.com/org/repo",
        "digest": {
          "git_commit": "c4b75d454655c1755ab116947e88a59ac03e28a9"
        }
      },
      {
        "uri": "pkg:docker/alpine@sha256:69e70a79f2d41ab5d637de98c1e0b055206ba40a8145e7bddb55ccc04e13cf8f",
        "digest": {
          "sha256": "69e70a79f2d41ab5d637de98c1e0b055206ba40a8145e7bddb55ccc04e13cf8f"
        }
      },
      {
        "uri": "pkg:docker/alpine@sha256:69e70a79f2d41ab5d637de98c1e0b055206ba40a8145e7bddb55ccc04e13cf8f",
        "digest": {
          "sha256": "69e70a79f2d41ab5d637de98c1e0b055206ba40a8145e7bddb55ccc04e13cf8f"
        }
      },
      {
        "uri": "pkg:docker/org/build/openjdk-11@sha256:51aa63475b5e1e2e22d1dc416556a14658a7e03a0d3c88bb9dd7b6e3411ae34a?repository_url=gcr.io",
        "digest": {
          "sha256": "51aa63475b5e1e2e22d1dc416556a14658a7e03a0d3c88bb9dd7b6e3411ae34a"
        }
      }
    ]
  }
}
```
