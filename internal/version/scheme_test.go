package version

import (
	"testing"
	"time"
)

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "simple semver",
			input: "1.2.3",
			want: &Version{
				Scheme: SchemeSemVer,
				Raw:    "1.2.3",
				Major:  1,
				Minor:  2,
				Patch:  3,
			},
		},
		{
			name:  "semver with prerelease",
			input: "1.2.3-rc.1",
			want: &Version{
				Scheme: SchemeSemVer,
				Raw:    "1.2.3-rc.1",
				Major:  1,
				Minor:  2,
				Patch:  3,
				Pre:    "rc.1",
			},
		},
		{
			name:  "semver with metadata",
			input: "1.2.3+build.123",
			want: &Version{
				Scheme: SchemeSemVer,
				Raw:    "1.2.3+build.123",
				Major:  1,
				Minor:  2,
				Patch:  3,
				Meta:   "build.123",
			},
		},
		{
			name:  "semver with prerelease and metadata",
			input: "1.2.3-rc.1+build.123",
			want: &Version{
				Scheme: SchemeSemVer,
				Raw:    "1.2.3-rc.1+build.123",
				Major:  1,
				Minor:  2,
				Patch:  3,
				Pre:    "rc.1",
				Meta:   "build.123",
			},
		},
		{
			name:    "invalid format",
			input:   "1.2",
			wantErr: true,
		},
		{
			name:    "non-numeric major",
			input:   "a.2.3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSemVer(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSemVer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
					t.Errorf("ParseSemVer() version = %d.%d.%d, want %d.%d.%d",
						got.Major, got.Minor, got.Patch,
						tt.want.Major, tt.want.Minor, tt.want.Patch)
				}
				if got.Pre != tt.want.Pre {
					t.Errorf("ParseSemVer() prerelease = %q, want %q", got.Pre, tt.want.Pre)
				}
				if got.Meta != tt.want.Meta {
					t.Errorf("ParseSemVer() metadata = %q, want %q", got.Meta, tt.want.Meta)
				}
			}
		})
	}
}

func TestBumpSemVer(t *testing.T) {
	tests := []struct {
		name    string
		version *Version
		bump    BumpType
		wantStr string
	}{
		{
			name:    "bump patch",
			version: &Version{Scheme: SchemeSemVer, Major: 1, Minor: 2, Patch: 3},
			bump:    BumpPatch,
			wantStr: "1.2.4",
		},
		{
			name:    "bump minor",
			version: &Version{Scheme: SchemeSemVer, Major: 1, Minor: 2, Patch: 3},
			bump:    BumpMinor,
			wantStr: "1.3.0",
		},
		{
			name:    "bump major",
			version: &Version{Scheme: SchemeSemVer, Major: 1, Minor: 2, Patch: 3},
			bump:    BumpMajor,
			wantStr: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.BumpSemVer(tt.bump)
			if got.String() != tt.wantStr {
				t.Errorf("BumpSemVer() = %v, want %v", got.String(), tt.wantStr)
			}
		})
	}
}

