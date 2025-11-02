package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/run"
)

// Builder handles Docker image building and tagging.
type Builder struct {
	repoDir    string
	dockerfile string
	context    string
	dryRun     bool
}

// NewBuilder creates a new Docker builder.
func NewBuilder(repoDir, dockerfile, context string, dryRun bool) *Builder {
	return &Builder{
		repoDir:    repoDir,
		dockerfile: dockerfile,
		context:    context,
		dryRun:     dryRun,
	}
}

// HasDockerfile checks if a Dockerfile exists.
func (b *Builder) HasDockerfile() bool {
	dockerfilePath := filepath.Join(b.repoDir, b.dockerfile)
	_, err := os.Stat(dockerfilePath)
	return err == nil
}

// TemplateData holds data for tag templating.
type TemplateData struct {
	Version     string
	CalVer      string
	Commit      string
	ShortCommit string
	Date        string
}

// expandTag applies template expansion to a tag string.
func expandTag(tagTemplate string, data TemplateData) (string, error) {
	tmpl, err := template.New("tag").Parse(tagTemplate)
	if err != nil {
		return "", fmt.Errorf("parse tag template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute tag template: %w", err)
	}

	return buf.String(), nil
}

func expandBuildArgs(argsTemplate string, data TemplateData) (string, error) {
	tmpl, err := template.New("build_args").Parse(argsTemplate)
	if err != nil {
		return "", fmt.Errorf("parse build_args template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute build_args template: %w", err)
	}

	return buf.String(), nil
}

// BuildOptions holds options for building a Docker image.
type BuildOptions struct {
	Repository   string   // Single repository, use Repositories for multiple
	Repositories []string // Multiple repositories to tag and push to
	Tags         []string // template strings
	Platforms    []string
	BuildArgs    map[string]string
	Push         bool
	Version      string
	Commit       string
	ShortCommit  string
}

// GetRepositories returns all configured repositories.
// If Repositories is set, it returns that. Otherwise, it returns Repository as a single-element slice for backward compatibility.
// Returns empty slice if neither is set.
func (opts *BuildOptions) GetRepositories() []string {
	if len(opts.Repositories) > 0 {
		return opts.Repositories
	}
	if opts.Repository != "" {
		return []string{opts.Repository}
	}
	return []string{}
}

