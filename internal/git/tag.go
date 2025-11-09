package git

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alexjoedt/forge/internal/log"
	"github.com/alexjoedt/forge/internal/run"
	"github.com/alexjoedt/forge/internal/version"
)

// Tagger handles git tag operations.
type Tagger struct {
	repoDir string
	prefix  string
	dryRun  bool
}

// NewTagger creates a new Tagger for the given repository directory.
func NewTagger(repoDir, prefix string, dryRun bool) *Tagger {
	return &Tagger{
		repoDir: repoDir,
		prefix:  prefix,
		dryRun:  dryRun,
	}
}

// LatestTag returns the latest tag with the configured prefix, or empty string if none exists.
func (t *Tagger) LatestTag(ctx context.Context) (string, error) {
	logger := log.FromContext(ctx)

	// List all tags matching the prefix, sorted by version
	result := run.CmdInDir(ctx, t.repoDir, "git", "tag", "-l", t.prefix+"*", "--sort=-version:refname")
	if !result.Success() {
		// If git tag fails, it might be because there are no tags yet
		if result.ExitCode == 0 || strings.Contains(result.Stderr, "not a git repository") {
			return "", fmt.Errorf("not a git repository or git not available: %s", result.Stderr)
		}
		// Empty output is fine - no tags yet
		if result.Stdout == "" {
			return "", nil
		}
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) == 0 || lines[0] == "" {
		logger.Debugf("no tags found with prefix %s", t.prefix)
		return "", nil
	}

	latestTag := lines[0]
	logger.Debugf("found latest tag: %s", latestTag)
	return latestTag, nil
}

// ParseLatestVersion returns the parsed version of the latest tag, or nil if no tag exists.
func (t *Tagger) ParseLatestVersion(ctx context.Context, scheme version.Scheme) (*version.Version, error) {
	tag, err := t.LatestTag(ctx)
	if err != nil {
		return nil, err
	}

	if tag == "" {
		return nil, nil
	}

	// Strip prefix
	versionStr := version.StripPrefix(tag, t.prefix)

	switch scheme {
	case version.SchemeSemVer:
		return version.ParseSemVer(versionStr)
	case version.SchemeCalVer:
		return version.ParseCalVer(versionStr)
	default:
		return nil, fmt.Errorf("unknown version scheme: %s", scheme)
	}
}

// TagExists checks if a tag already exists.
func (t *Tagger) TagExists(ctx context.Context, tag string) (bool, error) {
	result := run.CmdInDir(ctx, t.repoDir, "git", "tag", "-l", tag)
	if !result.Success() {
		return false, result.MustSucceed("check if tag exists")
	}

	return strings.TrimSpace(result.Stdout) != "", nil
}

// CreateTag creates an annotated tag with the given name and message.
// If dryRun is true, only logs the operation without creating the tag.
func (t *Tagger) CreateTag(ctx context.Context, tag, message string) error {
	logger := log.FromContext(ctx)

	if t.dryRun {
		logger.Debugf("dry-run: would create tag %s with message %s", tag, message)
		return nil
	}

	// Check if tag already exists
	exists, err := t.TagExists(ctx, tag)
	if err != nil {
		return fmt.Errorf("check tag existence: %w", err)
	}
	if exists {
		return fmt.Errorf("tag %s already exists", tag)
	}

	result := run.CmdInDir(ctx, t.repoDir, "git", "tag", "-a", tag, "-m", message)
	if err := result.MustSucceed("create tag"); err != nil {
		return err
	}

	logger.Debugf("created tag: %s", tag)
	return nil
}

// PushTag pushes the tag to the remote repository.
// If dryRun is true, only logs the operation without pushing.
func (t *Tagger) PushTag(ctx context.Context, tag string) error {
	logger := log.FromContext(ctx)

	if t.dryRun {
		logger.Debugf("dry-run: would push tag: %s", tag)
		return nil
	}

	result := run.CmdInDir(ctx, t.repoDir, "git", "push", "origin", tag)
	if err := result.MustSucceed("push tag"); err != nil {
		return err
	}

	logger.Debugf("pushed tag: %s", tag)
	return nil
}

