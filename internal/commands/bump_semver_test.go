package commands

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/alexjoedt/forge/internal/output"
	"github.com/alexjoedt/forge/internal/testutil"
)

func TestBumpSemverLevels(t *testing.T) {
	for _, tc := range []struct{ level, want string }{
		{"major", "v2.0.0"},
		{"minor", "v1.3.0"},
		{"patch", "v1.2.4"},
	} {
		t.Run(tc.level, func(t *testing.T) {
			dir := testutil.InitRepo(t, semverConfig)
			testutil.Git(t, dir, "tag", "-a", "v1.2.3", "-m", "v1.2.3")

			if _, stderr, err := runForge(t, "bump", "--bump", tc.level, "--repo-dir", dir); err != nil {
				t.Fatalf("bump: %v (stderr: %s)", err, stderr)
			}
			if tags := testutil.Tags(t, dir); !slices.Contains(tags, tc.want) {
				t.Fatalf("tags = %v, want %s", tags, tc.want)
			}
			head := testutil.Git(t, dir, "rev-parse", "HEAD")
			if got := testutil.Git(t, dir, "rev-list", "-n1", tc.want); got != head {
				t.Errorf("%s on %s, want HEAD %s", tc.want, got, head)
			}
		})
	}
}

func TestBumpSemverNoTags(t *testing.T) {
	dir := testutil.InitRepo(t, semverConfig)

	_, _, err := runForge(t, "bump", "--bump", "patch", "--repo-dir", dir)
	var fe *ForgeError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v, want *ForgeError", err)
	}
	if tags := testutil.Tags(t, dir); len(tags) != 0 {
		t.Errorf("tags = %v, want none", tags)
	}

	if _, stderr, initErr := runForge(t, "bump", "--initial", "1.0.0", "--repo-dir", dir); initErr != nil {
		t.Fatalf("bump --initial: %v (stderr: %s)", initErr, stderr)
	}
	if tags := testutil.Tags(t, dir); len(tags) != 1 || tags[0] != "v1.0.0" {
		t.Errorf("tags = %v, want [v1.0.0]", tags)
	}
}

func TestBumpSemverDryRunAndJSON(t *testing.T) {
	dir := testutil.InitRepo(t, semverConfig)
	testutil.Git(t, dir, "tag", "-a", "v1.2.3", "-m", "v1.2.3")

	if _, stderr, err := runForge(t, "bump", "--bump", "minor", "--dry-run", "--repo-dir", dir); err != nil {
		t.Fatalf("dry-run: %v (stderr: %s)", err, stderr)
	}
	if tags := testutil.Tags(t, dir); len(tags) != 1 {
		t.Fatalf("dry-run changed tags: %v", tags)
	}

	stdout, stderr, err := runForge(t, "bump", "--bump", "minor", "--json", "--repo-dir", dir)
	if err != nil {
		t.Fatalf("bump --json: %v (stderr: %s)", err, stderr)
	}
	var got output.TagResult
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("decode JSON %q: %v", stdout, jsonErr)
	}
	if got.Tag != "v1.3.0" || got.Version != "v1.3.0" {
		t.Errorf("got %+v, want v1.3.0", got)
	}
}

func TestBumpSemverDirtyTree(t *testing.T) {
	dir := testutil.InitRepo(t, semverConfig)
	testutil.Git(t, dir, "tag", "-a", "v1.2.3", "-m", "v1.2.3")
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := runForge(t, "bump", "--bump", "patch", "--repo-dir", dir); err == nil {
		t.Fatal("bump on dirty tree succeeded, want error")
	}
	if tags := testutil.Tags(t, dir); len(tags) != 1 {
		t.Errorf("tags = %v, want only v1.2.3", tags)
	}

	if _, stderr, err := runForge(t, "bump", "--bump", "patch", "--force", "--repo-dir", dir); err != nil {
		t.Fatalf("bump --force: %v (stderr: %s)", err, stderr)
	}
}