func TestParseCalVer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "simple calver",
			input: "2025.10.02",
			want: &Version{
				Scheme:     SchemeCalVer,
				Raw:        "2025.10.02",
				CalVerDate: "2025.10.02",
			},
		},
		{
			name:  "calver with sequence",
			input: "2025.10.02.1",
			want: &Version{
				Scheme:         SchemeCalVer,
				Raw:            "2025.10.02.1",
				CalVerDate:     "2025.10.02",
				CalVerSequence: 1,
			},
		},
		{
			name:  "calver with prerelease",
			input: "2025.10.02-rc.1",
			want: &Version{
				Scheme:     SchemeCalVer,
				Raw:        "2025.10.02-rc.1",
				CalVerDate: "2025.10.02",
				Pre:        "rc.1",
			},
		},
		{
			name:  "calver week format without sequence",
			input: "2025.40",
			want: &Version{
				Scheme:     SchemeCalVer,
				Raw:        "2025.40",
				CalVerDate: "2025.40",
			},
		},
		{
			name:  "calver week format with sequence",
			input: "2025.40.1",
			want: &Version{
				Scheme:         SchemeCalVer,
				Raw:            "2025.40.1",
				CalVerDate:     "2025.40",
				CalVerSequence: 1,
			},
		},
		{
			name:  "calver week format with higher sequence",
			input: "2025.40.5",
			want: &Version{
				Scheme:         SchemeCalVer,
				Raw:            "2025.40.5",
				CalVerDate:     "2025.40",
				CalVerSequence: 5,
			},
		},
		{
			name:  "calver week format with prerelease",
			input: "2025.40.1-beta.1",
			want: &Version{
				Scheme:         SchemeCalVer,
				Raw:            "2025.40.1-beta.1",
				CalVerDate:     "2025.40",
				CalVerSequence: 1,
				Pre:            "beta.1",
			},
		},
		{
			name:    "invalid format - too few parts",
			input:   "2025",
			wantErr: true,
		},
		{
			name:    "invalid format - ambiguous 2-part",
			input:   "2025.10",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCalVer(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCalVer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				if got.CalVerDate != tt.want.CalVerDate {
					t.Errorf("ParseCalVer() date = %q, want %q", got.CalVerDate, tt.want.CalVerDate)
				}
				if got.CalVerSequence != tt.want.CalVerSequence {
					t.Errorf("ParseCalVer() sequence = %d, want %d", got.CalVerSequence, tt.want.CalVerSequence)
				}
				if got.Pre != tt.want.Pre {
					t.Errorf("ParseCalVer() prerelease = %q, want %q", got.Pre, tt.want.Pre)
				}
			}
		})
	}
}

