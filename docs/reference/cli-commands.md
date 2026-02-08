# CLI Commands

Complete reference for all Forge CLI commands and their flags.

## Global Flags

Available on every command:

```bash
forge [global flags] <command> [command flags]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable debug logging |
| `--json` | | Output results in JSON format |
| `--version` | | Print Forge version |
| `--help` | `-h` | Show help text |

---

## `forge init`

Initialize a new `forge.yaml` configuration file.

```bash
forge init [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--output` | `-o` | Output path for the config file | `forge.yaml` |
| `--force` | | Overwrite existing config file | `false` |
| `--multi` | | Generate multi-app (monorepo) config | `false` |
| `--dry-run` | | Preview without creating the file | `false` |

**Examples:**

```bash
forge init                     # Create forge.yaml
forge init --multi             # Create monorepo config
forge init -o .forge.yaml      # Custom output path
forge init --force             # Overwrite existing
```

---

## `forge bump`

Bump the version and create a git tag. Alias: `forge tag`.

```bash
forge bump [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--bump` | | SemVer bump type: `major`, `minor`, `patch` | `patch` |
| `--initial` | `-i` | Create initial version tag | |
| `--scheme` | | Override version scheme | from config |
| `--calver-format` | | Override CalVer format | from config |
| `--prefix` | | Override tag prefix | from config |
| `--push` | | Push tag to remote | `false` |
| `--force` | | Skip git clean check | `false` |
| `--dry-run` | | Preview without creating tag | `false` |
| `--app` | | Target app (monorepo) | `defaultApp` |
| `--repo-dir` | | Repository directory | `.` |
| `--pre` | | *[ALPHA]* Prerelease identifier | |
| `--meta` | | *[ALPHA]* Build metadata | |

**Examples:**

```bash
forge bump                             # Interactive mode
forge bump --bump minor --push         # Bump minor, push
forge bump --initial 1.0.0 --push      # First tag
forge bump --bump patch --dry-run      # Preview only
forge bump --bump minor --app api      # Monorepo
```

---

## `forge version`

Show the current version from git tags.

```bash
forge version [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--repo-dir` | Repository directory | `.` |
| `--app` | Target app (monorepo) | `defaultApp` |

**Output (single app):**
```
Current Version: v1.2.3
Scheme:          semver
Commit:          abc1234f
```

**Output (monorepo, no --app):**
```
 App       Current    Scheme   Last Tag      Date              Commit
 api       1.2.3      semver   api/v1.2.3    2025-01-15        abc12345
 worker    2025.44    calver   worker/v...   2025-01-14        def67890
```

### `forge version list`

List all version tags in history. Alias: `forge version ls`.

```bash
forge version list [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--limit` | `-n` | Limit number of versions | all |
| `--repo-dir` | | Repository directory | `.` |
| `--app` | | Target app | `defaultApp` |

### `forge version next`

Preview the next version without creating a tag.

```bash
forge version next [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--bump` | SemVer bump type | `patch` |
| `--scheme` | Override version scheme | from config |
| `--calver-format` | Override CalVer format | from config |
| `--pre` | Prerelease identifier | |
| `--meta` | Build metadata | |
| `--repo-dir` | Repository directory | `.` |
| `--app` | Target app | `defaultApp` |

---

## `forge hotfix`

Manage hotfix branches and versions.

### `forge hotfix create`

Create a hotfix branch from a release tag.

```bash
forge hotfix create <base-tag> [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--app` | `-a` | App name (auto-detected from tag) | |
| `--no-checkout` | | Don't switch to the branch | `false` |
| `--dry-run` | | Preview without making changes | `false` |

### `forge hotfix bump`

Create a hotfix tag on the current hotfix branch.

```bash
forge hotfix bump [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--base` | `-b` | Create branch from tag + bump in one step | |
| `--message` | `-m` | Custom tag message | `Hotfix <tag>` |
| `--push` | | Push tag to remote | `false` |
| `--dry-run` | | Preview without making changes | `false` |

### `forge hotfix status`

Show the current hotfix branch status.

```bash
forge hotfix status
```

### `forge hotfix list`

List all hotfix tags for a base version.

```bash
forge hotfix list [base-tag] [flags]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--app` | `-a` | Filter by app name |

---

## `forge changelog`

Generate a changelog from git commit history.

```bash
forge changelog [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--from` | `-f` | Starting tag | latest tag |
| `--to` | `-t` | Ending tag or commit | `HEAD` |
| `--format` | `--fmt` | Output format: `markdown`, `json`, `plain` | `markdown` |
| `--output` | `-o` | Output file path | stdout |
| `--app` | `-a` | Application name (monorepo) | |

**Examples:**

```bash
forge changelog                                          # Since last tag
forge changelog --from v1.0.0 --to v1.1.0                # Between tags
forge changelog --format json                            # JSON output
forge changelog --output CHANGELOG.md                    # Save to file
forge changelog --app api --from api/v1.0.0              # Monorepo
```

---

## `forge build`

Build Go binaries for multiple platforms.

```bash
forge build [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--targets` | Comma-separated OS/ARCH targets | from config |
| `--ldflags` | Ldflags template string | from config |
| `--out` | Output directory | `dist` |
| `--version` | Version string | auto-detected |
| `--repo-dir` | Repository directory | `.` |
| `--app` | Target app (monorepo) | `defaultApp` |
| `--dry-run` | Preview without building | `false` |

---

## `forge docker`

Build and push Docker images. Alias: `forge image`.

```bash
forge docker [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--dockerfile` | Path to Dockerfile | `./Dockerfile` |
| `--context` | Build context path | `.` |
| `--repository` | Image repository (overrides config) | from config |
| `--tags` | Comma-separated tag templates | from config |
| `--platforms` | Comma-separated platforms | from config |
| `--build-arg` | Build argument (`key=value`, repeatable) | |
| `--push` | Push image to registry | `false` |
| `--version` | Version string | auto-detected |
| `--repo-dir` | Repository directory | `.` |
| `--app` | Target app (monorepo) | `defaultApp` |
| `--dry-run` | Preview without building | `false` |

---

## `forge validate`

Validate configuration and git repository state.

```bash
forge validate [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--repo-dir` | Repository directory | `.` |
| `--app` | Target app (monorepo) | `defaultApp` |

Checks performed:
- Git repository exists
- `forge.yaml` is valid and loadable
- Version scheme is valid
- CalVer format is set (if using CalVer)
- Git tag prefix is configured
- Working directory state
- Existing version tags

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Configuration error / missing feature |
