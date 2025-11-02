package docker

import (
	"reflect"
	"testing"
)

func TestBuildOptions_GetRepositories(t *testing.T) {
	tests := []struct {
		name     string
		opts     BuildOptions
		expected []string
	}{
		{
			name: "repositories set - should use repositories",
			opts: BuildOptions{
				Repositories: []string{"ghcr.io/user/app", "docker.io/user/app"},
				Repository:   "old.io/user/app", // should be ignored
			},
			expected: []string{"ghcr.io/user/app", "docker.io/user/app"},
		},
		{
			name: "only repository set - backward compatibility",
			opts: BuildOptions{
				Repository: "ghcr.io/user/app",
			},
			expected: []string{"ghcr.io/user/app"},
		},
		{
			name:     "neither set - should return empty",
			opts:     BuildOptions{},
			expected: []string{},
		},
		{
			name: "empty repositories slice - should use repository",
			opts: BuildOptions{
				Repositories: []string{},
				Repository:   "ghcr.io/user/app",
			},
			expected: []string{"ghcr.io/user/app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.opts.GetRepositories()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GetRepositories() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractAppendix(t *testing.T) {

	in := "v2025.40.1-dirty-124"
	want := "-dirty-124"

	got := extractVersionAppendix(in)
	if got != want {
		t.Errorf("got '%s'; want '%s'", got, want)
		t.Fail()
	}
}

func TestGenerateAdditionalTags(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected []string
	}{
		{
			name:     "clean semver with v prefix",
			version:  "v1.2.3",
			expected: []string{"1.2.3", "1.2", "1"},
		},
		{
			name:     "clean semver without v prefix",
			version:  "1.2.3",
			expected: []string{"1.2.3", "1.2", "1"},
		},
		{
			name:     "dirty build should return only full version",
			version:  "v1.2.3-dirty-abc123",
			expected: []string{"v1.2.3-dirty-abc123"},
		},
		{
			name:     "version with prerelease should return only full version",
			version:  "v1.2.3-beta.1",
			expected: []string{"1.2.3-beta.1"},
		},
		{
			name:     "version with metadata should return only full version",
			version:  "v1.2.3+build.123",
			expected: []string{"1.2.3+build.123"},
		},
		{
			name:     "version with both prerelease and metadata",
			version:  "v1.2.3-rc.1+build.456",
			expected: []string{"1.2.3-rc.1+build.456"},
		},
		{
			name:     "major version only",
			version:  "v2.0.0",
			expected: []string{"2.0.0", "2.0", "2"},
		},
		{
			name:     "multi-digit versions",
			version:  "v12.34.56",
			expected: []string{"12.34.56", "12.34", "12"},
		},
		{
			name:     "calver should return only full version",
			version:  "2025.40.1",
			expected: []string{"2025.40.1", "2025.40", "2025"},
		},
		{
			name:     "invalid semver format returns as-is",
			version:  "v1.2",
			expected: []string{"1.2"},
		},
		{
			name:     "non-numeric version returns as-is",
			version:  "latest",
			expected: []string{"latest"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateAdditionalTags(tt.version)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("generateAdditionalTags(%q) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}
