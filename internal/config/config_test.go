package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	if monitoring.Git.TagPrefix != "monitoring/v" {
		t.Errorf("Expected monitoring.Git.TagPrefix to be 'monitoring/v', got '%s'", monitoring.Git.TagPrefix)
	}

	// Verify hems app
	hems, ok := cfg.Apps["hems"]
	if !ok {
		t.Fatalf("hems app not found in config")
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

// Hotfix Config Tests

func TestAppConfig_GetHotfixConfig(t *testing.T) {
	tests := []struct {
		name   string
		config AppConfig
		want   HotfixConfig
	}{
		{
			name: "explicit config",
			config: AppConfig{
				Git: GitConfig{
					Hotfix: &HotfixConfig{
						BranchPrefix: "hotfix/",
						Suffix:       "patch",
					},
				},
			},
			want: HotfixConfig{
				BranchPrefix: "hotfix/",
				Suffix:       "patch",
			},
		},
		{
			name: "empty branch prefix gets default",
			config: AppConfig{
				Git: GitConfig{
					Hotfix: &HotfixConfig{
						BranchPrefix: "",
						Suffix:       "hotfix",
					},
				},
			},
			want: HotfixConfig{
				BranchPrefix: "release/",
				Suffix:       "hotfix",
			},
		},
		{
			name: "empty suffix gets default",
			config: AppConfig{
				Git: GitConfig{
					Hotfix: &HotfixConfig{
						BranchPrefix: "release/",
						Suffix:       "",
					},
				},
			},
			want: HotfixConfig{
				BranchPrefix: "release/",
				Suffix:       "hotfix",
			},
		},
		{
			name: "nil hotfix config returns all defaults",
			config: AppConfig{
				Git: GitConfig{
					Hotfix: nil,
				},
			},
			want: HotfixConfig{
				BranchPrefix: "release/",
				Suffix:       "hotfix",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetHotfixConfig()
			if got.BranchPrefix != tt.want.BranchPrefix {
				t.Errorf("GetHotfixConfig().BranchPrefix = %q, want %q", got.BranchPrefix, tt.want.BranchPrefix)
			}
			if got.Suffix != tt.want.Suffix {
				t.Errorf("GetHotfixConfig().Suffix = %q, want %q", got.Suffix, tt.want.Suffix)
			}
		})
	}
}

func TestConfig_DetectAppFromTag(t *testing.T) {
	multiAppConfig := &Config{
		DefaultApp: "api",
		Apps: map[string]AppConfig{
			"api": {
				Git: GitConfig{TagPrefix: "api/v"},
			},
			"worker": {
				Git: GitConfig{TagPrefix: "worker/v"},
			},
			"web": {
				Git: GitConfig{TagPrefix: "web/"},
			},
		},
	}

	singleAppConfig := &Config{
		Apps: map[string]AppConfig{
			"single": {
				Git: GitConfig{TagPrefix: "v"},
			},
		},
	}

	tests := []struct {
		name    string
		config  *Config
		tag     string
		want    string
		wantErr bool
	}{
		{
			name:   "api tag detected",
			config: multiAppConfig,
			tag:    "api/v1.0.0",
			want:   "api",
		},
		{
			name:   "worker tag detected",
			config: multiAppConfig,
			tag:    "worker/v2.3.0",
			want:   "worker",
		},
		{
			name:   "web tag detected",
			config: multiAppConfig,
			tag:    "web/1.0.0",
			want:   "web",
		},
		{
			name:   "no match returns default app",
			config: multiAppConfig,
			tag:    "v1.0.0",
			want:   "api",
		},
		{
			name:   "single app returns empty",
			config: singleAppConfig,
			tag:    "v1.0.0",
			want:   "",
		},
		{
			name: "no match and no default app returns error",
			config: &Config{
				Apps: map[string]AppConfig{
					"api":    {Git: GitConfig{TagPrefix: "api/v"}},
					"worker": {Git: GitConfig{TagPrefix: "worker/v"}},
				},
			},
			tag:     "v1.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.DetectAppFromTag(tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectAppFromTag(%q) error = %v, wantErr %v", tt.tag, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DetectAppFromTag(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}

func TestConfig_ValidateAppTag(t *testing.T) {
	config := &Config{
		Apps: map[string]AppConfig{
			"api": {
				Git: GitConfig{TagPrefix: "api/v"},
			},
			"worker": {
				Git: GitConfig{TagPrefix: "worker/v"},
			},
		},
	}

	tests := []struct {
		name    string
		appName string
		tag     string
		wantErr bool
	}{
		{
			name:    "matching tag prefix",
			appName: "api",
			tag:     "api/v1.0.0",
			wantErr: false,
		},
		{
			name:    "non-matching tag prefix logs warning but no error",
			appName: "api",
			tag:     "worker/v1.0.0",
			wantErr: false,
		},
		{
			name:    "invalid app name",
			appName: "invalid",
			tag:     "v1.0.0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateAppTag(tt.appName, tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAppTag(%q, %q) error = %v, wantErr %v", tt.appName, tt.tag, err, tt.wantErr)
			}
		})
	}
}
