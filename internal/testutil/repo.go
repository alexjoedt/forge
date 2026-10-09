// Package testutil provides helpers for tests that need a real git repository.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexjoedt/forge/internal/run"
)

// InitRepo creates a temporary git repository with a test identity and an
// initial commit. A non-empty forgeYAML is written to forge.yaml and included
// in that commit, so the working tree starts clean. Global and system git
// config are disabled for the rest of the test, so tests using it cannot run
// in parallel.
func InitRepo(t *testing.T, forgeYAML string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	Git(t, dir, "init", "-q", "-b", "main")
	Git(t, dir, "config", "user.email", "test@example.com")
	Git(t, dir, "config", "user.name", "Test User")
	Git(t, dir, "config", "commit.gpgsign", "false")
	Git(t, dir, "config", "tag.gpgsign", "false")
	if forgeYAML != "" {
		if err := os.WriteFile(filepath.Join(dir, "forge.yaml"), []byte(forgeYAML), 0o600); err != nil {
			t.Fatalf("write forge.yaml: %v", err)
		}
		Git(t, dir, "add", "forge.yaml")
	}
	Git(t, dir, "commit", "-q", "--allow-empty", "-m", "initial commit")
	return dir
}

// Git runs git with args in dir and fails the test on error. It returns trimmed stdout.
func Git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	r := run.CmdInDir(context.Background(), dir, "git", args...)
	if !r.Success() {
		t.Fatalf("git %v failed: %s", args, r.Stderr)
	}
	return strings.TrimSpace(r.Stdout)
}

// Tags returns the tags of the repository in dir, sorted by refname.
func Tags(t *testing.T, dir string) []string {
	t.Helper()
	out := Git(t, dir, "tag", "--list")
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}
