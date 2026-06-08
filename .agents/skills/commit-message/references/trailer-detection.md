# Author Detection and Trailer Generation

Guide for detecting author information and generating required commit trailers.

## Required Trailers

Every commit must include:

1. **Signed-off-by**: Author's name and email (DCO requirement)
2. **Assisted-by**: AI model information (when AI assists)

## Signed-off-by Detection

### Priority 1: Environment Variables (Highest)

```bash
echo "$GIT_AUTHOR_NAME <$GIT_AUTHOR_EMAIL>"
```

Common in dev containers and CI environments.

### Priority 2: Git Configuration (Fallback)

```bash
git config user.name
git config user.email
```

Checks repository config first, then global, then system.

### Priority 3: Ask User (Last Resort)

If neither source is available, ask the user to provide name and email.

## Assisted-by Trailer

**Format**: `Assisted-by: Model Name (via Tool Name)`

**Examples**:

```text
Assisted-by: <Model name> (via <Tool>)
```

Always use the actual model and tool name.

## Trailer Order and Spacing

- **Blank line before trailers**: Separate body from trailers
- **No blank lines between trailers**: Trailers are consecutive
- **Signed-off-by first**, Assisted-by second

**Correct**:

```text
feat(storage): add handler

Implements storage support.

Signed-off-by: Developer Name <developer@example.com>
Assisted-by: <Model name> (via <Tool>)
```
