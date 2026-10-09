package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/alexjoedt/forge/internal/output"
	"github.com/alexjoedt/forge/internal/testutil"
)

const semverConfig = `default_branch: main
scheme: semver
prefix: v
`

// runForge runs `forge <args>` in-process and returns what it wrote to stdout and stderr.
func runForge(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	root := Root(BuildInfo{Version: "test"})
	root.Writer = &stdout
	root.ErrWriter = &stderr
	root.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	err := root.Run(context.Background(), append([]string{"forge"}, args...))
	return stdout.String(), stderr.String(), err
}

func TestVersionJSONSmoke(t *testing.T) {
	dir := testutil.InitRepo(t, semverConfig)
	testutil.Git(t, dir, "tag", "-a", "v1.2.3", "-m", "v1.2.3")

	stdout, stderr, err := runForge(t, "version", "--json", "--repo-dir", dir)
	if err != nil {
		t.Fatalf("forge version: %v (stderr: %s)", err, stderr)
	}

	var got output.VersionResult
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("decode JSON %q: %v", stdout, jsonErr)
	}
	if got.Version != "1.2.3" || got.Scheme != "semver" || got.Commit == "" || got.Dirty {
		t.Errorf("got %+v, want version 1.2.3, scheme semver, a commit, clean tree", got)
	}
	if tags := testutil.Tags(t, dir); len(tags) != 1 || tags[0] != "v1.2.3" {
		t.Errorf("tags = %v, want [v1.2.3]", tags)
	}
}

func TestVersionFlag(t *testing.T) {
	stdout, _, err := runForge(t, "--version")
	if err != nil {
		t.Fatalf("forge --version: %v", err)
	}
	if stdout != "forge version test\n" {
		t.Errorf("stdout = %q", stdout)
	}
}
