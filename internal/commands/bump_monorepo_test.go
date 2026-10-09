package commands

import (
	"slices"
	"testing"
	"time"

	"github.com/alexjoedt/forge/internal/testutil"
	"github.com/alexjoedt/forge/internal/version"
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

func newTags(t *testing.T, dir string, before []string) []string {
	t.Helper()
	var got []string
	for _, tag := range testutil.Tags(t, dir) {
		if !slices.Contains(before, tag) {
			got = append(got, tag)
		}
	}
	return got
}

func TestBumpMonorepoWorkerApp(t *testing.T) {
	dir := monorepo(t)
	seed := testutil.Tags(t, dir)

	// api/v9.0.0 must not leak into worker's version; --bump minor is ignored for calver.
	before := time.Now()
	if _, stderr, err := runForge(t, "bump", "--bump", "minor", "--app", "worker", "--repo-dir", dir); err != nil {
		t.Fatalf("bump worker: %v (stderr: %s)", err, stderr)
	}
	after := time.Now()
	want := []string{
		"worker/" + version.NextCalVer(nil, "2006.01.02", before).String(),
		"worker/" + version.NextCalVer(nil, "2006.01.02", after).String(),
	}
	got := newTags(t, dir, seed)
	if len(got) != 1 || !slices.Contains(want, got[0]) {
		t.Fatalf("new tags = %v, want one of %v", got, want)
	}
	if !slices.Contains(testutil.Tags(t, dir), "api/v9.0.0") {
		t.Errorf("api tag lost")
	}
}

func TestBumpMonorepoDefaultApp(t *testing.T) {
	dir := monorepo(t)
	seed := testutil.Tags(t, dir)

	if _, stderr, err := runForge(t, "bump", "--bump", "patch", "--repo-dir", dir); err != nil {
		t.Fatalf("bump default: %v (stderr: %s)", err, stderr)
	}
	if got := newTags(t, dir, seed); len(got) != 1 || got[0] != "api/v9.0.1" {
		t.Errorf("new tags = %v, want [api/v9.0.1]", got)
	}
}
