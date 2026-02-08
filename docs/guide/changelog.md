# Changelog Generation

Forge generates changelogs from your git commit history using the [Conventional Commits](https://www.conventionalcommits.org/) specification.

## Basic Usage

```bash
# Generate changelog from last tag to HEAD
forge changelog

# Between two specific tags
forge changelog --from v1.0.0 --to v1.1.0

# Save to file
forge changelog --output CHANGELOG.md
```

## Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--from` | `-f` | Starting tag | latest tag |
| `--to` | `-t` | Ending tag or commit | `HEAD` |
| `--format` | `--fmt` | Output format: `markdown`, `json`, `plain` | `markdown` |
| `--output` | `-o` | Output file path (stdout if omitted) | |
| `--app` | `-a` | Application name (for monorepos) | |

## Conventional Commits

Forge parses commit messages following the Conventional Commits format:

```
<type>(<scope>): <subject>

[optional body]

[optional footer(s)]
```

### Supported Commit Types

| Type | Description | Changelog Section |
|------|-------------|-------------------|
| `feat` | New features | 🚀 Features |
| `fix` | Bug fixes | 🐛 Bug Fixes |
| `docs` | Documentation changes | 📚 Documentation |
| `style` | Code style changes | 💎 Styles |
| `refactor` | Code refactoring | ♻️ Code Refactoring |
| `perf` | Performance improvements | ⚡ Performance |
| `test` | Test changes | 🧪 Tests |
| `build` | Build system changes | 🏗️ Build |
| `ci` | CI/CD changes | 👷 CI |
| `chore` | Maintenance tasks | 🔧 Chores |

### Breaking Changes

Breaking changes are highlighted in the changelog. Mark them in two ways:

**With `!` after the type:**
```
feat(api)!: change response format
```

**With a `BREAKING CHANGE` footer:**
```
feat(api): change response format

BREAKING CHANGE: the response now returns an array instead of an object
```

### Scopes

Scopes are optional and appear in bold in the changelog:

```
feat(auth): add OAuth2 support
fix(parser): handle empty strings
docs: update README
```

### PR Numbers

Pull request numbers in the format `(#123)` are automatically detected:

```
feat(api): add pagination support (#42)
```

## Output Formats

### Markdown (default)

```bash
forge changelog --format markdown
```

Produces GitHub-flavored Markdown:

```markdown
# v1.1.0 (v1.0.0...v1.1.0)

*2025-01-15*

## ⚠ BREAKING CHANGES

* **api:** change response format (abc1234)

## 🚀 Features

* **auth:** add OAuth2 support (#42) (def5678)
* add dark mode (ghi9012)

## 🐛 Bug Fixes

* **parser:** handle empty strings (jkl3456)
```

### JSON

```bash
forge changelog --format json
```

Machine-readable output for tooling integration:

```json
{
  "from_tag": "v1.0.0",
  "to_tag": "v1.1.0",
  "commits": [
    {
      "hash": "abc1234...",
      "type": "feat",
      "scope": "auth",
      "subject": "add OAuth2 support",
      "breaking": false
    }
  ]
}
```

### Plain Text

```bash
forge changelog --format plain
```

Simple text output suitable for terminal display.

## Monorepo Usage

Use `--app` to scope the changelog to a specific application:

```bash
forge changelog --app api --from api/v1.0.0 --to api/v1.1.0
```

## Workflow Example

A typical release workflow with changelogs:

```bash
# 1. Preview changes since last release
forge changelog --from v1.2.3 --to HEAD

# 2. Create the version tag
forge bump --bump minor --push

# 3. Generate the changelog
forge changelog --from v1.2.3 --to v1.3.0 --output CHANGELOG.md

# 4. Commit the changelog
git add CHANGELOG.md
git commit -m "chore: update changelog for v1.3.0"
git push
```
