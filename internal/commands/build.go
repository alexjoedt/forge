package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alexjoedt/forge/internal/build"
	"github.com/alexjoedt/forge/internal/config"
	"github.com/alexjoedt/forge/internal/git"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/output"
	"github.com/urfave/cli/v3"
)

// Build returns the build command that builds binaries for multiple platforms.
func Build() *cli.Command {
	return &cli.Command{
		Name:  "build",
		Usage: "Build Go binaries for multiple platforms (optional)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "targets",
				Usage: "comma-separated list of OS/ARCH targets",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "ldflags",
				Usage: "ldflags template string",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "out",
				Usage: "output directory",
				Value: "dist",
			},
			&cli.StringFlag{
				Name:  "repo-dir",
				Usage: "repository directory",
				Value: ".",
			},
			&cli.StringFlag{
				Name:  "version",
				Usage: "version string (if empty, detected from git tag)",
				Value: "",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "show what would be done without doing it",
			},
			appFlag,
		},
		Action: buildAction,
	}
}

func buildAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	repoDir := cmd.String("repo-dir")
	outputDir := cmd.String("out")
	dryRun := cmd.Bool("dry-run")

	// Validate requirements
	if err := ValidateRequirements(ctx, repoDir); err != nil {
		return err
	}

	// Load config
	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	appName := cmd.String("app")
	appConfig, err := cfg.GetAppConfig(appName)
	if err != nil {
		return err
	}

	// Check if build configuration is present
	if len(appConfig.Build.Targets) == 0 && cmd.String("targets") == "" {
		return cli.Exit("build configuration not found - forge build requires build targets to be configured in forge.yaml\nYou can use forge for version management only with 'forge bump' and 'forge version' commands", 2)
	}

	// Get targets
	targetsStr := cmd.String("targets")
	var targets []string
	if targetsStr != "" {
		targets = strings.Split(targetsStr, ",")
	} else {
		targets = appConfig.Build.Targets
	}

	// Get ldflags
	ldflags := cmd.String("ldflags")
	if ldflags == "" {
		ldflags = appConfig.Build.LDFlags
	}

	tagPrefix := appConfig.Git.TagPrefix

	// Get version
	versionStr := cmd.String("version")
	if versionStr == "" {
		// Try to detect from git tag with dirty check
		tagger := git.NewTagger(repoDir, tagPrefix, false)
		detectedVersion, err := tagger.GetVersionWithDirtyCheck(ctx)
		if err != nil {
			logger.Warnf("failed to detect version from git, using default: %v", err)
			versionStr = "0.0.0-dev"
		} else {
			versionStr = detectedVersion
		}
	}

	// Get commit info
	tagger := git.NewTagger(repoDir, tagPrefix, false)
	commit, err := tagger.CurrentCommit(ctx)
	if err != nil {
		logger.Warnf("failed to get commit: %v", err)
		commit = "unknown"
	}

	shortCommit, err := tagger.ShortCommit(ctx)
	if err != nil {
		logger.Warnf("failed to get short commit: %v", err)
		shortCommit = "unknown"
	}

	// Get current date
	date := time.Now().UTC().Format("2006-01-02")

	logger.Debugf("building version '%s' for %d targets", versionStr, len(targets))

	// Create builder
	builder := build.NewBuilder(repoDir, outputDir, dryRun)

	var binaryNames []string

	// Check if we have multiple binaries configured
	if len(appConfig.Build.Binaries) > 0 {
		// Multi-binary build
		logger.Debugf("multi-binary build with %d binaries", len(appConfig.Build.Binaries))

		binaries := make([]build.BinaryBuildSpec, len(appConfig.Build.Binaries))
		for i, bin := range appConfig.Build.Binaries {
			binaries[i] = build.BinaryBuildSpec{
				Name:    bin.Name,
				Path:    bin.Path,
				LDFlags: bin.LDFlags,
			}
			binaryNames = append(binaryNames, bin.Name)
		}

		buildOpts := build.BuildMultiOptions{
			MainPath:    appConfig.Build.MainPath,
			Targets:     targets,
			Binaries:    binaries,
			LDFlags:     ldflags,
			Version:     versionStr,
			Commit:      commit,
			ShortCommit: shortCommit,
			Date:        date,
		}

		if err := builder.BuildMulti(ctx, buildOpts); err != nil {
			return fmt.Errorf("build: %w", err)
		}
	} else {
		// Single binary build (backward compatibility)
		buildOpts := build.BuildAllOptions{
			Targets:     targets,
			LDFlags:     ldflags,
			Version:     versionStr,
			Commit:      commit,
			ShortCommit: shortCommit,
			Date:        date,
			BinaryName:  appConfig.Build.Name,
			MainPath:    appConfig.Build.MainPath,
		}

		if err := builder.BuildAll(ctx, buildOpts); err != nil {
			return fmt.Errorf("build: %w", err)
		}

		if appConfig.Build.Name != "" {
			binaryNames = append(binaryNames, appConfig.Build.Name)
		}
	}

	// Output based on format
	if out.IsJSON() {
		result := output.BuildResult{
			Version:     versionStr,
			Commit:      commit,
			ShortCommit: shortCommit,
			Date:        date,
			OutputDir:   outputDir,
			Targets:     targets,
			Binaries:    binaryNames,
			Message:     "Build completed",
		}
		return out.Print(result)
	}

	logger.Success("Build completed: %s (version: %s)", outputDir, versionStr)
	return nil
}
