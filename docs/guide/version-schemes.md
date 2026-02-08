# Version Schemes

Forge supports two versioning schemes: **Semantic Versioning (SemVer)** and **Calendar Versioning (CalVer)**. You choose your scheme in `forge.yaml` and Forge handles the rest.

## Semantic Versioning (SemVer)

SemVer follows the `MAJOR.MINOR.PATCH` pattern as defined by [semver.org](https://semver.org).

```yaml
version:
  scheme: semver
  prefix: v
```

### Bump Types

| Bump | When to use | Example |
|------|------------|---------|
| `patch` | Bug fixes, small changes | `v1.2.3` → `v1.2.4` |
| `minor` | New features, backwards compatible | `v1.2.3` → `v1.3.0` |
| `major` | Breaking changes | `v1.2.3` → `v2.0.0` |

```bash
forge bump --bump patch
forge bump --bump minor
forge bump --bump major
```

### Version Prefix

The `prefix` field adds a prefix to the version string displayed in outputs. The `git.tag_prefix` field determines the actual git tag prefix:

```yaml
version:
  prefix: v        # Display: v1.2.3

git:
  tag_prefix: v    # Git tag: v1.2.3
```

## Calendar Versioning (CalVer)

CalVer uses date-based version numbers. This is useful for projects with regular release cadences where "compatibility" isn't the primary concern.

```yaml
version:
  scheme: calver
  prefix: v
  calver_format: "2006.WW"
```

### Supported Formats

| Format | Example | Description |
|--------|---------|-------------|
| `2006.WW` | `2025.44` | Year.Week (ISO week number, 01–53) ⭐ Popular |
| `2006.01.02` | `2025.11.02` | Year.Month.Day |
| `2006.01` | `2025.11` | Year.Month |

::: tip
`WW` is a special code recognized by Forge for ISO week numbers. The rest of the format follows Go's [time formatting](https://pkg.go.dev/time#pkg-constants) conventions.
:::

### Sequence Numbers

When you release multiple versions within the same calendar period, Forge automatically appends a sequence number:

```
v2025.44       # First release in week 44
v2025.44.1     # Second release in week 44
v2025.44.2     # Third release in week 44
v2025.45       # First release in week 45 (resets)
```

With CalVer, the `--bump` flag is **ignored** — versions are determined automatically by the current date and existing tags.

```bash
# Just run bump — no bump type needed for CalVer
forge bump
```

## Prerelease and Build Metadata

::: warning ALPHA
The `pre` (prerelease) and `meta` (build metadata) options are in early alpha and **not fully implemented**. Do not use these in production environments.
:::

```yaml
version:
  scheme: semver
  prefix: v
  pre: "rc.1"           # → v1.2.3-rc.1
  meta: "build.456"     # → v1.2.3+build.456
```

These can also be set via CLI flags:

```bash
forge bump --bump minor --pre rc.1
forge bump --bump patch --meta build.456
```

## Choosing a Scheme

| Criteria | SemVer | CalVer |
|----------|--------|--------|
| API compatibility matters | ✅ | ❌ |
| Regular release cadence | ❌ | ✅ |
| Breaking changes need signaling | ✅ | ❌ |
| Date-driven releases | ❌ | ✅ |
| Library / SDK projects | ✅ | ❌ |
| Internal services / applications | ✅ | ✅ |

## Mixing Schemes in a Monorepo

In a monorepo, each app can use a different versioning scheme:

```yaml
defaultApp: api

api:
  version:
    scheme: semver
    prefix: v
  git:
    tag_prefix: api/v

worker:
  version:
    scheme: calver
    calver_format: "2006.WW"
  git:
    tag_prefix: worker/v
```

See [Monorepo Setup](./monorepo) for more details.
