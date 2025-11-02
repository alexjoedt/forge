package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/alexjoedt/forge/internal/config"
	"github.com/alexjoedt/forge/internal/git"
	"github.com/alexjoedt/forge/internal/interactive"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/nodejs"
	"github.com/alexjoedt/forge/internal/output"
	"github.com/alexjoedt/forge/internal/version"
	"github.com/urfave/cli/v3"
)

var appFlag = &cli.StringFlag{
	Name:  "app",
	Usage: "app to bump",
	Value: "",
}

// Bump returns the bump command that creates and optionally pushes a git tag.
// This is the primary version management command.
func Bump() *cli.Command {
	return &cli.Command{
		Name:    "bump",
		Usage:   "Bump version and create a git tag",
		Aliases: []string{"tag"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "initial",
				Usage:   "create initial version tag (e.g., --initial 1.0.0)",
				Aliases: []string{"i"},
				Value:   "",
			},
			&cli.StringFlag{
				Name:  "scheme",
				Usage: "version scheme: semver or calver",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "bump",
				Usage: "semver bump type: major, minor, or patch",
				Value: "patch",
			},
			&cli.StringFlag{
				Name:  "calver-format",
				Usage: "calver format string (e.g., 2006.01.02)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "pre",
				Usage: "[ALPHA] prerelease identifier (e.g., rc.1) - not fully implemented",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "meta",
				Usage: "[ALPHA] build metadata - not fully implemented",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "prefix",
				Usage: "tag prefix (e.g., v)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "repo-dir",
				Usage: "repository directory",
				Value: ".",
			},
			&cli.BoolFlag{
				Name:  "push",
				Usage: "push the tag to remote",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "force tag creation even with uncommitted changes",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "show what would be done without doing it",
			},
			appFlag,
		},
		Action: tagAction,
	}
}

func tagAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	repoDir := cmd.String("repo-dir")
	dryRun := cmd.Bool("dry-run")
	initialVersion := cmd.String("initial")
	force := cmd.Bool("force")

	// Validate requirements
	if err := ValidateRequirements(ctx, repoDir); err != nil {
		return err
	}

	// Check for clean git state (unless --force or --dry-run)
	if !dryRun && !force {
		if err := CheckGitClean(ctx, repoDir, force); err != nil {
			return err
		}
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

	// Override config with flags
	scheme := cmd.String("scheme")
	if scheme == "" {
		scheme = appConfig.Version.Scheme
	}

	prefix := cmd.String("prefix")
	if prefix == "" {
		prefix = appConfig.Version.Prefix
	}

	tagPrefix := appConfig.Git.TagPrefix

	// Handle initial version creation
	if initialVersion != "" {
		return createInitialTag(ctx, repoDir, tagPrefix, initialVersion, dryRun, cmd.Bool("push"))
	}

	calverFormat := cmd.String("calver-format")
	if calverFormat == "" {
		calverFormat = appConfig.Version.CalVerFormat
	}

	pre := cmd.String("pre")
	if pre == "" {
		pre = appConfig.Version.Pre
	}
	if pre != "" {
		logger.Warnf("⚠️  --pre flag is in ALPHA state and not fully implemented. Do not use in production.")
	}

	meta := cmd.String("meta")
	if meta == "" {
		meta = appConfig.Version.Meta
	}
	if meta != "" {
		logger.Warnf("⚠️  --meta flag is in ALPHA state and not fully implemented. Do not use in production.")
	}

	// Create tagger for getting current version
	tagger := git.NewTagger(repoDir, tagPrefix, dryRun)
	
	// Check if any tags exist
	hasTags, err := CheckForExistingTags(ctx, repoDir, tagPrefix)
	if err != nil {
		return fmt.Errorf("failed to check for existing tags: %w", err)
	}
	
	if !hasTags {
		// No tags found - guide user to create first tag
		return NoTagsError(tagPrefix, "1.0.0")
	}
	
	// Get current version for interactive display
	currentVersion, err := tagger.GetVersionWithDirtyCheck(ctx)
	if err != nil {
		logger.Debugf("failed to detect current version: %v", err)
		currentVersion = "none"
	}

	// Interactive mode: if --bump flag is not explicitly set and we're in a TTY
	var bump version.BumpType
	isInteractive := interactive.IsInteractive() && !cmd.IsSet("bump") && !out.IsJSON()
	
	if isInteractive && scheme == "semver" {
		// Show interactive prompt for bump type selection
		logger.Debugf("entering interactive mode for bump selection")
		
		// Calculate preview versions for each bump type
		choices := []interactive.BumpChoice{}
		
		for _, bumpType := range []version.BumpType{version.BumpPatch, version.BumpMinor, version.BumpMajor} {
			previewVer, err := tagger.CalculateNextVersion(ctx, version.SchemeSemVer, bumpType, calverFormat, pre, meta)
			if err != nil {
				logger.Debugf("failed to calculate preview for %s: %v", bumpType, err)
				continue
			}
			
			var desc string
			switch bumpType {
			case version.BumpPatch:
				desc = "bug fixes and patches"
			case version.BumpMinor:
				desc = "new features, backwards compatible"
			case version.BumpMajor:
				desc = "breaking changes"
			}
			
			choices = append(choices, interactive.BumpChoice{
				Type:        interactive.BumpType(strings.ToLower(string(bumpType))),
				Description: desc,
				Preview:     version.WithPrefix(previewVer.String(), prefix),
			})
		}
		
		// Show selection prompt
		selected, err := interactive.PromptBumpType(currentVersion, choices)
		if err != nil {
			return fmt.Errorf("interactive selection: %w", err)
		}
		
		// Convert selected choice to bump type
		switch selected.Type {
		case "patch":
			bump = version.BumpPatch
		case "minor":
			bump = version.BumpMinor
		case "major":
			bump = version.BumpMajor
		default:
			return fmt.Errorf("invalid bump type selected: %s", selected.Type)
		}
		
		logger.Debugf("selected bump type: %s", bump)
	} else {
		// Non-interactive mode: use flag or default
		bumpStr := cmd.String("bump")
		switch bumpStr {
		case "major":
			bump = version.BumpMajor
		case "minor":
			bump = version.BumpMinor
		case "patch":
			bump = version.BumpPatch
		default:
			return fmt.Errorf("invalid bump type: %s", bumpStr)
		}
	}

	// Validate scheme
	var versionScheme version.Scheme
	switch scheme {
	case "semver":
		versionScheme = version.SchemeSemVer
	case "calver":
		versionScheme = version.SchemeCalVer
		// Warn if --bump flag is provided with calver
		if cmd.IsSet("bump") {
			logger.Warnf("--bump flag is ignored for calver scheme (versions are automatically determined by date/week)")
		}
	default:
		return fmt.Errorf("invalid scheme: %s (must be semver or calver)", scheme)
	}

	// Calculate next version (but don't create tag yet)
	nextVersion, err := tagger.CalculateNextVersion(ctx, versionScheme, bump, calverFormat, pre, meta)
	if err != nil {
		return fmt.Errorf("calculate next version: %w", err)
	}

	tag := version.WithPrefix(nextVersion.String(), prefix)
	cleanVersion := nextVersion.String()

	// Interactive confirmation before creating tag
	if isInteractive && !dryRun {
		preview := fmt.Sprintf("Current: %s → Next: %s", currentVersion, tag)
		confirmed, err := interactive.PromptConfirmation("Create this tag?", preview)
		if err != nil {
			return fmt.Errorf("confirmation: %w", err)
		}
		if !confirmed {
			logger.Infof("Tag creation canceled")
			return nil
		}
	}

	// Update package.json BEFORE creating the tag if Node.js integration is enabled
	if appConfig.NodeJS.Enabled {
		logger.Debugf("Node.js integration enabled, updating package.json")

		// Create Node.js updater
		nodeUpdater := nodejs.NewUpdater(repoDir, dryRun)

		// Update package.json
		updated, err := nodeUpdater.Update(ctx, appConfig.NodeJS.PackagePath, cleanVersion)
		if err != nil {
			return fmt.Errorf("update package.json: %w", err)
		}

		if updated && !dryRun {
			// Stage and commit the package.json change
			logger.Debugf("committing package.json version update")

			// Get package.json path for staging
			pkgPath := appConfig.NodeJS.PackagePath
			if pkgPath == "" {
				pkgPath = "package.json"
			}

			if err := tagger.CommitVersionUpdate(ctx, pkgPath, tag); err != nil {
				return fmt.Errorf("commit package.json: %w", err)
			}

			logger.Infof("committed package.json version update")
		}
	}

	// Now create the tag on the current commit (which includes package.json update if any)
	if err := tagger.CreateTag(ctx, tag, fmt.Sprintf("forge: release %s", tag)); err != nil {
		return fmt.Errorf("create tag: %w", err)
	}

	pushed := cmd.Bool("push")

	// Push if requested
	if pushed {
		if err := tagger.PushTag(ctx, tag); err != nil {
			return fmt.Errorf("push tag: %w", err)
		}
	}

	// Output based on format
	if out.IsJSON() {
		result := output.TagResult{
			Tag:     tag,
			Pushed:  pushed,
			Version: tag,
			Message: fmt.Sprintf("Tag created%s", map[bool]string{true: " and pushed", false: ""}[pushed]),
		}
		return out.Print(result)
	}

	if pushed {
		logger.Success("Tag created and pushed: %s", tag)
	} else {
		logger.Success("Tag created: %s", tag)
	}

	return nil
}

// createInitialTag creates the first version tag for a project
func createInitialTag(ctx context.Context, repoDir, tagPrefix, version string, dryRun, push bool) error {
	logger := log.FromContext(ctx)

	// Validate version format
	if version == "" {
		version = "1.0.0"
	}

	// Add prefix if not present
	fullTag := version
	if tagPrefix != "" && !strings.HasPrefix(version, tagPrefix) {
		fullTag = tagPrefix + version
	}

	logger.Infof("creating initial version tag: %s", fullTag)

	if dryRun {
		logger.Infof("dry-run: would create tag %s", fullTag)
		if push {
			logger.Infof("dry-run: would push tag to remote")
		}
		return nil
	}

	// Create tagger
	tagger := git.NewTagger(repoDir, tagPrefix, dryRun)

	// Create the tag
	if err := tagger.CreateTag(ctx, fullTag, fmt.Sprintf("forge: initial release %s", fullTag)); err != nil {
		return fmt.Errorf("create initial tag: %w", err)
	}

	logger.Success("Created initial tag: %s", fullTag)

	// Push if requested
	if push {
		if err := tagger.PushTag(ctx, fullTag); err != nil {
			return fmt.Errorf("push tag: %w", err)
		}
		logger.Success("Pushed tag to remote: %s", fullTag)
	} else {
		logger.Infof("tag created locally - use --push to push to remote")
	}

	return nil
}
