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
