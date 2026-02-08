# Getting Started

Forge is a CLI tool for automated git version tagging and changelog generation. It is designed for Go projects and monorepos, supporting both **Semantic Versioning** (SemVer) and **Calendar Versioning** (CalVer).

## What Forge Does

Forge manages the full version lifecycle of your project:

- **Creates and manages git tags** following SemVer or CalVer conventions
- **Interactive version bumping** with preview in your terminal
- **Hotfix workflows** for patching released versions without touching `main`
- **Monorepo support** with per-app versioning and namespaced tags
- **Changelog generation** from Conventional Commits
- **Go binary builds** for multiple platforms *(basic — use [GoReleaser](https://goreleaser.com) for production)*
- **Docker image builds** with multi-registry and multi-platform support
- **Node.js integration** for syncing `package.json` versions

## How It Works

Forge reads a `forge.yaml` configuration file in your project root and uses git tags to track versions. When you run `forge bump`, it:

1. Reads the latest git tag matching your configured prefix
2. Calculates the next version based on your chosen scheme
3. Creates an annotated git tag
4. Optionally pushes the tag to the remote

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  forge.yaml  │────▶│  forge bump  │────▶│   git tag    │
│  (config)    │     │  (calculate) │     │   (create)   │
└──────────────┘     └──────────────┘     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  git push    │
                     │  (optional)  │
                     └──────────────┘
```

## Prerequisites

- **Go** 1.21 or later (for installation via `go install`)
- **Git** repository initialized in your project
- A terminal that supports interactive prompts (optional, for interactive mode)

## Next Steps

- [Install Forge →](./installation)
- [Quick Start →](./quick-start)
- [Configuration Reference →](../reference/configuration)
