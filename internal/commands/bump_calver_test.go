package commands

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/alexjoedt/forge/internal/output"
	"github.com/alexjoedt/forge/internal/testutil"
	"github.com/alexjoedt/forge/internal/version"
)

func calverConfig(format string) string {
	return "default_branch: main\nscheme: calver\nprefix: v\ncalver_format: " + format + "\n"
}

func bumpJSON(t *testing.T, dir string) string {
	t.Helper()
	stdout, stderr, err := runForge(t, "bump", "--json", "--repo-dir", dir)
	if err != nil {
		t.Fatalf("bump: %v (stderr: %s)", err, stderr)
	}
	var got output.TagResult
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("decode JSON %q: %v", stdout, jsonErr)
	}
	if got.Tag == "" || got.Version != got.Tag {
		t.Fatalf("got %+v, want tag and version set", got)
	}
	return got.Tag
}

func TestBumpCalver(t *testing.T) {
	for _, format := range []string{"2006.01.02", "2006.WW"} {
		t.Run(format, func(t *testing.T) {
			dir := testutil.InitRepo(t, calverConfig(format))
			testutil.Git(t, dir, "tag", "-a", "v2020.01.01", "-m", "old")

			// The clock is not injectable: accept the tag for either side of a period rollover.
			before := time.Now()
			first := bumpJSON(t, dir)
			after := time.Now()
			want := []string{
				"v" + version.NextCalVer(nil, format, before).String(),
				"v" + version.NextCalVer(nil, format, after).String(),
			}
			if !slices.Contains(want, first) {
				t.Errorf("first tag = %s, want one of %v", first, want)
			}

			second := bumpJSON(t, dir)
			if second == first {
				t.Errorf("second bump duplicated %s", first)
			}
			if tags := testutil.Tags(t, dir); len(tags) != 3 || !slices.Contains(tags, second) {
				t.Errorf("tags = %v, want old, %s, %s", tags, first, second)
			}
		})
	}
}
