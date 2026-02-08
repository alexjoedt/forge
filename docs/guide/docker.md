# Docker Images

Forge can build and push Docker images for your applications, with support for multiple registries, platforms, and template-based tag naming.

## Basic Usage

```bash
# Build the Docker image
forge docker

# Build and push
forge docker --push
```

The `docker` command also has the alias `image`:

```bash
forge image --push
```

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--dockerfile` | Path to the Dockerfile | `./Dockerfile` |
| `--context` | Build context path | `.` |
| `--repository` | Image repository (overrides config) | from config |
| `--tags` | Comma-separated list of tag templates | from config |
| `--platforms` | Comma-separated list of platforms | from config |
| `--build-arg` | Build arguments (`key=value`, repeatable) | |
| `--push` | Push the image to registry | `false` |
| `--version` | Version string (auto-detected) | auto |
| `--repo-dir` | Repository directory | `.` |
| `--app` | Target app in a monorepo | default |
| `--dry-run` | Show what would be done | `false` |

## Configuration

### Single Repository

```yaml
docker:
  enabled: true
  repository: ghcr.io/username/myapp
  dockerfile: ./Dockerfile
  tags:
    - "{{ .Version }}"
    - "latest"
  platforms:
    - linux/amd64
    - linux/arm64
  build_args:
    GO_VERSION: "1.21"
```

### Multiple Repositories

Push to multiple registries simultaneously:

```yaml
docker:
  enabled: true
  repositories:
    - ghcr.io/username/myapp
    - docker.io/username/myapp
    - registry.example.com/myapp
  dockerfile: ./Dockerfile
  tags:
    - "{{ .Version }}"
    - "latest"
  platforms:
    - linux/amd64
    - linux/arm64
```

With version `v1.2.3`, this creates tags for **each** repository:

```
ghcr.io/username/myapp:v1.2.3
ghcr.io/username/myapp:latest
docker.io/username/myapp:v1.2.3
docker.io/username/myapp:latest
registry.example.com/myapp:v1.2.3
registry.example.com/myapp:latest
```

::: warning
If both `repository` (singular) and `repositories` (plural) are set, `repositories` takes precedence and `repository` is **ignored**. Forge will show a warning.
:::

### Tag Templates

Tag templates use Go template syntax with these variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{ .Version }}` | Current version | `v1.2.3` |
| `{{ .Commit }}` | Full commit hash | `abc123...` |
| `{{ .ShortCommit }}` | Short commit hash | `abc123de` |
| `{{ .Date }}` | Build date | `2025-01-15` |

### Build Arguments

Build arguments can be set in the config and/or via CLI flags:

```yaml
docker:
  build_args:
    GO_VERSION: "1.21"
    APP_ENV: "production"
```

CLI flags override config values:

```bash
forge docker --build-arg GO_VERSION=1.22 --build-arg APP_ENV=staging
```

## Enabling Docker

Docker builds are **opt-in**. To enable, set `docker.enabled: true` in your config. If not enabled, `forge docker` exits with a helpful message:

```
docker configuration not enabled — forge image requires docker
to be enabled in forge.yaml
```

## JSON Output

```bash
forge --json docker
```

Returns build details in JSON format for pipeline integration.
