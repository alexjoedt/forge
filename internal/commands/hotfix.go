package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alexjoedt/forge/internal/config"
	"github.com/alexjoedt/forge/internal/git"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/output"
	"github.com/urfave/cli/v3"
)

// Hotfix returns the hotfix command group.
func Hotfix() *cli.Command {
	return &cli.Command{
		Name:     "hotfix",
		Usage:    "Manage hotfix branches and versions",
		Category: "Version Management",
		Commands: []*cli.Command{
			hotfixCreate(),
			hotfixBump(),
			hotfixStatus(),
			hotfixList(),
		},
	}
}

// hotfixCreate returns the hotfix create command.
func hotfixCreate() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Create a hotfix branch from a release tag",
		ArgsUsage: "<base-tag>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "app",
				Aliases: []string{"a"},
				Usage:   "Specify app name (optional, auto-detected from tag)",
			},
			&cli.BoolFlag{
				Name:  "no-checkout",
				Usage: "Create branch without checking it out",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show what would happen without making changes",
			},
		},
		Action: hotfixCreateAction,
	}
}

// HotfixCreateOutput represents the output of hotfix create command.
type HotfixCreateOutput struct {
	Branch  string `json:"branch"`
	BaseTag string `json:"base_tag"`
	Created bool   `json:"created"`
	Message string `json:"message"`
}

func hotfixCreateAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	// 1. Parse arguments
	baseTag := cmd.Args().First()
	if baseTag == "" {
		return fmt.Errorf("base tag argument required\n\nUsage: forge hotfix create <base-tag>\n\nExample:\n  forge hotfix create v1.0.0")
	}

	dryRun := cmd.Bool("dry-run")

	// 2. Validate requirements
	repoDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := ValidateRequirements(ctx, repoDir); err != nil {
		return err
	}

	// 3. Load config
	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 4. Detect or validate app
	appName := cmd.String("app")
	if appName == "" {
		appName, err = cfg.DetectAppFromTag(baseTag)
		if err != nil {
			return err
		}
	} else {
		if err := cfg.ValidateAppTag(appName, baseTag); err != nil {
			return err
		}
	}

	// 5. Get app config
	appConfig, err := cfg.GetAppConfig(appName)
	if err != nil {
		return err
	}

	// 6. Validate base tag
	if err := git.ValidateHotfixBaseTag(ctx, repoDir, baseTag); err != nil {
		return err
	}

	// 7. Get hotfix config with defaults
	hotfixCfg := appConfig.GetHotfixConfig()

	// 8. Create branch
	tagger := git.NewTagger(repoDir, appConfig.Git.TagPrefix, dryRun)
	checkout := !cmd.Bool("no-checkout")

	branchName, err := tagger.CreateHotfixBranch(ctx, baseTag, hotfixCfg.BranchPrefix, checkout)
	if err != nil {
		return err
	}

	// 9. Output result
	result := HotfixCreateOutput{
		Branch:  branchName,
		BaseTag: baseTag,
		Created: !dryRun,
		Message: fmt.Sprintf("Created hotfix branch from %s", baseTag),
	}

	if dryRun {
		result.Message = fmt.Sprintf("Would create hotfix branch from %s", baseTag)
	}

	if err := out.Print(result); err != nil {
		return err
	}

	// Text-only hints
	if !cmd.Bool("json") && !dryRun {
		logger.Println("\nNext steps:")
		logger.Println("  1. Apply fixes and commit changes")
		logger.Println("  2. Run 'forge hotfix bump' to create hotfix tag")
	}

	return nil
}

// hotfixBump returns the hotfix bump command.
func hotfixBump() *cli.Command {
	return &cli.Command{
		Name:  "bump",
		Usage: "Bump hotfix version on current hotfix branch",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "base",
				Aliases: []string{"b"},
				Usage:   "Create hotfix branch from base tag and bump in one step",
			},
			&cli.StringFlag{
				Name:    "message",
				Aliases: []string{"m"},
				Usage:   "Custom tag message (default: 'Hotfix <tag>')",
			},
			&cli.BoolFlag{
				Name:  "push",
				Usage: "Push tag to remote after creation",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show what would happen without making changes",
			},
		},
		Action: hotfixBumpAction,
	}
}

