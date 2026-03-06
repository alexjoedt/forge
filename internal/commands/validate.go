package commands

import (
	"context"
	"fmt"

	"github.com/alexjoedt/forge/internal/config"
	"github.com/alexjoedt/forge/internal/git"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/output"
	"github.com/urfave/cli/v3"
)

// Validate returns the validate command that checks forge.yaml and git state
func Validate() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate forge.yaml configuration and git repository state",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "repo-dir",
				Usage: "repository directory",
				Value: ".",
			},
			appFlag,
		},
		Action: validateAction,
	}
}

//nolint:gocognit,nestif // validateAction orchestrates multiple validation checks; keeping logic in one place aids CLI flow clarity
func validateAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	repoDir := cmd.String("repo-dir")

	// Track validation issues
	issues := []string{}
	warnings := []string{}

	// Check git repository
	logger.Debugf("Checking git repository...")
	tagger := git.NewTagger(repoDir, "", false)
	_, err := tagger.CurrentCommit(ctx)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Not a git repository: %v", err))
	} else {
		logger.Debugf("✓ Git repository found")
	}

	// Check forge.yaml
	logger.Debugf("Loading forge.yaml configuration...")
	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Failed to load forge.yaml: %v", err))

		// Early exit if config can't be loaded
		if out.IsJSON() {
			result := map[string]any{
				"valid":    false,
				"issues":   issues,
				"warnings": warnings,
			}
			return out.Print(result)
		}

		logger.Errorf("Validation failed!")
		for _, issue := range issues {
			logger.Errorf("  ✗ %s", issue)
		}
		return fmt.Errorf("validation failed with %d issue(s)", len(issues))
	}
	logger.Debugf("✓ forge.yaml loaded successfully")

	// Validate app configuration
	appName := cmd.String("app")
	appConfig, appErr := cfg.GetAppConfig(appName)
	if appErr != nil {
		issues = append(issues, fmt.Sprintf("Failed to get app config: %v", appErr))
	} else {
		logger.Debugf("✓ App configuration found")

		// Check version scheme
		if appConfig.Scheme != "semver" && appConfig.Scheme != "calver" {
			issues = append(issues, fmt.Sprintf("Invalid version scheme '%s' (must be 'semver' or 'calver')", appConfig.Scheme))
		} else {
			logger.Debugf("✓ Version scheme: %s", appConfig.Scheme)
		}

		// Check calver format if calver scheme
		if appConfig.Scheme == "calver" {
			if appConfig.CalVerFormat == "" {
				warnings = append(warnings, "CalVer scheme without calver_format (will use default)")
			} else {
				logger.Debugf("✓ CalVer format: %s", appConfig.CalVerFormat)
			}
		}

		// Check tag prefix
		if appConfig.Prefix == "" {
			warnings = append(warnings, "No tag prefix configured (tags will have no prefix)")
		} else {
			logger.Debugf("✓ Tag prefix: %s", appConfig.Prefix)
		}

		// Check for existing tags
		appTagger := git.NewTagger(repoDir, appConfig.Prefix, false)
		tags, err := appTagger.ListAllTags(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to list tags: %v", err))
		} else if len(tags) == 0 {
			warnings = append(warnings, "No version tags found in repository (use 'forge bump' to create first tag)")
		} else {
			logger.Debugf("✓ Found %d version tag(s)", len(tags))
		}

		// Check working directory state
		isDirty, err := appTagger.HasUncommittedChanges(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Failed to check working tree: %v", err))
		} else if isDirty {
			warnings = append(warnings, "Working directory has uncommitted changes")
		} else {
			logger.Debugf("✓ Working directory is clean")
		}
	}

	// Output results
	if out.IsJSON() {
		result := map[string]any{
			"valid":    len(issues) == 0,
			"issues":   issues,
			"warnings": warnings,
		}
		return out.Print(result)
	}

	// Text output
	if len(issues) == 0 && len(warnings) == 0 {
		logger.Success("✓ Validation passed - configuration is valid")
		return nil
	}

	if len(issues) > 0 {
		logger.Errorf("Validation failed with %d issue(s):", len(issues))
		for _, issue := range issues {
			logger.Errorf("  ✗ %s", issue)
		}
	}

	if len(warnings) > 0 {
		logger.Warnf("Validation completed with %d warning(s):", len(warnings))
		for _, warning := range warnings {
			logger.Warnf("  ⚠ %s", warning)
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("validation failed with %d issue(s)", len(issues))
	}

	return nil
}