// Build builds and optionally pushes a Docker image.
func (b *Builder) Build(ctx context.Context, opts BuildOptions) error {
	logger := log.FromContext(ctx)

	repositories := opts.GetRepositories()

	logger.Debugf("docker build started (repositories: %v, tag_templates: %v, platforms: %v, push: %t, version: %s, commit: %s, short_commit: %s, dockerfile: %s, context: %s, repo_dir: %s, dry_run: %t)", repositories, opts.Tags, opts.Platforms, opts.Push, opts.Version, opts.Commit, opts.ShortCommit, b.dockerfile, b.context, b.repoDir, b.dryRun)

	// Check if Dockerfile exists
	dockerfilePath := filepath.Join(b.repoDir, b.dockerfile)
	logger.Debugf("checking for Dockerfile at %s", dockerfilePath)

	if !b.HasDockerfile() {
		logger.Debugf("no Dockerfile found at %s, skipping image build", dockerfilePath)
		return nil
	}

	logger.Debugf("Dockerfile found at %s", dockerfilePath)

	// Prepare template data
	data := TemplateData{
		Version:     opts.Version,
		CalVer:      opts.Version, // TODO: extract calver from version if applicable
		Commit:      opts.Commit,
		ShortCommit: opts.ShortCommit,
		Date:        time.Now().UTC().Format("2006-01-02"),
	}

	// Expand tag templates
	expandedTagTemplates := []string{}
	logger.Debugf("expanding tag templates (templates: %v, template_data: %+v)", opts.Tags, data)

	for _, tagTemplate := range opts.Tags {
		tag, err := expandTag(tagTemplate, data)
		if err != nil {
			logger.Warnf("failed to expand tag template %s, using as-is: %v", tagTemplate, err)
			tag = tagTemplate
		}

		// When we have a latest tag, we also want to add here the appendix from the version tag
		if tag == "latest" {
			appendix := extractVersionAppendix(opts.Version)
			tag = tag + appendix
		}

		expandedTagTemplates = append(expandedTagTemplates, tag)
		logger.Debugf("expanded tag from template %s to %s", tagTemplate, tag)
	}

	// Generate additional semver tags (e.g., 1.2.3 -> 1.2, 1)
	// Only for clean (non-dirty) versions
	additionalVersionTags := generateAdditionalTags(opts.Version)
	logger.Debugf("generated additional version tags for %s: %v", opts.Version, additionalVersionTags)

	// Track which tags we've already added to avoid duplicates
	tagSet := make(map[string]bool)
	for _, tag := range expandedTagTemplates {
		tagSet[tag] = true
	}

	// Add additional semver tags if they don't already exist
	for _, versionTag := range additionalVersionTags {
		if !tagSet[versionTag] {
			expandedTagTemplates = append(expandedTagTemplates, versionTag)
			tagSet[versionTag] = true
			logger.Debugf("added additional semver tag: %s", versionTag)
		}
	}

	if len(expandedTagTemplates) == 0 {
		return fmt.Errorf("no tags specified")
	}

	// Build full tags by combining repositories with tags
	expandedTags := []string{}
	for _, repo := range repositories {
		for _, tag := range expandedTagTemplates {
			fullTag := fmt.Sprintf("%s:%s", repo, tag)
			expandedTags = append(expandedTags, fullTag)
			logger.Debugf("created full tag: %s", fullTag)
		}
	}

	if len(expandedTags) == 0 {
		return fmt.Errorf("no repositories specified")
	}

	for argKey, argTemplate := range opts.BuildArgs {
		arg, err := expandBuildArgs(argTemplate, data)
		if err != nil {
			logger.Warnf("failed to expand build arg template %s, using as-is: %v", argTemplate, err)
			arg = argTemplate
		}
		opts.BuildArgs[argKey] = arg
		fullArg := fmt.Sprintf("%s=%s", argKey, arg)
		logger.Debugf("expanded build argument from template %s to %s", argTemplate, fullArg)
	}

	// Determine build strategy based on platforms and push flag
	// Strategy 1: Push multi-platform manifest (when push is enabled)
	// Strategy 2: Build each platform separately and load (when push is disabled)
	if opts.Push && len(opts.Platforms) > 1 {
		// Multi-platform push: build all platforms together with manifest
		return b.buildMultiPlatformPush(ctx, expandedTags, opts)
	} else if len(opts.Platforms) > 1 {
		// Multi-platform local: build each platform separately to enable --load
		return b.buildMultiPlatformLoad(ctx, expandedTags, opts)
	} else {
		// Single platform: standard build with --load or --push
		return b.buildSinglePlatform(ctx, expandedTags, opts)
	}
}

// buildMultiPlatformPush builds all platforms together and pushes as a manifest list
func (b *Builder) buildMultiPlatformPush(ctx context.Context, tags []string, opts BuildOptions) error {
	logger := log.FromContext(ctx)

	logger.Debugf("building multi-platform docker image with push (dockerfile: %s, tags: %v, platforms: %v)", b.dockerfile, tags, opts.Platforms)

	args := []string{"buildx", "build"}

	// Add all platforms
	platformStr := strings.Join(opts.Platforms, ",")
	args = append(args, "--platform", platformStr)

	// Push the manifest
	args = append(args, "--push")

	// Add tags
	for _, tag := range tags {
		args = append(args, "-t", tag)
	}

	// Add build args
	for key, value := range opts.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add dockerfile and context
	args = append(args, "-f", filepath.Join(b.repoDir, b.dockerfile))
	contextPath := filepath.Join(b.repoDir, b.context)
	args = append(args, contextPath)

	logger.Debugf("executing multi-platform push build (args: %v, workdir: %s)", args, b.repoDir)

	if b.dryRun {
		logger.Debugf("dry-run: would build and push multi-platform image (dockerfile: %s, context: %s, tags: %v, platforms: %v)", b.dockerfile, b.context, tags, opts.Platforms)
		return nil
	}

	return b.executeDockerBuild(ctx, args, tags, true)
}

