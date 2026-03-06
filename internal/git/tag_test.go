package git

import (
	"context"
	"fmt"
	"testing"

	"github.com/alexjoedt/forge/internal/run"
	"github.com/alexjoedt/forge/internal/version"
)

func TestTagger(t *testing.T) {

	tgg := NewTagger(".", "teatapp/v", true)
	s, err := tgg.LatestTag(t.Context())
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	fmt.Println(s)

	v, err := tgg.ParseLatestVersion(t.Context(), version.SchemeSemVer)
	must(t, err)

	fmt.Printf("%v\n", v)

	got, err := tgg.IsTagOnCurrentCommit(t.Context(), s)
	must(t, err)
	fmt.Println(got)
}

func must(t *testing.T, err error) {
	if err != nil {
		t.FailNow()
	}
}

// Hotfix Git Operation Tests

func TestIsHotfixBranch(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		prefix     string
		want       bool
	}{
		{
			name:       "standard release branch",
			branchName: "release/v1.0.0",
			prefix:     "release/",
			want:       true,
		},
		{
			name:       "multi-app release branch",
			branchName: "release/api/v1.0.0",
			prefix:     "release/",
			want:       true,
		},
		{
			name:       "main branch",
			branchName: "main",
			prefix:     "release/",
			want:       false,
		},
		{
			name:       "feature branch",
			branchName: "feature/new-thing",
			prefix:     "release/",
			want:       false,
		},
		{
			name:       "custom hotfix pattern",
			branchName: "hotfix/v1.0.0",
			prefix:     "hotfix/",
			want:       true,
		},
		{
			name:       "custom support pattern",
			branchName: "support/v1.0.0",
			prefix:     "support/",
			want:       true,
		},
		{
			name:       "wrong prefix",
			branchName: "hotfix/v1.0.0",
			prefix:     "release/",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHotfixBranch(tt.branchName, tt.prefix)
			if got != tt.want {
				t.Errorf("IsHotfixBranch(%q, %q) = %v, want %v", tt.branchName, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestExtractTagFromBranch(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		prefix     string
		wantTag    string
		wantErr    bool
	}{
		{
			name:       "simple tag",
			branchName: "release/v1.0.0",
			prefix:     "release/",
			wantTag:    "v1.0.0",
		},
		{
			name:       "namespaced tag",
			branchName: "release/api/v1.0.0",
			prefix:     "release/",
			wantTag:    "api/v1.0.0",
		},
		{
			name:       "custom hotfix pattern",
			branchName: "hotfix/v2.1.0",
			prefix:     "hotfix/",
			wantTag:    "v2.1.0",
		},
		{
			name:       "no match",
			branchName: "main",
			prefix:     "release/",
			wantErr:    true,
		},
		{
			name:       "wrong prefix",
			branchName: "hotfix/v1.0.0",
			prefix:     "release/",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractTagFromBranch(tt.branchName, tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTagFromBranch(%q, %q) error = %v, wantErr %v", tt.branchName, tt.prefix, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantTag {
				t.Errorf("ExtractTagFromBranch(%q, %q) = %q, want %q", tt.branchName, tt.prefix, got, tt.wantTag)
			}
		})
	}
}

func TestParseHotfixSequence(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		baseTag string
		suffix  string
		wantSeq int
		wantErr bool
	}{
		{
			name:    "seq 1",
			tag:     "v1.0.0-hotfix.1",
			baseTag: "v1.0.0",
			suffix:  "hotfix",
			wantSeq: 1,
		},
		{
			name:    "seq 10",
			tag:     "v1.0.0-hotfix.10",
			baseTag: "v1.0.0",
			suffix:  "hotfix",
			wantSeq: 10,
		},
		{
			name:    "custom suffix",
			tag:     "v1.0.0-patch.5",
			baseTag: "v1.0.0",
			suffix:  "patch",
			wantSeq: 5,
		},
		{
			name:    "wrong base tag",
			tag:     "v1.0.0-hotfix.1",
			baseTag: "v2.0.0",
			suffix:  "hotfix",
			wantErr: true,
		},
		{
			name:    "wrong suffix",
			tag:     "v1.0.0-hotfix.1",
			baseTag: "v1.0.0",
			suffix:  "patch",
			wantErr: true,
		},
		{
			name:    "invalid sequence",
			tag:     "v1.0.0-hotfix.abc",
			baseTag: "v1.0.0",
			suffix:  "hotfix",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHotfixSequence(tt.tag, tt.baseTag, tt.suffix)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHotfixSequence(%q, %q, %q) error = %v, wantErr %v", tt.tag, tt.baseTag, tt.suffix, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantSeq {
				t.Errorf("parseHotfixSequence(%q, %q, %q) = %d, want %d", tt.tag, tt.baseTag, tt.suffix, got, tt.wantSeq)
			}
		})
	}
}

