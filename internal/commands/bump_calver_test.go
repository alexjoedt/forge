package commands

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alexjoedt/forge/internal/testutil"
	"github.com/alexjoedt/forge/internal/version"
)

func calverConfig(format string) string {
	return "default_branch: main\nscheme: calver\nprefix: v\ncalver_format: " + format + "\n"
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

			parsed, err := version.ParseCalVer(strings.TrimPrefix(first, "v"))
			if err != nil {
				t.Fatalf("parse %s: %v", first, err)
			}
			before = time.Now()
			second := bumpJSON(t, dir)
			after = time.Now()
			want = []string{
				"v" + version.NextCalVer(parsed, format, before).String(),
				"v" + version.NextCalVer(parsed, format, after).String(),
			}
			if !slices.Contains(want, second) {
				t.Errorf("second tag = %s, want one of %v", second, want)
			}
			if tags := testutil.Tags(t, dir); len(tags) != 3 || !slices.Contains(tags, second) {
				t.Errorf("tags = %v, want old, %s, %s", tags, first, second)
			}
		})
	}
}
