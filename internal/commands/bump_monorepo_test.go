package commands

import (
	"slices"
	"strings"
	"testing"

	"github.com/alexjoedt/forge/internal/testutil"
)

const monorepoConfig = `defaultApp: api
api:
  default_branch: main
  scheme: semver
  prefix: api/v
worker:
  default_branch: main
  scheme: calver
  prefix: worker/
  calver_format: 2006.01.02
`

func monorepo(t *testing.T) string {
	t.Helper()
	dir := testutil.InitRepo(t, monorepoConfig)
	for _, tag := range []string{"api/v9.0.0", "worker/2020.01.01"} {
		testutil.Git(t, dir, "tag", "-a", tag, "-m", tag)
	}
	return dir
}

func newTags(t *testing.T, dir string) []string {
	t.Helper()
	var got []string
	for _, tag := range testutil.Tags(t, dir) {
		if tag != "api/v9.0.0" && tag != "worker/2020.01.01" {
			got = append(got, tag)
		}
	}
	return got
}

func TestBumpMonorepoWorkerApp(t *testing.T) {
	dir := monorepo(t)

	if _, stderr, err := runForge(t, "bump", "--bump", "minor", "--app", "worker", "--repo-dir", dir); err != nil {
		t.Fatalf("bump worker: %v (stderr: %s)", err, stderr)
	}
	// api/v9.0.0 must not leak into worker's version.
	got := newTags(t, dir)
	if len(got) != 1 || !strings.HasPrefix(got[0], "worker/20") || strings.HasPrefix(got[0], "worker/9") {
		t.Fatalf("new tags = %v, want one worker/<calver> tag", got)
	}
	if !slices.Contains(testutil.Tags(t, dir), "api/v9.0.0") {
		t.Errorf("api tag lost")
	}
}

func TestBumpMonorepoDefaultApp(t *testing.T) {
	dir := monorepo(t)

	if _, stderr, err := runForge(t, "bump", "--bump", "patch", "--repo-dir", dir); err != nil {
		t.Fatalf("bump default: %v (stderr: %s)", err, stderr)
	}
	if got := newTags(t, dir); len(got) != 1 || got[0] != "api/v9.0.1" {
		t.Errorf("new tags = %v, want [api/v9.0.1]", got)
	}
}
