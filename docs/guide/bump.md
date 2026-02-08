# Bump Command

The `bump` command is the primary way to create version tags. It calculates the next version, creates an annotated git tag, and optionally pushes it to the remote.

## Basic Usage

```bash
# Interactive mode (TTY only)
forge bump

# Explicit bump type
forge bump --bump patch
forge bump --bump minor
forge bump --bump major
```

The `bump` command also has the alias `tag`, so these are equivalent:

```bash
forge bump --bump minor
forge tag --bump minor
```

## Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--bump` | | SemVer bump type: `major`, `minor`, or `patch` | `patch` |
| `--initial` | `-i` | Create the first version tag (e.g., `--initial 1.0.0`) | |
| `--scheme` | | Override version scheme: `semver` or `calver` | from config |
| `--calver-format` | | Override CalVer format string | from config |
| `--prefix` | | Override tag prefix | from config |
| `--push` | | Push the tag to remote after creation | `false` |
| `--force` | | Create tag even with uncommitted changes | `false` |
| `--dry-run` | | Show what would happen without creating a tag | `false` |
| `--app` | | Target app in a monorepo | from `defaultApp` |
| `--repo-dir` | | Path to the repository directory | `.` |
| `--pre` | | *[ALPHA]* Prerelease identifier | |
| `--meta` | | *[ALPHA]* Build metadata | |

## Creating the First Tag

When no version tags exist yet, Forge guides you:

```bash
forge bump
```

```
Error: No version tags found

  This appears to be the first version tag for this project.

  Suggestions:
    • Create your first tag: forge bump --initial 1.0.0
    • Or use: forge bump --initial to use default (1.0.0)
    • Or manually: git tag v1.0.0 && git push --tags
```

Create the initial tag:

```bash
forge bump --initial 1.0.0 --push
```

## Git State Checks

By default, Forge refuses to create a tag if the working directory has uncommitted changes:

```
Error: Working directory has uncommitted changes

  Git working directory is not clean. Forge requires a clean
  state before creating version tags.

  Suggestions:
    • Commit your changes: git add . && git commit -m 'Your message'
    • Stash your changes: git stash
    • Use --force to create tag anyway (not recommended)
```

Use `--force` to skip this check:

```bash
forge bump --bump patch --force
```

## Dry Run

Preview the tag that would be created without actually creating it:

```bash
forge bump --bump minor --dry-run
```

## JSON Output

For scripting and CI/CD pipelines, use the global `--json` flag:

```bash
forge --json bump --bump minor
```

```json
{
  "tag": "v1.3.0",
  "pushed": false,
  "version": "v1.3.0",
  "message": "Tag created"
}
```

## Monorepo Usage

In a monorepo, specify the app with `--app`:

```bash
forge bump --bump minor --app api --push
forge bump --bump patch --app worker
```

If `defaultApp` is configured, omitting `--app` targets the default.

## Node.js Integration

When Node.js integration is enabled in your config, the `bump` command automatically updates `package.json` before creating the tag:

```yaml
nodejs:
  enabled: true
  package_path: ""  # defaults to ./package.json
```

Forge will:
1. Update the `version` field in `package.json`
2. Stage and commit the change
3. Create the git tag on the new commit

See [Node.js Integration](./nodejs) for details.
