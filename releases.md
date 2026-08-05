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
[ghcr.io/tektoncd/chains/controller-92006fd957c0afd31de6a40b3e33b39f](https://github.com/tektoncd/chains/pkgs/container/chains%2Fcontroller-92006fd957c0afd31de6a40b3e33b39f).

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

## Release

### v0.29 (LTS)
- **Latest Release**: [v0.29.0][v0-29-0] (2026-08-04) ([docs][v0-29-0-docs], [examples][v0-29-0-examples])
- **Initial Release**: [v0.29.0][v0-29-0] (2026-08-04)
- **End of Life**: 2027-08-04
- **Patch Releases**: [v0.29.0][v0-29-0]

### v0.28 (LTS)
- **Latest Release**: [v0.28.1][v0-28-1] (2026-07-17) ([docs][v0-28-1-docs], [examples][v0-28-1-examples])
- **Initial Release**: [v0.28.0][v0-28-0] (2026-07-10)
- **End of Life**: 2027-07-10
- **Patch Releases**: [v0.28.0][v0-28-0], [v0.28.1][v0-28-1]

### v0.27 (LTS)
- **Latest Release**: [v0.27.6][v0-27-6] (2026-08-03) ([docs][v0-27-6-docs], [examples][v0-27-6-examples])
- **Initial Release**: [v0.27.0][v0-27-0] (2026-05-28)
- **End of Life**: 2027-05-28
- **Patch Releases**: [v0.27.0][v0-27-0], [v0.27.1][v0-27-1], [v0.27.2][v0-27-2], [v0.27.3][v0-27-3], [v0.27.4][v0-27-4], [v0.27.5][v0-27-5], [v0.27.6][v0-27-6]

### v0.26 (LTS)
- **Latest Release**: [v0.26.8][v0-26-8] (2026-08-03) ([docs][v0-26-8-docs], [examples][v0-26-8-examples])
- **Initial Release**: [v0.26.0][v0-26-0] (2025-11-05)
- **End of Life**: 2026-11-05
- **Patch Releases**: [v0.26.0][v0-26-0], [v0.26.1][v0-26-1], [v0.26.2][v0-26-2], [v0.26.3][v0-26-3], [v0.26.4][v0-26-4], [v0.26.5][v0-26-5], [v0.26.6][v0-26-6], [v0.26.7][v0-26-7], [v0.26.8][v0-26-8]

## End of Life Releases

Older releases are EOL and available on [GitHub][tekton-chains-releases].

### v0.25 (LTS)
- **Latest Release**: [v0.25.2][v0-25-2] (2026-04-22) ([docs][v0-25-2-docs], [examples][v0-25-2-examples])
- **Initial Release**: [v0.25.0][v0-25-0] (2025-05-13)
- **End of Life**: 2026-05-13
- **Patch Releases**: [v0.25.0][v0-25-0], [v0.25.1][v0-25-1], [v0.25.2][v0-25-2]

### v0.24 (LTS)
- **Latest Release**: [v0.24.0][v0-24-0] (2025-02-11) ([docs][v0-24-0-docs], [examples][v0-24-0-examples])
- **Initial Release**: [v0.24.0][v0-24-0] (2025-02-11)
- **End of Life**: 2026-02-11
- **Patch Releases**: [v0.24.0][v0-24-0]

### v0.23 (LTS)
- **Latest Release**: [v0.23.1][v0-23-1] (2025-01-21) ([docs][v0-23-1-docs], [examples][v0-23-1-examples])
- **Initial Release**: [v0.23.0][v0-23-0] (2024-11-06)
- **End of Life**: 2025-11-06
- **Patch Releases**: [v0.23.0][v0-23-0], [v0.23.1][v0-23-1]

### v0.22 (LTS)
- **Latest Release**: [v0.22.3][v0-22-3] (2025-01-20) ([docs][v0-22-3-docs], [examples][v0-22-3-examples])
- **Initial Release**: [v0.22.0][v0-22-0] (2024-08-19)
- **End of Life**: 2025-08-19
- **Patch Releases**: [v0.22.0][v0-22-0], [v0.22.3][v0-22-3]

### v0.20 (LTS)
- **Latest Release**: [v0.20.0][v0-20-0] (2023-11-06) ([docs][v0-20-0-docs], [examples][v0-20-0-examples])
- **Initial Release**: [v0.20.0][v0-20-0] (2023-11-06)
- **End of Life**: 2024-11-06
- **Patch Releases**: [v0.20.0][v0-20-0]

[release-policy]: https://github.com/tektoncd/community/blob/main/releases.md
[sigstore]: https://sigstore.dev
[tekton-chains]: https://github.com/tektoncd/chains
[tekton-chains-releases]: https://github.com/tektoncd/chains/releases
[chains-public-key]: https://github.com/tektoncd/chains/blob/main/tekton.pub
[tekton-releases-docs]: release/README.md
[tekton-installation]: README.md#installation
[release-notes-standards]:
  https://github.com/tektoncd/community/blob/main/standards.md#release-notes

[v0-29-0]: https://github.com/tektoncd/chains/releases/tag/v0.29.0
[v0-28-1]: https://github.com/tektoncd/chains/releases/tag/v0.28.1
[v0-28-0]: https://github.com/tektoncd/chains/releases/tag/v0.28.0
[v0-27-6]: https://github.com/tektoncd/chains/releases/tag/v0.27.6
[v0-27-5]: https://github.com/tektoncd/chains/releases/tag/v0.27.5
[v0-27-4]: https://github.com/tektoncd/chains/releases/tag/v0.27.4
[v0-27-3]: https://github.com/tektoncd/chains/releases/tag/v0.27.3
[v0-27-2]: https://github.com/tektoncd/chains/releases/tag/v0.27.2
[v0-27-1]: https://github.com/tektoncd/chains/releases/tag/v0.27.1
[v0-27-0]: https://github.com/tektoncd/chains/releases/tag/v0.27.0
[v0-26-8]: https://github.com/tektoncd/chains/releases/tag/v0.26.8
[v0-26-7]: https://github.com/tektoncd/chains/releases/tag/v0.26.7
[v0-26-6]: https://github.com/tektoncd/chains/releases/tag/v0.26.6
[v0-26-5]: https://github.com/tektoncd/chains/releases/tag/v0.26.5
[v0-26-4]: https://github.com/tektoncd/chains/releases/tag/v0.26.4
[v0-26-3]: https://github.com/tektoncd/chains/releases/tag/v0.26.3
[v0-26-2]: https://github.com/tektoncd/chains/releases/tag/v0.26.2
[v0-26-1]: https://github.com/tektoncd/chains/releases/tag/v0.26.1
[v0-26-0]: https://github.com/tektoncd/chains/releases/tag/v0.26.0
[v0-25-2]: https://github.com/tektoncd/chains/releases/tag/v0.25.2
[v0-25-1]: https://github.com/tektoncd/chains/releases/tag/v0.25.1
[v0-25-0]: https://github.com/tektoncd/chains/releases/tag/v0.25.0
[v0-24-0]: https://github.com/tektoncd/chains/releases/tag/v0.24.0
[v0-23-1]: https://github.com/tektoncd/chains/releases/tag/v0.23.1
[v0-23-0]: https://github.com/tektoncd/chains/releases/tag/v0.23.0
[v0-22-3]: https://github.com/tektoncd/chains/releases/tag/v0.22.3
[v0-22-0]: https://github.com/tektoncd/chains/releases/tag/v0.22.0
[v0-20-0]: https://github.com/tektoncd/chains/releases/tag/v0.20.0

[v0-29-0-docs]: https://github.com/tektoncd/chains/tree/v0.29.0/docs
[v0-28-1-docs]: https://github.com/tektoncd/chains/tree/v0.28.1/docs
[v0-27-6-docs]: https://github.com/tektoncd/chains/tree/v0.27.6/docs
[v0-26-8-docs]: https://github.com/tektoncd/chains/tree/v0.26.8/docs
[v0-25-2-docs]: https://github.com/tektoncd/chains/tree/v0.25.2/docs
[v0-24-0-docs]: https://github.com/tektoncd/chains/tree/v0.24.0/docs
[v0-23-1-docs]: https://github.com/tektoncd/chains/tree/v0.23.1/docs
[v0-22-3-docs]: https://github.com/tektoncd/chains/tree/v0.22.3/docs
[v0-20-0-docs]: https://github.com/tektoncd/chains/tree/v0.20.0/docs

[v0-29-0-examples]: https://github.com/tektoncd/chains/tree/v0.29.0/examples
[v0-28-1-examples]: https://github.com/tektoncd/chains/tree/v0.28.1/examples
[v0-27-6-examples]: https://github.com/tektoncd/chains/tree/v0.27.6/examples
[v0-26-8-examples]: https://github.com/tektoncd/chains/tree/v0.26.8/examples
[v0-25-2-examples]: https://github.com/tektoncd/chains/tree/v0.25.2/examples
[v0-24-0-examples]: https://github.com/tektoncd/chains/tree/v0.24.0/examples
[v0-23-1-examples]: https://github.com/tektoncd/chains/tree/v0.23.1/examples
[v0-22-3-examples]: https://github.com/tektoncd/chains/tree/v0.22.3/examples
[v0-20-0-examples]: https://github.com/tektoncd/chains/tree/v0.20.0/examples