// HotfixBumpOutput represents the output of hotfix bump command.
type HotfixBumpOutput struct {
	Tag      string `json:"tag"`
	Version  string `json:"version"`
	BaseTag  string `json:"base_tag"`
	Sequence int    `json:"sequence"`
	Branch   string `json:"branch,omitempty"`
	Created  bool   `json:"created"`
	Pushed   bool   `json:"pushed"`
	Message  string `json:"message"`
}

func hotfixBumpAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)
	dryRun := cmd.Bool("dry-run")

	repoDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Check if --base flag provided (create + bump in one step)
	if baseTag := cmd.String("base"); baseTag != "" {
		return quickHotfixBump(ctx, cmd, baseTag, out, dryRun)
	}

	// Normal workflow: must be on hotfix branch
	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch(repoDir)
	if err != nil {
		return err
	}

	// Detect hotfix context from current branch
	var appConfig *config.AppConfig
	var baseTag string
	var hotfixCfg config.HotfixConfig

	for _, app := range cfg.GetAllAppConfigs() {
		hotfixCfg = app.GetHotfixConfig()

		if git.IsHotfixBranch(currentBranch, hotfixCfg.BranchPrefix) {
			baseTag, err = git.ExtractTagFromBranch(currentBranch, hotfixCfg.BranchPrefix)
			if err != nil {
				return fmt.Errorf("invalid hotfix branch format: %w", err)
			}
			appConfig = app
			break
		}
	}

	if appConfig == nil {
		return fmt.Errorf("not on a hotfix branch\n\nUse one of these commands:\n  forge hotfix create <tag>   - Create hotfix branch first\n  forge hotfix bump --base <tag>  - Create and bump in one step")
	}

	// Validate working tree is clean
	if err := git.ValidateWorkingTreeClean(ctx, repoDir); err != nil {
		return err
	}

	// Create tagger
	tagger := git.NewTagger(repoDir, appConfig.Git.TagPrefix, dryRun)

	// Get next hotfix tag
	nextTag, seq, err := tagger.GetNextHotfixTag(ctx, baseTag, hotfixCfg.Suffix)
	if err != nil {
		return err
	}

	// Create tag
	message := cmd.String("message")
	if message == "" {
		message = fmt.Sprintf("Hotfix %s", nextTag)
	}

	if err := tagger.CreateHotfixTag(ctx, nextTag, message); err != nil {
		return err
	}

	if !dryRun {
		logger.Success("✓ Created hotfix tag: %s", nextTag)
	}

	// Push if requested
	pushed := false
	if cmd.Bool("push") && !dryRun {
		if err := tagger.PushTag(ctx, nextTag); err != nil {
			return fmt.Errorf("failed to push tag: %w", err)
		}
		logger.Success("✓ Pushed tag to remote: %s", nextTag)
		pushed = true
	}

	// Output result
	result := HotfixBumpOutput{
		Tag:      nextTag,
		Version:  strings.TrimPrefix(nextTag, appConfig.Git.TagPrefix),
		BaseTag:  baseTag,
		Sequence: seq,
		Branch:   currentBranch,
		Created:  !dryRun,
		Pushed:   pushed,
		Message:  fmt.Sprintf("Created hotfix tag %s", nextTag),
	}

	if dryRun {
		result.Message = fmt.Sprintf("Would create hotfix tag %s", nextTag)
	}

	return out.Print(result)
}

