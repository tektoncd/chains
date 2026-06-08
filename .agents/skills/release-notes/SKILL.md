---
name: release-notes
description: This skill should be used when the user asks to "create release note", "generate release notes", "release notes", "release changelog", "update GitHub release", or wants to generate categorized release notes between tags. Gathers PR/commit data via gh CLI, categorizes changes, and outputs formatted release notes. The user can optionally specify a release version (e.g. "create release note v0.25.0") to bypass auto-detection.
version: 0.1.0
---

# Release Notes Generation

Generate categorized release notes for Tekton Chains releases by gathering PR/commit data between two git tags and producing formatted markdown output.

## Purpose

Skill that:

- Auto-detects current and previous tags
- Gathers PR and commit data via `gh` CLI
- Categorizes changes intelligently
- Outputs release notes matching the project's established format
- Optionally updates the GitHub release

## Workflow

### Step 1: Pull latest tags

```bash
git pull origin --tags
```

### Step 2: Detect repository info

```bash
gh repo view --json owner,name
```

### Step 3: Detect tags

**If the user specified a version** (e.g. "create release note v0.25.0"), use that directly. Validate that the tag exists locally:

```bash
git tag --list 'v0.25.0'
```

**Otherwise, auto-detect current tag:**

```bash
git tag --points-at HEAD
```

Filter for `v*` prefixed tags. If no tag points at HEAD, list recent tags and ask the user:

```bash
git tag --list 'v*' --sort=-version:refname | head -10
```

**Previous tag:**

```bash
git tag --list 'v*' --sort=-version:refname
```

Find the entry immediately after the current tag in the version-sorted list.

**CRITICAL**: Confirm both tags with the user before proceeding.

### Step 4: Verify prerequisites

```bash
gh auth status
```

Validate both tags exist on GitHub:

```bash
gh api repos/{owner}/{repo}/git/ref/tags/{current_tag}
gh api repos/{owner}/{repo}/git/ref/tags/{previous_tag}
```

### Step 5: Gather PR/commit data

```bash
gh api repos/{owner}/{repo}/compare/{previous_tag}...{current_tag} --jq '.commits[].sha'
```

For each commit, find associated PRs:

```bash
gh api repos/{owner}/{repo}/commits/{sha}/pulls
```

Deduplicate by PR number. Extract: PR number, title, body, author, URL, and labels.

### Step 6: Categorize changes

Sections (skip empty ones):

- `## ✨ Major changes and Features`
- `## 🐛 Bug Fixes`
- `## 📚 Documentation Updates`
- `## ⚙️ Chores`

**Categorization guidelines:**

- New capabilities, enhancements → Features
- Bug fixes, error corrections → Bug Fixes
- Documentation-only changes → Documentation Updates
- Dependencies, CI/CD, refactoring, test-only → Chores

**Internal vs user-facing**: CI/CD, release infra, test infra, build system, developer tooling, and internal refactoring belong in Chores even if prefixed with `feat:`.

### Step 7: Assemble release notes

Use `references/release-notes-format.md` for header, entry format, and installation section.

Append GitHub auto-generated changelog:

```bash
gh api repos/{owner}/{repo}/releases/generate-notes -f tag_name="{current_tag}" -f previous_tag_name="{previous_tag}"
```

### Step 8: Output and optional GitHub release update

1. Write to `/tmp/release-notes-{current_tag}.md`
2. Display to user
3. Ask if they want to update the GitHub release

```bash
gh release edit {current_tag} --notes-file /tmp/release-notes-{current_tag}.md
```

## Error Handling

| Scenario | Action |
| --- | --- |
| No tag at HEAD | List recent tags, ask user to pick |
| Tag doesn't exist on GitHub | Stop and report error |
| `gh` not authenticated | Instruct user to run `gh auth login` |
| No previous tag found | Ask user to provide one |
| Release already published | Ask for confirmation before overriding |

## User Confirmation Requirements

**CRITICAL**: Always confirm tags before gathering data. Always confirm before updating a GitHub release.
