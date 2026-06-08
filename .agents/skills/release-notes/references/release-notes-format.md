# Release Notes Format Reference

## Entry Format

Each release note entry MUST follow this exact format. The Link line MUST be indented with two spaces so it renders as a nested sub-bullet:

```markdown
* **Bold title:** One-sentence description of the change.
  * Link: <PR_OR_COMMIT_URL>
```

### Rules

- The first bullet MUST start with `*` (no indent) with a bold title followed by a colon and a description.
- The Link line MUST start with `* Link:` (two-space indent) with the PR or commit URL.
- Do NOT add a Contributors section.

## Header Template

```markdown
# Tekton Chains {tag}

Release of Tekton Chains {tag}.
```

## Installation Section Template

```markdown
## Installation

To install this release:

kubectl apply -f https://storage.googleapis.com/tekton-releases/chains/previous/{tag}/release.yaml
```
