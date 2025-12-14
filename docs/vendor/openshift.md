<!--

---
linkTitle: "OpenShift"
---
-->

# OpenShift

## Installation

Prerequisite: you'll need
[Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/main/docs/install.md#installing-tekton-pipelines-on-openshift)
installed on your OpenShift cluster before you install Chains.

To install Tekton Chains on OpenShift, you must first apply the `nonroot`
security context constraint to the `tekton-chains-controller` service account.
This is required to run the controller pod. See
[Security Context Constraints](https://docs.openshift.com/container-platform/4.3/authentication/managing-security-context-constraints.html)
for more information.

1. Log on as a user with `cluster-admin` privileges. The following example uses
   the default `system:admin` user:

   ```bash
   oc login -u system:admin
   ```

1. Set up the namespace (project) and configure the service account:

   ```bash
   oc new-project tekton-chains
   oc adm policy add-scc-to-user nonroot -z tekton-chains-controller
   ```

1. Install Tekton Chains:

   ```bash
   oc apply --filename https://infra.tekton.dev/tekton-releases/chains/latest/release.yaml
   ```

   See the
   [OpenShift CLI documentation](https://docs.openshift.com/container-platform/4.3/cli_reference/openshift_cli/getting-started-cli.html)
   for more information on the `oc` command.

1. Monitor the installation using the following command until all components
   show a `Running` status:

   ```bash
   oc get pods --namespace tekton-chains --watch
   ```
