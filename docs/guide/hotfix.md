# Hotfix Workflow

Forge provides a dedicated hotfix workflow for patching released versions without including newer changes from `main`. This is essential when a critical bug is found in production while development has already moved ahead.

## The Problem

Imagine your production runs `v1.5.0`, but development on `main` is already at `v1.7.0`. A critical bug is found in production. You can't just release `v1.7.1` because it would include all the unreleased changes from `v1.6.0` and `v1.7.0`.

## The Solution

Forge's hotfix workflow creates a branch from the release tag, lets you apply fixes, and creates sequenced hotfix tags:

```
main:     v1.5.0 ── v1.6.0 ── v1.7.0 ── ...
              │
              └── release/v1.5.0 (hotfix branch)
                       │
                       ├── v1.5.0-hotfix.1  (first fix)
                       └── v1.5.0-hotfix.2  (second fix)
```

## Commands

### Create a Hotfix Branch

```bash
forge hotfix create v1.5.0
```

This creates and checks out a branch named `release/v1.5.0` from the `v1.5.0` tag.

| Flag | Description |
|------|-------------|
| `--app`, `-a` | App name (auto-detected from tag in monorepos) |
| `--no-checkout` | Create the branch without switching to it |
| `--dry-run` | Preview without making changes |

### Apply Fixes and Bump

After committing your fixes on the hotfix branch:

```bash
git commit -m "fix: critical security issue"
forge hotfix bump --push
```

This creates the tag `v1.5.0-hotfix.1` and pushes it to the remote.

| Flag | Description |
|------|-------------|
| `--push` | Push the tag to remote |
| `--message`, `-m` | Custom tag message |
| `--dry-run` | Preview without making changes |

Each subsequent `forge hotfix bump` increments the sequence number:

```
v1.5.0-hotfix.1
v1.5.0-hotfix.2
v1.5.0-hotfix.3
```

### Quick Hotfix (Create + Bump)

Use `--base` to create the hotfix branch and tag in one step:

```bash
forge hotfix bump --base v1.5.0 --push
```

### Check Hotfix Status

```bash
forge hotfix status
```

Shows the current hotfix branch state, base tag, last hotfix tag, next tag, and all active hotfix branches.

### List Hotfix Tags

```bash
forge hotfix list v1.5.0
```

Lists all hotfix tags for a given base version. If you're already on a hotfix branch, the base tag is auto-detected:

```bash
forge hotfix list
```

## Configuration

Hotfix behavior is configured under `git.hotfix`:

```yaml
git:
  tag_prefix: v
  default_branch: main
  hotfix:
    branch_prefix: "release/"    # Branch: release/v1.5.0
    suffix: "hotfix"             # Tag: v1.5.0-hotfix.1
```

### Custom Naming

You can customize both the branch prefix and the tag suffix:

```yaml
git:
  hotfix:
    branch_prefix: "hotfix/"    # Branch: hotfix/v1.5.0
    suffix: "patch"             # Tag: v1.5.0-patch.1
```

### Defaults

If `git.hotfix` is omitted, Forge uses these defaults:

| Setting | Default | Example |
|---------|---------|---------|
| `branch_prefix` | `release/` | `release/v1.5.0` |
| `suffix` | `hotfix` | `v1.5.0-hotfix.1` |

## Monorepo Hotfixes

In a monorepo, Forge automatically detects the app from the tag prefix:

```bash
# Auto-detects 'api' from the tag prefix
forge hotfix create api/v1.0.0

# Creates branch: release/api/v1.0.0
# Hotfix tag: api/v1.0.0-hotfix.1
```

Each app can have its own hotfix configuration:

```yaml
api:
  git:
    tag_prefix: api/v
    hotfix:
      branch_prefix: "hotfix/"
      suffix: "patch"

worker:
  git:
    tag_prefix: worker/v
    # Uses defaults: release/, hotfix
```

## Full Workflow Example

```bash
# 1. Production is running v1.5.0, critical bug found
forge hotfix create v1.5.0
# ✓ Created hotfix branch: release/v1.5.0

# 2. Fix the bug
vim src/critical_handler.go
git add . && git commit -m "fix: critical security vulnerability"

# 3. Create the hotfix tag
forge hotfix bump --push
# ✓ Created hotfix tag: v1.5.0-hotfix.1
# ✓ Pushed tag to remote

# 4. Another fix needed
git commit -m "fix: edge case in auth flow"
forge hotfix bump --push
# ✓ Created hotfix tag: v1.5.0-hotfix.2

# 5. Check the status
forge hotfix status

# 6. List all hotfixes
forge hotfix list v1.5.0
```
