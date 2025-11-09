package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alexjoedt/forge/internal/commands"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/output"
	"github.com/urfave/cli/v3"
)

var (
	// version is set via ldflags at build time
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

func init() {
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Fprintf(cmd.Root().Writer, "%s version %s\n", cmd.Name, version)
		if commit != "" {
			fmt.Fprintf(cmd.Root().Writer, "commit:  %s\n", commit)
		}
		if date != "" {
			fmt.Fprintf(cmd.Root().Writer, "built:   %s\n", date)
		}
		if builtBy != "" {
			fmt.Fprintf(cmd.Root().Writer, "by:      %s\n", builtBy)
		}
	}
}

func main() {
	cmd := &cli.Command{
		Name:    "forge",
		Usage:   "Git version management for Go projects and monorepos",
		Version: version,
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
			// Setup logging
			verbose := c.Bool("verbose")
			jsonOutput := c.Bool("json")

			// When JSON output is enabled, suppress verbose logging
			if jsonOutput {
				verbose = false
			}
			log.Setup(verbose)

			// Setup output manager
			var format output.Format
			if jsonOutput {
				format = output.FormatJSON
			} else {
				format = output.FormatText
			}
			outputManager := output.New(format)
			ctx = output.WithManager(ctx, outputManager)

			return ctx, nil
		},
		Commands: []*cli.Command{
			commands.Init(),
			commands.Bump(),
			commands.Hotfix(),
			commands.Build(),
			commands.Docker(),
			commands.Version(),
			commands.Changelog(),
			commands.Validate(),
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.DefaultLogger.Errorf("command failed: %v", err)
		os.Exit(1)
	}
}
