# SLSA Branding of Tekton Chains Provenance Format

`Tekton Chains` is migrating the naming of provenance formats from `intotoite6` to `slsa`.
As different versions of provenance formats are rolled out in the future, they will take 
the form `slsa/v1, slsa/v2` and the `package names` will be `v1`, `v2` and so on, respectively.

**Note**: the `slsa` branding in `Tekton Chains` does not map directly to the slsa predicate release.
Tekton chains `slsa/v1` is not the same as `slsav1.0`.

Shown below is the mapping between Tekton chains proveance and SLSA predicate.

|Tekton Chains Provenance Format version     | SLSA predicate | Notes |
|:------------------------------------------|---------------:|------:|
|**slsa/v1**| **slsa v0.2**  | same as currently supported `in-toto` format|
|**slsa/v2alpha1** [DEPRECATED]| **slsa v0.2**  | contains complete build instructions as in [TEP0122](https://github.com/tektoncd/community/pull/820). This is still a WIP and currently only available for taskrun level provenance. |
|**slsa/v2alpha2** [DEPRECATED]| **slsa v1.0**  | contains SLSAv1.0 predicate. The parameters are complete. Support still needs to be added for surfacing builder version and builder dependencies information.|
|**slsa/v2alpha3**| **slsa v1.0**  | contains SLSAv1.0 predicate. The parameters are complete. Support still needs to be added for surfacing builder version and builder dependencies information. Support for V1 Tekton Objects|
