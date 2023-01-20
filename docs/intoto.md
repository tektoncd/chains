<!--
---
linkTitle: "In-toto Attestation Formatter"
weight: 40
---
-->

# In-toto attestation formatter

## Overview

The in-toto attestation spec is defined [here](https://github.com/in-toto/attestation/tree/v0.1.0/spec).

In-toto attestations can be generated for `TaskRuns` or `PipelineRuns`.
Tekton Chains generates in-toto attestations with the `slsa-provenance` predicate [format](https://slsa.dev/provenance/v0.2).

### Standard in-toto predicate

The in-toto format can be enabled by running:

```
# For TaskRuns
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"artifacts.taskrun.format": "slsa/v1"}}'
# For PipelineRuns
kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"artifacts.pipelinerun.format": "slsa/v1"}}'
```
**Note**, you can continue to use the older alias of `slsa/v1`: `in-toto`.

To provide a git URL/commit as material, add a parameter named
`CHAINS-GIT_COMMIT` and `CHAINS-GIT_URL`. The value of these
parameters should be fed by some VCS task (e.g like this
[task](https://github.com/tektoncd/catalog/blob/main/task/git-clone/0.3/git-clone.yaml#L81)).
A `PipeLine` example where another task `checkout` has URL/commit as
task results:

```
    - name: build
      params:
        - name: CHAINS-GIT_COMMIT
          value: "$(tasks.checkout.results.commit)"
        - name: CHAINS-GIT_URL
          value: "$(tasks.checkout.results.url)"
```

Alternatively, `CHAINS-GIT_COMMIT` and `CHAINS-GIT_URL` can be results instead.
Another `Pipeline` example where results are used:

```
spec:
  results:
    - description: Repository URL used for buiding the image.
      name: CHAINS-GIT_URL
      value: $(tasks.checkout.results.url)
    - description: Repository commit used for building the image.
      name: CHAINS-GIT_COMMIT
      value: $(tasks.checkout.results.commit)
  tasks:
  - name: checkout

```

### Type Hinting

To capture artifacts created by a task, Chains will scan the `TaskRun`
and `PipelineRun` result for a result name `*_DIGEST`. The result shall
be a string on the format `alg:digest`, alg is common `sha256`. If the
result is named `Foo_DIGEST`, Chains will try to find a parameter or
result (in that order) with the name `Foo`. The parameter or result named
`Foo` shall contain the name (reference) to the built artifact.

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

### Limitations
This is an MVP implementation of the in-toto attestation
format. More work would be required to properly capture the
`Entrypoint` field in the provenance predicate, now the `TaskRef`'s name
is used. Also metadata related to hermeticity/reproducibility are
currently not populated.

### Examples

Example `TaskRun` attestation:

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

Example `PipelineRun` attestation:

```
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "https://slsa.dev/provenance/v0.2",
  "subject": [
    {
      "name": "registry.example.com/minimal-container/min",
      "digest": {
        "sha256": "41a8ace7b880ae40708daa60387d2f181c41ecec667c93010294d1529d58c27e"
      }
    }
  ],
  "predicate": {
    "builder": {
      "id": "https://tekton.dev/chains/v2"
    },
    "buildType": "https://tekton.dev/attestations/chains/pipelinerun@v2",
    "invocation": {
      "configSource": {},
      "parameters": {
        "git-repo": "https://github.com/lcarva/minimal-container",
        "git-revision": "main",
        "output-image": "registry.example.com/minimal-container/min:latest"
      }
    },
    "buildConfig": {
      "tasks": [
        {
          "name": "git-clone",
          "ref": {
            "name": "git-clone",
            "kind": "Task"
          },
          "startedOn": "2022-08-29T18:42:04Z",
          "finishedOn": "2022-08-29T18:42:23Z",
          "status": "Succeeded",
          "steps": [
            {
              "entryPoint": "#!/usr/bin/env sh\nset -eu\n\nif [ \"${PARAM_VERBOSE}\" = \"true\" ] ; then\n  set -x\nfi\n\n\nif [ \"${WORKSPACE_BASIC_AUTH_DIRECTORY_BOUND}\" = \"true\" ] ; then\n  cp \"${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.git-credentials\" \"${PARAM_USER_HOME}/.git-credentials\"\n  cp \"${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.gitconfig\" \"${PARAM_USER_HOME}/.gitconfig\"\n  chmod 400 \"${PARAM_USER_HOME}/.git-credentials\"\n  chmod 400 \"${PARAM_USER_HOME}/.gitconfig\"\nfi\n\nif [ \"${WORKSPACE_SSH_DIRECTORY_BOUND}\" = \"true\" ] ; then\n  cp -R \"${WORKSPACE_SSH_DIRECTORY_PATH}\" \"${PARAM_USER_HOME}\"/.ssh\n  chmod 700 \"${PARAM_USER_HOME}\"/.ssh\n  chmod -R 400 \"${PARAM_USER_HOME}\"/.ssh/*\nfi\n\nif [ \"${WORKSPACE_SSL_CA_DIRECTORY_BOUND}\" = \"true\" ] ; then\n   export GIT_SSL_CAPATH=\"${WORKSPACE_SSL_CA_DIRECTORY_PATH}\"\nfi\nCHECKOUT_DIR=\"${WORKSPACE_OUTPUT_PATH}/${PARAM_SUBDIRECTORY}\"\n\ncleandir() {\n  # Delete any existing contents of the repo directory if it exists.\n  #\n  # We don't just \"rm -rf ${CHECKOUT_DIR}\" because ${CHECKOUT_DIR} might be \"/\"\n  # or the root of a mounted volume.\n  if [ -d \"${CHECKOUT_DIR}\" ] ; then\n    # Delete non-hidden files and directories\n    rm -rf \"${CHECKOUT_DIR:?}\"/*\n    # Delete files and directories starting with . but excluding ..\n    rm -rf \"${CHECKOUT_DIR}\"/.[!.]*\n    # Delete files and directories starting with .. plus any other character\n    rm -rf \"${CHECKOUT_DIR}\"/..?*\n  fi\n}\n\nif [ \"${PARAM_DELETE_EXISTING}\" = \"true\" ] ; then\n  cleandir\nfi\n\ntest -z \"${PARAM_HTTP_PROXY}\" || export HTTP_PROXY=\"${PARAM_HTTP_PROXY}\"\ntest -z \"${PARAM_HTTPS_PROXY}\" || export HTTPS_PROXY=\"${PARAM_HTTPS_PROXY}\"\ntest -z \"${PARAM_NO_PROXY}\" || export NO_PROXY=\"${PARAM_NO_PROXY}\"\n\n/ko-app/git-init \\\n  -url=\"${PARAM_URL}\" \\\n  -revision=\"${PARAM_REVISION}\" \\\n  -refspec=\"${PARAM_REFSPEC}\" \\\n  -path=\"${CHECKOUT_DIR}\" \\\n  -sslVerify=\"${PARAM_SSL_VERIFY}\" \\\n  -submodules=\"${PARAM_SUBMODULES}\" \\\n  -depth=\"${PARAM_DEPTH}\" \\\n  -sparseCheckoutDirectories=\"${PARAM_SPARSE_CHECKOUT_DIRECTORIES}\"\ncd \"${CHECKOUT_DIR}\"\nRESULT_SHA=\"$(git rev-parse HEAD)\"\nEXIT_CODE=\"$?\"\nif [ \"${EXIT_CODE}\" != 0 ] ; then\n  exit \"${EXIT_CODE}\"\nfi\nprintf \"%s\" \"${RESULT_SHA}\" > \"$(results.commit.path)\"\nprintf \"%s\" \"${PARAM_URL}\" > \"$(results.url.path)\"\n",
              "arguments": null,
              "environment": {
                "container": "clone",
                "image": "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:45dca0972541546d3625d99ee8a8fbcc768b01fc9c199d1251ebd7dfd1b8874c"
              },
              "annotations": null
            }
          ],
          "invocation": {
            "configSource": {},
            "parameters": {
              "deleteExisting": "true",
              "depth": "1",
              "gitInitImage": "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.29.0",
              "httpProxy": "",
              "httpsProxy": "",
              "noProxy": "",
              "refspec": "",
              "revision": "main",
              "sparseCheckoutDirectories": "",
              "sslVerify": "true",
              "subdirectory": "",
              "submodules": "true",
              "url": "https://github.com/lcarva/minimal-container",
              "userHome": "/tekton/home",
              "verbose": "true"
            }
          },
          "results": [
            {
              "name": "commit",
              "value": "89dedecaca1b85346600c7db9939a4fe090a42ef"
            },
            {
              "name": "url",
              "value": "https://github.com/lcarva/minimal-container"
            }
          ]
        },
        {
          "name": "source-security-scan",
          "after": [
            "git-clone"
          ],
          "ref": {
            "name": "trivy-scanner",
            "kind": "Task",
            "bundle": "gcr.io/tekton-releases/catalog/upstream/trivy-scanner:0.1"
          },
          "startedOn": "2022-08-29T18:42:24Z",
          "finishedOn": "2022-08-29T18:42:40Z",
          "status": "Succeeded",
          "steps": [
            {
              "entryPoint": "#!/usr/bin/env sh\n  cmd=\"trivy $* $(params.IMAGE_PATH)\"\n  echo \"Running trivy task with command below\"\n  echo \"$cmd\"\n  eval \"$cmd\"\n",
              "arguments": [
                "$(params.ARGS)"
              ],
              "environment": {
                "container": "trivy-scan",
                "image": "docker.io/aquasec/trivy@sha256:dea76d4b50c75125cada676a87ac23de2b7ba4374752c6f908253c3b839201d9"
              },
              "annotations": null
            }
          ],
          "invocation": {
            "configSource": {},
            "parameters": {
              "ARGS": [
                "filesystem"
              ],
              "IMAGE_PATH": ".",
              "TRIVY_IMAGE": "docker.io/aquasec/trivy@sha256:dea76d4b50c75125cada676a87ac23de2b7ba4374752c6f908253c3b839201d9"
            }
          }
        },
        {
          "name": "image-build",
          "after": [
            "source-security-scan"
          ],
          "ref": {
            "name": "buildah",
            "kind": "ClusterTask"
          },
          "startedOn": "2022-08-29T18:42:41Z",
          "finishedOn": "2022-08-29T18:43:03Z",
          "status": "Succeeded",
          "steps": [
            {
              "entryPoint": "[[ \"$(workspaces.sslcertdir.bound)\" == \"true\" ]] && CERT_DIR_FLAG=\"--cert-dir $(workspaces.sslcertdir.path)\"\nbuildah ${CERT_DIR_FLAG} --storage-driver=$(params.STORAGE_DRIVER) bud \\\n  $(params.BUILD_EXTRA_ARGS) --format=$(params.FORMAT) \\\n  --tls-verify=$(params.TLSVERIFY) --no-cache \\\n  -f $(params.DOCKERFILE) -t $(params.IMAGE) $(params.CONTEXT)\n",
              "arguments": null,
              "environment": {
                "container": "build",
                "image": "quay.io/buildah/stable@sha256:0ceadda5ead6601f347a801c935e668888a72ff858ef0c7b826aca10273f9a77"
              },
              "annotations": null
            },
            {
              "entryPoint": "[[ \"$(params.SKIP_PUSH)\" == \"true\" ]] && echo \"Push skipped\" && exit 0\n[[ \"$(workspaces.sslcertdir.bound)\" == \"true\" ]] && CERT_DIR_FLAG=\"--cert-dir $(workspaces.sslcertdir.path)\"\nbuildah ${CERT_DIR_FLAG} --storage-driver=$(params.STORAGE_DRIVER) push \\\n  $(params.PUSH_EXTRA_ARGS) --tls-verify=$(params.TLSVERIFY) \\\n  --digestfile $(workspaces.source.path)/image-digest $(params.IMAGE) \\\n  docker://$(params.IMAGE)\n",
              "arguments": null,
              "environment": {
                "container": "push",
                "image": "quay.io/buildah/stable@sha256:0ceadda5ead6601f347a801c935e668888a72ff858ef0c7b826aca10273f9a77"
              },
              "annotations": null
            },
            {
              "entryPoint": "cat \"$(workspaces.source.path)\"/image-digest | tee $(results.IMAGE_DIGEST.path)\necho \"$(params.IMAGE)\" | tee $(results.IMAGE_URL.path)\n",
              "arguments": null,
              "environment": {
                "container": "digest-to-results",
                "image": "quay.io/buildah/stable@sha256:0ceadda5ead6601f347a801c935e668888a72ff858ef0c7b826aca10273f9a77"
              },
              "annotations": null
            }
          ],
          "invocation": {
            "configSource": {},
            "parameters": {
              "BUILDER_IMAGE": "quay.io/buildah/stable:v1.18.0",
              "BUILD_EXTRA_ARGS": "",
              "CONTEXT": ".",
              "DOCKERFILE": "./Dockerfile",
              "FORMAT": "oci",
              "IMAGE": "registry.example.com/minimal-container/min:latest",
              "PUSH_EXTRA_ARGS": "",
              "SKIP_PUSH": "false",
              "STORAGE_DRIVER": "vfs",
              "TLSVERIFY": "true"
            }
          },
          "results": [
            {
              "name": "IMAGE_DIGEST",
              "value": "sha256:41a8ace7b880ae40708daa60387d2f181c41ecec667c93010294d1529d58c27e"
            },
            {
              "name": "IMAGE_URL",
              "value": "registry.example.com/minimal-container/min:latest\n"
            }
          ]
        },
        {
          "name": "image-security-scan",
          "after": [
            "image-build"
          ],
          "ref": {
            "name": "trivy-scanner",
            "kind": "Task",
            "bundle": "gcr.io/tekton-releases/catalog/upstream/trivy-scanner@sha256:e4c2916f25ce2d42ec7016c3dc3392e527442c307f43aae3ea63f4622ee5cfe4"
          },
          "startedOn": "2022-08-29T18:43:03Z",
          "finishedOn": "2022-08-29T18:43:14Z",
          "status": "Succeeded",
          "steps": [
            {
              "entryPoint": "#!/usr/bin/env sh\n  cmd=\"trivy $* $(params.IMAGE_PATH)\"\n  echo \"Running trivy task with command below\"\n  echo \"$cmd\"\n  eval \"$cmd\"\n",
              "arguments": [
                "$(params.ARGS)"
              ],
              "environment": {
                "container": "trivy-scan",
                "image": "docker.io/aquasec/trivy@sha256:dea76d4b50c75125cada676a87ac23de2b7ba4374752c6f908253c3b839201d9"
              },
              "annotations": null
            }
          ],
          "invocation": {
            "configSource": {},
            "parameters": {
              "ARGS": [
                "image"
              ],
              "IMAGE_PATH": "registry.example.com/minimal-container/min:latest\n",
              "TRIVY_IMAGE": "docker.io/aquasec/trivy@sha256:dea76d4b50c75125cada676a87ac23de2b7ba4374752c6f908253c3b839201d9"
            }
          }
        }
      ]
    },
    "metadata": {
      "buildStartedOn": "2022-08-29T18:42:04Z",
      "buildFinishedOn": "2022-08-29T18:43:14Z",
      "completeness": {
        "parameters": false,
        "environment": false,
        "materials": false
      },
      "reproducible": false
    },
    "materials": [
      {
        "uri": "git+https://github.com/lcarva/minimal-container.git",
        "digest": {
          "sha1": "89dedecaca1b85346600c7db9939a4fe090a42ef"
        }
      }
    ]
  }
}

```

### Structured Result Type Hinting

**_A new feature will be implemented to have better support for artifact provenance retrieval. More details can be found in [Tekton Pipelines](https://github.com/tektoncd/pipeline/issues/5455)._**

The feature requires **Tekton Pipeline v0.38** or later.

To capture artifacts created by a task in a structured manner, Tekton Chains integrated with structured results and retrieve artifacts' provenances.
The result should be in the following format in a Task:
``` yaml
results:
  - name: {artifact_name}-ARTIFACT_INPUTS
    description: Digest of the image just built.
    type: object
    properties:
      uri:
        type: string
      digest:
        type:string
  - name: {artifact_name}-ARTIFACT_OUTPUTS
    description: Digest of the image just built.
    type: object
    properties:
      uri:
        type: string
      digest:
        type:string
```
Suffix `-ARTIFACT_INPUTS` will retrieve the artifact provenance and put them in [Intoto Materials](https://github.com/in-toto/attestation/blob/v0.1.0/spec/predicates/provenance.md#fields), and `-ARTIFACT_OUTPUTS` will retrieve the artifact provenance and put them in [Intoto Subjects](https://github.com/in-toto/attestation/tree/v0.1.0/spec#statement).

`uri` is the unique identifier for this artifact, and `digest` needs to be a string on the format `alg:digest`.

An example structured result in a TaskRun:
``` yaml
results:
        - name: img_1-ARTIFACT_INPUTS
           value:
               uri: gcr.io/foo/bar
               digest: sha123@89dedecaca1b85346600c7db9939a4fe090a42ef
        - name: mvn1_pkg-ARTIFACT_OUTPUTS
           value:
               uri: maven-test-0.0.1.jar
               digest: sha256@89dedecaca1b85346600c7db9939a4fe090a42ee
        - name: mvn1_pom-ARTIFACT_OUTPUTS
           value:
               uri: maven-test-0.0.1.pom
               digest: sha256@89dedecaca1b85346600c7db9939a4fe090a42eg
        - name: mvn1_src-ARTIFACT_OUTPUTS
           value:
               uri: maven-test-0.0.1-sources.jar
               digest: sha256@89dedecaca1b85346600c7db9939a4fe090a42ez
```