// buildMultiPlatformLoad builds each platform separately so they can be loaded
func (b *Builder) buildMultiPlatformLoad(ctx context.Context, tags []string, opts BuildOptions) error {
	logger := log.FromContext(ctx)

	logger.Debugf("building multi-platform docker image with separate platform builds (dockerfile: %s, tags: %v, platforms: %v) - building each platform separately to enable local loading", b.dockerfile, tags, opts.Platforms)

	if b.dryRun {
		for _, platform := range opts.Platforms {
			logger.Debugf("dry-run: would build image for platform %s (dockerfile: %s, context: %s, tags: %v)", platform, b.dockerfile, b.context, tags)
		}
		return nil
	}

	// Build each platform separately
	for i, platform := range opts.Platforms {
		logger.Debugf("building platform image for %s (%d/%d)", platform, i+1, len(opts.Platforms))

		// Create platform-specific tags
		platformTags := make([]string, len(tags))
		for j, tag := range tags {
			// Add platform suffix to tag for clarity
			// e.g., myapp:v1.0.0 -> myapp:v1.0.0-linux-amd64
			platformSuffix := strings.ReplaceAll(platform, "/", "-")
			platformTags[j] = fmt.Sprintf("%s-%s", tag, platformSuffix)
		}

		args := []string{"buildx", "build"}

		// Single platform
		args = append(args, "--platform", platform)

		// Load to local docker
		args = append(args, "--load")

		// Add platform-specific tags
		for _, tag := range platformTags {
			args = append(args, "-t", tag)
		}

		// Add build args
		for key, value := range opts.BuildArgs {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
		}

		// Add dockerfile and context
		args = append(args, "-f", filepath.Join(b.repoDir, b.dockerfile))
		contextPath := filepath.Join(b.repoDir, b.context)
		args = append(args, contextPath)

		logger.Debugf("executing platform-specific build for %s (args: %v, workdir: %s)", platform, args, b.repoDir)

		if err := b.executeDockerBuild(ctx, args, platformTags, false); err != nil {
			return fmt.Errorf("build platform %s: %w", platform, err)
		}

		logger.Infof("platform image built and loaded for %s (tags: %v)", platform, platformTags)
	}

	logger.Infof("all platform images built and loaded (%d platforms)", len(opts.Platforms))

	return nil
}

// buildSinglePlatform builds a single platform image
func (b *Builder) buildSinglePlatform(ctx context.Context, tags []string, opts BuildOptions) error {
	logger := log.FromContext(ctx)

	platform := ""
	if len(opts.Platforms) == 1 {
		platform = opts.Platforms[0]
	}

	logger.Infof("building single platform docker image (dockerfile: %s, tags: %v, platform: %s, push: %t)", b.dockerfile, tags, platform, opts.Push)

	args := []string{"buildx", "build"}

	// Add platform if specified
	if platform != "" {
		args = append(args, "--platform", platform)
	}

	// Add push or load flag
	if opts.Push {
		args = append(args, "--push")
	} else {
		args = append(args, "--load")
	}

	// Add tags
	for _, tag := range tags {
		args = append(args, "-t", tag)
	}

	// Add build args
	for key, value := range opts.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add dockerfile and context
	args = append(args, "-f", filepath.Join(b.repoDir, b.dockerfile))
	contextPath := filepath.Join(b.repoDir, b.context)
	args = append(args, contextPath)

	logger.Debugf("executing single platform build (args: %v, workdir: %s)", args, b.repoDir)

	if b.dryRun {
		logger.Infof("dry-run: would build docker image (dockerfile: %s, context: %s, tags: %v, platform: %s, push: %t)", b.dockerfile, b.context, tags, platform, opts.Push)
		return nil
	}

	return b.executeDockerBuild(ctx, args, tags, opts.Push)
}

