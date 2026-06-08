---
name: commit-message
description: This skill should be used when the user asks to "create a commit", "generate commit message", "commit changes", "make a commit", mentions "conventional commits", or discusses commit message formatting. Provides guided workflow for creating properly formatted commit messages with line length validation and required trailers.
version: 0.2.0
---

# Conventional Commit Message Creation

Create properly formatted conventional commit messages following project standards with line length validation and required trailers.

## Purpose

Generate commit messages that:

- Follow conventional commits format (`type(scope): description`)
- Use component names or GitHub issue numbers as scope
- Respect line length limits (50 for subject, 72 for body)
- Include required trailers (Signed-off-by, Assisted-by)
- Match [Tekton commit message standards](https://github.com/tektoncd/community/blob/master/standards.md#commit-messages)

## Quick Workflow

1. **Analyze changes**: Run git status and git diff to understand modifications
2. **Determine scope**: Use component name from changed files, or GitHub issue number if available
3. **Generate message**: Create conventional commit message with proper formatting
4. **Add trailers**: Include Signed-off-by and Assisted-by trailers
5. **Confirm with user**: Display message and wait for approval before committing

**CRITICAL**: Never commit without explicit user confirmation.

## Conventional Commit Format

### Structure

```text
<type>(<scope>): <description>

[optional body]

Signed-off-by: <name> <email>
Assisted-by: <model-name> (via <Tool>)
```

### Type Selection

Choose the appropriate commit type based on changes:

| Type | Description | Example |
| ------ | ------------- | --------- |
| `feat` | New features | `feat(storage): add Archivista backend` |
| `fix` | Bug fixes | `fix(signing): resolve KMS race condition` |
| `docs` | Documentation | `docs(README): update installation steps` |
| `refactor` | Code refactoring | `refactor(formats): simplify SLSA builder` |
| `test` | Test changes | `test(e2e): add OCI storage tests` |
| `chore` | Maintenance | `chore(deps): update go dependencies` |
| `build` | Build system | `build(Makefile): add vendor target` |
| `ci` | CI/CD changes | `ci(github): add golangci-lint action` |
| `perf` | Performance | `perf(reconciler): reduce requeue latency` |
| `style` | Code style | `style(format): run goimports` |
| `revert` | Revert commit | `revert: undo breaking config change` |

For complete type reference, see `references/commit-types.md`.

### Scope Rules

#### Priority 1: Component from changed files

Analyze staged files to identify the primary component:

```bash
git diff --cached --name-only
```

| File pattern | Scope | Example commit |
| ------------ | ----- | -------------- |
| `pkg/chains/signing/*` | `signing` | `fix(signing): resolve x509 cert issue` |
| `pkg/chains/storage/*` | `storage` | `feat(storage): add PubSub backend` |
| `pkg/chains/formats/*` | `formats` | `refactor(formats): simplify SLSA v2` |
| `pkg/reconciler/taskrun/*` | `taskrun` | `fix(taskrun): handle nil status` |
| `pkg/reconciler/pipelinerun/*` | `pipelinerun` | `feat(pipelinerun): add deep inspection` |
| `pkg/config/*` | `config` | `feat(config): add new signing key` |
| `pkg/artifacts/*` | `artifacts` | `refactor(artifacts): simplify extraction` |
| `cmd/*` | `controller` | `feat(controller): add new flag` |
| `docs/*` | `docs` or filename | `docs(config): update signing options` |
| `test/*` | component being tested | `test(storage): add GCS tests` |
| Root files | filename | `chore(Makefile): add lint target` |
| `AGENTS.md`, `CLAUDE.md` | `docs` | `docs(AGENTS.md): update conventions` |

#### Priority 2: GitHub issue number (optional)

If the work is tracked in a GitHub issue and the user provides one, it can be used as the scope:

```text
Fixes #123
```

Add `Fixes #NNN` or `Closes #NNN` in the commit body (not the scope).

#### Priority 3: Ask user

If changed files span multiple components or scope is unclear, ask the user which component is the primary focus.

## Line Length Requirements

### Subject Line

- **Target**: 50 characters maximum
- **Hard limit**: 72 characters (Tekton community standard)
- **Format**: `type(scope): description` counts toward limit
- **Tips**: Use present tense, no period at end

### Body

- **Wrap at 72 characters per line**
- **Blank line** required between subject and body
- **Content**: Explain why, not what (code shows what)

## Required Trailers

### Signed-off-by

**Always include**: `Signed-off-by: <name> <email>`

This certifies the Developer Certificate of Origin (DCO) — required by tektoncd upstream.

**Detection priority order**:

1. Environment variables: `$GIT_AUTHOR_NAME` and `$GIT_AUTHOR_EMAIL`
2. Git config: `git config user.name` and `git config user.email`
3. If neither configured, ask user to provide details

For complete detection logic, see `references/trailer-detection.md`.

### Assisted-by

**Always include**: `Assisted-by: <model-name> (via <Tool>)`

Use the actual model name and tool name.

## User Confirmation Requirement

**CRITICAL RULE**: Always ask for user confirmation before executing `git commit`.

1. **Generate** the commit message following all rules above
2. **Display** the complete message to the user
3. **Ask**: "Should I commit with this message? (y/n)"
4. **Wait** for user response
5. **Commit** only if user confirms

## Commit Execution

Use heredoc format for proper multi-line handling:

```bash
git commit -m "$(cat <<'EOF'
feat(storage): add Archivista backend

Enable storing attestations in Archivista for improved
discoverability and querying.

Signed-off-by: Developer Name <developer@example.com>
Assisted-by: <Model name> (via <Tool>)
EOF
)"
```

**Never use**:

- `--no-verify` (skips pre-commit hooks)
- `--no-gpg-sign` (skips signing)
- `--amend` (unless explicitly requested and safe)

## Format standards

Follow [Tekton commit message standards](https://github.com/tektoncd/community/blob/master/standards.md#commit-messages):

- Conventional commit format (`type(scope): description`)
- Subject line: target 50 characters, hard limit 72
- Body lines wrapped at 72 characters
- Required `Signed-off-by` trailer (DCO)
- No trailing punctuation on the subject

## Auto-Detection Summary

1. Run `git status` and `git diff` for staged/unstaged changes
2. Identify primary component from staged file paths
3. If scope unclear, ask user
4. Analyze staged files to determine commit type
5. Detect author info from environment variables or git config
6. Ensure subject line is ≤50 characters (max 72)
7. Wrap body text at 72 characters per line
8. Add required trailers (Signed-off-by and Assisted-by)
9. **Display message and ask for user confirmation**
10. Only commit after receiving confirmation

## Additional Resources

- **`references/commit-types.md`** - Complete commit type reference with descriptions
- **`references/trailer-detection.md`** - Author detection logic and priority order
