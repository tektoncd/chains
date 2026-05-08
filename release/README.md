# Tekton Chains Official Release Cheat Sheet

These steps provide a no-frills guide to performing an official release
of Tekton Chains. Releases are now largely automated via
[Pipelines-as-Code](https://pipelinesascode.com) (PAC) on the `oci-ci-cd`
cluster.

## How releases work

### Initial releases (e.g. v0.27.0)

1. Create a release branch named `release-v<major>.<minor>.x` (e.g.
   `release-v0.27.x`) from the desired commit on `main`.
2. PAC automatically detects the branch creation and triggers the release
   pipeline defined in `.tekton/release.yaml`.
3. The version is derived from the branch name: `release-v0.27.x` → `v0.27.0`.
4. Monitor the PipelineRun on the
   [Tekton Dashboard](https://tekton.infra.tekton.dev/#/namespaces/releases-chains/pipelineruns).

### Patch releases (e.g. v0.26.4)

Patch releases happen in two ways:

- **Automatically**: A weekly cron (Thursday 10:00 UTC) in
  `.github/workflows/patch-release.yaml` scans all `release-v*` branches.
  If new commits exist since the last tag, it triggers a patch release via
  PAC incoming webhook.
- **Manually**: Run the "Patch Release" workflow from GitHub Actions
  (`workflow_dispatch`) with the branch and version as inputs.

Both methods trigger the release pipeline defined in `.tekton/release-patch.yaml`
on the `oci-ci-cd` cluster.

## Post-release steps

Once the release pipeline completes successfully:

1. Check the PipelineRun results on the
   [Tekton Dashboard](https://tekton.infra.tekton.dev/#/namespaces/releases-chains/pipelineruns):

    ```
    📝 Results

    NAME                    VALUE
    ∙ commit-sha            420adfcdf225326605f2b2c2264b42a2f7b86e4e
    ∙ release-file           https://infra.tekton.dev/tekton-releases/chains/previous/v0.26.4/release.yaml
    ∙ release-file-no-tag    https://infra.tekton.dev/tekton-releases/chains/previous/v0.26.4/release.notags.yaml
    ```

2. Create the GitHub release:

    1. Find the Rekor UUID for the release

        ```bash
        TEKTON_VERSION=v0.26.4  # set to your version
        TEKTON_RELEASE_GIT_SHA=<commit-sha-from-results>
        TEKTON_OLD_VERSION=v0.26.3  # previous release tag
        TEKTON_PACKAGE=tektoncd/chains
        TEKTON_REPO_NAME=chains

        RELEASE_FILE=https://infra.tekton.dev/tekton-releases/chains/previous/${TEKTON_VERSION}/release.yaml
        CONTROLLER_IMAGE_SHA=$(curl -L $RELEASE_FILE | egrep 'ghcr.io.*controller' | cut -d'@' -f2)
        REKOR_UUID=$(rekor-cli search --sha $CONTROLLER_IMAGE_SHA | grep -v Found | head -1)
        echo -e "CONTROLLER_IMAGE_SHA: ${CONTROLLER_IMAGE_SHA}\nREKOR_UUID: ${REKOR_UUID}"
        ```

    1. Execute the Draft Release Pipeline on the dogfooding cluster:

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

        POD_TEMPLATE=$(mktemp /tmp/pod-template.XXXXXX.yaml)
        cat <<'EOF' > $POD_TEMPLATE
        securityContext:
          fsGroup: 65532
          runAsUser: 65532
          runAsNonRoot: true
        EOF

        tkn pipeline start \
          --workspace name=shared,volumeClaimTemplateFile="${WORKSPACE_TEMPLATE}" \
          --workspace name=credentials,secret=oci-release-secret \
          --pod-template "${POD_TEMPLATE}" \
          -p package="${TEKTON_PACKAGE}" \
          -p git-revision="$TEKTON_RELEASE_GIT_SHA" \
          -p release-tag="${TEKTON_VERSION}" \
          -p previous-release-tag="${TEKTON_OLD_VERSION}" \
          -p repo-name="${TEKTON_REPO_NAME}" \
          -p bucket="tekton-releases" \
          -p rekor-uuid="$REKOR_UUID" \
          release-draft-oci
        ```

    1. On successful completion, visit the logged URL and review the release notes.
    1. Un-check "This is a pre-release" and publish the GitHub release.

3. Edit `releases.md` on the `main` branch, add an entry for the release.

4. Test the release against your own cluster:

    ```bash
    kubectl apply --filename https://infra.tekton.dev/tekton-releases/chains/previous/$TEKTON_VERSION/release.yaml
    ```

5. Announce the release in Slack channels #general, #chains and #announcements.

## Recovery

If a release PipelineRun fails:

- **Re-run via GitHub Checks**: Click "Re-Run" on the failed check in the
  PR/branch checks tab.
- **Re-run via GitOps**: Comment `/retest` on the commit or PR.
- **Incoming webhook**: Use the manual `workflow_dispatch` trigger in
  GitHub Actions to re-trigger the release.

## Manual release (fallback)

If automation is unavailable, you can still release manually. See
[the old release process](https://github.com/tektoncd/chains/blob/v0.26.3/release/README.md)
for the full manual steps using `tkn pipeline start`.

## Cherry-picking commits for patch releases

The easiest way to cherry-pick a commit into a release branch is to use the
"cherrypicker" plugin (see https://prow.tekton.dev/plugins for documentation).
Comment `/cherry-pick <branch-to-cherry-pick-onto>` on the pull request
containing the commits that need to be cherry-picked.

If there are merge conflicts, manually cherry-pick:

```sh
git fetch upstream <branchname>
git checkout upstream/<branchname>
git cherry-pick <commit-hash>
# resolve conflicts, then:
git add <changed-files>
git cherry-pick --continue
# push and open PR against upstream branch
```
