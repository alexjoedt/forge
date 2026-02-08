# Installation

## Using `go install` (Recommended)

The simplest way to install Forge is via Go:

```bash
go install github.com/alexjoedt/forge@latest
```

This installs the `forge` binary to your `$GOPATH/bin` directory. Make sure it's in your `$PATH`.

## Building from Source

Clone the repository and build manually:

```bash
git clone https://github.com/alexjoedt/forge.git
cd forge
go build -o forge .
```

Move the binary to a directory in your `$PATH`:

```bash
mv forge /usr/local/bin/
```

## Verify Installation

After installation, verify that Forge is available:

```bash
forge --version
```

You should see output like:

```
forge version 1.x.x
commit:  abc1234
built:   2025-01-15
```

## Shell Completions

Forge uses [urfave/cli](https://github.com/urfave/cli), which supports shell completions. Refer to the urfave/cli documentation for setting up completions for your shell.

## Next Steps

- [Quick Start →](./quick-start)
