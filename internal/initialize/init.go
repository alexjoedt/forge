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

const multiAppConfigHeader = `# Forge Multi-App Configuration
# This file configures the forge CLI tool for managing multiple apps in a monorepo.
# Each application has its own version scheme and git tag prefix.
#
# Usage:
#   forge bump minor --app api            # Tag the API app
#   forge version --app api               # Show current version
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
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(outputPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	logger.Infof("config file created at %s", outputPath)
	return nil
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
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(outputPath, data, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	logger.Infof("config file created at %s", outputPath)
	return nil
}
