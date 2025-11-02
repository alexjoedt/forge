package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/run"
)

// ForgeError represents a user friendly error with actionable suggestions
type ForgeError struct {
	Title       string
	Description string
	Suggestions []string
}

func (e *ForgeError) Error() string {
	msg := fmt.Sprintf("Error: %s\n", e.Title)
	if e.Description != "" {
		msg += fmt.Sprintf("\n  %s\n", e.Description)
	}
	if len(e.Suggestions) > 0 {
		msg += "\n  Suggestions:\n"
		for _, suggestion := range e.Suggestions {
			msg += fmt.Sprintf("    • %s\n", suggestion)
		}
	}
	return msg
}

// ValidateRequirements checks for forge.yaml and git repository.
// This should be called for commands that require these dependencies.
func ValidateRequirements(ctx context.Context, repoDir string) error {
	logger := log.FromContext(ctx)

	// Check for forge.yaml or .forge.yaml
	forgeYaml := filepath.Join(repoDir, "forge.yaml")
	dotForgeYaml := filepath.Join(repoDir, ".forge.yaml")

	forgeYamlExists := false
	if _, err := os.Stat(forgeYaml); err == nil {
		forgeYamlExists = true
	} else if _, err := os.Stat(dotForgeYaml); err == nil {
		forgeYamlExists = true
	}

	if !forgeYamlExists {
		absPath, _ := filepath.Abs(repoDir)
		return &ForgeError{
			Title:       "Configuration file not found",
			Description: fmt.Sprintf("No forge.yaml or .forge.yaml found in %s", absPath),
			Suggestions: []string{
				"Run 'forge init' to create a new configuration",
				"If you're in the wrong directory, use --repo-dir to specify the correct path",
				"Check that forge.yaml exists in your project root",
			},
		}
	}

	// Check for git repository
	result := run.CmdInDir(ctx, repoDir, "git", "rev-parse", "--git-dir")
	if !result.Success() {
		absPath, _ := filepath.Abs(repoDir)
		return &ForgeError{
			Title:       "Not a git repository",
			Description: fmt.Sprintf("Directory %s is not a git repository", absPath),
			Suggestions: []string{
				"Run 'git init' to initialize a git repository",
				"Clone your repository with 'git clone <url>'",
				"Forge requires a git repository to manage versions",
			},
		}
	}

	logger.Debugf("requirements validated")
	return nil
}

// CheckGitClean checks if the git working directory has uncommitted changes
func CheckGitClean(ctx context.Context, repoDir string, allowDirty bool) error {
	if allowDirty {
		return nil
	}

	result := run.CmdInDir(ctx, repoDir, "git", "status", "--porcelain")
	if !result.Success() {
		return fmt.Errorf("failed to check git status: %s", result.Stderr)
	}

	if result.Stdout != "" {
		return &ForgeError{
			Title:       "Working directory has uncommitted changes",
			Description: "Git working directory is not clean. Forge requires a clean state before creating version tags.",
			Suggestions: []string{
				"Commit your changes: git add . && git commit -m 'Your message'",
				"Stash your changes: git stash",
				"Use --force to create tag anyway (not recommended)",
			},
		}
	}

	return nil
}

// CheckForExistingTags checks if any version tags exist
func CheckForExistingTags(ctx context.Context, repoDir, tagPrefix string) (bool, error) {
	pattern := tagPrefix + "*"
	result := run.CmdInDir(ctx, repoDir, "git", "tag", "-l", pattern)

	if !result.Success() {
		return false, fmt.Errorf("failed to list tags: %s", result.Stderr)
	}

	return result.Stdout != "", nil
}

// NoTagsError returns an error for when no version tags are found
func NoTagsError(tagPrefix, initialVersion string) error {
	if initialVersion == "" {
		initialVersion = "1.0.0"
	}

	return &ForgeError{
		Title:       "No version tags found",
		Description: "This appears to be the first version tag for this project.",
		Suggestions: []string{
			fmt.Sprintf("Create your first tag: forge bump --initial %s", initialVersion),
			"Or use: forge bump --initial to use default (1.0.0)",
			fmt.Sprintf("Or manually: git tag %s%s && git push --tags", tagPrefix, initialVersion),
		},
	}
}
