# Building Binaries

Forge includes basic Go cross-compilation support for building binaries for multiple platforms.

::: warning Recommendation
For production releases, use [GoReleaser](https://goreleaser.com). Forge's build features are basic and intended for simple use cases. Forge excels at **version management** — use it for tagging, then let GoReleaser handle the build and release process.
:::

## Basic Usage

```bash
forge build
```

This builds the binary for all configured target platforms and places the output in the configured output directory.

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--targets` | Comma-separated list of OS/ARCH targets | from config |
| `--ldflags` | Ldflags template string | from config |
| `--out` | Output directory | `dist` |
| `--version` | Version string (auto-detected from git tag) | auto |
| `--repo-dir` | Repository directory | `.` |
| `--app` | Target app in a monorepo | default |
| `--dry-run` | Show what would be done | `false` |

## Configuration

### Single Binary

```yaml
build:
  name: myapp
  main_path: ./cmd/main.go
  targets:
    - linux/amd64
    - linux/arm64
    - darwin/amd64
    - darwin/arm64
    - windows/amd64
  ldflags: "-s -w -X main.version={{ .Version }}"
  output_dir: dist
```

### Multiple Binaries

For projects that produce multiple binaries from a single repository:

```yaml
build:
  targets:
    - linux/amd64
    - linux/arm64
    - darwin/amd64
    - darwin/arm64
  ldflags: "-s -w -X main.version={{ .Version }}"
  output_dir: dist
  binaries:
    - name: myapp
      path: ./cmd/myapp
    - name: myapp-cli
      path: ./cmd/cli
      ldflags: "-s -w -X main.version={{ .Version }} -X main.appName=cli"
```

Each binary can have its own `ldflags` override. If omitted, the top-level `ldflags` is used.

## Ldflags Templates

The `ldflags` field supports Go template syntax with these variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `{{ .Version }}` | Current version string | `1.2.3` |
| `{{ .Commit }}` | Full commit hash | `abc123def456...` |
| `{{ .ShortCommit }}` | Short commit hash (8 chars) | `abc123de` |
| `{{ .Date }}` | Build date (UTC) | `2025-01-15` |
| `{{ .OS }}` | Target operating system | `linux` |
| `{{ .Arch }}` | Target architecture | `amd64` |

Example:

```yaml
ldflags: >-
  -s -w
  -X main.version={{ .Version }}
  -X main.commit={{ .ShortCommit }}
  -X main.date={{ .Date }}
```

## Version Detection

If `--version` is not provided, Forge auto-detects the version from the latest git tag. If the working directory is dirty (uncommitted changes), a `-dirty` suffix is appended:

```
1.2.3         # Clean working directory
1.2.3-dirty   # Uncommitted changes present
```

## JSON Output

```bash
forge --json build
```

Returns build results in JSON format for CI/CD integration.

## Target Format

Targets follow the Go `GOOS/GOARCH` convention:

```
linux/amd64
linux/arm64
darwin/amd64
darwin/arm64
windows/amd64
windows/arm64
```

See the [Go documentation](https://go.dev/doc/install/source#environment) for all supported OS/ARCH combinations.
