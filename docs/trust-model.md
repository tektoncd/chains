<!--
---
linkTitle: "Trust Model"
weight: 15
---
-->

# Trust Model

This doc describes what Tekton Chains trusts, what a signed attestation from
Chains therefore proves, and which permissions an operator has to keep tight for
that statement to stay true.

## Chains is a post-execution observer

Chains does not run the build and does not observe it while it happens. It
watches `TaskRun` and `PipelineRun` objects, and when one is complete it reads
the object's status and signs what it finds: the inputs, the outputs, the task
spec and the results. It has no independent source of truth about what the build
actually executed.

So the trust boundary is the object itself, not the process behind it. Whatever
wrote a `TaskRun`'s status is what Chains signs. In a standard installation that
is the Tekton Pipelines controller, which is why Chains can be described as
trusting the Pipelines controller, and through it the cluster's own RBAC.

## `taskruns/status` is the boundary that matters

Anyone who can update `taskruns/status` (or `pipelineruns/status`) can change
what Chains signs: the results that become provenance subjects, the task spec
that becomes the build definition, the start and completion times.

Chains' own controller is granted that permission, to the
`tekton-chains-controller` service account through the
`tekton-chains-controller-cluster-access` ClusterRole in
[`config/100-rolebinding.yaml`](../config/100-rolebinding.yaml). Chains itself
only patches finalizers and labels on the objects it reconciles, but the role
carries the status verbs, so anyone who can assume that service account can also
write them:

```yaml
- apiGroups: ["tekton.dev"]
  resources: ["tasks/status", "clustertasks/status", "taskruns/status", "pipelines/status", "pipelineruns/status", "pipelineresources/status", "runs/status"]
  verbs: ["get", "list", "create", "update", "delete", "patch", "watch"]
```

In a standard Tekton Pipelines install the Pipelines controller holds the same
permission.
Any principal you add to that set — a custom controller, an operator's own
reconciler, a `ClusterRoleBinding` written for an integration — becomes part of
the trust boundary as well. Review it the way you would review write access to
the signing key, because that is what it is in practice.

On a cluster where several teams share the control plane, prefer:

- no role other than the Pipelines and Chains controllers with `update` on
  `taskruns/status` and `pipelineruns/status`;
- Task and Pipeline definitions owned by the platform team, or a curated catalog,
  rather than accepted from tenants one at a time;
- a reviewed list of who may create `TaskRun`s in the namespaces Chains watches
  (see [Namespaces Restrictions](config.md#namespaces-restrictions-in-chains-controller)).

## Results are user-controlled

Task results such as `IMAGE_URL` and `IMAGE_DIGEST` are whatever the Task wrote
them to be. Chains uses them to decide which artifact the attestation is about,
so a Task that reports the wrong digest produces a correctly signed attestation
about the wrong image.

If Task authorship is untrusted, the provenance is only as good as the catalog it
came from. That is why the type-hinting tasks in the Tekton catalog report the
url and revision they actually resolved at build time
(see [SLSA Provenance](slsa-provenance.md)).

## Where the signed output goes

`storage.oci.repository` decides where attestations are stored (see
[Chains Configuration](config.md#oci-configuration)). When it is unset, signatures and
attestations are stored alongside the image they refer to, in the same repository
the build pushed to. In a multi-tenant cluster that means the tenant's own
credentials decide where its provenance lands, and the tenant can push to that
location directly.

For multi-tenant deployments, set `storage.oci.repository` to a repository that
the build's credentials cannot write to and that the verification path trusts,
and keep its credentials separate from the build's. Treat the value as
security-critical configuration, not as a convenience default.

One exception: with `storage.oci.encoding-format: sigstore-bundle` the override is
ignored, because a referrer has to live in the repository of the image it points
at. In that mode the tenant's own image repository is where the attestation
lands, whatever `storage.oci.repository` says. See
[Things to keep in mind in `sigstore-bundle` mode](oci-encoding-format.md#things-to-keep-in-mind-in-sigstore-bundle-mode).

## What the provenance proves

Chains produces SLSA provenance describing what the build platform observed: who
ran which Task or Pipeline, with which inputs, producing which outputs, at what
time, and it signs that statement with a key the platform controls.

That is the kind of evidence [SLSA Build L1–L2](https://slsa.dev/spec/v1.0/levels)
rests on: the build platform attests to the build, and a verifier that trusts the
platform's key inherits that trust. It is not end-to-end content verification. Provenance does not prove that the build was
hermetic, that its dependencies were the ones the lockfile names, that no
compromised step ran inside the build, or that the image is safe to deploy.
Those are properties of the pipeline, not of the attestation, and the verifier
has to establish them separately — for example with a policy that pins the
allowed builders, source repositories and entry points.

## Checklist for a multi-tenant deployment

- `taskruns/status` and `pipelineruns/status` write access limited to the
  Pipelines and Chains controllers.
- Task and Pipeline definitions from a trusted catalog, not from tenants.
- `storage.oci.repository` set, with credentials the build cannot write with.
- Verification policy pinning the builder identity, the source repository and
  the expected entry point, in addition to the signature itself.
- Whoever consumes the attestations must know which key they trust, and where
  that key is kept.
