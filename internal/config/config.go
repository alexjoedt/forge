package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Scheme        string        `yaml:"scheme"`                    // "semver" or "calver"
	Prefix        string        `yaml:"prefix"`                    // Tag prefix, e.g., "v", "api/v"
	DefaultBranch string        `yaml:"default_branch"`            // e.g., "main"
	CalVerFormat  string        `yaml:"calver_format,omitempty"`   // e.g., "2006.01.02", "2006.WW"
	Pre           string        `yaml:"pre,omitempty"`             // [ALPHA] prerelease identifier
	Meta          string        `yaml:"meta,omitempty"`            // [ALPHA] build metadata
	Hotfix        *HotfixConfig `yaml:"hotfix,omitempty"`          // Hotfix workflow settings
	NodeJS        NodeJSConfig  `yaml:"nodejs,omitempty"`          // Node.js package.json sync
}

// HotfixConfig holds hotfix workflow configuration.
type HotfixConfig struct {
	// Branch naming prefix. The full branch name is prefix + tag.
	// Examples: "release/" creates "release/v1.0.0"
	BranchPrefix string `yaml:"branch_prefix"` // Default: "release/"

	// Version suffix for hotfix tags
	// Examples: "hotfix", "patch", "fix"
	Suffix string `yaml:"suffix"` // Default: "hotfix"
}

// NodeJSConfig holds Node.js/npm package.json version sync settings.
type NodeJSConfig struct {
	Enabled     bool   `yaml:"enabled"`      // Enable package.json version updates
	PackagePath string `yaml:"package_path"` // Path to package.json (relative to repo root, defaults to "./package.json")
}

// Validate checks if the AppConfig has all required fields.
func (ac *AppConfig) Validate() error {
	if ac.Scheme == "" {
		return fmt.Errorf("scheme is required\n\n" +
			"  Add to your forge.yaml:\n" +
			"    scheme: semver  # or calver\n\n" +
			"  Valid schemes:\n" +
			"    • semver - Semantic Versioning (e.g., v1.2.3)\n" +
			"    • calver - Calendar Versioning (e.g., 2025.44.1)")
	}

	if ac.Scheme != "semver" && ac.Scheme != "calver" {
		return fmt.Errorf("invalid scheme: '%s'\n\n"+
			"  Valid schemes:\n"+
			"    • semver - Semantic Versioning (e.g., v1.2.3)\n"+
			"    • calver - Calendar Versioning (e.g., 2025.44.1)\n\n"+
			"  Fix your forge.yaml:\n"+
			"    scheme: semver  # or calver",
			ac.Scheme)
	}

	if ac.Prefix == "" {
		return fmt.Errorf("prefix is required\n\n" +
			"  Add to your forge.yaml:\n" +
			"    prefix: v\n\n" +
			"  Examples:\n" +
			"    • 'v' for tags like v1.2.3\n" +
			"    • 'api/v' for monorepo app tags like api/v1.2.3")
	}

	if ac.DefaultBranch == "" {
		return fmt.Errorf("default_branch is required\n\n" +
			"  Add to your forge.yaml:\n" +
			"    default_branch: main  # or master, develop, etc.")
	}

	// CalVer format required for CalVer scheme
	if ac.Scheme == "calver" && ac.CalVerFormat == "" {
		return fmt.Errorf("calver_format is required when using calver scheme\n\n" +
			"  Add to your forge.yaml:\n" +
			"    scheme: calver\n" +
			"    calver_format: \"2006.WW\"  # Choose a format below\n\n" +
			"  Supported CalVer formats:\n" +
			"    • \"2006.01.02\"     - Year.Month.Day (e.g., 2025.11.02)\n" +
			"    • \"2006.WW\"        - Year.Week (e.g., 2025.44) ⭐ Popular\n" +
			"    • \"2006.01\"        - Year.Month (e.g., 2025.11)\n" +
			"    • \"2006.01.02.03\"  - Year.Month.Day.Sequence\n\n" +
			"  Note: WW is a special code for ISO week number (01-53)\n" +
			"        Sequence numbers are auto-incremented for same-period releases")
	}

	return nil
}

// Default returns a default AppConfig for single app configuration.
func Default() *AppConfig {
	return &AppConfig{
		Scheme:        "semver",
		Prefix:        "v",
		DefaultBranch: "main",
		CalVerFormat:  "2006.01.02",
	}
}

// DefaultMulti returns a default Config for multi app configuration.
func DefaultMulti() *Config {
	apiConfig := Default()
	apiConfig.Prefix = "api/v"

	workerConfig := Default()
	workerConfig.Scheme = "calver"
	workerConfig.CalVerFormat = "2006.WW"
	workerConfig.Prefix = "worker/v"

	return &Config{
		DefaultApp: "api",
		Apps: map[string]AppConfig{
			"api":    *apiConfig,
			"worker": *workerConfig,
		},
	}
}

