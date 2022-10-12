# Tekton Chains Releases

## Release Frequency

Tekton Chains follows the Tekton community [release policy][release-policy]
as follows:

- Versions are numbered according to semantic versioning: `vX.Y.Z`
- At a minimum four LTS release are produced. Additional releases are
  produced based on availability of new features to be released
- Four releases a year are chosen for [long term support (LTS)](https://github.com/tektoncd/community/blob/main/releases.md#support-policy).
  All remaining releases are supported for approximately 1 month.
    - LTS releases take place in January, April, July and October every year
    - The first Tekton Chains LTS release will be **v0.13.0** in October 2022

Tekton Chains produces nightly builds, publicly available on
`gcr.io/tekton-nightly`. 

### Transition Process

Before release v0.13 Tekton Chains has worked on the basis of an undocumented
support period, providing patch releases when needed. While transitioning to
the new support model, v0.11 and v0.12 will be supported for four months from
the initial publishing date.

## Release Process

Tekton Chains releases are made of YAML manifests and container images.
Manifests are published to cloud object-storage as well as
[GitHub][tekton-chains-releases]. Container images are signed by
[Sigstore][sigstore] via [Tekton Chains][tekton-chains]; signatures can be
verified through the [public key][chains-public-key] hosted by the Tekton Chains
project.

Further documentation available:

- The Tekton Chains [release process][tekton-releases-docs]
- [Installing Tekton][tekton-installation]
- Standard for [release notes][release-notes-standards]

## Releases

### v0.12

- **Latest Release**: [v0.12.0][v0-12-0] (2022-08-31) ([docs][v0-12-0-docs], [examples][v0-12-0-examples])
- **Initial Release**: [v0.12.0][v0-12-0] (2022-08-31)
- **End of Life**: 2022-12-30
- **Patch Releases**: [v0.12.0][v0-12-0]

### v0.11

- **Latest Release**: [v0.11.0][v0-11-0] (2022-07-26) ([docs][v0-11-0-docs], [examples][v0-11-0-examples])
- **Initial Release**: [v0.11.0][v0-11-0] (2022-07-26)
- **End of Life**: 2022-12-17
- **Patch Releases**: [v0.11.0][v0-11-0]

## End of Life Releases

Older releases are EOL and available on [GitHub][tekton-pipeline-releases].


[release-policy]: https://github.com/tektoncd/community/blob/main/releases.md
[sigstore]: https://sigstore.dev
[tekton-chains]: https://github.com/tektoncd/chains
[tekton-chains-releases]: https://github.com/tektoncd/chains/releases
[chains-public-key]: https://github.com/tektoncd/chains/blob/main/tekton.pub
[tekton-releases-docs]: release/README.md
[tekton-installation]: README.md#installation
[release-notes-standards]:
    https://github.com/tektoncd/community/blob/main/standards.md#release-notes

[v0-12-2]: https://github.com/tektoncd/chains/releases/tag/v0.12.2
[v0-12-1]: https://github.com/tektoncd/chains/releases/tag/v0.12.1
[v0-12-0]: https://github.com/tektoncd/chains/releases/tag/v0.12.0
[v0-11-0]: https://github.com/tektoncd/chains/releases/tag/v0.11.0

[v0-12-2-docs]: https://github.com/tektoncd/chains/tree/v0.12.2/docs#tekton-pipelines
[v0-11-0-docs]: https://github.com/tektoncd/chains/tree/v0.11.0/docs#tekton-pipelines

[v0-12-2-examples]: https://github.com/tektoncd/chains/tree/v0.12.2/examples#examples
[v0-11-0-examples]: https://github.com/tektoncd/chains/tree/v0.11.0/examples#examples
[v0-38-4-examples]: https://github.com/tektoncd/chains/tree/v0.38.4/examples#examples
[v0-37-5-examples]: https://github.com/tektoncd/chains/tree/v0.37.5/examples#examples