func quickHotfixBump(ctx context.Context, cmd *cli.Command, baseTag string, out *output.Manager, dryRun bool) error {
	logger := log.FromContext(ctx)

	repoDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if err := ValidateRequirements(ctx, repoDir); err != nil {
		return err
	}

	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Detect app
	appName, err := cfg.DetectAppFromTag(baseTag)
	if err != nil {
		return err
	}

	appConfig, err := cfg.GetAppConfig(appName)
	if err != nil {
		return err
	}

	// Validate base tag
	if err := git.ValidateHotfixBaseTag(ctx, repoDir, baseTag); err != nil {
		return err
	}

	hotfixCfg := appConfig.GetHotfixConfig()

	// Create branch
	tagger := git.NewTagger(repoDir, appConfig.Git.TagPrefix, dryRun)
	branchName, err := tagger.CreateHotfixBranch(ctx, baseTag, hotfixCfg.BranchPrefix, true)
	if err != nil {
		return err
	}

	if !dryRun {
		logger.Success("✓ Created and checked out hotfix branch: %s", branchName)
	}

	// Validate working tree is clean
	if err := git.ValidateWorkingTreeClean(ctx, repoDir); err != nil {
		return err
	}

	// Get next hotfix tag
	nextTag, seq, err := tagger.GetNextHotfixTag(ctx, baseTag, hotfixCfg.Suffix)
	if err != nil {
		return err
	}

	// Create tag
	message := cmd.String("message")
	if message == "" {
		message = fmt.Sprintf("Hotfix %s", nextTag)
	}

	if err := tagger.CreateHotfixTag(ctx, nextTag, message); err != nil {
		return err
	}

	if !dryRun {
		logger.Success("✓ Created hotfix tag: %s", nextTag)
	}

	// Push if requested
	pushed := false
	if cmd.Bool("push") && !dryRun {
		if err := tagger.PushTag(ctx, nextTag); err != nil {
			return fmt.Errorf("failed to push tag: %w", err)
		}
		logger.Success("✓ Pushed tag to remote: %s", nextTag)
		pushed = true
	}

	// Output result
	result := HotfixBumpOutput{
		Tag:      nextTag,
		Version:  strings.TrimPrefix(nextTag, appConfig.Git.TagPrefix),
		BaseTag:  baseTag,
		Sequence: seq,
		Branch:   branchName,
		Created:  !dryRun,
		Pushed:   pushed,
		Message:  fmt.Sprintf("Created hotfix tag %s", nextTag),
	}

	if dryRun {
		result.Message = fmt.Sprintf("Would create hotfix branch and tag %s", nextTag)
	}

	return out.Print(result)
}

// hotfixStatus returns the hotfix status command.
func hotfixStatus() *cli.Command {
	return &cli.Command{
		Name:   "status",
		Usage:  "Show current hotfix branch status",
		Action: hotfixStatusAction,
	}
}

// HotfixStatusOutput represents the output of hotfix status command.
type HotfixStatusOutput struct {
	OnHotfixBranch bool           `json:"on_hotfix_branch"`
	CurrentBranch  string         `json:"current_branch"`
	BaseTag        string         `json:"base_tag,omitempty"`
	LastHotfix     string         `json:"last_hotfix,omitempty"`
	NextHotfix     string         `json:"next_hotfix,omitempty"`
	HotfixCount    int            `json:"hotfix_count"`
	ActiveHotfixes []ActiveHotfix `json:"active_hotfixes,omitempty"`
}

// ActiveHotfix represents an active hotfix branch.
type ActiveHotfix struct {
	Branch  string `json:"branch"`
	BaseTag string `json:"base_tag"`
	LastTag string `json:"last_tag"`
	Count   int    `json:"count"`
}

