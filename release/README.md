# Tekton Chains Official Release Cheat Sheet

Follow these steps to perform an official release of Tekton Chains! To follow
these steps you'll need a checkout of the chains repo, a terminal window and a
text editor.

1. [Setup a context to connect to the dogfooding cluster](#setup-dogfooding-context)
   if you haven't already.

1. `cd` to the root of the chains repo

1. Apply release Tekton resources

   - [`publish-chains-release`](release-pipeline.yaml) - This task uses
     [ko](https://github.com/google/ko) to build all container images we release
     and generate the `release.yaml`

     ```sh
     kubectl apply -f release/publish.yaml
     ```

   - [`chains-release`](release-pipeline.yaml) - This is the pipeline that
     stitches everything together.

     ```sh
     kubectl apply -f release/release-pipeline.yaml
     ```

1. Select the commit you would like to build the release from, most likely the
   most recent commit at https://github.com/tektoncd/chains/commits/main and
   note the commit's hash.

1. Create environment variables for bash scripts in later steps.

   ```bash
   CHAINS_VERSION_TAG=# UPDATE THIS. Example: v0.6.2
   CHAINS_RELEASE_GIT_SHA=# SHA of the release to be released
   ```

1. Confirm commit SHA matches what you want to release.

   ```bash
   git show $CHAINS_RELEASE_GIT_SHA
   ```

1. Create a workspace template file:

   ```bash
     cat <<EOF > workspace-template.yaml
     spec:
       accessModes:
       - ReadWriteOnce
       resources:
         requests:
           storage: 1Gi
   EOF
   ```

1. Execute the release pipeline.

   ```bash
   tkn --context dogfooding pipeline start chains-release \
     --param=gitRevision="${CHAINS_RELEASE_GIT_SHA}" \
     --param=versionTag="${CHAINS_VERSION_TAG}" \
     --param=serviceAccountPath=release.json \
     --param=serviceAccountImagesPath=credentials \
     --param=releaseBucket=gs://tekton-releases/chains \
     --workspace name=release-secret,secret=release-secret \
     --workspace name=release-images-secret,secret=ghcr-creds \
     --use-param-defaults \
     --workspace name=workarea,volumeClaimTemplateFile=workspace-template.yaml
   ```

1. Watch logs of chains-release.

1. Once the pipeline run is complete, check its results:

   ```bash
   tkn --context dogfooding pr describe <pipeline-run-name>

   (...)
   📝 Results

   NAME                    VALUE
   commit-sha                 420adfcdf225326605f2b2c2264b42a2f7b86e4e
   release-file               https://storage.googleapis.com/tekton-releases/chains/previous/v0.13.0/release.yaml
   release-file-no-tag        https://storage.googleapis.com/tekton-releases/chains/previous/v0.13.0/release.notag.yaml

   (...)
   ```

   The `commit-sha` should match `$CHAINS_RELEASE_GIT_SHA`. The two URLs can be
   opened in the browser or via `curl` to download the release manifests.

   1. The YAMLs are now released! Anyone installing Tekton Chains will now get
      the new version. Time to create a new GitHub release announcement:

   1. Find the Rekor UUID for the release

      ```bash
      RELEASE_FILE=https://storage.googleapis.com/tekton-releases/chains/previous/${CHAINS_VERSION_TAG}/release.yaml
      CONTROLLER_IMAGE_SHA=$(curl $RELEASE_FILE | egrep 'ghcr.io.*controller' | cut -d'@' -f2)
      REKOR_UUID=$(rekor-cli search --sha $CONTROLLER_IMAGE_SHA | grep -v Found | head -1)
      echo -e "CONTROLLER_IMAGE_SHA: ${CONTROLLER_IMAGE_SHA}\nREKOR_UUID: ${REKOR_UUID}"
      ```

   1. Create additional environment variables

      ```bash
      CHAINS_OLD_VERSION=# Example: v0.11.1
      CHAINS_RELEASE_NAME=$CHAINS_VERSION_TAG
      CHAINS_PACKAGE=tektoncd/chains
      ```

   1. Execute the Draft Release task.

      ```bash
      tkn --context dogfooding pipeline start \
        --workspace name=shared,volumeClaimTemplateFile=workspace-template.yaml \
        --workspace name=credentials,secret=release-secret \
        -p package="${CHAINS_PACKAGE}" \
        -p git-revision="$CHAINS_RELEASE_GIT_SHA" \
        -p release-tag="${CHAINS_VERSION_TAG}" \
        -p previous-release-tag="${CHAINS_OLD_VERSION}" \
        -p release-name="${CHAINS_RELEASE_NAME}" \
        -p bucket="gs://tekton-releases/chains" \
        -p rekor-uuid="$REKOR_UUID" \
        release-draft
      ```

   1. Watch logs of create-draft-release

   1. On successful completion, a URL will be logged. Visit that URL and look
      through the release notes.

      1. Manually add upgrade and deprecation notices based on the generated
         release notes
      1. Double-check that the list of commits here matches your expectations
         for the release. You might need to remove incorrect commits or
         copy/paste commits from the release branch. Refer to previous releases
         to confirm the expected format.

   1. Un-check the "This is a pre-release" checkbox since you're making a legit
      for-reals release!

   1. Publish the GitHub release once all notes are correct and in order.

1. Create a branch for the release named `release-<version number>x`, e.g.
   `release-v0.28.x` and push it to the repo https://github.com/tektoncd/chains.
   Make sure to fetch the commit specified in `$CHAINS_RELEASE_GIT_SHA` to
   create the released branch.

1. Test release that you just made against your own cluster (note
   `--context my-dev-cluster`):

   ```bash
   # Test latest
   kubectl --context my-dev-cluster apply --filename https://storage.googleapis.com/tekton-releases/chains/latest/release.yaml
   ```

   ```bash
   # Test backport
   kubectl --context my-dev-cluster apply --filename https://storage.googleapis.com/tekton-releases/chains/previous/$CHAINS_VERSION_TAG/release.yaml
   ```

1. Update releases page at
   [releases/.md](https://github.com/tektoncd/chains/blob/main/releases.md)

1. Announce the release in Slack channels #general, #chains and #announcements.

Congratulations, you're done!

## Setup dogfooding context

1. Configure `kubectl` to connect to
   [the dogfooding cluster](https://github.com/tektoncd/plumbing/blob/main/docs/dogfooding.md):

   ```bash
   gcloud container clusters get-credentials dogfooding --zone us-central1-a --project tekton-releases
   ```

1. Give
   [the context](https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters/)
   a short memorable name such as `dogfooding`:

   ```bash
   kubectl config rename-context gke_tekton-releases_us-central1-a_dogfooding dogfooding
   ```

## Important: Switch `kubectl` back to your own cluster by default.

```bash
    kubectl config use-context my-dev-cluster
```
