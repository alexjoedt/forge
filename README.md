<p align="center">
  <img src="assets/logo.png" alt="Forge Logo" width="400"/>
</p>

# Forge

A CLI for automated git version tagging and changelog generation. Supports SemVer, CalVer, and monorepo workflows.

I built this for my personal projects to handle versioning without the hassle.

## What It Does

- **Version Management**: SemVer and CalVer (with ISO week support)
- **Hotfix Workflow**: Safe patching of released versions with dedicated branches
- **Monorepo Support**: Per-app versioning with namespaced tags
- **Changelog Generation**: Parses conventional commits
- **Interactive CLI**: Quick bumps with Bubble Tea UI

> **⚠️ ALPHA Features**: The `meta` (build metadata) configuration option is in early alpha state and not fully implemented. Do not use in production environments. The `pre` (prerelease) feature is available via `forge bump pre`.

## Quick Start

```bash
# Install
go install github.com/alexjoedt/forge@latest

# Initialize
forge init

# Bump version (interactive)
forge bump

# Or direct
forge bump minor --push

# Create prerelease tag (rc channel)
forge bump pre rc --bump minor

# Move a tag to a different commit
forge retag v1.2.3
```

## Core Commands

```bash
forge bump [major|minor|patch]    # Create version tag
forge bump pre <channel>          # Manage prerelease lifecycle
forge retag <tag> [<commit>]      # Move an existing tag to a different commit
forge hotfix create <tag>         # Create hotfix branch from release
forge hotfix bump                 # Bump hotfix version
forge hotfix status               # Show hotfix branch status
forge version                     # Show current version(s)
forge changelog                   # Generate changelog
forge validate                    # Check config and git state
```

## Configuration

Minimal `forge.yaml`:

```yaml
version:
  scheme: semver    # or: calver
  prefix: v

git:
  tag_prefix: v
  default_branch: main
```

Monorepo:

```yaml
defaultApp: api

api:
  version:
    scheme: semver
    prefix: v
  git:
    tag_prefix: api/v
    default_branch: main

worker:
  version:
    scheme: calver
    calver_format: "2006.WW"  # Year.Week
  git:
    tag_prefix: worker/
    default_branch: main
```

## Workflow

### Normal Development

```bash
# Make changes
git commit -m "feat: new feature"

# Tag version
forge bump minor --push

# Generate changelog
forge changelog --output CHANGELOG.md
```

### Prerelease Workflow

```bash
# Start a release candidate from v1.5.0
forge bump pre rc --bump minor    # v1.5.0 → v1.6.0-rc.1

# Increment the prerelease number
forge bump pre rc                 # v1.6.0-rc.1 → v1.6.0-rc.2

# Graduate to stable release
forge bump pre release            # v1.6.0-rc.2 → v1.6.0
```

### Re-tagging

```bash
# Move a tag to HEAD
forge retag v1.2.3

# Move a tag to a specific commit
forge retag v1.2.3 abc1234

# Move and push
forge retag v1.2.3 --push
```

### Hotfix Workflow

```bash
# Production has v1.5.0, main is at v1.7.0
# Critical bug found in production

# Create hotfix branch from production tag
forge hotfix create v1.5.0

# Apply fixes and commit
git commit -m "fix: critical security issue"

# Create hotfix tag (v1.5.0-hotfix.1)
forge hotfix bump --push

# Check status anytime
forge hotfix status

# List all hotfixes for a version
forge hotfix list v1.5.0
```
