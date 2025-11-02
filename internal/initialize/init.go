package initialize

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexjoedt/forge/internal/config"
	"github.com/alexjoedt/forge/internal/log"
	"gopkg.in/yaml.v3"
)

const defaultConfigTemplate = `# Forge Configuration
# This file configures the forge CLI tool for managing releases, builds, and Docker images.

version:
  # Version scheme: "semver" (semantic versioning) or "calver" (calendar versioning)
  scheme: %s
  
  # Prefix for git tags (e.g., "v" creates tags like "v1.2.3")
  prefix: %s
  
  # CalVer format string (Go time format) - used when scheme is "calver"
  calver_format: "%s"
  
  # Prerelease identifier (e.g., "rc.1", "beta.2")
  pre: "%s"
  
  # Build metadata (e.g., "build.123")
  meta: "%s"

build:
  # Target platforms for builds (format: "OS/ARCH")
  targets:
%s
  
  # Linker flags template (supports Go template syntax)
  # Available variables: .Version, .Commit, .ShortCommit, .Date, .OS, .Arch
  ldflags: "%s"
  
  # Output directory for built binaries
  output_dir: %s

docker:
  # Enable/disable Docker image builds
  enabled: %t
  
  # Docker image repository (e.g., "ghcr.io/username/project")
  repository: %s
  
  # Path to Dockerfile
  dockerfile: %s
  
  # Image tags (supports Go template syntax)
  # Available variables: .Version, .CalVer, .Commit, .ShortCommit, .Date
  tags:
%s
  
  # Target platforms for multi-arch builds
  platforms:
%s
  
  # Build arguments (key-value pairs)
  build_args: {}

git:
  # Tag prefix (e.g., "v")
  tag_prefix: %s
  
  # Default branch name
  default_branch: %s
`

const multiAppConfigHeader = `# Forge Multi-App Configuration
# This file configures the forge CLI tool for managing multiple applications in a monorepo.
# Each application has its own versioning, build, and Docker settings.
#
# Usage:
#   forge bump mino --app api             # Tag the API app
#   forge build --app worker              # Build the worker app
#   forge image --app api --push          # Build and push API Docker image
#
# If no --app flag is specified, the defaultApp will be used.

`

// Options holds options for initializing a forge.yaml file.
type Options struct {
	OutputPath string
	Force      bool
	DryRun     bool
	Multi      bool
}

// Init creates a new forge.yaml configuration file with default values.
func Init(ctx context.Context, opts Options) error {
	logger := log.FromContext(ctx)

	// Determine output path
	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = "forge.yaml"
	}

	// Make path absolute
	if !filepath.IsAbs(outputPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		outputPath = filepath.Join(wd, outputPath)
	}

	logger.Debugf("initializing forge config (path: %s, force: %t, dry_run: %t)", outputPath, opts.Force, opts.DryRun)

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		if !opts.Force {
			return fmt.Errorf("config file already exists: %s (use --force to overwrite)", outputPath)
		}
		logger.Warnf("overwriting existing config file at %s", outputPath)
	}

	// Get default config

	var content string
	var err error
	if opts.Multi {
		cfg := config.DefaultMulti()
		// Generate YAML content with header
		yamlContent, err := generateContent(cfg)
		if err != nil {
			return fmt.Errorf("generate YAML content: %w", err)
		}
		content = multiAppConfigHeader + yamlContent
	} else {
		content, err = generateContent(config.Default())
		if err != nil {
			return fmt.Errorf("generate YAML content: %w", err)
		}
	}

	if opts.DryRun {
		logger.Infof("dry-run: would create config file at %s", outputPath)
		fmt.Println("---")
		fmt.Println(content)
		fmt.Println("---")
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	logger.Infof("config file created at %s", outputPath)
	return nil
}

// generateYAMLContent generates formatted YAML content from a Config struct.
func generateYAMLContent(cfg *config.AppConfig) (string, error) {
	// Format targets as YAML list
	targetsYAML := ""
	for _, target := range cfg.Build.Targets {
		targetsYAML += fmt.Sprintf("    - %s\n", target)
	}

	// Format docker tags as YAML list
	tagsYAML := ""
	for _, tag := range cfg.Docker.Tags {
		tagsYAML += fmt.Sprintf("    - \"%s\"\n", tag)
	}

	// Format docker platforms as YAML list
	platformsYAML := ""
	for _, platform := range cfg.Docker.Platforms {
		platformsYAML += fmt.Sprintf("    - %s\n", platform)
	}

	content := fmt.Sprintf(defaultConfigTemplate,
		cfg.Version.Scheme,
		cfg.Version.Prefix,
		cfg.Version.CalVerFormat,
		cfg.Version.Pre,
		cfg.Version.Meta,
		targetsYAML,
		cfg.Build.LDFlags,
		cfg.Build.OutputDir,
		cfg.Docker.Enabled,
		cfg.Docker.Repository,
		cfg.Docker.Dockerfile,
		tagsYAML,
		platformsYAML,
		cfg.Git.TagPrefix,
		cfg.Git.DefaultBranch,
	)

	return content, nil
}

func generateYAMLContentMulti(cfg *config.Config) (string, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func generateContent(v any) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// InitWithCustomConfig creates a forge.yaml with a custom configuration.
func InitWithCustomConfig(ctx context.Context, cfg *config.AppConfig, opts Options) error {
	logger := log.FromContext(ctx)

	// Determine output path
	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = "forge.yaml"
	}

	// Make path absolute
	if !filepath.IsAbs(outputPath) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		outputPath = filepath.Join(wd, outputPath)
	}

	logger.Debugf("initializing forge config (path: %s, force: %t, dry_run: %t)", outputPath, opts.Force, opts.DryRun)

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		if !opts.Force {
			return fmt.Errorf("config file already exists: %s (use --force to overwrite)", outputPath)
		}
		logger.Warnf("overwriting existing config file at %s", outputPath)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config to YAML: %w", err)
	}

	if opts.DryRun {
		logger.Infof("dry-run: would create config file at %s", outputPath)
		fmt.Println("---")
		fmt.Println(string(data))
		fmt.Println("---")
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	logger.Infof("config file created at %s", outputPath)
	return nil
}
