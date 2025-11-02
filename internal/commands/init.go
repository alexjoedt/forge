package commands

import (
	"context"
	"fmt"

	"github.com/alexjoedt/forge/internal/initialize"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/output"
	"github.com/urfave/cli/v3"
)

// Init returns the init command that initializes a new forge.yaml configuration file.
func Init() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize forge.yaml for version management",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "output path for the config file",
				Value:   "forge.yaml",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "overwrite existing config file",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "show what would be done without doing it",
			},
			&cli.BoolFlag{
				Name:  "multi",
				Usage: "initialzises a configuration for multiple apps",
			},
		},
		Action: initAction,
	}
}

func initAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	outputPath := cmd.String("output")
	force := cmd.Bool("force")
	dryRun := cmd.Bool("dry-run")
	multi := cmd.Bool("multi")

	logger.Debugf("initializing forge configuration: %s", outputPath)

	opts := initialize.Options{
		OutputPath: outputPath,
		Force:      force,
		DryRun:     dryRun,
		Multi:      multi,
	}

	if err := initialize.Init(ctx, opts); err != nil {
		return fmt.Errorf("initialize config: %w", err)
	}

	// Output based on format
	if out.IsJSON() {
		result := output.InitResult{
			OutputPath: outputPath,
			Created:    !dryRun,
			Message:    fmt.Sprintf("%sCreated forge configuration", map[bool]string{true: "Would ", false: ""}[dryRun]),
		}
		return out.Print(result)
	}

	if !dryRun {
		logger.Success("Created forge configuration at %s", outputPath)
	} else {
		logger.Success("Would create forge configuration at %s", outputPath)
	}

	return nil
}
