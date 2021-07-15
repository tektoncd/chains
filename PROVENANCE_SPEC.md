# Provenance Spec

## Model

This provenance model has the following properties:
* Each attestation has exaclty one `subject`, which is the same as an artifact or output of the build pipeline (e.g. OCI Images)
* `materials` defines the inputs to the build system (e.g. Git repos)
* `recipe` is the set of steps executed to build the `subject`, typically each container run by the TaskRun is a step in the `recipe`

In this model, there can be multiple `materials` and multiple steps in the `recipe`. 
However, there is exactly one `subject`.
If a build pipeline outputs multiple artifacts, each will have its own attestation.

![Diagram](images/provenance-diagram.png)


## Schema

```json
{
  "_type": "",
  "name": "",
  "predicateType": "",
  "subject": {
    "name": "",
    "digest": null
  },
  "predicate": {
    "invocation": {
      "parameters": null,
      "uri": "",
      "event_id": "",
      "id": "",
    },
    "recipe": {
      "steps": [
        {
          "entryPoint": "",
          "arguments": null,
          "environment": null,
          "annotations": null
        }
      ]
    },
    "metadata": {
      "buildStartedOn": null,
      "buildFinishedOn": null,
      "reproducible": false
    },
    "materials": [
      {
        "uri": "",
        "digest": null
      }
    ]
  }
}
```

## Fields

`subject`, required, `object`
This is the artifact the build pipeline has output.
For example, it could be an OCI Image, a binary artifact, or a package for a language package manager.

`subject.name`, required, `string`
The name of the artifact.

`subject.digest`, required, `DigestSet`
A set of cryptographic digests for the artifact, e.g. the `sha256` digest of an OCI image.

`invocation`, required, `object`
Desribes the event that kicked off the build.

`invocation.id`, required, `string`
A URI of the build system, e.g. "actions.github.com/myrepo"

`invocation.parameters`, optional, `object`
Parameters specified in the TaskRun and by the user.

`invocation.event_id`, optional, `string`
Opaque string, reserved for use by the builder to correlate with internal auditing or event mechanisms.

`invocation.recipe_uri`, optional, `string`
Desribes the location of the script that was used in the build.
This should be set if the build was created from a template or other type of file defined out of the build request itself.

For a TaskRun, this could be a PipelineRun, an OCI image, or reference a Task in-cluster, with the following prefixes:

* `oci://`
* `task://`
* `clusterTask://`
* `pipeline://`

`recipe`, required, object
Describes the steps taken to build the artifact.
For a TaskRun, typically each contaienr corresponds to one step in the `recipe`.

`recipe.steps.entrypoint`, optional, `string`
The entrypoint of the image. 
For a TaskRun, this could be the `command` or the `script` specified for the container.

`recipe.steps.arguments`, optional, `object`
Additional arguments passed to the `entrypoint`.

`recipe.steps.environment`, optional, `object`
Environment includes additional information about the step needed to execute it.
For a TaskRun, this includes:
* The image name, qualified by digest
* The container ID


`recipe.steps.annotations`, optional, `object`
TODO

`materials`, optional, `array of objects`
Describes the inputs used to build the `subject`.

`materials.uri`, optional, `string`
URI describing the material.

`materials.digest`, optional, `object`
Pins down the material to an immutable version (e.g. pins down a Git repo to a specific commit)


## Example

```json
  {
    "_type": "",
    "name": "build-distroless-sr6pn",
    "predicateType": "https://tekton.dev/chains/provenance",
    "subject": {
      "name": "gcr.io/foo/bar",
      "digest": {
        "sha256": "1ce00912e1f4df41a03704e9d1b0af569fa8f75e889505602be6424f3040011c"
      }
    },
    "predicate": {
      "invocation": {
        "parameters": null,
        "recipe_uri": "",
        "event_id": "0537b684-8463-4d9e-bd2c-08da6e3dae53",
        "builder.id": "tekton-chains"
      },
      "recipe": {
        "steps": [
          {
            "entryPoint": "",
            "arguments": null,
            "environment": {
              "container": "git-source-repo-jwqcl",
              "image": "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init@sha256:b963f6e7a69617db57b685893256f978436277094c21d43b153994acd8a01247"
            },
            "annotations": null
          },
          {
            "entryPoint": "#!/usr/bin/env bash\nset -x\n\ncd $(inputs.resources.repo.path)\n\nbazel build --host_force_python=PY2 //package_manager:dpkg_parser.par\ncp bazel-bin/package_manager/dpkg_parser.par .\n\nbazel build //base:static_root_amd64_debian10.tar\n\ncp bazel-bin/base/static_root_amd64_debian10.tar .\n\nfind /workspace/repo\npwd\n\necho \"gcr.io/foo/bar\" > $(results.IMAGE_URL.path)\n",
            "arguments": null,
            "environment": {
              "container": "build",
              "image": "gcr.io/cloud-marketplace-containers/google/bazel@sha256:010a1ecd1a8c3610f12039a25b823e3a17bd3e8ae455a53e340dcfdd37a49964"
            },
            "annotations": null
          },
          {
            "entryPoint": "sh",
            "arguments": [
              "-c",
              "set -x; cd /workspace/repo && crane digest --tarball=static_root_amd64_debian10.tar > $(results.IMAGE_DIGEST.path) && cat $(results.IMAGE_DIGEST.path)"
            ],
            "environment": {
              "container": "crane",
              "image": "gcr.io/go-containerregistry/crane@sha256:746291589a530c825103f606a7fbe7633ba65fe573f614fe2f115892ecac48ad"
            },
            "annotations": null
          }
        ]
      },
      "metadata": {
        "buildStartedOn": "2021-07-09T18:08:35Z",
        "buildFinishedOn": "2021-07-09T18:15:54Z",
        "reproducible": false
      },
      "materials": [
        {
          "uri": "https://github.com/GoogleContainerTools/distroless",
          "digest": {
            "revision": "50c56a48cfb3a5a80fa36ed91c739bdac8381cbe"
          }
        }
      ]
    }
  }
```
