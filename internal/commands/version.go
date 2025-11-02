package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/alexjoedt/forge/internal/config"
	"github.com/alexjoedt/forge/internal/git"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/output"
	"github.com/alexjoedt/forge/internal/table"
	"github.com/alexjoedt/forge/internal/version"
	"github.com/urfave/cli/v3"
)

// Version returns the version command that prints the current version based on the last valid git tag.
func Version() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show current version from git tags",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "repo-dir",
				Usage: "repository directory",
				Value: ".",
			},
			appFlag,
		},
		Action: versionAction,
		Commands: []*cli.Command{
			versionListCommand(),
			versionNextCommand(),
		},
	}
}

func versionAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	repoDir := cmd.String("repo-dir")

	// Validate requirements
	if err := ValidateRequirements(ctx, repoDir); err != nil {
		return err
	}

	// Load config
	logger.Debugf("Load configuration from: %s", repoDir)
	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	appName := cmd.String("app")

	// Check if multi-app and no specific app requested
	if cfg.IsMultiApp() && appName == "" && !out.IsJSON() {
		return versionMultiAppAction(ctx, cfg, repoDir, out)
	}

	// Single app or specific app requested
	appConfig, err := cfg.GetAppConfig(appName)
	if err != nil {
		return err
	}

	tagPrefix := appConfig.Git.TagPrefix

	// Create tagger
	tagger := git.NewTagger(repoDir, tagPrefix, false)

	// Get version with dirty check (same logic as build/image commands)
	versionStr, err := tagger.GetVersionWithDirtyCheck(ctx)
	if err != nil {
		logger.Warnf("failed to detect version from git, using default: %v", err)
		versionStr = "0.0.0-dev"
	}

	// Check if dirty
	dirty := strings.HasSuffix(versionStr, "-dirty")

	// Get current commit hash
	commit, err := tagger.CurrentCommit(ctx)
	if err != nil {
		logger.Warnf("failed to get current commit: %v", err)
		commit = "unknown"
	}

	// Output based on format
	if out.IsJSON() {
		result := output.VersionResult{
			Version: versionStr,
			Scheme:  appConfig.Version.Scheme,
			Commit:  commit,
			Dirty:   dirty,
		}
		return out.Print(result)
	}

	// Enhanced single-app display
	fmt.Printf("Current Version: %s\n", table.CurrentVersion(versionStr))
	fmt.Printf("Scheme:          %s\n", table.Scheme(appConfig.Version.Scheme))
	fmt.Printf("Commit:          %s\n", table.Commit(commit))
	if dirty {
		fmt.Printf("Status:          %s\n", table.Date("dirty (uncommitted changes)"))
	}

	return nil
}

// versionMultiAppAction displays all app versions in a table
func versionMultiAppAction(ctx context.Context, cfg *config.Config, repoDir string, out *output.Manager) error {
	logger := log.FromContext(ctx)

	// Create table
	tbl := table.New([]table.Column{
		{Header: "App", Width: 10, Align: table.AlignLeft},
		{Header: "Current", Width: 15, Align: table.AlignLeft},
		{Header: "Scheme", Width: 8, Align: table.AlignLeft},
		{Header: "Last Tag", Width: 15, Align: table.AlignLeft},
		{Header: "Date", Width: 19, Align: table.AlignLeft},
		{Header: "Commit", Width: 8, Align: table.AlignLeft},
	})

	// Get all apps and their versions
	apps := cfg.GetAllApps()
	for appName, appConfig := range apps {
		tagPrefix := appConfig.Git.TagPrefix
		tagger := git.NewTagger(repoDir, tagPrefix, false)

		// Get version
		versionStr, err := tagger.GetVersionWithDirtyCheck(ctx)
		if err != nil {
			logger.Debugf("failed to get version for %s: %v", appName, err)
			versionStr = "none"
		}

		// Get last tag info
		latestTag, err := tagger.LatestTag(ctx)
		if err != nil {
			logger.Debugf("failed to get latest tag for %s: %v", appName, err)
			latestTag = "none"
		}

		// Get tag info for date and commit
		var dateStr, commitStr string
		if latestTag != "" && latestTag != "none" {
			tagInfo, err := tagger.GetTagInfo(ctx, latestTag)
			if err == nil {
				dateStr = tagInfo.Date
				if len(tagInfo.Commit) >= 8 {
					commitStr = tagInfo.Commit[:8]
				} else {
					commitStr = tagInfo.Commit
				}
			}
		}

		if dateStr == "" {
			dateStr = "-"
		}
		if commitStr == "" {
			commitStr = "-"
		}

		// Add row with styling
		tbl.AddRow(
			appName,
			table.CurrentVersion(versionStr),
			table.Scheme(appConfig.Version.Scheme),
			latestTag,
			table.Date(dateStr),
			table.Commit(commitStr),
		)
	}

	// Print table
	fmt.Println(tbl.Render())

	return nil
}

