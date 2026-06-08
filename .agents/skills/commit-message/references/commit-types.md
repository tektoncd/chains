# Commit Types Reference

Complete reference for conventional commit types used in the Tekton Chains project.

## Scope and issue references

- Prefer **component names** as scope (e.g. `signing`, `storage`, `config`, `reconciler`).
- To link work to a GitHub issue, add `Fixes #NNN` or `Closes #NNN` in the **commit body** (not the subject scope). Merging the PR then closes the issue automatically.

## Standard Types

### feat - New Features

Use for new features or functionality added to the codebase.

**Examples**:

- `feat(storage): add Archivista backend`
- `feat(signing): implement Vault KMS support`
- `feat(formats): add SLSA v2alpha4 provenance`
- `feat(config): expose new signing option`

**When to use**:

- Adding new capabilities
- Introducing new storage/signing/format backends
- Implementing new APIs or configuration options

### fix - Bug Fixes

Use for bug fixes that resolve incorrect behavior.

**Examples**:

- `fix(reconciler): resolve TaskRun race condition`
- `fix(signing): correct x509 certificate chain`
- `fix(storage): handle OCI registry timeout`
- `fix(config): prevent nil pointer on missing key`

**When to use**:

- Fixing crashes or errors
- Resolving incorrect behavior
- Correcting logic errors

**CVE / security fixes**: Use scope `cve` and cite the advisory in the subject (see examples on `main`). Routine dependency bumps from Dependabot use `chore(deps):`, not `fix`.

### docs - Documentation

Use for documentation-only changes.

**Examples**:

- `docs(README): update installation steps`
- `docs(config): document new signing options`
- `docs(tutorials): add GCS storage walkthrough`
- `docs(AGENTS.md): update architecture section`

**When to use**:

- Updating README or docs/ files
- Adding or improving code comments
- Updating developer guides

### refactor - Code Refactoring

Use for code changes that neither fix bugs nor add features.

**Examples**:

- `refactor(formats): extract common SLSA logic`
- `refactor(storage): consolidate backend interface`
- `refactor(chains): simplify ObjectSigner flow`

**When to use**:

- Improving code readability
- Simplifying complex logic
- Reorganizing code structure

**Must not** change observable behavior. If behavior changes, use `fix` (bug) or `feat` (new capability) instead.

### test - Testing

Use for adding or updating tests.

**Examples**:

- `test(e2e): add OCI storage integration test`
- `test(signing): improve x509 test coverage`
- `test(formats): add SLSA v2 edge case tests`

**When to use**:

- Adding new test cases
- Improving test coverage
- Fixing flaky tests

### chore - Maintenance Tasks

Use for routine maintenance tasks and tooling.

**Examples**:

- `chore(deps): update go dependencies`
- `chore(vendor): run make vendor`
- `chore(tools): update golangci-lint version`

**When to use**:

- Dependency updates
- Tooling configuration
- Repository maintenance

### build - Build System

Use for changes to how the project is built (not routine dependency bumps — those are `chore(deps):`).

**Examples**:

- `build(Makefile): add yamllint target`
- `build(ko): update base image`
- `build(go.mod): bump Go version to 1.24`

**When to use**:

- Changes to the root `Makefile` or build scripts
- Container or `ko` image build configuration
- Bumping the Go version in `go.mod`

### ci - CI/CD Changes

Use for changes to continuous integration and release automation.

**Examples**:

- `ci(.github/workflows): add golangci-lint to CI`
- `ci(.github/workflows): update e2e matrix workflow`
- `ci(.tekton): update release pipeline`

**When to use**:

- Changes under `.github/workflows/`
- Changes under `.tekton/`

### perf - Performance Improvements

Use for changes that improve performance.

**Examples**:

- `perf(reconciler): reduce requeue latency`
- `perf(storage): batch OCI uploads`
- `perf(signing): cache KMS connections`

### style - Code Style

Use for formatting and style changes.

**Examples**:

- `style(format): run goimports`
- `style(lint): fix golangci-lint warnings`

### revert - Revert Previous Commit

Use for reverting previous commits.

**Examples**:

- `revert: undo breaking config change`
- `revert(signing): revert KMS refactoring`

**Format**: Include reference to original commit in body.

## Breaking Changes

For any commit type, add `!` after type/scope to indicate breaking change:

- `feat(config)!: change signing key format`
- `fix(storage)!: remove deprecated GCS options`

Body should include:

```text
BREAKING CHANGE: <description and migration path>
```

## Type Selection Guide

1. **Does it add new functionality?** → `feat`
2. **Does it fix a bug?** → `fix`
3. **Is it documentation only?** → `docs`
4. **Does it change code structure without behavior change?** → `refactor`
5. **Is it test-related?** → `test`
6. **Is it dependency/maintenance?** → `chore`
7. **Is it build system related?** → `build`
8. **Is it CI/CD related?** → `ci`
9. **Does it improve performance?** → `perf`
10. **Is it formatting/style only?** → `style`
11. **Does it revert a previous commit?** → `revert`