// initTestRepo creates a temporary git repository with an initial empty commit.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
		{"git", "commit", "--allow-empty", "-m", "initial commit"},
	}
	for _, args := range cmds {
		r := run.CmdInDir(ctx, dir, args[0], args[1:]...)
		if !r.Success() {
			t.Fatalf("repo setup %v failed: %s", args, r.Stderr)
		}
	}
	return dir
}

// addAnnotatedTag creates an annotated tag in the test repo.
func addAnnotatedTag(t *testing.T, dir, tag string) {
	t.Helper()
	ctx := context.Background()
	// Add an empty commit so each tag sits on its own commit.
	r := run.CmdInDir(ctx, dir, "git", "commit", "--allow-empty", "-m", "chore: bump "+tag)
	if !r.Success() {
		t.Fatalf("empty commit before tag %s failed: %s", tag, r.Stderr)
	}
	r = run.CmdInDir(ctx, dir, "git", "tag", "-a", tag, "-m", tag)
	if !r.Success() {
		t.Fatalf("create tag %s failed: %s", tag, r.Stderr)
	}
}

func TestCalculatePreRelease(t *testing.T) {
	tests := []struct {
		name      string
		// tags are applied in order before calling CalculatePreRelease.
		tags      []string
		prefix    string
		channel   string
		bumpType  string
		wantVer   string
		wantErr   bool
	}{
		{
			name:     "no existing tags, bump minor, alpha channel",
			tags:     nil,
			prefix:   "v",
			channel:  "alpha",
			bumpType: "minor",
			wantVer:  "0.1.0-alpha.1",
		},
		{
			name:     "stable base, bump minor, alpha channel",
			tags:     []string{"v1.0.0"},
			prefix:   "v",
			channel:  "alpha",
			bumpType: "minor",
			wantVer:  "1.1.0-alpha.1",
		},
		{
			name:     "stable base, bump major, rc channel",
			tags:     []string{"v1.0.0"},
			prefix:   "v",
			channel:  "rc",
			bumpType: "major",
			wantVer:  "2.0.0-rc.1",
		},
		{
			name:     "stable base, bump patch, beta channel",
			tags:     []string{"v1.2.3"},
			prefix:   "v",
			channel:  "beta",
			bumpType: "patch",
			wantVer:  "1.2.4-beta.1",
		},
		{
			name:     "increment within same channel",
			tags:     []string{"v1.0.0", "v1.1.0-alpha.1"},
			prefix:   "v",
			channel:  "alpha",
			bumpType: "",
			wantVer:  "1.1.0-alpha.2",
		},
		{
			name:     "promote alpha to rc",
			tags:     []string{"v1.0.0", "v1.1.0-alpha.2"},
			prefix:   "v",
			channel:  "rc",
			bumpType: "",
			wantVer:  "1.1.0-rc.1",
		},
		{
			name:     "graduate rc to stable",
			tags:     []string{"v1.0.0", "v1.1.0-rc.1"},
			prefix:   "v",
			channel:  "release",
			bumpType: "",
			wantVer:  "1.1.0",
		},
		{
			name:     "graduate alpha to stable",
			tags:     []string{"v1.0.0", "v1.1.0-alpha.3"},
			prefix:   "v",
			channel:  "release",
			bumpType: "",
			wantVer:  "1.1.0",
		},
		{
			name:     "error: stable current without bumpType",
			tags:     []string{"v1.0.0"},
			prefix:   "v",
			channel:  "alpha",
			bumpType: "",
			wantErr:  true,
		},
		{
			name:     "error: invalid bumpType",
			tags:     []string{"v1.0.0"},
			prefix:   "v",
			channel:  "alpha",
			bumpType: "invalid",
			wantErr:  true,
		},
		{
			name:     "error: graduate stable version",
			tags:     []string{"v1.0.0"},
			prefix:   "v",
			channel:  "release",
			bumpType: "",
			wantErr:  true,
		},
		{
			name:     "error: graduate with no existing version",
			tags:     nil,
			prefix:   "v",
			channel:  "release",
			bumpType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initTestRepo(t)
			for _, tag := range tt.tags {
				addAnnotatedTag(t, dir, tag)
			}

			tagger := NewTagger(dir, tt.prefix, false)
			got, err := tagger.CalculatePreRelease(t.Context(), tt.channel, tt.bumpType)

			if (err != nil) != tt.wantErr {
				t.Fatalf("CalculatePreRelease(%q, %q) error = %v, wantErr %v", tt.channel, tt.bumpType, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			gotStr := got.String()
			if gotStr != tt.wantVer {
				t.Errorf("CalculatePreRelease(%q, %q) = %q, want %q", tt.channel, tt.bumpType, gotStr, tt.wantVer)
			}

			// Ensure the result parses as valid SemVer.
			if _, err := version.ParseSemVer(gotStr); err != nil {
				t.Errorf("result %q is not valid SemVer: %v", gotStr, err)
			}
		})
	}
}
