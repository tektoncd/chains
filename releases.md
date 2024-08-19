# Tekton Chains Releases

## Release Frequency

Tekton Chains follows the Tekton community [release policy][release-policy] as
follows:

- Versions are numbered according to semantic versioning: `vX.Y.Z`
- At a minimum four LTS release are produced. Additional releases are produced
  based on availability of new features to be released
- Four releases a year are chosen for
  [long term support (LTS)](https://github.com/tektoncd/community/blob/main/releases.md#support-policy).
  All remaining releases are supported for approximately 1 month.
  - The first Tekton Chains LTS release will be **v0.13.0** in October 2022

Tekton Chains produces nightly builds, publicly available on
`gcr.io/tekton-nightly`.

### Transition Process

Before release v0.13 Tekton Chains has worked on the basis of an undocumented
support period, providing patch releases when needed. While transitioning to the
new support model, v0.11 and v0.12 will be supported for four months from the
initial publishing date.

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

## Backwards Incompatible Changes

In general we aim for new features to be added in a backwards compatible way,
but sometimes we will need to make breaking changes. This policy outlines how we
will make and communicate these.

**NOTE**: Tekton Chains is working towards a formal beta release. Until then,
all features are technically considered `alpha` (though we continue to do our
best to retain backwards compatibility).

### Stability levels

| Stability Level | Deprecation Window                                       |
| --------------- | -------------------------------------------------------- |
| alpha           | none                                                     |
| beta            | 3 months or 3 minor releases (which ever is longer)      |
| stable          | 12 months or 3 LTS minor releases (which ever is longer) |

Example: if a beta feature is announced as deprecated in v0.10.0, it can be
completely removed in v0.13.0.

### What's in scope

- [User Configuration](docs/config.md)
- Input mechanisms from Pipelines (e.g.
  [Type Hinting](https://github.com/tektoncd/chains/blob/main/docs/config.md#chains-type-hinting))
- Chains generated provenance output formats
- Exported Client Libraries (https://pkg.go.dev/github.com/tektoncd/chains)

<!-- TODO(wlynch): if/when config type aliases are a thing, document policy around how these can change -->

#### Exceptions

We reserve the right to make breaking changes (regardless of stability level)
under certain conditions:

- Security

  We will make breaking changes for security reasons (e.g. in response to CVEs
  or other vulnerabilities) if necessary.

- Backwards incompatible changes from dependencies

  We try to keep Chains up-to-date with minor versions of its dependencies with
  Dependabot. Chains assumes its dependencies adhere to
  [Go module versioning](https://go.dev/doc/modules/developing#versioning) for
  backwards compatibility. In cases where dependencies break this expectation,
  we will try and work around these changes as best as we can and make breaking
  changes as a last resort.

  Example: If tektoncd/pipeline makes a breaking change to its client that
  renders Chains unable to operate, we will update Chains to get things working
  again even if it requires a breaking change to Chains config.

### Deprecation Notice

Deprecation notices will be included in
[release notes](https://github.com/tektoncd/chains/releases). Any relevant
deprecation window timers begin from when the release is published.

## Current Releases

### v0.22 (LTS)
- **Latest Release**: [v0.22.0][v0-22-0] (2024-08-19)
- **Initial Release**: [v0.22.0][v0-22-0] (2024-08-19)

[v0-22-0]: https://github.com/tektoncd/chains/releases/tag/v0.22.0

### v0.21 (LTS)
- **Latest Release**: [v0.21.1][v0-21-1] (2024-06-10)
- **Initial Release**: [v0.21.0][v0-21-0] (2024-05-23)

[v0-21-0]: https://github.com/tektoncd/chains/releases/tag/v0.21.0
[v0-21-1]: https://github.com/tektoncd/chains/releases/tag/v0.21.1

### v0.20 (LTS)

- **Latest Release**: [v0.20.0][v0-20-0] (2024-01-31)
- **Initial Release**: [v0.20.0][v0-20-0] (2024-01-31)

[v0-20-0]: https://github.com/tektoncd/chains/releases/tag/v0.20.0

### v0.19 (LTS)

- **Latest Release**: [v0.19.0][v0-19-0] (2023-10-26)
- **Initial Release**: [v0.19.0][v0-19-0] (2023-10-26)

[v0-19-0]: https://github.com/tektoncd/chains/releases/tag/v0.19.0

## Older Releases

Older releases are EOL and available on [GitHub][tekton-chains-releases].

[release-policy]: https://github.com/tektoncd/community/blob/main/releases.md
[sigstore]: https://sigstore.dev
[tekton-chains]: https://github.com/tektoncd/chains
[tekton-chains-releases]: https://github.com/tektoncd/chains/releases
[chains-public-key]: https://github.com/tektoncd/chains/blob/main/tekton.pub
[tekton-releases-docs]: release/README.md
[tekton-installation]: README.md#installation
[release-notes-standards]:
  https://github.com/tektoncd/community/blob/main/standards.md#release-notes
