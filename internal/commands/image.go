package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/alexjoedt/forge/internal/config"
	"github.com/alexjoedt/forge/internal/docker"
	"github.com/alexjoedt/forge/internal/git"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/output"
	"github.com/urfave/cli/v3"
)

// Docker returns the docker command that builds and optionally pushes Docker images.
func Docker() *cli.Command {
	return &cli.Command{
		Name:    "docker",
		Usage:   "Build and push Docker images (optional)",
		Aliases: []string{"image"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "dockerfile",
				Usage: "path to Dockerfile",
				Value: "./Dockerfile",
			},
			&cli.StringFlag{
				Name:  "context",
				Usage: "build context path",
				Value: ".",
			},
			&cli.StringFlag{
				Name:  "repository",
				Usage: "image repository (e.g., ghcr.io/USER/APP)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "tags",
				Usage: "comma-separated list of tag templates",
				Value: "",
			},
			&cli.BoolFlag{
				Name:  "push",
				Usage: "push the image to registry",
			},
			&cli.StringFlag{
				Name:  "platforms",
				Usage: "comma-separated list of platforms",
				Value: "",
			},
			&cli.StringSliceFlag{
				Name:  "build-arg",
				Usage: "build arguments (key=value)",
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
		Action: imageAction,
	}
}

func imageAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	repoDir := cmd.String("repo-dir")
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

	// Check if docker configuration is present
	if !appConfig.Docker.Enabled {
		return cli.Exit("docker configuration not enabled - forge image requires docker to be enabled in forge.yaml\nYou can use forge for version management only with 'forge bump' and 'forge version' commands", 2)
	}

	dockerRepositories := appConfig.Docker.GetRepositories()
	cmdRepository := cmd.String("repository")
	
	// If command line repository is provided, use it (overrides config)
	if cmdRepository != "" {
		dockerRepositories = []string{cmdRepository}
	}
	
	// Check if we have at least one repository
	if len(dockerRepositories) == 0 {
		return cli.Exit("docker repository not configured - forge image requires at least one repository to be configured in forge.yaml or via --repository flag\nYou can use forge for version management only with 'forge bump' and 'forge version' commands", 2)
	}

	tagPrefix := appConfig.Git.TagPrefix

	// Get repository (backward compatibility - for single repo)
	repository := ""
	if len(dockerRepositories) > 0 {
		repository = dockerRepositories[0]
	}

	// Get tags
	tagsStr := cmd.String("tags")
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	} else {
		tags = appConfig.Docker.Tags
	}

	// Get platforms
	platformsStr := cmd.String("platforms")
	var platforms []string
	if platformsStr != "" {
		platforms = strings.Split(platformsStr, ",")
	} else {
		platforms = appConfig.Docker.Platforms
	}

	// Parse build args
	buildArgs := make(map[string]string)
	for _, arg := range cmd.StringSlice("build-arg") {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			buildArgs[parts[0]] = parts[1]
		}
	}
	// Merge with config build args
	for k, v := range appConfig.Docker.BuildArgs {
		if _, exists := buildArgs[k]; !exists {
			buildArgs[k] = v
		}
	}

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

	logger.Debugf("building docker image for version %s in repositories %v", versionStr, dockerRepositories)

	// Create builder
	dockerfilePath := cmd.String("dockerfile")
	contextPath := cmd.String("context")
	builder := docker.NewBuilder(repoDir, dockerfilePath, contextPath, dryRun)

	// Build options
	pushed := cmd.Bool("push")
	opts := docker.BuildOptions{
		Repositories: dockerRepositories,
		Repository:   repository, // Keep for backward compatibility
		Tags:         tags,
		Platforms:    platforms,
		BuildArgs:    buildArgs,
		Push:         pushed,
		Version:      versionStr,
		Commit:       commit,
		ShortCommit:  shortCommit,
	}

	// Build image
	if err := builder.Build(ctx, opts); err != nil {
		return fmt.Errorf("build image: %w", err)
	}

	// Output based on format
	if out.IsJSON() {
		result := output.ImageResult{
			Version:     versionStr,
			Commit:      commit,
			ShortCommit: shortCommit,
			Repository:  repository,
			Tags:        tags,
			Platforms:   platforms,
			Pushed:      pushed,
			Message:     fmt.Sprintf("Image built%s", map[bool]string{true: " and pushed", false: ""}[pushed]),
		}
		return out.Print(result)
	}

	// Format repository display
	repoDisplay := repository
	if len(dockerRepositories) > 1 {
		repoDisplay = fmt.Sprintf("%s (and %d more)", dockerRepositories[0], len(dockerRepositories)-1)
	}

	if pushed {
		logger.Success("Image built and pushed: %s (version: %s)", repoDisplay, versionStr)
	} else {
		logger.Success("Image built: %s (version: %s)", repoDisplay, versionStr)
	}
	return nil
}
