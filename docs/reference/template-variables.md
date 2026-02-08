# Template Variables

Forge uses Go template syntax in several configuration fields. This page lists all available template variables and where they can be used.

## Available Variables

| Variable | Description | Example Value |
|----------|-------------|---------------|
| `{{ .Version }}` | Current version string (without prefix) | `1.2.3` |
| `{{ .Commit }}` | Full git commit hash | `abc123def456789...` |
| `{{ .ShortCommit }}` | Short commit hash (8 characters) | `abc123de` |
| `{{ .Date }}` | Build date in UTC (`YYYY-MM-DD`) | `2025-01-15` |
| `{{ .OS }}` | Target operating system | `linux` |
| `{{ .Arch }}` | Target architecture | `amd64` |
| `{{ .CalVer }}` | CalVer date string (if using CalVer) | `2025.44` |

## Where Templates Are Used

### `build.ldflags`

Linker flags for Go builds:

```yaml
build:
  ldflags: "-s -w -X main.version={{ .Version }} -X main.commit={{ .ShortCommit }} -X main.date={{ .Date }}"
```

This compiles to:
```
-s -w -X main.version=1.2.3 -X main.commit=abc123de -X main.date=2025-01-15
```

### `build.binaries[].ldflags`

Per-binary ldflags override:

```yaml
build:
  binaries:
    - name: myapp
      path: ./cmd/myapp
      ldflags: "-X main.version={{ .Version }} -X main.appName=myapp"
```

### `docker.tags`

Docker image tag templates:

```yaml
docker:
  tags:
    - "{{ .Version }}"
    - "latest"
    - "{{ .ShortCommit }}"
```

With version `1.2.3` and commit `abc123de`, this produces:
```
ghcr.io/user/app:1.2.3
ghcr.io/user/app:latest
ghcr.io/user/app:abc123de
```

## Common Patterns

### Standard Go ldflags

```yaml
ldflags: >-
  -s -w
  -X main.version={{ .Version }}
  -X main.commit={{ .ShortCommit }}
  -X main.date={{ .Date }}
```

### Docker tags with version variations

```yaml
docker:
  tags:
    - "{{ .Version }}"
    - "latest"
```

### Build metadata in ldflags

```yaml
ldflags: >-
  -X main.version={{ .Version }}
  -X main.commit={{ .Commit }}
  -X main.builtBy=forge
```
