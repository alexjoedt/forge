package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexjoedt/forge/internal/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultApp string               `yaml:"defaultApp"`
	Apps       map[string]AppConfig `yaml:",inline"`
}

// type Config map[string]AppConfig

// AppConfig represents the forge.yaml configuration file structure.
type AppConfig struct {
	Version VersionConfig `yaml:"version"`
	Build   BuildConfig   `yaml:"build"`
	Docker  DockerConfig  `yaml:"docker"`
	Git     GitConfig     `yaml:"git"`
	NodeJS  NodeJSConfig  `yaml:"nodejs"`
}

// VersionConfig holds version scheme settings.
type VersionConfig struct {
	Scheme       string `yaml:"scheme"`        // "semver" or "calver"
	Prefix       string `yaml:"prefix"`        // e.g., "v"
	CalVerFormat string `yaml:"calver_format"` // e.g., "2006.01.02", "2006.WW" (supports WW for ISO week)
	Pre          string `yaml:"pre"`           // [ALPHA] prerelease identifier - not fully implemented, do not use in production
	Meta         string `yaml:"meta"`          // [ALPHA] build metadata - not fully implemented, do not use in production
}

// Binary represents a single binary to build.
type Binary struct {
	Name    string `yaml:"name"`    // Binary name (e.g., "forge", "cli-tool")
	Path    string `yaml:"path"`    // Path to main.go (e.g., "./cmd/forge", ".")
	LDFlags string `yaml:"ldflags"` // Optional ldflags override for this binary
}

// BuildConfig holds build settings.
type BuildConfig struct {
	Name      string   `yaml:"name"`       // Binary name for single-app builds (optional, defaults to repo dir basename)
	MainPath  string   `yaml:"main_path"`  // Path to main.go (e.g., "./cmd/main.go")
	Targets   []string `yaml:"targets"`    // ["linux/amd64", "darwin/arm64", ...]
	LDFlags   string   `yaml:"ldflags"`    // template allowed (default for all binaries)
	OutputDir string   `yaml:"output_dir"` // default "dist"
	Binaries  []Binary `yaml:"binaries"`   // List of binaries to build (optional, defaults to single binary)
}

// DockerConfig holds Docker image build settings.
type DockerConfig struct {
	Enabled      bool              `yaml:"enabled"`
	Repository   string            `yaml:"repository"`   // Single repository, use Repositories for multiple (e.g., "ghcr.io/USER/forge")
	Repositories []string          `yaml:"repositories"` // Multiple repositories (e.g., ["ghcr.io/USER/forge", "docker.io/USER/forge"])
	Dockerfile   string            `yaml:"dockerfile"`   // default "./Dockerfile"
	Tags         []string          `yaml:"tags"`         // template strings
	Platforms    []string          `yaml:"platforms"`    // ["linux/amd64", "linux/arm64"]
	BuildArgs    map[string]string `yaml:"build_args"`
}

// GetRepositories returns all configured repositories.
// If Repositories is set, it returns that. Otherwise, it returns Repository as a single-element slice for backward compatibility.
// Returns empty slice if neither is set.
func (dc *DockerConfig) GetRepositories() []string {
	if len(dc.Repositories) > 0 {
		return dc.Repositories
	}
	if dc.Repository != "" {
		return []string{dc.Repository}
	}
	return []string{}
}

// GitConfig holds git-related settings.
type GitConfig struct {
	TagPrefix     string `yaml:"tag_prefix"`     // e.g., "v"
	DefaultBranch string `yaml:"default_branch"` // e.g., "main"
}

// NodeJSConfig holds Node.js/npm package.json version sync settings.
type NodeJSConfig struct {
	Enabled     bool   `yaml:"enabled"`      // Enable package.json version updates
	PackagePath string `yaml:"package_path"` // Path to package.json (relative to repo root, defaults to "./package.json")
}

