# Configuration Reference

Forge is configured via a `forge.yaml` (or `.forge.yaml`) file in your project root. This page documents every configuration option.

## File Location

Forge looks for configuration files in this order:

1. `forge.yaml`
2. `.forge.yaml`

Use `forge init` to generate a default configuration file.

## Single App Configuration

```yaml
version:
  scheme: semver
  prefix: v
  calver_format: "2006.01.02"
  pre: ""
  meta: ""

git:
  tag_prefix: v
  default_branch: main
  hotfix:
    branch_prefix: "release/"
    suffix: "hotfix"

nodejs:
  enabled: false
  package_path: ""
```

## Multi-App Configuration (Monorepo)

```yaml
defaultApp: api

api:
  version: { ... }
  git: { ... }
  nodejs: { ... }

worker:
  version: { ... }
  git: { ... }
```

---

## `version`

Version scheme settings.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `scheme` | `string` | ✅ | — | Versioning scheme: `semver` or `calver` |
| `prefix` | `string` | | `""` | Prefix for displayed version (e.g., `v`) |
| `calver_format` | `string` | ✅ (if calver) | — | CalVer format string |
| `pre` | `string` | | `""` | ⚠️ *[ALPHA]* Prerelease identifier |
| `meta` | `string` | | `""` | ⚠️ *[ALPHA]* Build metadata |

### `calver_format` Values

| Format | Output | Description |
|--------|--------|-------------|
| `2006.WW` | `2025.44` | Year.Week (ISO week, 01–53) |
| `2006.01.02` | `2025.11.02` | Year.Month.Day |
| `2006.01` | `2025.11` | Year.Month |

`WW` is a special Forge extension for ISO week numbers.

---

## `git`

Git-related settings.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `tag_prefix` | `string` | ✅ | — | Git tag prefix (e.g., `v`, `api/v`) |
| `default_branch` | `string` | ✅ | — | Default branch name |
| `hotfix` | `HotfixConfig` | | defaults | Hotfix workflow settings |

### `git.hotfix`

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `branch_prefix` | `string` | | `release/` | Hotfix branch prefix |
| `suffix` | `string` | | `hotfix` | Hotfix tag suffix |

Hotfix branch name: `{branch_prefix}{tag}` (e.g., `release/v1.0.0`)
Hotfix tag name: `{tag}-{suffix}.{n}` (e.g., `v1.0.0-hotfix.1`)

---

## `nodejs`

Node.js `package.json` version sync settings. **Optional**.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | `bool` | | `false` | Enable package.json updates |
| `package_path` | `string` | | `""` | Path to `package.json` (relative to repo root, defaults to `./package.json`) |

---

## `defaultApp`

*Monorepo only.* The name of the default application when `--app` is not specified.

```yaml
defaultApp: api
```

---

## Global CLI Flags

These flags are available on all commands:

| Flag | Description |
|------|-------------|
| `--verbose`, `-v` | Enable debug logging |
| `--json` | Output results in JSON format |
| `--version` | Show Forge version |
| `--help`, `-h` | Show help |

---

## Minimal Configuration

The simplest valid `forge.yaml`:

```yaml
version:
  scheme: semver
  prefix: v

git:
  tag_prefix: v
  default_branch: main
```

This gives you `forge bump`, `forge version`, `forge changelog`, `forge validate`, and `forge hotfix` — everything needed for version management without builds or Docker.
