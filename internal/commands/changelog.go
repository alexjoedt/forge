package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/alexjoedt/forge/internal/changelog"
	"github.com/alexjoedt/forge/internal/config"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/urfave/cli/v3"
)

// Changelog returns the changelog command
func Changelog() *cli.Command {
	return &cli.Command{
		Name:  "changelog",
		Usage: "Generate changelog from git commit history",
		Description: `Generate a formatted changelog from git commits between two tags.

Supports Conventional Commits format (https://www.conventionalcommits.org/):
  - feat: New feature
  - fix: Bug fix
  - docs: Documentation changes
  - style: Code style changes
  - refactor: Code refactoring
  - perf: Performance improvements
  - test: Test changes
  - build: Build system changes
  - ci: CI/CD changes
  - chore: Maintenance tasks

Examples:
  # Generate changelog from last tag to HEAD
  forge changelog

  # Generate changelog between two tags
  forge changelog --from v1.0.0 --to v1.1.0

  # Output as JSON
  forge changelog --format json

  # Save to file
  forge changelog --output CHANGELOG.md

  # Multi-app changelog
  forge changelog --app api --from api/v1.0.0`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "from",
				Aliases: []string{"f"},
				Usage:   "Starting tag (defaults to latest tag)",
			},
			&cli.StringFlag{
				Name:    "to",
				Aliases: []string{"t"},
				Usage:   "Ending tag or commit (defaults to HEAD)",
				Value:   "HEAD",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"fmt"},
				Usage:   "Output format (markdown, json, plain)",
				Value:   "markdown",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file (defaults to stdout)",
			},
			&cli.StringFlag{
				Name:    "app",
				Aliases: []string{"a"},
				Usage:   "Application name (for multi-app repos)",
			},
		},
		Action: changelogAction,
	}
}

func changelogAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	repoDir := "."
	app := cmd.String("app")

	// Validate environment
	if err := ValidateRequirements(ctx, repoDir); err != nil {
		return err
	}

	// Load config
	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Get app config
	appConfig, err := cfg.GetAppConfig(app)
	if err != nil {
		return err
	}

	// Get tags
	from := cmd.String("from")
	to := cmd.String("to")
	format := cmd.String("format")
	output := cmd.String("output")

	// If no from tag specified, use latest tag
	if from == "" {
		// TODO: Get latest tag from git
		logger.Warnf("No --from tag specified, using all commits up to HEAD")
	}

	// Validate format
	var changelogFormat changelog.Format
	switch format {
	case "markdown", "md":
		changelogFormat = changelog.MarkdownFormat
	case "json":
		changelogFormat = changelog.JSONFormat
	case "plain", "text":
		changelogFormat = changelog.PlainFormat
	default:
		return fmt.Errorf("unsupported format: %s (use markdown, json, or plain)", format)
	}

	// Parse commits
	logger.Infof("Parsing git commits...")
	parser := changelog.NewParser(repoDir, appConfig.Git.TagPrefix)
	
	cl, err := parser.Parse(ctx, from, to)
	if err != nil {
		return fmt.Errorf("parse changelog: %w", err)
	}

	if len(cl.Commits) == 0 {
		logger.Warnf("No commits found in range")
		return nil
	}

	logger.Infof("Found %d commits", len(cl.Commits))

	// Format changelog
	var formatted string
	switch changelogFormat {
	case "markdown":
		formatted = changelog.FormatMarkdown(cl)
	case "json":
		formatted, err = changelog.FormatJSON(cl)
		if err != nil {
			return fmt.Errorf("format JSON: %w", err)
		}
	case "plain":
		formatted = changelog.FormatPlain(cl)
	}

	// Output
	if output != "" {
		if err := os.WriteFile(output, []byte(formatted), 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		logger.Success("Changelog written to %s", output)
	} else {
		fmt.Println(formatted)
	}

	return nil
}
