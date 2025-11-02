package nodejs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alexjoedt/forge/internal/log"
)

// PackageJSON represents the minimal structure of a package.json file
// for version management purposes.
type PackageJSON struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version"`
}

// Updater handles package.json version updates.
type Updater struct {
	repoDir string
	dryRun  bool
}

// NewUpdater creates a new package.json updater for the given repository directory.
func NewUpdater(repoDir string, dryRun bool) *Updater {
	return &Updater{
		repoDir: repoDir,
		dryRun:  dryRun,
	}
}

// FindPackageJSON searches for package.json in the repository.
// If path is provided, it uses that path (relative to repoDir).
// Otherwise, it looks for package.json in the repository root.
func (u *Updater) FindPackageJSON(ctx context.Context, path string) (string, error) {
	logger := log.FromContext(ctx)

	var packagePath string
	if path != "" {
		// Use provided path
		if filepath.IsAbs(path) {
			packagePath = path
		} else {
			packagePath = filepath.Join(u.repoDir, path)
		}
	} else {
		// Look in repository root
		packagePath = filepath.Join(u.repoDir, "package.json")
	}

	// Check if file exists
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		logger.Debugf("package.json not found at %s", packagePath)
		return "", nil
	}

	logger.Debugf("found package.json at %s", packagePath)
	return packagePath, nil
}

// ReadVersion reads the current version from package.json.
func (u *Updater) ReadVersion(ctx context.Context, packagePath string) (string, error) {
	data, err := os.ReadFile(packagePath)
	if err != nil {
		return "", fmt.Errorf("read package.json: %w", err)
	}

	// Parse as generic map to preserve all fields
	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", fmt.Errorf("parse package.json: %w", err)
	}

	version, ok := pkg["version"].(string)
	if !ok {
		return "", fmt.Errorf("version field not found or not a string in package.json")
	}

	return version, nil
}

// UpdateVersion updates the version in package.json while preserving formatting and comments.
// It uses regex to replace only the version field value, leaving everything else intact.
// Returns true if the version was changed, false if it was already the target version.
func (u *Updater) UpdateVersion(ctx context.Context, packagePath, newVersion string) (bool, error) {
	logger := log.FromContext(ctx)

	if u.dryRun {
		logger.Debugf("dry-run: would update version in %s to %s", packagePath, newVersion)
		return true, nil
	}

	// Read the file as raw text
	data, err := os.ReadFile(packagePath)
	if err != nil {
		return false, fmt.Errorf("read package.json: %w", err)
	}

	content := string(data)

	// Try to validate it as JSON (may fail if it has comments, which is OK for package.json)
	// We'll still try to update it using regex
	var pkg map[string]interface{}
	var oldVersion string
	
	// Create a version without comments for validation
	contentNoComments := stripJSONComments(content)
	if err := json.Unmarshal([]byte(contentNoComments), &pkg); err != nil {
		return false, fmt.Errorf("parse package.json: %w", err)
	}

	if v, ok := pkg["version"].(string); ok {
		oldVersion = v
	} else {
		return false, fmt.Errorf("version field not found or not a string in package.json")
	}

	// Check if version is already the target version
	if oldVersion == newVersion {
		logger.Debugf("package.json version is already %s, no update needed", newVersion)
		return false, nil
	}

	// Use regex to find and replace the version field value
	// This pattern matches "version": "x.x.x" with various whitespace/quote styles
	// Pattern explanation:
	// - "version"\s*:\s* matches "version" followed by colon with optional whitespace
	// - (["']) captures the quote type (single or double)
	// - [^"']+ matches the current version (anything except quotes)
	// - (["']) matches the closing quote
	versionPattern := regexp.MustCompile(`("version"\s*:\s*)(["'])([^"']+)(["'])`)

	// Check if pattern matches
	if !versionPattern.MatchString(content) {
		return false, fmt.Errorf("could not find version field in package.json")
	}

	// Replace the version value while preserving quotes and formatting
	newContent := versionPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the parts using submatch
		parts := versionPattern.FindStringSubmatch(match)
		if len(parts) != 5 {
			return match
		}
		// parts[1] = "version": 
		// parts[2] = opening quote
		// parts[3] = old version
		// parts[4] = closing quote
		return parts[1] + parts[2] + newVersion + parts[4]
	})

	// Verify the replacement worked by checking if new version is in the content
	if !strings.Contains(newContent, newVersion) {
		return false, fmt.Errorf("failed to update version in package.json")
	}

	// Write back the modified content
	if err := os.WriteFile(packagePath, []byte(newContent), 0644); err != nil {
		return false, fmt.Errorf("write package.json: %w", err)
	}

	logger.Debugf("updated package.json version from %s to %s", oldVersion, newVersion)
	return true, nil
}

// stripJSONComments removes // and /* */ style comments from JSON content
// to allow validation of JSONC/JSON5 files that may have comments.
// It's careful not to remove comment-like sequences inside strings.
func stripJSONComments(content string) string {
	var result strings.Builder
	inString := false
	inSingleLineComment := false
	inMultiLineComment := false
	escape := false
	
	chars := []rune(content)
	for i := 0; i < len(chars); i++ {
		ch := chars[i]
		
		// Handle escape sequences in strings
		if inString && escape {
			result.WriteRune(ch)
			escape = false
			continue
		}
		
		if inString && ch == '\\' {
			result.WriteRune(ch)
			escape = true
			continue
		}
		
		// Toggle string state
		if ch == '"' && !inSingleLineComment && !inMultiLineComment {
			inString = !inString
			result.WriteRune(ch)
			continue
		}
		
		// If we're in a string, just write the character
		if inString {
			result.WriteRune(ch)
			continue
		}
		
		// Handle end of single-line comment
		if inSingleLineComment {
			if ch == '\n' {
				inSingleLineComment = false
				result.WriteRune(ch) // Keep the newline
			}
			// Skip characters in comment
			continue
		}
		
		// Handle end of multi-line comment
		if inMultiLineComment {
			if ch == '*' && i+1 < len(chars) && chars[i+1] == '/' {
				inMultiLineComment = false
				i++ // Skip the '/'
			}
			// Skip characters in comment
			continue
		}
		
		// Check for start of comments
		if ch == '/' && i+1 < len(chars) {
			next := chars[i+1]
			if next == '/' {
				inSingleLineComment = true
				i++ // Skip the second '/'
				continue
			}
			if next == '*' {
				inMultiLineComment = true
				i++ // Skip the '*'
				continue
			}
		}
		
		// Regular character, not in comment or string
		result.WriteRune(ch)
	}
	
	return result.String()
}

// Update finds and updates the version in package.json if it exists.
// Returns whether the file was updated and any error.
func (u *Updater) Update(ctx context.Context, packagePath, newVersion string) (bool, error) {
	logger := log.FromContext(ctx)

	// Find package.json
	pkgPath, err := u.FindPackageJSON(ctx, packagePath)
	if err != nil {
		return false, err
	}

	// If not found, nothing to do
	if pkgPath == "" {
		logger.Debugf("no package.json found, skipping Node.js version update")
		return false, nil
	}

	// Update version
	changed, err := u.UpdateVersion(ctx, pkgPath, newVersion)
	if err != nil {
		return false, fmt.Errorf("update package.json version: %w", err)
	}

	// If no change was made (version already matches), return false
	if !changed {
		return false, nil
	}

	if u.dryRun {
		logger.Infof("dry-run: would update package.json version to %s", newVersion)
	} else {
		logger.Infof("updated package.json version to %s", newVersion)
	}

	return true, nil
}