// CurrentCommit returns the current commit hash.
func (t *Tagger) CurrentCommit(ctx context.Context) (string, error) {
	result := run.CmdInDir(ctx, t.repoDir, "git", "rev-parse", "HEAD")
	if err := result.MustSucceed("get current commit"); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// ShortCommit returns the short commit hash (first 7 characters).
func (t *Tagger) ShortCommit(ctx context.Context) (string, error) {
	result := run.CmdInDir(ctx, t.repoDir, "git", "rev-parse", "--short", "HEAD")
	if err := result.MustSucceed("get short commit"); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// HasUncommittedChanges checks if there are uncommitted changes in the repository.
func (t *Tagger) HasUncommittedChanges(ctx context.Context) (bool, error) {
	// Check for modified/added/deleted files
	result := run.CmdInDir(ctx, t.repoDir, "git", "status", "--porcelain")
	if !result.Success() {
		return false, result.MustSucceed("check git status")
	}
	return strings.TrimSpace(result.Stdout) != "", nil
}

// IsTagOnCurrentCommit checks if the given tag points to the current commit.
func (t *Tagger) IsTagOnCurrentCommit(ctx context.Context, tag string) (bool, error) {
	if tag == "" {
		return false, nil
	}

	// Get commit hash for the tag
	tagResult := run.CmdInDir(ctx, t.repoDir, "git", "rev-parse", tag+"^{}")
	if !tagResult.Success() {
		// Tag might not exist or be invalid
		return false, nil
	}
	tagCommit := strings.TrimSpace(tagResult.Stdout)

	// Get current commit hash
	headResult := run.CmdInDir(ctx, t.repoDir, "git", "rev-parse", "HEAD")
	if !headResult.Success() {
		return false, headResult.MustSucceed("get current commit")
	}
	headCommit := strings.TrimSpace(headResult.Stdout)

	return tagCommit == headCommit, nil
}

// CalculateNextVersion calculates the next version without creating a tag.
// This is useful when you need to know the version before making changes (e.g., updating package.json).
func (t *Tagger) CalculateNextVersion(ctx context.Context, scheme version.Scheme, bump version.BumpType, calverFormat, pre, meta string) (*version.Version, error) {
	// Get current version
	current, err := t.ParseLatestVersion(ctx, scheme)
	if err != nil {
		return nil, fmt.Errorf("parse latest version: %w", err)
	}

	var next *version.Version

	switch scheme {
	case version.SchemeSemVer:
		if current == nil {
			// No previous version, start at 0.1.0 or 1.0.0 depending on bump
			if bump == version.BumpMajor {
				next = &version.Version{Scheme: version.SchemeSemVer, Major: 1, Minor: 0, Patch: 0}
			} else {
				next = &version.Version{Scheme: version.SchemeSemVer, Major: 0, Minor: 1, Patch: 0}
			}
		} else {
			next = current.BumpSemVer(bump)
		}

	case version.SchemeCalVer:
		next = version.NextCalVer(current, calverFormat, time.Now())

	default:
		return nil, fmt.Errorf("unknown version scheme: %s", scheme)
	}

	// Apply prerelease and metadata
	if pre != "" {
		next = next.WithPrerelease(pre)
	}
	if meta != "" {
		next = next.WithMetadata(meta)
	}

	return next, nil
}

// CommitVersionUpdate commits a version file update (like package.json).
// It stages the file, creates a commit with a standard message.
func (t *Tagger) CommitVersionUpdate(ctx context.Context, filePath, version string) error {
	logger := log.FromContext(ctx)

	if t.dryRun {
		logger.Debugf("dry-run: would commit version update for %s", filePath)
		return nil
	}

	// Stage the file
	result := run.CmdInDir(ctx, t.repoDir, "git", "add", filePath)
	if err := result.MustSucceed("stage file"); err != nil {
		return err
	}

	// Create commit
	commitMsg := fmt.Sprintf("chore: bump version to %s", version)
	result = run.CmdInDir(ctx, t.repoDir, "git", "commit", "-m", commitMsg)
	if err := result.MustSucceed("commit version update"); err != nil {
		return err
	}

	logger.Debugf("committed version update: %s", commitMsg)
	return nil
}

// CreateNextTag generates the next tag based on the scheme and creates it.
// For semver, bump is required. For calver, the current date is used.
func (t *Tagger) CreateNextTag(ctx context.Context, scheme version.Scheme, bump version.BumpType, calverFormat, pre, meta string) (string, error) {
	logger := log.FromContext(ctx)

	// Get current version
	current, err := t.ParseLatestVersion(ctx, scheme)
	if err != nil {
		return "", fmt.Errorf("parse latest version: %w", err)
	}

	var next *version.Version

	switch scheme {
	case version.SchemeSemVer:
		if current == nil {
			// No previous version, start at 0.1.0 or 1.0.0 depending on bump
			if bump == version.BumpMajor {
				next = &version.Version{Scheme: version.SchemeSemVer, Major: 1, Minor: 0, Patch: 0}
			} else {
				next = &version.Version{Scheme: version.SchemeSemVer, Major: 0, Minor: 1, Patch: 0}
			}
		} else {
			next = current.BumpSemVer(bump)
		}

	case version.SchemeCalVer:
		next = version.NextCalVer(current, calverFormat, time.Now())

	default:
		return "", fmt.Errorf("unknown version scheme: %s", scheme)
	}

	// Apply prerelease and metadata
	if pre != "" {
		next = next.WithPrerelease(pre)
	}
	if meta != "" {
		next = next.WithMetadata(meta)
	}

	tag := version.WithPrefix(next.String(), t.prefix)
	message := fmt.Sprintf("forge: release %s", tag)

	logger.Debugf("creating next tag %s using %s scheme", tag, scheme)

	if err := t.CreateTag(ctx, tag, message); err != nil {
		return "", err
	}

	return tag, nil
}

// GetVersionWithDirtyCheck returns the version string, appending "-dirty-<short-commit>"
// if there are uncommitted changes or if the latest tag is not on the current commit.
// If no tags exist, it returns "0.0.0-dev".
func (t *Tagger) GetVersionWithDirtyCheck(ctx context.Context) (string, error) {
	logger := log.FromContext(ctx)

	// Get latest tag
	tag, err := t.LatestTag(ctx)
	if err != nil {
		return "", fmt.Errorf("get latest tag: %w", err)
	}

	// Get short commit
	shortCommit, err := t.ShortCommit(ctx)
	if err != nil {
		return "", fmt.Errorf("get short commit: %w", err)
	}

	// If no tags exist, return dev version
	if tag == "" {
		logger.Debugf("no tags found, using dev version")
		return fmt.Sprintf("0.0.0-dev-%s", shortCommit), nil
	}

	// Strip prefix from tag to get version
	versionStr := version.StripPrefix(tag, t.prefix)

	// Check if we need to mark as dirty
	isDirty := false

	// Check for uncommitted changes
	hasChanges, err := t.HasUncommittedChanges(ctx)
	if err != nil {
		logger.Warnf("failed to check for uncommitted changes: %v", err)
	} else if hasChanges {
		isDirty = true
		logger.Debugf("detected uncommitted changes")
	}

	// Check if tag is on current commit
	tagOnHead, err := t.IsTagOnCurrentCommit(ctx, tag)
	if err != nil {
		logger.Warnf("failed to check if tag is on current commit: %v", err)
	} else if !tagOnHead {
		isDirty = true
		logger.Debugf("latest tag is not on current commit")
	}

	// Append dirty suffix if needed
	if isDirty {
		versionStr = fmt.Sprintf("%s-dirty-%s", versionStr, shortCommit)
		logger.Debugf("marked version as dirty: %s", versionStr)
	}

	return versionStr, nil
}

// TagInfo represents information about a version tag.
type TagInfo struct {
	Tag     string
	Version string
	Commit  string
	Date    string
	Message string
}

// ListAllTags returns all tags with the configured prefix, sorted by version (newest first).
// For each tag, it includes the commit hash, date, and message.
func (t *Tagger) ListAllTags(ctx context.Context) ([]TagInfo, error) {
	logger := log.FromContext(ctx)

	// List all tags matching the prefix, sorted by version
	result := run.CmdInDir(ctx, t.repoDir, "git", "tag", "-l", t.prefix+"*", "--sort=-version:refname")
	if !result.Success() {
		// If git tag fails, it might be because there are no tags yet
		if result.ExitCode == 0 || strings.Contains(result.Stderr, "not a git repository") {
			return nil, fmt.Errorf("not a git repository or git not available: %s", result.Stderr)
		}
		// Empty output is fine - no tags yet
		if result.Stdout == "" {
			return []TagInfo{}, nil
		}
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	if len(lines) == 0 || lines[0] == "" {
		logger.Debugf("no tags found with prefix %s", t.prefix)
		return []TagInfo{}, nil
	}

	var tags []TagInfo
	for _, tag := range lines {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}

		// Get commit hash for the tag
		commitResult := run.CmdInDir(ctx, t.repoDir, "git", "rev-parse", tag+"^{}")
		if !commitResult.Success() {
			logger.Warnf("failed to get commit for tag %s: %v", tag, commitResult.Stderr)
			continue
		}
		commit := strings.TrimSpace(commitResult.Stdout)

		// Get commit date and message (from the commit, not the tag annotation)
		dateResult := run.CmdInDir(ctx, t.repoDir, "git", "log", "-1", "--format=%ci", commit)
		date := strings.TrimSpace(dateResult.Stdout)
		if !dateResult.Success() {
			logger.Warnf("failed to get date for tag %s", tag)
			date = ""
		}

		// Get commit message (the actual commit message, not the tag annotation)
		msgResult := run.CmdInDir(ctx, t.repoDir, "git", "log", "-1", "--format=%s", commit)
		message := strings.TrimSpace(msgResult.Stdout)
		if !msgResult.Success() {
			logger.Debugf("failed to get message for tag %s", tag)
			message = ""
		}

		// Strip prefix from tag to get version
		versionStr := version.StripPrefix(tag, t.prefix)

		tags = append(tags, TagInfo{
			Tag:     tag,
			Version: versionStr,
			Commit:  commit,
			Date:    date,
			Message: message,
		})
	}

	logger.Debugf("found %d tags with prefix %s", len(tags), t.prefix)
	return tags, nil
}

// GetTagInfo retrieves detailed information for a specific tag.
// It handles prefix auto-detection by trying both the exact tag name and with the configured prefix.
// Returns TagInfo with all fields populated, or an error if the tag doesn't exist.
func (t *Tagger) GetTagInfo(ctx context.Context, tagName string) (*TagInfo, error) {
	logger := log.FromContext(ctx)

	// Try multiple variations of the tag name to handle prefix auto-detection
	tagsToTry := []string{
		tagName,                    // Exact name as provided
		t.prefix + tagName,         // With configured prefix
	}

	// Remove duplicates (if tagName already has prefix)
	var uniqueTags []string
	seen := make(map[string]bool)
	for _, tag := range tagsToTry {
		if !seen[tag] && tag != "" {
			seen[tag] = true
			uniqueTags = append(uniqueTags, tag)
		}
	}

	// Try each variation until we find one that exists
	var foundTag string
	for _, tag := range uniqueTags {
		exists, err := t.TagExists(ctx, tag)
		if err != nil {
			return nil, fmt.Errorf("check tag existence: %w", err)
		}
		if exists {
			foundTag = tag
			logger.Debugf("found tag: %s", foundTag)
			break
		}
	}

	// If no tag found, return error
	if foundTag == "" {
		return nil, fmt.Errorf("tag not found: %s (tried with prefix '%s')", tagName, t.prefix)
	}

	// Get commit hash for the tag
	commitResult := run.CmdInDir(ctx, t.repoDir, "git", "rev-parse", foundTag+"^{}")
	if !commitResult.Success() {
		return nil, fmt.Errorf("get commit for tag %s: %s", foundTag, commitResult.Stderr)
	}
	commit := strings.TrimSpace(commitResult.Stdout)

	// Get commit date
	dateResult := run.CmdInDir(ctx, t.repoDir, "git", "log", "-1", "--format=%ci", commit)
	date := strings.TrimSpace(dateResult.Stdout)
	if !dateResult.Success() {
		logger.Warnf("failed to get date for tag %s", foundTag)
		date = ""
	}

	// Get commit message
	msgResult := run.CmdInDir(ctx, t.repoDir, "git", "log", "-1", "--format=%s", commit)
	message := strings.TrimSpace(msgResult.Stdout)
	if !msgResult.Success() {
		logger.Debugf("failed to get message for tag %s", foundTag)
		message = ""
	}

	// Strip prefix from tag to get version
	versionStr := version.StripPrefix(foundTag, t.prefix)

	return &TagInfo{
		Tag:     foundTag,
		Version: versionStr,
		Commit:  commit,
		Date:    date,
		Message: message,
	}, nil
}

// ============================================================================
// Hotfix Workflow Functions
// ============================================================================

// GetCurrentBranch returns the currently checked out branch name.
func GetCurrentBranch(repoDir string) (string, error) {
	result := run.Cmd(context.Background(), "git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	if !result.Success() {
		return "", fmt.Errorf("failed to get current branch: %s", result.Stderr)
	}
	return strings.TrimSpace(result.Stdout), nil
}

// IsHotfixBranch checks if current branch matches hotfix pattern.
func IsHotfixBranch(branchName, prefix string) bool {
	return strings.HasPrefix(branchName, prefix)
}

// ExtractTagFromBranch extracts the base tag from hotfix branch name.
// Example: "release/api/v1.0.0" with prefix "release/" → "api/v1.0.0"
func ExtractTagFromBranch(branchName, prefix string) (string, error) {
	if !strings.HasPrefix(branchName, prefix) {
		return "", fmt.Errorf("branch %q does not match prefix %q", branchName, prefix)
	}
	return strings.TrimPrefix(branchName, prefix), nil
}

// CreateHotfixBranch creates a new hotfix branch from specified tag.
func (t *Tagger) CreateHotfixBranch(ctx context.Context, tag, branchPrefix string, checkout bool) (string, error) {
	logger := log.FromContext(ctx)

	// Branch name is prefix + full tag
	branchName := branchPrefix + tag

	// Validate tag exists
	exists, err := t.TagExists(ctx, tag)
	if err != nil {
		return "", fmt.Errorf("failed to check if tag exists: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("tag %q does not exist", tag)
	}

	// Check if branch already exists
	if t.branchExists(ctx, branchName) {
		return "", fmt.Errorf("branch %q already exists\nCheckout with: git checkout %s", branchName, branchName)
	}

	if t.dryRun {
		logger.Debugf("dry-run: would create branch %s from tag %s", branchName, tag)
		return branchName, nil
	}

	// Create branch from tag
	result := run.CmdInDir(ctx, t.repoDir, "git", "branch", branchName, tag)
	if err := result.MustSucceed("create hotfix branch"); err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}

	logger.Debugf("created hotfix branch: %s", branchName)

	// Checkout if requested
	if checkout {
		result := run.CmdInDir(ctx, t.repoDir, "git", "checkout", branchName)
		if err := result.MustSucceed("checkout hotfix branch"); err != nil {
			return "", fmt.Errorf("failed to checkout branch: %w", err)
		}
		logger.Debugf("checked out branch: %s", branchName)
	}

	return branchName, nil
}

// GetNextHotfixTag determines next hotfix version from base tag.
// Example: base "v1.0.0", existing "v1.0.0-hotfix.2" → returns "v1.0.0-hotfix.3", seq 3
func (t *Tagger) GetNextHotfixTag(ctx context.Context, baseTag, suffix string) (string, int, error) {
	// List all hotfix tags for this base
	pattern := fmt.Sprintf("%s-%s.*", baseTag, suffix)
	tags, err := t.listTags(ctx, pattern)
	if err != nil {
		return "", 0, err
	}

	// Find highest sequence number
	maxSeq := 0
	for _, tag := range tags {
		seq, err := parseHotfixSequence(tag, baseTag, suffix)
		if err != nil {
			continue // Skip malformed tags
		}
		if seq > maxSeq {
			maxSeq = seq
		}
	}

	// Next sequence
	nextSeq := maxSeq + 1
	nextTag := fmt.Sprintf("%s-%s.%d", baseTag, suffix, nextSeq)

	return nextTag, nextSeq, nil
}

// CreateHotfixTag creates a hotfix tag from current HEAD.
func (t *Tagger) CreateHotfixTag(ctx context.Context, tag, message string) error {
	return t.CreateTag(ctx, tag, message)
}

// ListBranches returns all branches in the repository.
func ListBranches(repoDir string) ([]string, error) {
	result := run.Cmd(context.Background(), "git", "-C", repoDir, "branch", "--format=%(refname:short)")
	if !result.Success() {
		return nil, fmt.Errorf("failed to list branches: %s", result.Stderr)
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	branches := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// ValidateHotfixBaseTag ensures tag is valid for hotfix creation.
func ValidateHotfixBaseTag(ctx context.Context, repoDir, tag string) error {
	// Check if tag exists
	tagger := NewTagger(repoDir, "", false)
	exists, err := tagger.TagExists(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to check tag: %w", err)
	}
	if !exists {
		return fmt.Errorf("tag %q does not exist", tag)
	}

	// Cannot be a hotfix tag itself
	if version.IsHotfixVersion(tag) {
		base, _, _, _ := version.ParseHotfixVersion(tag)
		if base != nil {
			return fmt.Errorf("cannot create hotfix from hotfix version %q\nUse the base version instead: %s", tag, base.String())
		}
		return fmt.Errorf("cannot create hotfix from hotfix version %q", tag)
	}

	return nil
}

// ValidateWorkingTreeClean ensures working tree is clean.
func ValidateWorkingTreeClean(ctx context.Context, repoDir string) error {
	tagger := NewTagger(repoDir, "", false)
	hasChanges, err := tagger.HasUncommittedChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to check working tree: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("working tree has uncommitted changes\nCommit or stash changes before creating hotfix tag")
	}
	return nil
}

// Helper functions

// branchExists checks if a branch exists.
func (t *Tagger) branchExists(ctx context.Context, branchName string) bool {
	result := run.CmdInDir(ctx, t.repoDir, "git", "rev-parse", "--verify", branchName)
	return result.Success()
}

// listTags lists all tags matching the pattern.
func (t *Tagger) listTags(ctx context.Context, pattern string) ([]string, error) {
	result := run.CmdInDir(ctx, t.repoDir, "git", "tag", "-l", pattern)
	if !result.Success() {
		return nil, fmt.Errorf("failed to list tags: %s", result.Stderr)
	}

	if result.Stdout == "" {
		return []string{}, nil
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	tags := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}
	return tags, nil
}

// parseHotfixSequence extracts the sequence number from a hotfix tag.
func parseHotfixSequence(tag, baseTag, suffix string) (int, error) {
	// Expected format: baseTag-suffix.N
	expectedPrefix := fmt.Sprintf("%s-%s.", baseTag, suffix)
	if !strings.HasPrefix(tag, expectedPrefix) {
		return 0, fmt.Errorf("tag does not match expected format")
	}

	seqStr := strings.TrimPrefix(tag, expectedPrefix)
	seq, err := fmt.Sscanf(seqStr, "%d", new(int))
	if err != nil || seq != 1 {
		return 0, fmt.Errorf("invalid sequence number: %s", seqStr)
	}

	var seqNum int
	fmt.Sscanf(seqStr, "%d", &seqNum)
	return seqNum, nil
}
