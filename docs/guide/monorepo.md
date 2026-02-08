# Monorepo Setup

Forge has first-class support for monorepos — repositories that contain multiple applications, each with independent versioning, builds, and Docker images.

## How It Works

In a monorepo configuration, each application gets:

- Its own **version scheme** (SemVer or CalVer)
- Its own **git tag prefix** (e.g., `api/v1.2.3`, `worker/v2025.44`)
- Its own **build and Docker settings**
- Its own **hotfix configuration**

## Configuration

A monorepo config uses top-level keys for each app, plus a `defaultApp` field:

```yaml
defaultApp: api

api:
  version:
    scheme: semver
    prefix: v
  git:
    tag_prefix: api/v
    default_branch: main
  build:
    name: api
    main_path: ./cmd/api/main.go
    targets:
      - linux/amd64
      - linux/arm64

worker:
  version:
    scheme: calver
    calver_format: "2006.WW"
    prefix: v
  git:
    tag_prefix: worker/v
    default_branch: main
  build:
    name: worker
    main_path: ./cmd/worker/main.go
    targets:
      - linux/amd64
      - linux/arm64
```

::: tip
Use `forge init --multi` to generate a monorepo configuration template.
:::

## The `defaultApp` Field

When `defaultApp` is set, commands that don't specify `--app` will target the default app:

```bash
# Targets the "api" app (the default)
forge bump --bump minor

# Targets the "worker" app explicitly
forge bump --bump minor --app worker
```

## Tag Namespacing

Each app's `git.tag_prefix` creates namespaced tags. This prevents collisions and lets Forge identify which tags belong to which app:

```
api/v1.0.0
api/v1.1.0
api/v1.2.0
worker/v2025.44
worker/v2025.44.1
worker/v2025.45
```

## Commands in a Monorepo

### Bumping Versions

```bash
forge bump --bump minor --app api --push
forge bump --app worker --push           # CalVer, no bump type needed
```

### Viewing Versions

Without `--app`, Forge displays a table of all apps:

```bash
forge version
```

```
 App       Current       Scheme   Last Tag        Date                 Commit
 api       1.2.3         semver   api/v1.2.3      2025-01-15 10:30     abc12345
 worker    2025.44       calver   worker/v2025.44 2025-01-14 09:00     def67890
```

For a specific app:

```bash
forge version --app api
```

### Listing Version History

```bash
forge version list --app api --limit 5
```

### Previewing Next Version

```bash
forge version next --app worker
```

### Building

```bash
forge build --app api
forge build --app worker
```

### Docker Images

```bash
forge docker --app api --push
```

### Changelog

```bash
forge changelog --app api --from api/v1.0.0 --to api/v1.2.0
```

### Validation

```bash
forge validate --app api
forge validate --app worker
```

## App Auto-Detection

For hotfix commands, Forge automatically detects the app from the tag prefix — no `--app` flag needed:

```bash
forge hotfix create api/v1.0.0
# Detects 'api' from the tag prefix

forge hotfix create worker/v2025.44
# Detects 'worker' from the tag prefix
```

## Example: Full Monorepo Workflow

```bash
# Initialize monorepo config
forge init --multi

# Create first tags
forge bump --initial 1.0.0 --app api --push
forge bump --initial --app worker --push

# Development cycle
git commit -m "feat(api): add user endpoint"
forge bump --bump minor --app api --push

git commit -m "fix(worker): fix queue processing"
forge bump --app worker --push

# View all versions
forge version

# Generate changelog for a specific app
forge changelog --app api --output CHANGELOG-api.md

# Build and deploy a specific app
forge build --app api
forge docker --app api --push
```
