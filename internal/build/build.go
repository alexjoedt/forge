package build

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/run"
)

// Target represents a GOOS/GOARCH build target.
type Target struct {
	OS   string
	Arch string
}

// ParseTarget parses a target string like "linux/amd64" into a Target.
func ParseTarget(s string) (Target, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return Target{}, fmt.Errorf("invalid target format: %s (expected OS/ARCH)", s)
	}
	return Target{OS: parts[0], Arch: parts[1]}, nil
}

// String returns the target as "OS/ARCH".
func (t Target) String() string {
	return fmt.Sprintf("%s/%s", t.OS, t.Arch)
}

// Builder handles building Go binaries for multiple targets.
type Builder struct {
	repoDir   string
	outputDir string
	dryRun    bool
}

// NewBuilder creates a new Builder.
func NewBuilder(repoDir, outputDir string, dryRun bool) *Builder {
	return &Builder{
		repoDir:   repoDir,
		outputDir: outputDir,
		dryRun:    dryRun,
	}
}

// TemplateData holds data for ldflags templating.
type TemplateData struct {
	Version     string
	Commit      string
	ShortCommit string
	Date        string
	OS          string
	Arch        string
}

// applyTemplate applies a template string with the given data.
func applyTemplate(tmpl string, data TemplateData) (string, error) {
	t, err := template.New("ldflags").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// BuildOptions holds options for building a binary.
type BuildOptions struct {
	Target      Target
	LDFlags     string
	Version     string
	Commit      string
	ShortCommit string
	Date        string
	BinaryName  string // Optional custom binary name (if empty, derived from repo dir)
	MainPath    string // Path to main.go directory  "./cmd/forge", "./cmd/server/main.og" or "."
}

// Build builds the binary with the given options.
func (b *Builder) Build(ctx context.Context, opts BuildOptions) error {
	logger := log.FromContext(ctx)

	// Determine binary name
	binaryName := filepath.Base(b.repoDir)
	if binaryName == "" || binaryName == "." {
		binaryName = "app"
	}

	// Add .exe for Windows
	if opts.Target.OS == "windows" {
		binaryName += ".exe"
	}

	// Create output directory path: dist/<os>-<arch>/
	targetDir := filepath.Join(b.outputDir, fmt.Sprintf("%s-%s", opts.Target.OS, opts.Target.Arch))
	outputPath := filepath.Join(targetDir, binaryName)

	// Create target directory
	if !b.dryRun {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	// Prepare template data with commit information
	data := TemplateData{
		Version:     opts.Version,
		Commit:      opts.Commit,
		ShortCommit: opts.ShortCommit,
		Date:        opts.Date,
		OS:          opts.Target.OS,
		Arch:        opts.Target.Arch,
	}

	// Apply ldflags template
	resolvedLDFlags := opts.LDFlags
	if opts.LDFlags != "" {
		var err error
		resolvedLDFlags, err = applyTemplate(opts.LDFlags, data)
		if err != nil {
			logger.Warnf("failed to apply ldflags template, using as-is: %v", err)
		}
	}

	logger.Debugf("building for target %s to output %s", opts.Target.String(), outputPath)

	if b.dryRun {
		logger.Debugf("dry-run: would build (GOOS: %s, GOARCH: %s, output: %s, ldflags: %s)", opts.Target.OS, opts.Target.Arch, outputPath, resolvedLDFlags)
		return nil
	}

	// Build command
	args := []string{"build"}
	if resolvedLDFlags != "" {
		args = append(args, "-ldflags", resolvedLDFlags)
	}
	args = append(args, "-o", outputPath, ".")

	// Execute build
	cmd := run.CmdInDir(ctx, b.repoDir, "go", args...)

	// Set environment variables
	// TODO: Use exec.Cmd directly to set env vars properly
	// For now, we willl need to enhance the run package or use a workaround

	if err := cmd.MustSucceed("build " + opts.Target.String()); err != nil {
		return err
	}

	logger.Debugf("built successfully for target %s to output %s", opts.Target.String(), outputPath)
	return nil
}

// BuildWithEnvVars builds with explicit environment variables using the shell.
func (b *Builder) BuildWithEnvVars(ctx context.Context, opts BuildOptions) error {
	logger := log.FromContext(ctx)

	// Determine binary name
	binaryName := opts.BinaryName
	if binaryName == "" {
		// If no custom name, derive from main path or repo dir
		if opts.MainPath != "" && opts.MainPath != "." {
			// Extract directory name, not file name
			// "./cmd/main.go" -> "cmd", "./cmd/api" -> "api"
			mainDir := filepath.Dir(opts.MainPath)
			if mainDir != "" && mainDir != "." {
				binaryName = filepath.Base(mainDir)
			} else {
				binaryName = filepath.Base(b.repoDir)
			}
		} else {
			binaryName = filepath.Base(b.repoDir)
		}
		if binaryName == "" || binaryName == "." {
			binaryName = "app"
		}
	}

	// Add .exe for Windows
	if opts.Target.OS == "windows" {
		binaryName += ".exe"
	}

	// Determine main package path
	mainPath := opts.MainPath
	if mainPath == "" {
		mainPath = "."
	}

	// Create output directory path: dist/<os>-<arch>/
	targetDir := filepath.Join(b.outputDir, fmt.Sprintf("%s-%s", opts.Target.OS, opts.Target.Arch))
	outputPath := filepath.Join(targetDir, binaryName)

	// Make output path absolute
	if !filepath.IsAbs(outputPath) {
		absPath := filepath.Join(b.repoDir, outputPath)
		outputPath = absPath
	}

	// Create target directory
	if !b.dryRun {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	// Prepare template data with commit information
	data := TemplateData{
		Version:     opts.Version,
		Commit:      opts.Commit,
		ShortCommit: opts.ShortCommit,
		Date:        opts.Date,
		OS:          opts.Target.OS,
		Arch:        opts.Target.Arch,
	}

	// Apply ldflags template
	resolvedLDFlags := opts.LDFlags
	if opts.LDFlags != "" {
		var err error
		resolvedLDFlags, err = applyTemplate(opts.LDFlags, data)
		if err != nil {
			logger.Warnf("failed to apply ldflags template, using as-is: %v", err)
		}
	}

	logger.Debugf("building for target %s to output %s", opts.Target.String(), outputPath)

	if b.dryRun {
		logger.Debugf("dry-run: would build (GOOS: %s, GOARCH: %s, output: %s, ldflags: %s)", opts.Target.OS, opts.Target.Arch, outputPath, resolvedLDFlags)
		return nil
	}

	// Build command string with env vars
	envVars := []string{
		fmt.Sprintf("GOOS=%s", opts.Target.OS),
		fmt.Sprintf("GOARCH=%s", opts.Target.Arch),
		"CGO_ENABLED=0",
	}

	// Build command
	cmdParts := []string{}
	cmdParts = append(cmdParts, envVars...)
	cmdParts = append(cmdParts, "go", "build")
	if resolvedLDFlags != "" {
		cmdParts = append(cmdParts, "-ldflags", fmt.Sprintf("%q", resolvedLDFlags))
	}
	cmdParts = append(cmdParts, "-o", outputPath, mainPath)

	cmdStr := strings.Join(cmdParts, " ")

	// Execute via shell to handle env vars
	result := run.CmdInDir(ctx, b.repoDir, "sh", "-c", cmdStr)
	if err := result.MustSucceed("build " + opts.Target.String()); err != nil {
		return err
	}

	logger.Debugf("built successfully for target %s to output %s", opts.Target.String(), outputPath)
	return nil
}

// BuildWithEnv builds with explicit environment variables (deprecated, use BuildWithEnvVars).
func (b *Builder) BuildWithEnv(ctx context.Context, target Target, ldflags, version string, env map[string]string) error {
	logger := log.FromContext(ctx)

	// Determine binary name
	binaryName := filepath.Base(b.repoDir)
	if binaryName == "" || binaryName == "." {
		binaryName = "app"
	}

	// Add .exe for Windows
	if target.OS == "windows" {
		binaryName += ".exe"
	}

	// Create output directory path: dist/<os>-<arch>/
	targetDir := filepath.Join(b.outputDir, fmt.Sprintf("%s-%s", target.OS, target.Arch))
	outputPath := filepath.Join(targetDir, binaryName)

	// Make output path absolute
	if !filepath.IsAbs(outputPath) {
		absPath := filepath.Join(b.repoDir, outputPath)
		outputPath = absPath
	}

	// Create target directory
	if !b.dryRun {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
	}

	// TODO: Get commit info for template data
	data := TemplateData{
		Version:     version,
		Commit:      "unknown",
		ShortCommit: "unknown",
		Date:        "unknown",
		OS:          target.OS,
		Arch:        target.Arch,
	}

	// Apply ldflags template
	resolvedLDFlags := ldflags
	if ldflags != "" {
		var err error
		resolvedLDFlags, err = applyTemplate(ldflags, data)
		if err != nil {
			logger.Warnf("failed to apply ldflags template, using as-is: %v", err)
		}
	}

	logger.Debugf("building for target %s to output %s", target.String(), outputPath)

	if b.dryRun {
		logger.Debugf("dry-run: would build (GOOS: %s, GOARCH: %s, output: %s, ldflags: %s, env: %v)", target.OS, target.Arch, outputPath, resolvedLDFlags, env)
		return nil
	}

	// Build command string with env vars
	envVars := []string{}
	for k, v := range env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}

	// Add GOOS and GOARCH
	envVars = append(envVars, fmt.Sprintf("GOOS=%s", target.OS))
	envVars = append(envVars, fmt.Sprintf("GOARCH=%s", target.Arch))
	envVars = append(envVars, "CGO_ENABLED=0")

	// Build command
	cmdParts := []string{}
	cmdParts = append(cmdParts, envVars...)
	cmdParts = append(cmdParts, "go", "build")
	if resolvedLDFlags != "" {
		cmdParts = append(cmdParts, "-ldflags", fmt.Sprintf("%q", resolvedLDFlags))
	}
	cmdParts = append(cmdParts, "-o", outputPath, ".")

	cmdStr := strings.Join(cmdParts, " ")

	// Execute via shell to handle env vars
	result := run.CmdInDir(ctx, b.repoDir, "sh", "-c", cmdStr)
	if err := result.MustSucceed("build " + target.String()); err != nil {
		return err
	}

	logger.Debugf("built successfully for target %s to output %s", target.String(), outputPath)
	return nil
}

// BuildAllOptions holds options for building all targets.
type BuildAllOptions struct {
	Targets     []string
	LDFlags     string
	Version     string
	Commit      string
	ShortCommit string
	Date        string
	BinaryName  string // Optional custom binary name
	MainPath    string // Optional path to main.go directory
}

// BuildAll builds for all specified targets.
func (b *Builder) BuildAll(ctx context.Context, opts BuildAllOptions) error {
	logger := log.FromContext(ctx)
	logger.Debugf("building for %d targets", len(opts.Targets))

	for _, targetStr := range opts.Targets {
		target, err := ParseTarget(targetStr)
		if err != nil {
			return fmt.Errorf("parse target %s: %w", targetStr, err)
		}

		// Build with environment variables for proper GOOS/GOARCH handling
		buildOpts := BuildOptions{
			Target:      target,
			LDFlags:     opts.LDFlags,
			Version:     opts.Version,
			Commit:      opts.Commit,
			ShortCommit: opts.ShortCommit,
			Date:        opts.Date,
			BinaryName:  opts.BinaryName,
			MainPath:    opts.MainPath,
		}

		if err := b.BuildWithEnvVars(ctx, buildOpts); err != nil {
			return fmt.Errorf("build %s: %w", target.String(), err)
		}
	}

	logger.Infof("all builds completed successfully")
	return nil
}

// BinaryBuildSpec specifies a binary to build.
type BinaryBuildSpec struct {
	Name    string // Binary name
	Path    string // Path to main.go directory
	LDFlags string // LDFlags for this binary (overrides default)
}

// BuildMultiOptions holds options for building multiple binaries across multiple targets.
type BuildMultiOptions struct {
	MainPath    string
	Targets     []string
	Binaries    []BinaryBuildSpec
	LDFlags     string // Default ldflags
	Version     string
	Commit      string
	ShortCommit string
	Date        string
}

// BuildMulti builds multiple binaries for all specified targets.
func (b *Builder) BuildMulti(ctx context.Context, opts BuildMultiOptions) error {
	logger := log.FromContext(ctx)
	logger.Infof("building %d binaries for %d targets", len(opts.Binaries), len(opts.Targets))

	for _, binary := range opts.Binaries {
		logger.Infof("building binary %s from path %s", binary.Name, binary.Path)

		// Determine ldflags (binary-specific or default)
		ldflags := binary.LDFlags
		if ldflags == "" {
			ldflags = opts.LDFlags
		}

		// Build for all targets
		for _, targetStr := range opts.Targets {
			target, err := ParseTarget(targetStr)
			if err != nil {
				return fmt.Errorf("parse target %s: %w", targetStr, err)
			}

			buildOpts := BuildOptions{
				Target:      target,
				LDFlags:     ldflags,
				Version:     opts.Version,
				Commit:      opts.Commit,
				ShortCommit: opts.ShortCommit,
				Date:        opts.Date,
				BinaryName:  binary.Name,
				MainPath:    binary.Path,
			}

			if err := b.BuildWithEnvVars(ctx, buildOpts); err != nil {
				return fmt.Errorf("build %s for %s: %w", binary.Name, target.String(), err)
			}
		}
	}

	logger.Infof("all builds completed successfully")
	return nil
}