// versionListCommand returns the "version list" subcommand
func versionListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Usage:   "List all version tags in history",
		Aliases: []string{"ls"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "repo-dir",
				Usage: "repository directory",
				Value: ".",
			},
			&cli.IntFlag{
				Name:    "limit",
				Aliases: []string{"n"},
				Usage:   "limit number of versions to display",
				Value:   0, // 0 means no limit
			},
			appFlag,
		},
		Action: versionListAction,
	}
}

func versionListAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	repoDir := cmd.String("repo-dir")

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

	tagPrefix := appConfig.Git.TagPrefix
	tagger := git.NewTagger(repoDir, tagPrefix, false)

	// Get all tags
	tags, err := tagger.ListAllTags(ctx)
	if err != nil {
		return fmt.Errorf("list tags: %w", err)
	}

	if len(tags) == 0 {
		logger.Debugf("no version tags found")
		if out.IsJSON() {
			return out.Print(output.VersionHistoryResult{
				Versions: []output.VersionHistoryEntry{},
				Count:    0,
			})
		}
		fmt.Println("No version tags found")
		return nil
	}

	// Apply limit if specified
	limit := cmd.Int("limit")
	if limit > 0 && limit < len(tags) {
		tags = tags[:limit]
	}

	// Output based on format
	if out.IsJSON() {
		entries := make([]output.VersionHistoryEntry, 0, len(tags))
		for _, tag := range tags {
			entries = append(entries, output.VersionHistoryEntry{
				Version: tag.Version,
				Tag:     tag.Tag,
				Commit:  tag.Commit,
				Date:    tag.Date,
				Message: tag.Message,
			})
		}
		result := output.VersionHistoryResult{
			Versions: entries,
			Count:    len(entries),
		}
		return out.Print(result)
	}

	// Create table for better formatting
	tbl := table.New([]table.Column{
		{Header: "Version", Width: 12, Align: table.AlignLeft},
		{Header: "Tag", Width: 15, Align: table.AlignLeft},
		{Header: "Commit", Width: 8, Align: table.AlignLeft},
		{Header: "Date", Width: 19, Align: table.AlignLeft},
	})
	tbl.Border = false

	// Add rows with styling
	for _, tag := range tags {
		commitShort := tag.Commit
		if len(commitShort) > 8 {
			commitShort = commitShort[:8]
		}
		
		tbl.AddRow(
			table.CurrentVersion(tag.Version),
			tag.Tag,
			table.Commit(commitShort),
			table.Date(tag.Date),
		)
	}

	fmt.Println(tbl.Render())

	return nil
}

// versionNextCommand returns the "version next" subcommand
func versionNextCommand() *cli.Command {
	return &cli.Command{
		Name:  "next",
		Usage: "Preview the next version without creating a tag",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "repo-dir",
				Usage: "repository directory",
				Value: ".",
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
				Usage: "prerelease identifier (e.g., rc.1)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "meta",
				Usage: "build metadata",
				Value: "",
			},
			appFlag,
		},
		Action: versionNextAction,
	}
}

func versionNextAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	repoDir := cmd.String("repo-dir")

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

	// Override config with flags
	scheme := cmd.String("scheme")
	if scheme == "" {
		scheme = appConfig.Version.Scheme
	}

	prefix := appConfig.Version.Prefix
	calverFormat := cmd.String("calver-format")
	if calverFormat == "" {
		calverFormat = appConfig.Version.CalVerFormat
	}

	pre := cmd.String("pre")
	if pre == "" {
		pre = appConfig.Version.Pre
	}

	meta := cmd.String("meta")
	if meta == "" {
		meta = appConfig.Version.Meta
	}

	bumpStr := cmd.String("bump")
	var bump version.BumpType
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

	// Validate scheme
	var versionScheme version.Scheme
	switch scheme {
	case "semver":
		versionScheme = version.SchemeSemVer
	case "calver":
		versionScheme = version.SchemeCalVer
		if cmd.IsSet("bump") {
			logger.Warnf("--bump flag is ignored for calver scheme (versions are automatically determined by date/week)")
		}
	default:
		return fmt.Errorf("invalid scheme: %s (must be semver or calver)", scheme)
	}

	// Create tagger (dry-run doesn't matter here since we're only calculating)
	tagger := git.NewTagger(repoDir, prefix, true)

	// Get current version
	currentVersion, err := tagger.GetVersionWithDirtyCheck(ctx)
	if err != nil {
		logger.Warnf("failed to detect current version from git: %v", err)
		currentVersion = "none"
	}

	// Calculate next version
	nextVersion, err := tagger.CalculateNextVersion(ctx, versionScheme, bump, calverFormat, pre, meta)
	if err != nil {
		return fmt.Errorf("calculate next version: %w", err)
	}

	tag := version.WithPrefix(nextVersion.String(), prefix)

	// Output based on format
	if out.IsJSON() {
		result := map[string]interface{}{
			"current": currentVersion,
			"next":    nextVersion.String(),
			"tag":     tag,
			"scheme":  scheme,
		}
		return out.Print(result)
	}

	// Text output
	fmt.Printf("Current:  %s\n", currentVersion)
	fmt.Printf("Next:     %s\n", nextVersion.String())
	fmt.Printf("Tag:      %s\n", tag)
	fmt.Printf("Scheme:   %s\n", scheme)

	return nil
}