func TestFormatCalVer(t *testing.T) {
	// October 2, 2025 is in week 40 of 2025 (ISO week)
	testTime := time.Date(2025, 10, 2, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{
			name:   "standard date format",
			format: "2006.01.02",
			want:   "2025.10.02",
		},
		{
			name:   "year and week format",
			format: "2006.WW",
			want:   "2025.40",
		},
		{
			name:   "YYYY.WW format",
			format: "YYYY.WW",
			want:   "2025.40",
		},
		{
			name:   "year-month format",
			format: "2006.01",
			want:   "2025.10",
		},
		{
			name:   "compact year format",
			format: "06.01.02",
			want:   "25.10.02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCalVer(testTime, tt.format)
			if got != tt.want {
				t.Errorf("FormatCalVer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNextCalVer(t *testing.T) {
	now := time.Date(2025, 10, 2, 12, 0, 0, 0, time.UTC)
	format := "2006.01.02"

	tests := []struct {
		name    string
		current *Version
		wantStr string
	}{
		{
			name:    "first version of the day",
			current: nil,
			wantStr: "2025.10.02",
		},
		{
			name: "second version of the day",
			current: &Version{
				Scheme:         SchemeCalVer,
				CalVerDate:     "2025.10.02",
				CalVerSequence: 0,
			},
			wantStr: "2025.10.02.1",
		},
		{
			name: "third version of the day",
			current: &Version{
				Scheme:         SchemeCalVer,
				CalVerDate:     "2025.10.02",
				CalVerSequence: 1,
			},
			wantStr: "2025.10.02.2",
		},
		{
			name: "new day resets sequence",
			current: &Version{
				Scheme:         SchemeCalVer,
				CalVerDate:     "2025.10.01",
				CalVerSequence: 5,
			},
			wantStr: "2025.10.02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextCalVer(tt.current, format, now)
			if got.String() != tt.wantStr {
				t.Errorf("NextCalVer() = %v, want %v", got.String(), tt.wantStr)
			}
		})
	}
}

func TestNextCalVerWithWeekFormat(t *testing.T) {
	// October 2, 2025 is in week 40
	now := time.Date(2025, 10, 2, 12, 0, 0, 0, time.UTC)
	format := "2006.WW"

	tests := []struct {
		name    string
		current *Version
		wantStr string
	}{
		{
			name:    "first version of the week",
			current: nil,
			wantStr: "2025.40.1",
		},
		{
			name: "second version of the week",
			current: &Version{
				Scheme:         SchemeCalVer,
				CalVerDate:     "2025.40",
				CalVerSequence: 1,
			},
			wantStr: "2025.40.2",
		},
		{
			name: "multiple builds in same week",
			current: &Version{
				Scheme:         SchemeCalVer,
				CalVerDate:     "2025.40",
				CalVerSequence: 2,
			},
			wantStr: "2025.40.3",
		},
		{
			name: "new week resets sequence",
			current: &Version{
				Scheme:         SchemeCalVer,
				CalVerDate:     "2025.39",
				CalVerSequence: 5,
			},
			wantStr: "2025.40.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NextCalVer(tt.current, format, now)
			if got.String() != tt.wantStr {
				t.Errorf("NextCalVer() = %v, want %v", got.String(), tt.wantStr)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		name    string
		version *Version
		want    string
	}{
		{
			name:    "semver",
			version: &Version{Scheme: SchemeSemVer, Major: 1, Minor: 2, Patch: 3},
			want:    "1.2.3",
		},
		{
			name:    "semver with prerelease",
			version: &Version{Scheme: SchemeSemVer, Major: 1, Minor: 2, Patch: 3, Pre: "rc.1"},
			want:    "1.2.3-rc.1",
		},
		{
			name:    "calver",
			version: &Version{Scheme: SchemeCalVer, CalVerDate: "2025.10.02"},
			want:    "2025.10.02",
		},
		{
			name:    "calver with sequence",
			version: &Version{Scheme: SchemeCalVer, CalVerDate: "2025.10.02", CalVerSequence: 1},
			want:    "2025.10.02.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.version.String()
			if got != tt.want {
				t.Errorf("Version.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		tag    string
		prefix string
		want   string
	}{
		{"v1.2.3", "v", "1.2.3"},
		{"1.2.3", "v", "1.2.3"},
		{"release/1.2.3", "release/", "1.2.3"},
	}

	for _, tt := range tests {
		got := StripPrefix(tt.tag, tt.prefix)
		if got != tt.want {
			t.Errorf("StripPrefix(%q, %q) = %q, want %q", tt.tag, tt.prefix, got, tt.want)
		}
	}
}

// Hotfix Version Tests

func TestIsHotfixVersion(t *testing.T) {
	tests := []struct {
		name  string
		tag   string
		want  bool
	}{
		{"semver hotfix", "v1.0.0-hotfix.1", true},
		{"semver hotfix seq", "v1.0.0-hotfix.5", true},
		{"calver hotfix", "2025.44.1-hotfix.2", true},
		{"custom suffix", "v2.0.0-patch.3", true},
		{"namespaced", "api/v1.0.0-hotfix.1", true},
		{"not hotfix", "v1.0.0", false},
		{"prerelease", "v1.0.0-rc.1", false},
		{"build metadata", "v1.0.0+build.123", false},
		{"invalid", "v1.0.0-hotfix", false},
		{"invalid seq", "v1.0.0-hotfix.abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHotfixVersion(tt.tag)
			if got != tt.want {
				t.Errorf("IsHotfixVersion(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestParseHotfixVersion(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantBase   string
		wantSuffix string
		wantSeq    int
		wantErr    bool
	}{
		{
			name:       "semver hotfix",
			input:      "v1.0.0-hotfix.1",
			wantBase:   "v1.0.0",
			wantSuffix: "hotfix",
			wantSeq:    1,
		},
		{
			name:       "semver hotfix seq 5",
			input:      "v1.0.0-hotfix.5",
			wantBase:   "v1.0.0",
			wantSuffix: "hotfix",
			wantSeq:    5,
		},
		{
			name:       "calver hotfix",
			input:      "2025.44.1-hotfix.2",
			wantBase:   "2025.44.1",
			wantSuffix: "hotfix",
			wantSeq:    2,
		},
		{
			name:       "custom suffix patch",
			input:      "v2.0.0-patch.3",
			wantBase:   "v2.0.0",
			wantSuffix: "patch",
			wantSeq:    3,
		},
		{
			name:       "custom suffix fix",
			input:      "v1.5.2-fix.10",
			wantBase:   "v1.5.2",
			wantSuffix: "fix",
			wantSeq:    10,
		},
		{
			name:    "not hotfix",
			input:   "v1.0.0",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "v1.0.0-hotfix",
			wantErr: true,
		},
		{
			name:    "invalid seq",
			input:   "v1.0.0-hotfix.abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, suffix, seq, err := ParseHotfixVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHotfixVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if base.String() != tt.wantBase {
				t.Errorf("ParseHotfixVersion(%q) base = %q, want %q", tt.input, base.String(), tt.wantBase)
			}
			if suffix != tt.wantSuffix {
				t.Errorf("ParseHotfixVersion(%q) suffix = %q, want %q", tt.input, suffix, tt.wantSuffix)
			}
			if seq != tt.wantSeq {
				t.Errorf("ParseHotfixVersion(%q) seq = %d, want %d", tt.input, seq, tt.wantSeq)
			}
		})
	}
}

func TestIncrementHotfixSequence(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "increment seq 1 to 2",
			input: "v1.0.0-hotfix.1",
			want:  "v1.0.0-hotfix.2",
		},
		{
			name:  "increment seq 5 to 6",
			input: "v1.0.0-hotfix.5",
			want:  "v1.0.0-hotfix.6",
		},
		{
			name:  "custom suffix",
			input: "v2.0.0-patch.3",
			want:  "v2.0.0-patch.4",
		},
		{
			name:  "calver",
			input: "2025.44-hotfix.2",
			want:  "2025.44-hotfix.3",
		},
		{
			name:    "not hotfix",
			input:   "v1.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IncrementHotfixSequence(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("IncrementHotfixSequence(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IncrementHotfixSequence(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// SemVer v2 spec compliance tests

func TestValidatePrereleaseIdentifiers(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty is valid", "", false},
		{"single alpha", "alpha", false},
		{"channel.number", "rc.1", false},
		{"multi-part", "alpha.1.beta.2", false},
		{"hyphenated", "pre-release.1", false},
		{"large number", "rc.999", false},
		{"empty identifier", "rc..1", true},
		{"leading dot", ".rc", true},
		{"trailing dot", "rc.", true},
		{"underscore", "rc_1", true},
		{"space", "rc 1", true},
		{"leading zero numeric", "rc.01", true},
		{"leading zero single segment", "01", true},
		{"zero is ok", "rc.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrereleaseIdentifiers(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrereleaseIdentifiers(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseSemVerValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid basic", "1.2.3", false},
		{"valid with pre", "1.2.3-rc.1", false},
		{"valid with meta", "1.2.3+build.001", false},
		{"leading zero major", "01.2.3", true},
		{"leading zero minor", "1.02.3", true},
		{"leading zero patch", "1.2.03", true},
		{"invalid pre underscore", "1.2.3-rc_1", true},
		{"invalid pre leading zero", "1.2.3-rc.01", true},
		{"empty pre identifier", "1.2.3-.rc", true},
		{"invalid meta underscore", "1.2.3+build_1", true},
		{"empty prerelease section", "1.2.3-", true},
		{"empty metadata section", "1.2.3+", true},
		{"empty prerelease with metadata", "1.2.3-+build.1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSemVer(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSemVer(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	// Spec §11 example ordering:
	// 1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-alpha.beta < 1.0.0-beta < 1.0.0-beta.2
	// < 1.0.0-beta.11 < 1.0.0-rc.1 < 1.0.0
	ordered := []string{
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0-alpha.beta",
		"1.0.0-beta",
		"1.0.0-beta.2",
		"1.0.0-beta.11",
		"1.0.0-rc.1",
		"1.0.0",
	}

	for i := 0; i < len(ordered)-1; i++ {
		a, err := ParseSemVer(ordered[i])
		if err != nil {
			t.Fatalf("ParseSemVer(%q) unexpected error: %v", ordered[i], err)
		}
		b, err := ParseSemVer(ordered[i+1])
		if err != nil {
			t.Fatalf("ParseSemVer(%q) unexpected error: %v", ordered[i+1], err)
		}
		if got := Compare(a, b); got != -1 {
			t.Errorf("Compare(%q, %q) = %d, want -1", ordered[i], ordered[i+1], got)
		}
		if got := Compare(b, a); got != 1 {
			t.Errorf("Compare(%q, %q) = %d, want 1", ordered[i+1], ordered[i], got)
		}
	}

	// Equality
	a, err := ParseSemVer("1.0.0")
	if err != nil {
		t.Fatalf("ParseSemVer(%q) unexpected error: %v", "1.0.0", err)
	}
	b, err := ParseSemVer("1.0.0")
	if err != nil {
		t.Fatalf("ParseSemVer(%q) unexpected error: %v", "1.0.0", err)
	}
	if got := Compare(a, b); got != 0 {
		t.Errorf("Compare equal versions = %d, want 0", got)
	}

	// Build metadata ignored for precedence (spec §10)
	c, err := ParseSemVer("1.0.0+build.1")
	if err != nil {
		t.Fatalf("ParseSemVer(%q) unexpected error: %v", "1.0.0+build.1", err)
	}
	d, err := ParseSemVer("1.0.0+build.2")
	if err != nil {
		t.Fatalf("ParseSemVer(%q) unexpected error: %v", "1.0.0+build.2", err)
	}
	if got := Compare(c, d); got != 0 {
		t.Errorf("Compare versions differing only in metadata = %d, want 0", got)
	}

	// Numeric vs alphanumeric: numeric has lower precedence
	e, err := ParseSemVer("1.0.0-1")
	if err != nil {
		t.Fatalf("ParseSemVer(%q) unexpected error: %v", "1.0.0-1", err)
	}
	f, err := ParseSemVer("1.0.0-alpha")
	if err != nil {
		t.Fatalf("ParseSemVer(%q) unexpected error: %v", "1.0.0-alpha", err)
	}
	if got := Compare(e, f); got != -1 {
		t.Errorf("Compare numeric < alpha = %d, want -1", got)
	}
}

func TestIsPrerelease(t *testing.T) {
	stable, err := ParseSemVer("1.2.3")
	if err != nil {
		t.Fatalf("ParseSemVer(%q) unexpected error: %v", "1.2.3", err)
	}
	pre, err := ParseSemVer("1.2.3-rc.1")
	if err != nil {
		t.Fatalf("ParseSemVer(%q) unexpected error: %v", "1.2.3-rc.1", err)
	}

	if stable.IsPrerelease() {
		t.Error("1.2.3 should not be a prerelease")
	}
	if !stable.IsStable() {
		t.Error("1.2.3 should be stable")
	}
	if !pre.IsPrerelease() {
		t.Error("1.2.3-rc.1 should be a prerelease")
	}
	if pre.IsStable() {
		t.Error("1.2.3-rc.1 should not be stable")
	}
}

func TestGraduate(t *testing.T) {
	pre, err := ParseSemVer("2.0.0-rc.2")
	if err != nil {
		t.Fatalf("ParseSemVer(%q) unexpected error: %v", "2.0.0-rc.2", err)
	}
	stable := pre.Graduate()
	if stable.String() != "2.0.0" {
		t.Errorf("Graduate() = %q, want %q", stable.String(), "2.0.0")
	}
	if stable.IsPrerelease() {
		t.Error("graduated version should be stable")
	}
}

func TestBumpPreRelease(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		channel string
		want    string
		wantErr bool
	}{
		{
			name:    "first alpha from stable",
			base:    "1.2.3",
			channel: "alpha",
			want:    "1.2.3-alpha.1",
		},
		{
			name:    "increment same channel",
			base:    "1.2.3-alpha.1",
			channel: "alpha",
			want:    "1.2.3-alpha.2",
		},
		{
			name:    "promote to rc",
			base:    "1.2.3-alpha.3",
			channel: "rc",
			want:    "1.2.3-rc.1",
		},
		{
			name:    "increment rc",
			base:    "2.0.0-rc.1",
			channel: "rc",
			want:    "2.0.0-rc.2",
		},
		{
			name:    "invalid channel with dot",
			base:    "1.0.0",
			channel: "rc.1",
			wantErr: true,
		},
		{
			name:    "invalid channel with underscore",
			base:    "1.0.0",
			channel: "my_channel",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := ParseSemVer(tt.base)
			if err != nil {
				t.Fatalf("ParseSemVer(%q) failed: %v", tt.base, err)
			}
			got, err := v.BumpPreRelease(tt.channel)
			if (err != nil) != tt.wantErr {
				t.Errorf("BumpPreRelease(%q) error = %v, wantErr %v", tt.channel, err, tt.wantErr)
				return
			}
			if err == nil && got.String() != tt.want {
				t.Errorf("BumpPreRelease(%q) = %q, want %q", tt.channel, got.String(), tt.want)
			}
		})
	}
}