// Validate checks if the AppConfig has all required fields
func (ac *AppConfig) Validate() error {
	// Version config is required
	if ac.Version.Scheme == "" {
		return fmt.Errorf("version.scheme is required\n\n" +
			"  Add to your forge.yaml:\n" +
			"    version:\n" +
			"      scheme: semver  # or calver\n\n" +
			"  Valid schemes:\n" +
			"    • semver - Semantic Versioning (e.g., v1.2.3)\n" +
			"    • calver - Calendar Versioning (e.g., 2025.44.1)")
	}

	if ac.Version.Scheme != "semver" && ac.Version.Scheme != "calver" {
		return fmt.Errorf("invalid version.scheme: '%s'\n\n"+
			"  Valid schemes:\n"+
			"    • semver - Semantic Versioning (e.g., v1.2.3)\n"+
			"    • calver - Calendar Versioning (e.g., 2025.44.1)\n\n"+
			"  Fix your forge.yaml:\n"+
			"    version:\n"+
			"      scheme: semver  # or calver",
			ac.Version.Scheme)
	}

	// Git config is required
	if ac.Git.TagPrefix == "" {
		return fmt.Errorf("git.tag_prefix is required\n\n" +
			"  Add to your forge.yaml:\n" +
			"    git:\n" +
			"      tag_prefix: v  # or empty string for no prefix\n\n" +
			"  Examples:\n" +
			"    • 'v' for tags like v1.2.3\n" +
			"    • 'api/v' for monorepo app tags like api/v1.2.3\n" +
			"    • '' (empty) for tags without prefix like 1.2.3")
	}

	if ac.Git.DefaultBranch == "" {
		return fmt.Errorf("git.default_branch is required\n\n" +
			"  Add to your forge.yaml:\n" +
			"    git:\n" +
			"      default_branch: main  # or master, develop, etc.")
	}

	// CalVer format required for CalVer scheme
	if ac.Version.Scheme == "calver" && ac.Version.CalVerFormat == "" {
		return fmt.Errorf("version.calver_format is required when using calver scheme\n\n" +
			"  Add to your forge.yaml:\n" +
			"    version:\n" +
			"      scheme: calver\n" +
			"      calver_format: \"2006.WW\"  # Choose a format below\n\n" +
			"  Supported CalVer formats:\n" +
			"    • \"2006.01.02\"     - Year.Month.Day (e.g., 2025.11.02)\n" +
			"    • \"2006.WW\"        - Year.Week (e.g., 2025.44) ⭐ Popular\n" +
			"    • \"2006.01\"        - Year.Month (e.g., 2025.11)\n" +
			"    • \"2006.01.02.03\"  - Year.Month.Day.Sequence\n\n" +
			"  Note: WW is a special code for ISO week number (01-53)\n" +
			"        Sequence numbers are auto-incremented for same-period releases")
	}

	// Warn if both repository and repositories are set
	if ac.Docker.Repository != "" && len(ac.Docker.Repositories) > 0 {
		log.DefaultLogger.Warnf("both 'docker.repository' and 'docker.repositories' are set in forge.yaml - 'docker.repository' will be ignored, only 'docker.repositories' will be used")
	}

	return nil
}

// Default returns a default AppConfig for single app configuration.
func Default() *AppConfig {
	return &AppConfig{
		Version: VersionConfig{
			Scheme:       "semver",
			Prefix:       "v",
			CalVerFormat: "2006.01.02",
			Pre:          "",
			Meta:         "",
		},
		Build: BuildConfig{
			Name:     "forge",
			MainPath: "./cmd/main.go",
			Targets: []string{
				"linux/amd64",
				"linux/arm64",
				"darwin/amd64",
				"darwin/arm64",
				"windows/amd64",
			},
			LDFlags:   "-s -w -X main.version={{ .Version }}",
			OutputDir: "dist",
			Binaries:  []Binary{},
		},
		Docker: DockerConfig{
			Enabled:    true,
			Repository: "ghcr.io/USER/forge",
			Dockerfile: "./Dockerfile",
			Tags:       []string{"{{ .Version }}", "latest"},
			Platforms:  []string{"linux/amd64", "linux/arm64"},
			BuildArgs:  make(map[string]string),
		},
		Git: GitConfig{
			TagPrefix:     "v",
			DefaultBranch: "main",
		},
		NodeJS: NodeJSConfig{
			Enabled:     false,
			PackagePath: "",
		},
	}
}

