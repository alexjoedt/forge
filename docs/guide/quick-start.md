# Quick Start

This guide walks you through your first version tag with Forge in under 2 minutes.

## 1. Initialize Configuration

Navigate to your Git repository and run:

```bash
forge init
```

This creates a `forge.yaml` file with sensible defaults:

```yaml
version:
  scheme: semver
  prefix: v

git:
  tag_prefix: v
  default_branch: main
```

::: tip
Use `forge init --multi` to generate a monorepo configuration with multiple apps.
:::

## 2. Create Your First Tag

If this is a new project with no existing tags, create an initial version:

```bash
forge bump --initial 1.0.0
```

This creates the git tag `v1.0.0` on your current commit.

## 3. Bump Versions

After making changes and committing, bump the version:

```bash
# Interactive mode — select bump type with arrow keys
forge bump

# Or specify the bump type directly
forge bump --bump patch    # v1.0.0 → v1.0.1
forge bump --bump minor    # v1.0.1 → v1.1.0
forge bump --bump major    # v1.1.0 → v2.0.0
```

## 4. Push Tags

Add `--push` to automatically push the tag to the remote:

```bash
forge bump --bump minor --push
```

## 5. Check Current Version

```bash
forge version
```

Output:

```
Current Version: v1.1.0
Scheme:          semver
Commit:          abc1234f
```

## 6. Preview Next Version

See what the next version would be without creating a tag:

```bash
forge version next --bump minor
```

Output:

```
Current:  1.1.0
Next:     1.2.0
Tag:      v1.2.0
Scheme:   semver
```

## 7. Generate a Changelog

```bash
forge changelog --from v1.0.0 --to v1.1.0
```

Or save it to a file:

```bash
forge changelog --output CHANGELOG.md
```

## What's Next?

| Goal | Guide |
|------|-------|
| Understand SemVer vs CalVer | [Version Schemes →](./version-schemes) |
| Learn the bump command in depth | [Bump Command →](./bump) |
| Set up a hotfix workflow | [Hotfix Workflow →](./hotfix) |
| Configure a monorepo | [Monorepo Setup →](./monorepo) |
| Full config reference | [Configuration →](../reference/configuration) |
