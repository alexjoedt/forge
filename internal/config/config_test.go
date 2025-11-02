package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDockerConfig_GetRepositories(t *testing.T) {
	tests := []struct {
		name     string
		config   DockerConfig
		expected []string
	}{
		{
			name: "repositories set - should use repositories",
			config: DockerConfig{
				Repositories: []string{"ghcr.io/user/app", "docker.io/user/app"},
				Repository:   "old.io/user/app", // should be ignored
			},
			expected: []string{"ghcr.io/user/app", "docker.io/user/app"},
		},
		{
			name: "only repository set - backward compatibility",
			config: DockerConfig{
				Repository: "ghcr.io/user/app",
			},
			expected: []string{"ghcr.io/user/app"},
		},
		{
			name:     "neither set - should return empty",
			config:   DockerConfig{},
			expected: []string{},
		},
		{
			name: "empty repositories slice - should use repository",
			config: DockerConfig{
				Repositories: []string{},
				Repository:   "ghcr.io/user/app",
			},
			expected: []string{"ghcr.io/user/app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetRepositories()
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GetRepositories() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppConfig_Validate_DockerRepositoryWarning(t *testing.T) {
	// Test that validation warns when both repository and repositories are set
	cfg := &AppConfig{
		Version: VersionConfig{
			Scheme: "semver",
		},
		Git: GitConfig{
			TagPrefix:     "v",
			DefaultBranch: "main",
		},
		Docker: DockerConfig{
			Repository:   "ghcr.io/user/app",
			Repositories: []string{"docker.io/user/app", "registry.io/user/app"},
		},
	}

	// Validate should not return an error, but should log a warning
	// (we can't easily test for the warning output without mocking the logger,
	// but we can ensure validation still passes)
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() should not error when both repository fields are set, got: %v", err)
	}

	// Verify GetRepositories returns repositories, not repository
	repos := cfg.Docker.GetRepositories()
	expected := []string{"docker.io/user/app", "registry.io/user/app"}
	if !reflect.DeepEqual(repos, expected) {
		t.Errorf("GetRepositories() = %v, want %v (repository field should be ignored)", repos, expected)
	}
}

func TestLoadMultiAppConfig(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "forge.yaml")

	// Write the problematic config
	configContent := `# Forge Multi-App Configuration
defaultApp: monitoring
monitoring:
    version:
        scheme: semver
        prefix: monitoring/v
        calver_format: 2006.01.02
        pre: ""
        meta: ""
    build:
        name: monitoring
        main_path: ./cmd/monitoring/main.go
        targets:
            - linux/amd64
            - linux/arm64
        ldflags: -s -w -X main.version={{ .Version }}
        output_dir: dist
        binaries: []
    docker:
        enabled: true
        repository: ghcr.io/USER/api
        dockerfile: ./Dockerfile
        tags:
            - '{{ .Version }}'
            - latest
        platforms:
            - linux/amd64
            - linux/arm64
        build_args: {}
    git:
        tag_prefix: monitoring/v
        default_branch: master
hems:
    version:
        scheme: semver
        prefix: hems/v
        calver_format: 2006.WW
        pre: ""
        meta: ""
    build:
        name: hems
        main_path: ./cmd/rbems/
        targets:
            - linux/amd64
            - linux/arm64
        ldflags: -s -w -X main.version={{ .Version }}
        output_dir: bin
        binaries: []
    docker:
        enabled: true
        repository: ghcr.io/USER/worker
        dockerfile: ./Dockerfile
        tags:
            - '{{ .Version }}'
            - latest
        platforms:
            - linux/amd64
            - linux/arm64
        build_args: {}
    git:
        tag_prefix: hems/v
        default_branch: master
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load the config
	cfg, err := LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify DefaultApp
	if cfg.DefaultApp != "monitoring" {
		t.Errorf("Expected DefaultApp to be 'monitoring', got '%s'", cfg.DefaultApp)
	}

	// Verify both apps are loaded
	if len(cfg.Apps) != 2 {
		t.Fatalf("Expected 2 apps, got %d. Apps: %v", len(cfg.Apps), cfg.Apps)
	}

	// Verify monitoring app
	monitoring, ok := cfg.Apps["monitoring"]
	if !ok {
		t.Fatalf("monitoring app not found in config")
	}
	if monitoring.Build.Name != "monitoring" {
		t.Errorf("Expected monitoring.Build.Name to be 'monitoring', got '%s'", monitoring.Build.Name)
	}
	if monitoring.Build.MainPath != "./cmd/monitoring/main.go" {
		t.Errorf("Expected monitoring.Build.MainPath to be './cmd/monitoring/main.go', got '%s'", monitoring.Build.MainPath)
	}
	if monitoring.Git.TagPrefix != "monitoring/v" {
		t.Errorf("Expected monitoring.Git.TagPrefix to be 'monitoring/v', got '%s'", monitoring.Git.TagPrefix)
	}

	// Verify hems app
	hems, ok := cfg.Apps["hems"]
	if !ok {
		t.Fatalf("hems app not found in config")
	}
	if hems.Build.Name != "hems" {
		t.Errorf("Expected hems.Build.Name to be 'hems', got '%s'", hems.Build.Name)
	}
	if hems.Build.MainPath != "./cmd/rbems/" {
		t.Errorf("Expected hems.Build.MainPath to be './cmd/rbems/', got '%s'", hems.Build.MainPath)
	}
	if hems.Git.TagPrefix != "hems/v" {
		t.Errorf("Expected hems.Git.TagPrefix to be 'hems/v', got '%s'", hems.Git.TagPrefix)
	}
}

func TestLoadSingleAppConfig(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "forge.yaml")

	// Write a single-app config
	configContent := `version:
    scheme: semver
    prefix: v
    calver_format: 2006.01.02
build:
    name: myapp
    main_path: ./cmd/main.go
    targets:
        - linux/amd64
        - darwin/arm64
    ldflags: -s -w -X main.version={{ .Version }}
    output_dir: dist
docker:
    enabled: true
    repository: ghcr.io/user/myapp
    dockerfile: ./Dockerfile
    tags:
        - '{{ .Version }}'
        - latest
    platforms:
        - linux/amd64
git:
    tag_prefix: v
    default_branch: main
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load the config
	cfg, err := LoadFromDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify it loaded as single app
	if len(cfg.Apps) != 1 {
		t.Fatalf("Expected 1 app (single mode), got %d. Apps: %v", len(cfg.Apps), cfg.Apps)
	}

	// Get the single app
	app, err := cfg.GetAppConfig("")
	if err != nil {
		t.Fatalf("Failed to get app config: %v", err)
	}

	// Verify fields
	if app.Build.Name != "myapp" {
		t.Errorf("Expected Build.Name to be 'myapp', got '%s'", app.Build.Name)
	}
	if app.Build.MainPath != "./cmd/main.go" {
		t.Errorf("Expected Build.MainPath to be './cmd/main.go', got '%s'", app.Build.MainPath)
	}
	if app.Git.TagPrefix != "v" {
		t.Errorf("Expected Git.TagPrefix to be 'v', got '%s'", app.Git.TagPrefix)
	}
	if app.Version.Scheme != "semver" {
		t.Errorf("Expected Version.Scheme to be 'semver', got '%s'", app.Version.Scheme)
	}
}

func TestValidateAppConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      AppConfig
		wantErr     bool
		errContains string // Check if error contains this substring
	}{
		{
			name: "valid config",
			config: AppConfig{
				Version: VersionConfig{
					Scheme: "semver",
					Prefix: "v",
				},
				Git: GitConfig{
					TagPrefix:     "v",
					DefaultBranch: "main",
				},
			},
			wantErr: false,
		},
		{
			name: "missing version scheme",
			config: AppConfig{
				Version: VersionConfig{
					Prefix: "v",
				},
				Git: GitConfig{
					TagPrefix:     "v",
					DefaultBranch: "main",
				},
			},
			wantErr:     true,
			errContains: "version.scheme is required",
		},
		{
			name: "invalid version scheme",
			config: AppConfig{
				Version: VersionConfig{
					Scheme: "invalid",
					Prefix: "v",
				},
				Git: GitConfig{
					TagPrefix:     "v",
					DefaultBranch: "main",
				},
			},
			wantErr:     true,
			errContains: "invalid version.scheme: 'invalid'",
		},
		{
			name: "missing git tag prefix",
			config: AppConfig{
				Version: VersionConfig{
					Scheme: "semver",
					Prefix: "v",
				},
				Git: GitConfig{
					DefaultBranch: "main",
				},
			},
			wantErr:     true,
			errContains: "git.tag_prefix is required",
		},
		{
			name: "missing git default branch",
			config: AppConfig{
				Version: VersionConfig{
					Scheme: "semver",
					Prefix: "v",
				},
				Git: GitConfig{
					TagPrefix: "v",
				},
			},
			wantErr:     true,
			errContains: "git.default_branch is required",
		},
		{
			name: "calver without format",
			config: AppConfig{
				Version: VersionConfig{
					Scheme: "calver",
					Prefix: "v",
				},
				Git: GitConfig{
					TagPrefix:     "v",
					DefaultBranch: "main",
				},
			},
			wantErr:     true,
			errContains: "version.calver_format is required",
		},
		{
			name: "valid calver config",
			config: AppConfig{
				Version: VersionConfig{
					Scheme:       "calver",
					Prefix:       "v",
					CalVerFormat: "2006.01.02",
				},
				Git: GitConfig{
					TagPrefix:     "v",
					DefaultBranch: "main",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateCalVerFormatErrorMessage(t *testing.T) {
	config := AppConfig{
		Version: VersionConfig{
			Scheme: "calver",
			Prefix: "v",
		},
		Git: GitConfig{
			TagPrefix:     "v",
			DefaultBranch: "main",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Fatal("Expected error for missing calver_format, got nil")
	}

	errMsg := err.Error()

	// Check that error message contains key information
	expectedParts := []string{
		"version.calver_format is required",
		"Supported CalVer formats",
		"2006.01.02",
		"2006.WW",
		"Year.Week",
		"ISO week number",
	}

	for _, part := range expectedParts {
		if !strings.Contains(errMsg, part) {
			t.Errorf("Error message should contain '%s', but got:\n%s", part, errMsg)
		}
	}
}

func TestLoadMultiAppConfigWithInvalidApp(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "forge.yaml")

	// Write a multi-app config with one invalid app (missing git section)
	configContent := `defaultApp: forge

forge:
  version:
    scheme: semver
    prefix: v
  git:
    tag_prefix: v
    default_branch: main

testapp:
  version:
    scheme: calver
    calver_format: "2006.01.02"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load the config - should fail validation
	_, err := LoadFromDir(tmpDir)
	if err == nil {
		t.Fatal("Expected error when loading config with invalid app, got nil")
	}

	// Check that the error mentions testapp and git.tag_prefix
	if !strings.Contains(err.Error(), "invalid config for app 'testapp'") {
		t.Errorf("Expected error to contain 'invalid config for app 'testapp'', got '%s'", err.Error())
	}
	if !strings.Contains(err.Error(), "git.tag_prefix is required") {
		t.Errorf("Expected error to contain 'git.tag_prefix is required', got '%s'", err.Error())
	}
}