// isOldNestedFormat checks whether a raw config map (or a nested app config map)
// uses the legacy version:/git: block structure from before the config was flattened.
func isOldNestedFormat(raw map[string]any) bool {
	_, hasVersion := raw["version"]
	_, hasGit := raw["git"]
	return hasVersion || hasGit
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
	var raw map[string]any
	if err = yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Detect the old nested format (version:/git: blocks) and return a clear migration error.
	// Check the top-level keys for old format (single-app case).
	if isOldNestedFormat(raw) {
		return nil, fmt.Errorf("your forge.yaml uses the old nested format (version:/git: blocks) " +
			"which is no longer supported\n\n" +
			"  Migrate to the new flat format:\n\n" +
			"    # Old format (no longer supported):\n" +
			"    version:\n" +
			"      scheme: semver\n" +
			"      prefix: v\n" +
			"    git:\n" +
			"      default_branch: main\n\n" +
			"    # New flat format:\n" +
			"    scheme: semver\n" +
			"    prefix: v\n" +
			"    default_branch: main\n\n" +
			"  Run 'forge init' to generate a new configuration file.")
	}

	// Check if this is a multi-app config by looking for defaultApp or multiple app configs
	hasDefaultApp := false
	appCount := 0
	for key := range raw {
		if key == "defaultApp" {
			hasDefaultApp = true
			continue
		}
		// Check if this key looks like an app config (has nested structure with scheme/prefix/etc.)
		if val, ok := raw[key].(map[string]interface{}); ok {
			// Detect old nested format inside a multi-app entry
			if isOldNestedFormat(val) {
				return nil, fmt.Errorf("app %q in forge.yaml uses the old nested format (version:/git: blocks) "+
					"which is no longer supported\n\n"+
					"  Migrate to the new flat format:\n\n"+
					"    # Old format (no longer supported):\n"+
					"    %s:\n"+
					"      version:\n"+
					"        scheme: semver\n"+
					"        prefix: v\n"+
					"      git:\n"+
					"        default_branch: main\n\n"+
					"    # New flat format:\n"+
					"    %s:\n"+
					"      scheme: semver\n"+
					"      prefix: v\n"+
					"      default_branch: main\n\n"+
					"  Run 'forge init' to generate a new configuration file.",
					key, key, key)
			}
			if _, hasScheme := val["scheme"]; hasScheme {
				appCount++
			} else if _, hasPrefix := val["prefix"]; hasPrefix {
				appCount++
			}
		}
	}

	// If we have defaultApp or multiple apps, treat as multi-app config
	if hasDefaultApp || appCount > 1 {
		log.DefaultLogger.Debugf(
			"loading multi app configuration (detected: defaultApp=%v, apps=%d)",
			hasDefaultApp,
			appCount,
		)
		cfg := &Config{Apps: make(map[string]AppConfig)}
		if err = yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("unmarshal multi-app config: %w", err)
		}

		// Validate each app config
		for appName, appCfg := range cfg.Apps {
			if err = appCfg.Validate(); err != nil {
				return nil, fmt.Errorf("invalid config for app '%s': %w", appName, err)
			}
		}

		return cfg, nil
	}

	// Single app config
	log.DefaultLogger.Debugf("loading single app configuration")
	single := &AppConfig{}
	if err = yaml.Unmarshal(data, single); err != nil {
		return nil, fmt.Errorf("unmarshal single-app config: %w", err)
	}

	// Validate single app config
	if err = single.Validate(); err != nil {
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

	return nil, errors.New("no config found")
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

// GetHotfixConfig returns hotfix config with defaults applied.
func (ac *AppConfig) GetHotfixConfig() HotfixConfig {
	if ac.Hotfix != nil {
		cfg := *ac.Hotfix
		// Apply defaults for empty fields
		if cfg.BranchPrefix == "" {
			cfg.BranchPrefix = "release/"
		}
		if cfg.Suffix == "" {
			cfg.Suffix = "hotfix"
		}
		return cfg
	}
	// Return all defaults
	return HotfixConfig{
		BranchPrefix: "release/",
		Suffix:       "hotfix",
	}
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

// GetAllAppConfigs returns all app configurations as pointers
func (c *Config) GetAllAppConfigs() []*AppConfig {
	configs := make([]*AppConfig, 0, len(c.Apps))
	for _, cfg := range c.Apps {
		cfgCopy := cfg
		configs = append(configs, &cfgCopy)
	}
	return configs
}

// DetectAppFromTag determines which app config to use based on tag prefix.
// Returns empty string for single-app configs.
func (c *Config) DetectAppFromTag(tag string) (string, error) {
	// If single-app config, return empty string (no app needed)
	if !c.IsMultiApp() {
		return "", nil
	}

	// Iterate through apps, check if tag starts with app.Prefix
	for appName, app := range c.Apps {
		if strings.HasPrefix(tag, app.Prefix) {
			return appName, nil
		}
	}

	// If no match and default app exists, return default app
	if c.DefaultApp != "" {
		return c.DefaultApp, nil
	}

	// Otherwise error with available apps listed
	appNames := make([]string, 0, len(c.Apps))
	for name := range c.Apps {
		appNames = append(appNames, name)
	}
	return "", fmt.Errorf("cannot determine app from tag %q\nAvailable apps: %v", tag, appNames)
}

// ValidateAppTag ensures app name matches tag prefix.
func (c *Config) ValidateAppTag(appName, tag string) error {
	app, err := c.GetAppConfig(appName)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(tag, app.Prefix) {
		// Detect what app the tag suggests
		detectedApp, _ := c.DetectAppFromTag(tag)
		if detectedApp != "" && detectedApp != appName {
			log.DefaultLogger.Warnf("Tag %q suggests app %q, but --app=%s was specified", tag, detectedApp, appName)
		}
	}

	return nil
}
