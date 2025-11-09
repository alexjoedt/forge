package git

import (
	"fmt"
	"testing"

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
