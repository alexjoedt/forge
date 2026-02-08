# Interactive Mode

Forge provides an interactive terminal UI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a smooth developer experience.

## How It Works

Interactive mode activates automatically when:

- Running in a **TTY** (not piped or redirected)
- No explicit `--bump` flag is provided
- JSON output mode (`--json`) is **not** enabled

Simply run:

```bash
forge bump
```

## Bump Type Selection

When interactive mode is active and the version scheme is SemVer, Forge presents a selection prompt with version previews:

```
? Select version bump type:
  ❯ patch — bug fixes and patches              (v1.2.3 → v1.2.4)
    minor — new features, backwards compatible  (v1.2.3 → v1.3.0)
    major — breaking changes                    (v1.2.3 → v2.0.0)
```

Use the **arrow keys** to navigate and **Enter** to confirm your selection.

## Confirmation Prompt

After selecting a bump type, Forge shows a confirmation before creating the tag:

```
Current: v1.2.3 → Next: v1.2.4
? Create this tag? (y/N)
```

This prevents accidental tag creation.

## Disabling Interactive Mode

There are several ways to run Forge non-interactively:

### Provide the `--bump` flag explicitly

```bash
forge bump --bump patch
```

### Use JSON output

```bash
forge --json bump --bump minor
```

### Pipe the output

```bash
forge bump --bump patch | cat
```

### CI/CD environments

In CI/CD, there's no TTY, so interactive mode is automatically disabled. Use explicit flags:

```bash
forge bump --bump minor --push --force
```

::: tip
For CI/CD pipelines, combine `--force` (skip git clean check) with `--push` and an explicit `--bump` type for fully automated releases.
:::
