# Tekton Chains Official Release Cheat Sheet

These steps provide a no-frills guide to performing an official release
of Tekton Chains. To follow these steps you'll need a checkout of
the chains repo, a terminal window and a text editor.

1. [Setup a context to connect to the dogfooding cluster](#setup-dogfooding-context) if you haven't already.

1. Install the [rekor CLI](https://docs.sigstore.dev/rekor/installation/) if you haven't already.

1. `cd` to root of Chains git checkout.

1. Select the commit you would like to build the release from (NOTE: the commit is full (40-digit) hash.)
    - Select the most recent commit on the ***main branch*** if you are cutting a major or minor release i.e. `x.0.0` or `0.x.0`
    - Select the most recent commit on the ***`release-<version number>x` branch***, e.g. [`release-v0.26.x`](https://github.com/tektoncd/chains/tree/release-v0.26.x) if you are patching a release i.e. `v0.26.2`.

1. Ensure the correct version of the release pipeline is installed on the cluster.
   To do that, the selected commit should be checked-out locally

    ```bash
    kubectl --context dogfooding apply -f release/release-pipeline.yaml
    ```

1. Create a `release.env` file with environment variables for bash scripts in later steps, and source it:

    ```bash
    cat <<EOF > release.env
    CHAINS_VERSION_TAG= # Example: v0.6.2
    CHAINS_RELEASE_GIT_SHA= # SHA of the release to be released, e.g. 5b082b1106753e093593d12152c82e1c4b0f37e5
    CHAINS_OLD_VERSION= # Example: v0.5.0
    CHAINS_PACKAGE=tektoncd/chains
    EOF
    . ./release.env
    ```

1. Confirm commit SHA matches what you want to release.

    ```bash
    git show $CHAINS_RELEASE_GIT_SHA
    ```

1. Create a workspace template file:

   ```bash
   WORKSPACE_TEMPLATE=$(mktemp /tmp/workspace-template.XXXXXX.yaml)
   cat <<'EOF' > $WORKSPACE_TEMPLATE
   spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi
   EOF
   ```

1. Execute the release pipeline (takes ~45 mins).
    
    **The minimum required tkn version is v0.30.0 or later**

    **If you are back-porting include this flag: `--param=releaseAsLatest="false"`**

   ```bash
    tkn --context dogfooding pipeline start chains-release \
      --param=gitRevision="${CHAINS_RELEASE_GIT_SHA}" \
      --param=versionTag="${CHAINS_VERSION_TAG}" \
      --param=serviceAccountImagesPath=credentials \
      --param=releaseBucket=tekton-releases \
      --param=releaseAsLatest="true" \
      --workspace name=release-secret,secret=oci-release-secret \
      --workspace name=release-images-secret,secret=ghcr-creds \
      --use-param-defaults \
      --workspace name=workarea,volumeClaimTemplateFile="${WORKSPACE_TEMPLATE}" \
      --tasks-timeout 2h \
      --pipeline-timeout 3h
   ```

    Accept the default values of the parameters (except for "releaseAsLatest" if backporting).

1. Watch logs of chains-release.

1. Once the pipeline is complete, check its results:

    ```bash
    tkn --context dogfooding pr describe <pipeline-run-name>

    (...)
    📝 Results

    NAME                    VALUE
    ∙ commit-sha            420adfcdf225326605f2b2c2264b42a2f7b86e4e
    ∙ release-file           https://infra.tekton.dev/tekton-releases/chains/previous/v0.13.0/release.yaml
    ∙ release-file-no-tag    https://infra.tekton.dev/tekton-releases/chains/previous/v0.13.0/release.notag.yaml

    (...)
    ```

    The `commit-sha` should match `$CHAINS_RELEASE_GIT_SHA`.
    The two URLs can be opened in the browser or via `curl` to download the release manifests.

1. The YAMLs are now released! Anyone installing Tekton Chains will get the new version. Time to create a new GitHub release announcement:

    1. Find the Rekor UUID for the release

    ```bash
    RELEASE_FILE=https://infra.tekton.dev/tekton-releases/chains/previous/${CHAINS_VERSION_TAG}/release.yaml
    CONTROLLER_IMAGE_SHA=$(curl -L $RELEASE_FILE | egrep 'ghcr.io.*controller' | cut -d'@' -f2)
    REKOR_UUID=$(rekor-cli search --sha $CONTROLLER_IMAGE_SHA | grep -v Found | head -1)
    echo -e "CONTROLLER_IMAGE_SHA: ${CONTROLLER_IMAGE_SHA}\nREKOR_UUID: ${REKOR_UUID}"
    ```

    1. Execute the Draft Release Pipeline.

        ```bash
        tkn --context dogfooding pipeline start \
          --workspace name=shared,volumeClaimTemplateFile="${WORKSPACE_TEMPLATE}" \
          --workspace name=credentials,secret=oci-release-secret \
          -p package="${CHAINS_PACKAGE}" \
          -p git-revision="$CHAINS_RELEASE_GIT_SHA" \
          -p release-tag="${CHAINS_VERSION_TAG}" \
          -p previous-release-tag="${CHAINS_OLD_VERSION}" \
          -p release-name="${CHAINS_VERSION_TAG}" \
          -p bucket="tekton-releases" \
          -p rekor-uuid="$REKOR_UUID" \
          release-draft
        ```

    1. Watch logs of resulting pipeline run on pipeline `release-draft`

    1. On successful completion, a URL will be logged. Visit that URL and look through the release notes.
      1. Manually add upgrade and deprecation notices based on the generated release notes
      1. Double-check that the list of commits here matches your expectations
         for the release. You might need to remove incorrect commits or copy/paste commits
         from the release branch. Refer to previous releases to confirm the expected format.

    1. Un-check the "This is a pre-release" checkbox since you're making a legit for-reals release!

    1. Publish the GitHub release once all notes are correct and in order.

1. Create a branch for the release named `release-<version number>x`, e.g. [`release-v0.26.x`](https://github.com/tektoncd/chains/tree/release-v0.26.x)
   and push it to the repo https://github.com/tektoncd/chains.
   (This can be done on the Github UI.)
   Make sure to fetch the commit specified in `CHAINS_RELEASE_GIT_SHA` to create the released branch.
   > Background: The reason why we need to create a branch for the release named `release-<version number>x` is for future patch releases. Cherrypicked PRs for the patch release will be merged to this branch. For example, [v0.26.0](https://github.com/tektoncd/chains/releases/tag/v0.26.0) has been already released, but later on we found that an important PR should have been included to that release. Therefore, we need to do a patch release i.e. v0.26.1 by cherrypicking this PR, which will trigger tekton-robot to create a new PR to merge the changes to the [release-v0.26.x branch](https://github.com/tektoncd/chains/tree/release-v0.26.x).

1. Edit `releases.md` on the `main` branch, add an entry for the release.
   - In case of a patch release, replace the latest release with the new one,
     including links to docs and examples. Append the new release to the list
     of patch releases as well.
   - In case of a minor or major release, add a new entry for the
     release, including links to docs and example
   - Check if any release is EOL, if so move it to the "End of Life Releases"
     section

1. Push & make PR for updated `releases.md`

1. Test release that you just made against your own cluster (note `--context my-dev-cluster`):

    ```bash
    # Test latest
    kubectl --context my-dev-cluster apply --filename https://infra.tekton.dev/tekton-releases/chains/latest/release.yaml
    ```

    ```bash
    # Test backport
    kubectl --context my-dev-cluster apply --filename https://infra.tekton.dev/tekton-releases/chains/previous/$CHAINS_VERSION_TAG/release.yaml
    ```

1. Announce the release in Slack channels #general, #chains and #announcements.

Congratulations, you're done!

## Setup dogfooding context

1. Configure `kubectl` to connect to
   [the dogfooding cluster](https://github.com/tektoncd/plumbing/blob/main/docs/dogfooding.md):

   The dogfooding cluster is currently an OKE cluster in oracle cloud. we need the Oracle Cloud CLI client. Install oracle cloud cli (https://docs.oracle.com/en-us/iaas/Content/API/SDKDocs/cliinstall.htm) 

    ```bash
    oci ce cluster create-kubeconfig --cluster-id <CLUSTER-OCID> --file $HOME/.kube/config --region <CLUSTER-REGION> --token-version 2.0.0  --kube-endpoint PUBLIC_ENDPOINT
    ```

1. Give [the context](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)
   a short memorable name such as `dogfooding`:

   ```bash
   kubectl config current-context
   ```
   get the context name and replace with current_context_name

   ```bash
   kubectl config rename-context <current_context_name> dogfooding
   ```

1. **Important: Switch `kubectl` back to your own cluster by default.**

    ```bash
    kubectl config use-context my-dev-cluster
    ```

## Cherry-picking commits for patch releases

The easiest way to cherry-pick a commit into a release branch is to use the "cherrypicker" plugin (see https://prow.tekton.dev/plugins for documentation).
To use the plugin, comment "/cherry-pick <branch-to-cherry-pick-onto>" on the pull request containing the commits that need to be cherry-picked.
Make sure this command is on its own line, and use one comment per branch that you're cherry-picking onto.
Automation will create a pull request cherry-picking the commits into the named branch, e.g. `release-v0.26.x`.

The cherrypicker plugin isn't able to resolve merge conflicts. If there are merge conflicts, you'll have to manually cherry-pick following these steps:
1. Fetch the branch you're backporting to and check it out:
```sh
git fetch upstream <branchname>
git checkout upstream/<branchname>
```
1. (Optional) Rename the local branch to make it easier to work with:
```sh
git switch -c <new-name-for-local-branch>
```
1. Find the 40-character commit hash to cherry-pick. Note: automation creates a new commit when merging contributors' commits into main.
You'll need to use the hash of the commit created by tekton-robot.

1. [Cherry-pick](https://git-scm.com/docs/git-cherry-pick) the commit onto the branch:
```sh
git cherry-pick <commit-hash>
```
1. Resolve any merge conflicts.
1. Finish the cherry-pick:
```sh
git add <changed-files>
git cherry-pick --continue
```
1. Push your changes to your fork and open a pull request against the upstream branch.
