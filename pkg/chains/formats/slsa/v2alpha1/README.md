# Provenance Format Example for slsa/v2alpha1
When running the following taskrun with bundle resolver referencing the [remote task](https://github.com/tektoncd/catalog/tree/main/task/git-clone/0.9):

```yaml
apiVersion: tekton.dev/v1
kind: TaskRun
metadata:
  generateName: bundles-resolver-
spec:
  workspaces:
    - name: output
      emptyDir: {}
  podTemplate:
    securityContext:
      fsGroup: 65532
  taskRef:
    resolver: bundles
    params:
      - name: bundle
        value: gcr.io/tekton-releases/catalog/upstream/git-clone:0.9
      - name: name
        value: git-clone
      - name: kind
        value: task
  params:
    - name: url
      value: https://github.com/kelseyhightower/nocode
    - name: revision
      value: master
```

The following output was generated. Notice the following below:
1. `ConfigSource` is correctly populated.
2. `Invocation.Parameters` contains `workspaces and podTemplate` from above spec correctly populated.
3. `Invocation.Environment` contains `tekton-pipelines-feature-flags` to indicate which feature flags were enabled during the taskrun.
4. `BuildConfig` contains the resolved `taskSpec`.

```json
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "https://slsa.dev/provenance/v0.2",
  "subject": null,
  "predicate": {
    "builder": {
      "id": "https://tekton.dev/chains/v2"
    },
    "buildType": "https://chains.tekton.dev/format/slsa/v2alpha1/type/tekton.dev/v1/TaskRun",
    "invocation": {
      "configSource": {
        "uri": "gcr.io/tekton-releases/catalog/upstream/git-clone",
        "digest": {
          "sha256": "1ee0f45892faf152aa631b0745f192e353c684408286bc55e9041edfaf22cb5e"
        },
        "entryPoint": "git-clone"
      },
      "parameters": {
        "ComputeResources": null,
        "Debug": null,
        "Params": [
          {
            "name": "url",
            "value": "https://github.com/kelseyhightower/nocode"
          },
          {
            "name": "revision",
            "value": "master"
          }
        ],
        "PodTemplate": {
          "securityContext": {
            "fsGroup": 65532
          }
        },
        "Resources": null,
        "Retries": 0,
        "ServiceAccountName": "default",
        "SidecarOverrides": null,
        "Status": "",
        "StatusMessage": "",
        "StepOverrides": null,
        "Timeout": "1h0m0s",
        "Workspaces": [
          {
            "name": "output",
            "emptyDir": {}
          }
        ]
      },
      "environment": {
        "tekton-pipelines-feature-flags": {
          "DisableAffinityAssistant": false,
          "DisableCredsInit": false,
          "RunningInEnvWithInjectedSidecars": true,
          "RequireGitSSHSecretKnownHosts": false,
          "EnableTektonOCIBundles": true,
          "ScopeWhenExpressionsToTask": false,
          "EnableAPIFields": "alpha",
          "SendCloudEventsForRuns": false,
          "AwaitSidecarReadiness": true,
          "EnforceNonfalsifiability": "",
          "ResourceVerificationMode": "skip",
          "EnableProvenanceInStatus": true,
          "ResultExtractionMethod": "termination-message",
          "MaxResultSize": 4096,
          "CustomTaskVersion": "v1"
        }
      }
    },
    "buildConfig": {
      "taskSpec": {
        "params": [
          {
            "name": "url",
            "type": "string",
            "description": "Repository URL to clone from."
          },
          {
            "name": "revision",
            "type": "string",
            "description": "Revision to checkout. (branch, tag, sha, ref, etc...)",
            "default": ""
          },
          {
            "name": "refspec",
            "type": "string",
            "description": "Refspec to fetch before checking out revision.",
            "default": ""
          },
          {
            "name": "submodules",
            "type": "string",
            "description": "Initialize and fetch git submodules.",
            "default": "true"
          },
          {
            "name": "depth",
            "type": "string",
            "description": "Perform a shallow clone, fetching only the most recent N commits.",
            "default": "1"
          },
          {
            "name": "sslVerify",
            "type": "string",
            "description": "Set the `http.sslVerify` global git config. Setting this to `false` is not advised unless you are sure that you trust your git remote.",
            "default": "true"
          },
          {
            "name": "crtFileName",
            "type": "string",
            "description": "file name of mounted crt using ssl-ca-directory workspace. default value is ca-bundle.crt.",
            "default": "ca-bundle.crt"
          },
          {
            "name": "subdirectory",
            "type": "string",
            "description": "Subdirectory inside the `output` Workspace to clone the repo into.",
            "default": ""
          },
          {
            "name": "sparseCheckoutDirectories",
            "type": "string",
            "description": "Define the directory patterns to match or exclude when performing a sparse checkout.",
            "default": ""
          },
          {
            "name": "deleteExisting",
            "type": "string",
            "description": "Clean out the contents of the destination directory if it already exists before cloning.",
            "default": "true"
          },
          {
            "name": "httpProxy",
            "type": "string",
            "description": "HTTP proxy server for non-SSL requests.",
            "default": ""
          },
          {
            "name": "httpsProxy",
            "type": "string",
            "description": "HTTPS proxy server for SSL requests.",
            "default": ""
          },
          {
            "name": "noProxy",
            "type": "string",
            "description": "Opt out of proxying HTTP/HTTPS requests.",
            "default": ""
          },
          {
            "name": "verbose",
            "type": "string",
            "description": "Log the commands that are executed during `git-clone`'s operation.",
            "default": "true"
          },
          {
            "name": "gitInitImage",
            "type": "string",
            "description": "The image providing the git-init binary that this Task runs.",
            "default": "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.40.2"
          },
          {
            "name": "userHome",
            "type": "string",
            "description": "Absolute path to the user's home directory.\n",
            "default": "/home/git"
          }
        ],
        "description": "These Tasks are Git tasks to work with repositories used by other tasks in your Pipeline.\nThe git-clone Task will clone a repo from the provided url into the output Workspace. By default the repo will be cloned into the root of your Workspace. You can clone into a subdirectory by setting this Task's subdirectory param. This Task also supports sparse checkouts. To perform a sparse checkout, pass a list of comma separated directory patterns to this Task's sparseCheckoutDirectories param.",
        "steps": [
          {
            "name": "clone",
            "image": "gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init:v0.40.2",
            "env": [
              {
                "name": "HOME",
                "value": "/home/git"
              },
              {
                "name": "PARAM_URL",
                "value": "https://github.com/kelseyhightower/nocode"
              },
              {
                "name": "PARAM_REVISION",
                "value": "master"
              },
              {
                "name": "PARAM_REFSPEC"
              },
              {
                "name": "PARAM_SUBMODULES",
                "value": "true"
              },
              {
                "name": "PARAM_DEPTH",
                "value": "1"
              },
              {
                "name": "PARAM_SSL_VERIFY",
                "value": "true"
              },
              {
                "name": "PARAM_CRT_FILENAME",
                "value": "ca-bundle.crt"
              },
              {
                "name": "PARAM_SUBDIRECTORY"
              },
              {
                "name": "PARAM_DELETE_EXISTING",
                "value": "true"
              },
              {
                "name": "PARAM_HTTP_PROXY"
              },
              {
                "name": "PARAM_HTTPS_PROXY"
              },
              {
                "name": "PARAM_NO_PROXY"
              },
              {
                "name": "PARAM_VERBOSE",
                "value": "true"
              },
              {
                "name": "PARAM_SPARSE_CHECKOUT_DIRECTORIES"
              },
              {
                "name": "PARAM_USER_HOME",
                "value": "/home/git"
              },
              {
                "name": "WORKSPACE_OUTPUT_PATH",
                "value": "/workspace/output"
              },
              {
                "name": "WORKSPACE_SSH_DIRECTORY_BOUND",
                "value": "false"
              },
              {
                "name": "WORKSPACE_SSH_DIRECTORY_PATH"
              },
              {
                "name": "WORKSPACE_BASIC_AUTH_DIRECTORY_BOUND",
                "value": "false"
              },
              {
                "name": "WORKSPACE_BASIC_AUTH_DIRECTORY_PATH"
              },
              {
                "name": "WORKSPACE_SSL_CA_DIRECTORY_BOUND",
                "value": "false"
              },
              {
                "name": "WORKSPACE_SSL_CA_DIRECTORY_PATH"
              }
            ],
            "resources": {},
            "securityContext": {
              "runAsUser": 65532,
              "runAsNonRoot": true
            },
            "script": "#!/usr/bin/env sh\nset -eu\n\nif [ \"${PARAM_VERBOSE}\" = \"true\" ] ; then\n  set -x\nfi\n\nif [ \"${WORKSPACE_BASIC_AUTH_DIRECTORY_BOUND}\" = \"true\" ] ; then\n  cp \"${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.git-credentials\" \"${PARAM_USER_HOME}/.git-credentials\"\n  cp \"${WORKSPACE_BASIC_AUTH_DIRECTORY_PATH}/.gitconfig\" \"${PARAM_USER_HOME}/.gitconfig\"\n  chmod 400 \"${PARAM_USER_HOME}/.git-credentials\"\n  chmod 400 \"${PARAM_USER_HOME}/.gitconfig\"\nfi\n\nif [ \"${WORKSPACE_SSH_DIRECTORY_BOUND}\" = \"true\" ] ; then\n  cp -R \"${WORKSPACE_SSH_DIRECTORY_PATH}\" \"${PARAM_USER_HOME}\"/.ssh\n  chmod 700 \"${PARAM_USER_HOME}\"/.ssh\n  chmod -R 400 \"${PARAM_USER_HOME}\"/.ssh/*\nfi\n\nif [ \"${WORKSPACE_SSL_CA_DIRECTORY_BOUND}\" = \"true\" ] ; then\n   export GIT_SSL_CAPATH=\"${WORKSPACE_SSL_CA_DIRECTORY_PATH}\"\n   if [ \"${PARAM_CRT_FILENAME}\" != \"\" ] ; then\n      export GIT_SSL_CAINFO=\"${WORKSPACE_SSL_CA_DIRECTORY_PATH}/${PARAM_CRT_FILENAME}\"\n   fi\nfi\nCHECKOUT_DIR=\"${WORKSPACE_OUTPUT_PATH}/${PARAM_SUBDIRECTORY}\"\n\ncleandir() {\n  # Delete any existing contents of the repo directory if it exists.\n  #\n  # We don't just \"rm -rf ${CHECKOUT_DIR}\" because ${CHECKOUT_DIR} might be \"/\"\n  # or the root of a mounted volume.\n  if [ -d \"${CHECKOUT_DIR}\" ] ; then\n    # Delete non-hidden files and directories\n    rm -rf \"${CHECKOUT_DIR:?}\"/*\n    # Delete files and directories starting with . but excluding ..\n    rm -rf \"${CHECKOUT_DIR}\"/.[!.]*\n    # Delete files and directories starting with .. plus any other character\n    rm -rf \"${CHECKOUT_DIR}\"/..?*\n  fi\n}\n\nif [ \"${PARAM_DELETE_EXISTING}\" = \"true\" ] ; then\n  cleandir || true\nfi\n\ntest -z \"${PARAM_HTTP_PROXY}\" || export HTTP_PROXY=\"${PARAM_HTTP_PROXY}\"\ntest -z \"${PARAM_HTTPS_PROXY}\" || export HTTPS_PROXY=\"${PARAM_HTTPS_PROXY}\"\ntest -z \"${PARAM_NO_PROXY}\" || export NO_PROXY=\"${PARAM_NO_PROXY}\"\n\ngit config --global --add safe.directory \"${WORKSPACE_OUTPUT_PATH}\"\n/ko-app/git-init \\\n  -url=\"${PARAM_URL}\" \\\n  -revision=\"${PARAM_REVISION}\" \\\n  -refspec=\"${PARAM_REFSPEC}\" \\\n  -path=\"${CHECKOUT_DIR}\" \\\n  -sslVerify=\"${PARAM_SSL_VERIFY}\" \\\n  -submodules=\"${PARAM_SUBMODULES}\" \\\n  -depth=\"${PARAM_DEPTH}\" \\\n  -sparseCheckoutDirectories=\"${PARAM_SPARSE_CHECKOUT_DIRECTORIES}\"\ncd \"${CHECKOUT_DIR}\"\nRESULT_SHA=\"$(git rev-parse HEAD)\"\nEXIT_CODE=\"$?\"\nif [ \"${EXIT_CODE}\" != 0 ] ; then\n  exit \"${EXIT_CODE}\"\nfi\nRESULT_COMMITTER_DATE=\"$(git log -1 --pretty=%ct)\"\nprintf \"%s\" \"${RESULT_COMMITTER_DATE}\" > \"/tekton/results/committer-date\"\nprintf \"%s\" \"${RESULT_SHA}\" > \"/tekton/results/commit\"\nprintf \"%s\" \"${PARAM_URL}\" > \"/tekton/results/url\"\n"
          }
        ],
        "workspaces": [
          {
            "name": "output",
            "description": "The git repo will be cloned onto the volume backing this Workspace."
          },
          {
            "name": "ssh-directory",
            "description": "A .ssh directory with private key, known_hosts, config, etc. Copied to\nthe user's home before git commands are executed. Used to authenticate\nwith the git remote when performing the clone. Binding a Secret to this\nWorkspace is strongly recommended over other volume types.\n",
            "optional": true
          },
          {
            "name": "basic-auth",
            "description": "A Workspace containing a .gitconfig and .git-credentials file. These\nwill be copied to the user's home before any git commands are run. Any\nother files in this Workspace are ignored. It is strongly recommended\nto use ssh-directory over basic-auth whenever possible and to bind a\nSecret to this Workspace over other volume types.\n",
            "optional": true
          },
          {
            "name": "ssl-ca-directory",
            "description": "A workspace containing CA certificates, this will be used by Git to\nverify the peer with when fetching or pushing over HTTPS.\n",
            "optional": true
          }
        ],
        "results": [
          {
            "name": "commit",
            "type": "string",
            "description": "The precise commit SHA that was fetched by this Task."
          },
          {
            "name": "url",
            "type": "string",
            "description": "The precise URL that was fetched by this Task."
          },
          {
            "name": "committer-date",
            "type": "string",
            "description": "The epoch timestamp of the commit that was fetched by this Task."
          }
        ]
      },
      "taskRunResults": [
        {
          "name": "commit",
          "type": "string",
          "value": "6c073b08f7987018cbb2cb9a5747c84913b3608e"
        },
        {
          "name": "committer-date",
          "type": "string",
          "value": "1579634710"
        },
        {
          "name": "url",
          "type": "string",
          "value": "https://github.com/kelseyhightower/nocode"
        }
      ]
    },
    "metadata": {
      "buildStartedOn": "2023-02-24T03:34:21Z",
      "buildFinishedOn": "2023-02-24T03:34:27Z",
      "completeness": {
        "parameters": false,
        "environment": false,
        "materials": false
      },
      "reproducible": false
    },
    "materials": [
      {
        "uri": "oci://gcr.io/tekton-releases/github.com/tektoncd/pipeline/cmd/git-init",
        "digest": {
          "sha256": "28ff94e63e4058afc3f15b4c11c08cf3b54fa91faa646a4bbac90380cd7158df"
        }
      }
    ]
  }
}
```
