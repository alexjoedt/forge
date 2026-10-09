package commands

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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

	stdout, stderr, err := runForge(t, "bump", "--bump", "minor", "--dry-run", "--repo-dir", dir)
	if err != nil {
		t.Fatalf("dry-run: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout+stderr, "v1.3.0") {
		t.Errorf("dry-run output lacks planned tag v1.3.0: %q %q", stdout, stderr)
	}
	if tags := testutil.Tags(t, dir); len(tags) != 1 {
		t.Fatalf("dry-run changed tags: %v", tags)
	}

	if got := bumpJSON(t, dir, "--bump", "minor"); got != "v1.3.0" {
		t.Errorf("tag = %s, want v1.3.0", got)
	}
}

func TestBumpSemverDirtyTree(t *testing.T) {
	dir := testutil.InitRepo(t, semverConfig)
	testutil.Git(t, dir, "tag", "-a", "v1.2.3", "-m", "v1.2.3")
	if err := os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := runForge(t, "bump", "--bump", "patch", "--repo-dir", dir)
	var fe *ForgeError
	if !errors.As(err, &fe) || !strings.Contains(fe.Title, "uncommitted changes") {
		t.Fatalf("err = %v, want uncommitted-changes ForgeError", err)
	}
	if tags := testutil.Tags(t, dir); len(tags) != 1 {
		t.Errorf("tags = %v, want only v1.2.3", tags)
	}

	if _, stderr, forceErr := runForge(t, "bump", "--bump", "patch", "--force", "--repo-dir", dir); forceErr != nil {
		t.Fatalf("bump --force: %v (stderr: %s)", forceErr, stderr)
	}
	head := testutil.Git(t, dir, "rev-parse", "HEAD")
	if got := testutil.Git(t, dir, "rev-list", "-n1", "v1.2.4"); got != head {
		t.Errorf("v1.2.4 on %q, want HEAD %s", got, head)
	}
}
