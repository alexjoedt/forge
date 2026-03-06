# Forge — Project Guidelines

`forge` is a CLI tool for automated git version tagging, SemVer/CalVer bumping, changelog generation, and hotfix branching. Built in Go using `urfave/cli/v3`.

## Build & Test

Uses [Taskfile](../Taskfile.yml) — requires `task` CLI.

```bash
task test      # go test --cover ./... && go mod verify
task tidy      # go fmt ./... && go mod tidy
task build     # goreleaser build --snapshot --clean --single-target (requires goreleaser)
task install   # build + copy to ~/.local/bin
```

Run tests for a specific package: `go test ./internal/version/...`

## Architecture

| Package | Responsibility |
|---|---|
| `main` | Wires CLI app, injects logger + output manager into `context.Context` via `Before` hook |
| `internal/commands` | One file per command: `tag.go` (bump), `changelog.go`, `hotfix.go`, `version.go`, `init.go`, `validate.go`, `retag.go`; `common.go` holds shared helpers and `ForgeError` |
| `internal/config` | Loads `forge.yaml` / `.forge.yaml`; single-app and monorepo configs |
| `internal/version` | Pure version math: `ParseSemVer`, `ParseCalVer`, `BumpSemVer`, `BumpCalVer` |
| `internal/git` | `Tagger` struct — wraps `git tag` operations |
| `internal/changelog` | `Parser` (git log → Conventional Commits) + `format.go` |
| `internal/run` | Thin `exec.Cmd` wrapper; all shell calls use `run.CmdInDir()` returning `Result{Stdout, Stderr, ExitCode}` |
| `internal/log` | Context-keyed logger (`log.FromContext`, `log.WithLogger`) |
| `internal/output` | Context-keyed output manager; `FormatText` / `FormatJSON`; result structs live here |
| `internal/interactive` | Bubble Tea TUI for interactive bump-type selection |
| `internal/nodejs` | Reads/writes `package.json` version on bump |

## Conventions

**Context threading** — `context.Context` is the first parameter of every function. Logger and output manager are retrieved via `log.FromContext(ctx)` and `output.FromContext(ctx)`; never use global instances directly.

**Command factories** — Each CLI command is a constructor `func() *cli.Command`. No global state; inject dependencies via context in the `Before` hook.

**Shell execution** — All `os/exec` calls go through `internal/run`. Never call `exec.Cmd` directly in business logic. Check results with `result.Success()` or `result.MustSucceed(msg)`.

**Error types** — User-facing errors are `ForgeError{Title, Description, Suggestions}` (in `internal/commands/common.go`) for actionable CLI messages.

**Dry-run** — Every mutating command supports `--dry-run`. The `Tagger` short-circuits writes when `dryRun == true`. New commands must honour this flag.

**Config override chain** — CLI flags override `forge.yaml`; never read from config when a flag is set.

**CalVer format strings** — Use Go's reference time (`2006` = year, `01` = month, `02` = day). `WW` is a custom extension for ISO week number.

**Struct tags** — Config structs use YAML tags only. Output result types (`TagResult`, `VersionResult`, etc.) use JSON tags.

## Testing Patterns

- Table-driven tests with `t.Run` subtests (see `internal/version/scheme_test.go`, `internal/config/config_test.go`)
- Prefer testing pure functions in `internal/version` and `internal/config` without mocking
- `internal/git` and `internal/run` tests may require a real git repo; set up a temp dir when needed

## Configuration (`forge.yaml`)

```yaml
# Single-app
scheme: semver       # or: calver
prefix: v
default_branch: main
calver_format: "2006.WW"   # CalVer only
nodejs:
  enabled: false

# Monorepo — flat map of app configs
defaultApp: api
api:
  scheme: semver
  prefix: api/v
worker:
  scheme: calver
  calver_format: "2006.WW"
  prefix: worker/v
```

> `pre` and `meta` fields exist in structs but are **ALPHA** — not production-ready.