func hotfixStatusAction(ctx context.Context, cmd *cli.Command) error {
	out := output.FromContext(ctx)

	repoDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	currentBranch, err := git.GetCurrentBranch(repoDir)
	if err != nil {
		return err
	}

	result := HotfixStatusOutput{
		CurrentBranch: currentBranch,
	}

	// Check current branch
	for _, app := range cfg.GetAllAppConfigs() {
		hotfixCfg := app.GetHotfixConfig()

		if git.IsHotfixBranch(currentBranch, hotfixCfg.BranchPrefix) {
			result.OnHotfixBranch = true
			result.BaseTag, _ = git.ExtractTagFromBranch(currentBranch, hotfixCfg.BranchPrefix)

			// Get hotfix info
			tagger := git.NewTagger(repoDir, app.Git.TagPrefix, false)
			nextTag, seq, _ := tagger.GetNextHotfixTag(ctx, result.BaseTag, hotfixCfg.Suffix)
			result.NextHotfix = nextTag
			result.HotfixCount = seq - 1

			if result.HotfixCount > 0 {
				result.LastHotfix = fmt.Sprintf("%s-%s.%d", result.BaseTag, hotfixCfg.Suffix, result.HotfixCount)
			}

			break
		}
	}

	// List all active hotfix branches
	branches, _ := git.ListBranches(repoDir)
	for _, app := range cfg.GetAllAppConfigs() {
		hotfixCfg := app.GetHotfixConfig()

		for _, branch := range branches {
			if git.IsHotfixBranch(branch, hotfixCfg.BranchPrefix) {
				baseTag, _ := git.ExtractTagFromBranch(branch, hotfixCfg.BranchPrefix)

				tagger := git.NewTagger(repoDir, app.Git.TagPrefix, false)
				_, seq, _ := tagger.GetNextHotfixTag(ctx, baseTag, hotfixCfg.Suffix)
				count := seq - 1

				var lastTag string
				if count > 0 {
					lastTag = fmt.Sprintf("%s-%s.%d", baseTag, hotfixCfg.Suffix, count)
				}

				result.ActiveHotfixes = append(result.ActiveHotfixes, ActiveHotfix{
					Branch:  branch,
					BaseTag: baseTag,
					LastTag: lastTag,
					Count:   count,
				})
			}
		}
	}

	return out.Print(result)
}

// hotfixList returns the hotfix list command.
func hotfixList() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List all hotfix tags for a base version",
		ArgsUsage: "[base-tag]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "app",
				Aliases: []string{"a"},
				Usage:   "Filter by app name",
			},
		},
		Action: hotfixListAction,
	}
}

// HotfixListOutput represents the output of hotfix list command.
type HotfixListOutput struct {
	BaseTag  string   `json:"base_tag"`
	Hotfixes []string `json:"hotfixes"`
	Count    int      `json:"count"`
}

func hotfixListAction(ctx context.Context, cmd *cli.Command) error {
	out := output.FromContext(ctx)

	repoDir, err := os.Getwd()
	if err != nil {
		return err
	}

	baseTag := cmd.Args().First()

	// If no base tag provided, try to detect from current branch
	if baseTag == "" {
		cfg, err := config.LoadFromDir(repoDir)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		currentBranch, err := git.GetCurrentBranch(repoDir)
		if err != nil {
			return err
		}

		for _, app := range cfg.GetAllAppConfigs() {
			hotfixCfg := app.GetHotfixConfig()
			if git.IsHotfixBranch(currentBranch, hotfixCfg.BranchPrefix) {
				baseTag, err = git.ExtractTagFromBranch(currentBranch, hotfixCfg.BranchPrefix)
				if err != nil {
					return err
				}
				break
			}
		}

		if baseTag == "" {
			return fmt.Errorf("base tag required (or run from hotfix branch)\n\nUsage: forge hotfix list <base-tag>\n\nExample:\n  forge hotfix list v1.0.0")
		}
	}

	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	appName := cmd.String("app")
	if appName == "" {
		appName, _ = cfg.DetectAppFromTag(baseTag)
	}

	appConfig, err := cfg.GetAppConfig(appName)
	if err != nil {
		return err
	}

	hotfixCfg := appConfig.GetHotfixConfig()

	// List all hotfix tags - we'll call the GetNextHotfixTag to get them
	tagger := git.NewTagger(repoDir, appConfig.Git.TagPrefix, false)

	// Get all tags and filter for hotfix pattern
	allTagsInfo, err := tagger.ListAllTags(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	// Filter for hotfix tags matching the base tag
	pattern := fmt.Sprintf("%s-%s.", baseTag, hotfixCfg.Suffix)
	hotfixTags := []string{}
	for _, tagInfo := range allTagsInfo {
		if strings.HasPrefix(tagInfo.Tag, pattern) {
			hotfixTags = append(hotfixTags, tagInfo.Tag)
		}
	}

	result := HotfixListOutput{
		BaseTag:  baseTag,
		Hotfixes: hotfixTags,
		Count:    len(hotfixTags),
	}

	return out.Print(result)
}
