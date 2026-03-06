package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/alexjoedt/forge/internal/config"
	"github.com/alexjoedt/forge/internal/git"
	"github.com/alexjoedt/forge/internal/interactive"
	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/output"
	"github.com/urfave/cli/v3"
)

// Retag returns the retag command that moves an existing tag to a different commit.
func Retag() *cli.Command {
	return &cli.Command{
		Name:      "retag",
		Usage:     "Move an existing tag to a different commit",
		ArgsUsage: "<tag> [<commit>]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "skip confirmation prompt (required in non-interactive mode)",
			},
			&cli.StringFlag{
				Name:    "message",
				Aliases: []string{"m"},
				Usage:   "annotation message for the moved tag",
				Value:   "",
			},
			&cli.BoolFlag{
				Name:  "push",
				Usage: "force-push the tag to remote after moving",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "show what would be done without doing it",
			},
			&cli.StringFlag{
				Name:  "prefix",
				Usage: "tag prefix override (e.g., v)",
				Value: "",
			},
			&cli.StringFlag{
				Name:  "repo-dir",
				Usage: "repository directory",
				Value: ".",
			},
			appFlag,
		},
		Action: retagAction,
	}
}

//nolint:gocognit,nestif // CLI handler requires branching and complexity; splitting would hurt readability
func retagAction(ctx context.Context, cmd *cli.Command) error {
	logger := log.FromContext(ctx)
	out := output.FromContext(ctx)

	args := cmd.Args().Slice()
	if len(args) < 1 {
		return &ForgeError{
			Title:       "Missing tag argument",
			Description: "Usage: forge retag <tag> [<commit>]",
			Suggestions: []string{
				"Example: forge retag v1.2.3",
				"Example: forge retag v1.2.3 abc1234",
			},
		}
	}

	tag := args[0]
	target := "HEAD"
	if len(args) >= 2 {
		target = args[1]
	}

	yes := cmd.Bool("yes")
	push := cmd.Bool("push")
	dryRun := cmd.Bool("dry-run")
	repoDir := cmd.String("repo-dir")
	message := cmd.String("message")
	appName := cmd.String("app")
	prefix := cmd.String("prefix")

	if err := ValidateRequirements(ctx, repoDir); err != nil {
		return err
	}

	cfg, err := config.LoadFromDir(repoDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	appConfig, err := cfg.GetAppConfig(appName)
	if err != nil {
		return fmt.Errorf("get app config: %w", err)
	}

	if prefix == "" {
		prefix = appConfig.Git.TagPrefix
	}

	tagger := git.NewTagger(repoDir, prefix, dryRun)

	exists, err := tagger.TagExists(ctx, tag)
	if err != nil {
		return fmt.Errorf("check tag: %w", err)
	}
	if !exists {
		return &ForgeError{
			Title:       fmt.Sprintf("Tag %q not found", tag),
			Description: "The tag must already exist to be moved.",
			Suggestions: []string{
				"List existing tags: git tag -l",
				"Create a new tag with: forge bump",
			},
		}
	}

	fromCommit, err := tagger.GetTagCommit(ctx, tag)
	if err != nil {
		return fmt.Errorf("resolve tag commit: %w", err)
	}

	toCommit, err := tagger.ResolveCommit(ctx, target)
	if err != nil {
		return &ForgeError{
			Title:       fmt.Sprintf("Cannot resolve commit %q", target),
			Description: err.Error(),
			Suggestions: []string{"Use a valid commit hash, branch name, or HEAD"},
		}
	}

	if fromCommit == toCommit {
		return &ForgeError{
			Title:       "Tag already points to this commit",
			Description: fmt.Sprintf("%s already points to %s", tag, fromCommit[:7]),
			Suggestions: []string{"Nothing to do."},
		}
	}

	if message == "" {
		message = fmt.Sprintf("retag %s to %s", tag, toCommit[:7])
	}

	if !dryRun {
		if interactive.IsInteractive() && !yes {
			preview := fmt.Sprintf("  from  %s\n  to    %s", fromCommit[:7], toCommit[:7])
			confirmed, err := interactive.PromptConfirmation(
				fmt.Sprintf("Move tag %s?", tag),
				preview,
			)
			if err != nil {
				return fmt.Errorf("confirmation: %w", err)
			}
			if !confirmed {
				fmt.Println("Aborted.")
				return nil
			}
		} else if !interactive.IsInteractive() && !yes {
			return &ForgeError{
				Title:       "Confirmation required",
				Description: "Moving a tag is a destructive operation that cannot run unattended without --yes.",
				Suggestions: []string{
					fmt.Sprintf("Add --yes to confirm: forge retag %s --yes", tag),
					"Use --dry-run to preview the operation first",
				},
			}
		}
	}

	logger.Debugf("moving tag %s from %s to %s", tag, fromCommit[:7], toCommit[:7])
	if err := tagger.MoveTag(ctx, tag, target, message); err != nil {
		return fmt.Errorf("move tag: %w", err)
	}

	pushed := false
	if push {
		if err := tagger.PushTagForce(ctx, tag); err != nil {
			return fmt.Errorf("force-push tag: %w", err)
		}
		pushed = !dryRun
	}

	result := &output.RetagResult{
		Tag:        tag,
		FromCommit: fromCommit,
		ToCommit:   toCommit,
		Pushed:     pushed,
	}

	if out.IsJSON() {
		return out.Print(result)
	}

	if dryRun {
		fmt.Fprintf(
			os.Stdout,
			"dry-run: would move tag %s\n  from  %s\n  to    %s\n",
			tag,
			fromCommit[:7],
			toCommit[:7],
		)
		if push {
			fmt.Fprintf(os.Stdout, "dry-run: would force-push tag %s to origin\n", tag)
		}
	} else {
		fmt.Fprintf(os.Stdout, "moved tag %s\n  from  %s\n  to    %s\n", tag, fromCommit[:7], toCommit[:7])
		if push {
			fmt.Fprintf(os.Stdout, "force-pushed tag %s to origin\n", tag)
		}
	}

	return nil
}