// DefaultMulti returns a default Config for multi app configuration.
func DefaultMulti() *Config {
	apiConfig := Default()
	apiConfig.Version.Prefix = "v"
	apiConfig.Git.TagPrefix = "api/v"
	apiConfig.Build.Name = "api"
	apiConfig.Build.MainPath = "./cmd/api/main.go"
	apiConfig.Docker.Repository = "ghcr.io/USER/api"

	workerConfig := Default()
	workerConfig.Version.Scheme = "calver"
	workerConfig.Version.CalVerFormat = "2006.WW"
	workerConfig.Version.Prefix = "v"
	workerConfig.Git.TagPrefix = "worker/v"
	workerConfig.Build.Name = "worker"
	workerConfig.Build.MainPath = "./cmd/worker/main.go"
	workerConfig.Build.Targets = []string{"linux/amd64", "linux/arm64"}
	workerConfig.Docker.Repository = "ghcr.io/USER/worker"

	return &Config{
		DefaultApp: "api",
		Apps: map[string]AppConfig{
			"api":    *apiConfig,
			"worker": *workerConfig,
		},
	}
}

// Load reads the configuration from the specified path.
// If the file doesn't exist, returns the default configuration.
func Load(path string) (*Config, error) {
	// If path is not absolute, make it relative to current directory
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		path = filepath.Join(wd, path)
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, return defaults
		return nil, fmt.Errorf("forge config file does not exists: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Try to detect config type by checking for defaultApp field
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Check if this is a multi-app config by looking for defaultApp or multiple app configs
	hasDefaultApp := false
	appCount := 0
	for key := range raw {
		if key == "defaultApp" {
			hasDefaultApp = true
			continue
		}
		// Check if this key looks like an app config (has nested structure with version/build/docker/git)
		if val, ok := raw[key].(map[string]interface{}); ok {
			// Look for config sections
			if _, hasVersion := val["version"]; hasVersion {
				appCount++
			} else if _, hasBuild := val["build"]; hasBuild {
				appCount++
			} else if _, hasDocker := val["docker"]; hasDocker {
				appCount++
			} else if _, hasGit := val["git"]; hasGit {
				appCount++
			}
		}
	}

	// If we have defaultApp or multiple apps, treat as multi-app config
	if hasDefaultApp || appCount > 1 {
		log.DefaultLogger.Debugf("loading multi app configuration (detected: defaultApp=%v, apps=%d)", hasDefaultApp, appCount)
		cfg := &Config{Apps: make(map[string]AppConfig)}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("unmarshal multi-app config: %w", err)
		}

		// Validate each app config
		for appName, appCfg := range cfg.Apps {
			if err := appCfg.Validate(); err != nil {
				return nil, fmt.Errorf("invalid config for app '%s': %w", appName, err)
			}
		}

		return cfg, nil
	}

	// Single app config
	log.DefaultLogger.Debugf("loading single app configuration")
	single := &AppConfig{}
	if err := yaml.Unmarshal(data, single); err != nil {
		return nil, fmt.Errorf("unmarshal single-app config: %w", err)
	}

	// Validate single app config
	if err := single.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Config{Apps: map[string]AppConfig{
		"single": *single,
	}}, nil
}

// LoadFromDir loads the forge.yaml configuration from the specified directory.
// It looks for forge.yaml or .forge.yaml in that directory.
func LoadFromDir(dir string) (*Config, error) {
	// Try forge.yaml first, then .forge.yaml
	for _, name := range []string{"forge.yaml", ".forge.yaml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}
	// No config file found, return defaults
	return nil, fmt.Errorf("no config found in repo: %s", dir)
}

func (c *Config) GetFirst() (*AppConfig, error) {
	for _, ac := range c.Apps {
		return &ac, nil
	}

	return nil, fmt.Errorf("no config found")
}

func (c *Config) GetAppConfig(app string) (*AppConfig, error) {
	if app == "" {
		app = c.DefaultApp
	}

	if app == "" {
		return c.GetFirst()
	}

	for name, appCfg := range c.Apps {
		if app == name {
			return &appCfg, nil
		}
	}

	return nil, fmt.Errorf("no config found for '%s'", app)
}

// IsMultiApp returns true if this is a multi-app configuration
func (c *Config) IsMultiApp() bool {
	// If there's more than one app, or if defaultApp is set, it's multi-app
	return len(c.Apps) > 1 || c.DefaultApp != ""
}

// GetAllApps returns all app configurations with their names
func (c *Config) GetAllApps() map[string]AppConfig {
	return c.Apps
}
