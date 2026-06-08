# Tekton Chains

Kubernetes controller for supply chain security. Watches TaskRun and
PipelineRun completions, signs them, generates attestations (in-toto,
SLSA), and stores signatures in configurable backends (OCI, Tekton
annotations, GCS, Grafeas, DocDB, Archivista, PubSub).

---

## Build & Test Commands

```bash
# Build
make bin/controller

# Unit tests (no cluster needed)
make test-unit
make test-unit-verbose-and-race
make test-unit PKG=./pkg/chains/storage/...  # single package (fast)

# Lint — must pass before every PR
make golangci-lint                          # all packages
make golangci-lint PKG=./pkg/chains/...     # single package (fast)
make yamllint                               # YAML linting

# Code generation — after changing pkg/config types
make generated

# Dependency update — after go.mod changes
make vendor

# Deploy locally (requires cluster + Tekton Pipelines installed)
ko apply -f config/
```

E2E tests require a live cluster and are tagged `//go:build e2e`.
See [DEVELOPMENT.md](./DEVELOPMENT.md) for cluster setup.

---

## Key Conventions

1. **Single binary.** Only `cmd/controller` — no CLI, no webhook binary.
   The controller registers both TaskRun and PipelineRun reconcilers.

2. **Vendored dependencies.** All builds use `-mod=vendor`. Run
   `make vendor` (calls `hack/update-deps.sh`) after any `go.mod` change.

3. **Knative controller runtime.** Reconcilers use `knative.dev/pkg`,
   not `controller-runtime`. Status conditions use
   `condSet.Manage(status).MarkTrue/MarkFalse`.

4. **ConfigMap-driven configuration.** All runtime config lives in
   `pkg/config/` and is loaded from the `chains-config` ConfigMap.
   See [docs/config.md](./docs/config.md) for all keys.

5. **Sigstore ecosystem.** Signing uses cosign/Fulcio/Rekor. KMS
   providers (AWS, Azure, GCP, Vault) are registered at startup in
   `cmd/controller/main.go`.

---

## Architecture

```
cmd/controller/             → Binary entrypoint, registers reconcilers + KMS
pkg/reconciler/taskrun/     → TaskRun controller — watches completions
pkg/reconciler/pipelinerun/ → PipelineRun controller
pkg/chains/                 → Core signing orchestration (ObjectSigner)
pkg/chains/signing/         → Signer interface + x509/KMS implementations
pkg/chains/formats/         → Attestation formats (simple, SLSA v1/v2alpha)
pkg/chains/storage/         → Storage backends (oci, tekton, gcs, grafeas…)
pkg/config/                 → ConfigMap parsing + deepcopy codegen
pkg/artifacts/              → Artifact/signable abstractions
config/                     → Kubernetes manifests (deployment, RBAC, ConfigMaps)
test/                       → E2E tests (Go + shell) and test data
```

---

## PR Conventions

- Pull requests must follow the repository PR template defined in `.github/pull_request_template.md`.
- Commit messages should follow [Tekton community standards](https://github.com/tektoncd/community/blob/master/standards.md#commit-messages).
- `make golangci-lint` must pass with zero issues.
- `make test-unit` must pass with zero failures.
- Run `make generated` and commit results after changing `pkg/config/` types.
- Commits require `Signed-off-by` (DCO).

---

## Windows checkout

`CLAUDE.md` points to `AGENTS.md`, and `.claude/skills` points to `.agents/skills`.
This works on Linux, macOS, and GitHub; on Windows, enable symlinks when cloning:

```bash
git clone -c core.symlinks=true https://github.com/tektoncd/chains.git
```

Alternatively, set `core.symlinks=true` in your git config before checkout.

---

## Skills

- **Commit messages**: Conventional commits with component scopes,
  line length validation, DCO Signed-off-by, and Assisted-by trailers.
- **Release notes**: Gather PRs between tags, categorize, output
  formatted markdown, optionally update GitHub release.
