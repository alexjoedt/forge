package commands

import (
	"context"
	"fmt"
	"sync"

	"github.com/urfave/cli/v3"

	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/output"
)

// BuildInfo holds the build metadata shown by `forge --version`.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}

const buildInfoKey = "buildInfo"

//nolint:gochecknoglobals // cli.VersionPrinter is a package global of urfave/cli.
var versionPrinterOnce sync.Once

// Root returns the root forge command with all subcommands registered.
func Root(info BuildInfo) *cli.Command {
	versionPrinterOnce.Do(func() {
		cli.VersionPrinter = printVersion //nolint:reassign // urfave/cli exposes no other hook
	})

	return &cli.Command{
		Name:     "forge",
		Usage:    "Git version management for Go projects and monorepos",
		Version:  info.Version,
		Metadata: map[string]any{buildInfoKey: info},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "enable verbose logging (debug level)",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "output results in JSON format for scripting",
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			jsonOutput := c.Bool("json")
			// When JSON output is enabled, suppress verbose logging.
			verbose := c.Bool("verbose") && !jsonOutput

			root := c.Root()
			format, logOut := output.FormatText, root.Writer
			if jsonOutput {
				// Keep stdout pure JSON: log lines go to stderr.
				format, logOut = output.FormatJSON, root.ErrWriter
			}
			ctx = output.WithManager(ctx, output.New(format, root.Writer, root.ErrWriter))
			ctx = log.WithLogger(ctx, log.New(logOut, verbose))

			return ctx, nil
		},
		Commands: []*cli.Command{
			Init(),
			Bump(),
			Hotfix(),
			Version(),
			Changelog(),
			Retag(),
			Validate(),
		},
	}
}

func printVersion(cmd *cli.Command) {
	root := cmd.Root()
	info, _ := root.Metadata[buildInfoKey].(BuildInfo)
	w := root.Writer
	fmt.Fprintf(w, "%s version %s\n", root.Name, root.Version)
	if info.Commit != "" {
		fmt.Fprintf(w, "commit:  %s\n", info.Commit)
	}
	if info.Date != "" {
		fmt.Fprintf(w, "built:   %s\n", info.Date)
	}
	if info.BuiltBy != "" {
		fmt.Fprintf(w, "by:      %s\n", info.BuiltBy)
	}
}