// executeDockerBuild runs the docker build command and handles output
func (b *Builder) executeDockerBuild(ctx context.Context, args []string, tags []string, pushed bool) error {
	logger := log.FromContext(ctx)

	logger.Infof("running docker buildx command: docker %s", strings.Join(args, " "))

	result := run.CmdInDir(ctx, b.repoDir, "docker", args...)

	// Print stdout and stderr to console for visibility
	if result.Stdout != "" {
		fmt.Println("=== Docker Build Output (stdout) ===")
		fmt.Println(result.Stdout)
		fmt.Println("=== End stdout ===")
	}
	if result.Stderr != "" {
		fmt.Println("=== Docker Build Output (stderr) ===")
		fmt.Println(result.Stderr)
		fmt.Println("=== End stderr ===")
	}

	// Log the result details
	logger.Debugf("docker build result",
		"exitCode", result.ExitCode,
		"stdout_length", len(result.Stdout),
		"stderr_length", len(result.Stderr),
		"success", result.Success())

	if err := result.MustSucceed("docker build"); err != nil {
		logger.Errorf("docker build failed",
			"error", err,
			"exitCode", result.ExitCode,
			"stderr", result.Stderr)
		return err
	}

	if pushed {
		logger.Infof("docker image built and pushed (tags: %v)", tags)
	} else {
		logger.Infof("docker image built (tags: %v)", tags)
	}

	return nil
}

// CheckDocker verifies that docker is available.
func CheckDocker(ctx context.Context) error {
	result := run.Cmd(ctx, "docker", "version")
	if !result.Success() {
		return fmt.Errorf("docker is not available: %w", result.Err)
	}
	return nil
}

// CheckBuildx verifies that docker buildx is available.
func CheckBuildx(ctx context.Context) error {
	result := run.Cmd(ctx, "docker", "buildx", "version")
	if !result.Success() {
		return fmt.Errorf("docker buildx is not available: %w", result.Err)
	}
	return nil
}

// extractVersionAppendix extracts appended strings like
// v2025.40.1-dirty-1234 -> -dirty-1234
// v1.3.1-dirty-1234 -> -dirty-1234
func extractVersionAppendix(version string) string {
	parts := strings.SplitN(version, "-", 2)
	if len(parts) < 2 {
		return ""
	}
	return "-" + parts[1]
}

// generateAdditionalTags creates additional Docker tags from a version string.
// For clean semver versions like "v1.2.3" or "1.2.3", it generates:
//   - "1.2.3" (full version)
//   - "1.2" (major.minor)
//   - "1" (major only)
//
// For dirty builds or non-semver versions, it returns only the original version.
func generateAdditionalTags(version string) []string {
	// Check if this is a dirty build - skip additional tags
	if strings.Contains(version, "-dirty") {
		return []string{version}
	}

	// Remove 'v' prefix if present
	cleanVersion := strings.TrimPrefix(version, "v")

	// Check if there are any suffixes (prerelease or metadata)
	// If there are, only use the full version
	if strings.Contains(cleanVersion, "-") || strings.Contains(cleanVersion, "+") {
		return []string{cleanVersion}
	}

	// Split by dots
	parts := strings.Split(cleanVersion, ".")

	// Must have at least 3 parts for semver (major.minor.patch)
	if len(parts) < 3 {
		return []string{cleanVersion}
	}

	// Verify all parts are numeric (basic semver validation)
	for _, part := range parts[:3] {
		if _, err := strconv.Atoi(part); err != nil {
			// Not a valid semver, return as-is
			return []string{cleanVersion}
		}
	}

	// Generate the tags: major.minor.patch, major.minor, major
	tags := []string{
		cleanVersion,                 // 1.2.3
		strings.Join(parts[:2], "."), // 1.2
		parts[0],                     // 1
	}

	return tags
}
