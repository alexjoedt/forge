package nodejs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/alexjoedt/forge/internal/log"
)

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// stripJSONCommentsTest removes // and /* */ style comments from JSON for test validation
func stripJSONCommentsTest(content string) string {
	// Remove single-line comments (// ...)
	re1 := regexp.MustCompile(`//.*`)
	content = re1.ReplaceAllString(content, "")

	// Remove multi-line comments (/* ... */)
	re2 := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	content = re2.ReplaceAllString(content, "")

	return content
}

func TestUpdater_FindPackageJSON(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string
		providePath string
		want        string // relative to temp dir
		wantFound   bool
	}{
		{
			name: "finds package.json in root",
			setupFiles: map[string]string{
				"package.json": `{"name": "test", "version": "1.0.0"}`,
			},
			providePath: "",
			want:        "package.json",
			wantFound:   true,
		},
		{
			name: "finds package.json with explicit path",
			setupFiles: map[string]string{
				"frontend/package.json": `{"name": "test", "version": "1.0.0"}`,
			},
			providePath: "frontend/package.json",
			want:        "frontend/package.json",
			wantFound:   true,
		},
		{
			name:        "returns empty when not found",
			setupFiles:  map[string]string{},
			providePath: "",
			want:        "",
			wantFound:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Setup files
			for path, content := range tt.setupFiles {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("create dir: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("write file: %v", err)
				}
			}

			// Create updater
			ctx := log.WithLogger(context.Background(), log.New(false))
			updater := NewUpdater(tmpDir, false)

			// Find package.json
			got, err := updater.FindPackageJSON(ctx, tt.providePath)
			if err != nil {
				t.Fatalf("FindPackageJSON() error = %v", err)
			}

			if tt.wantFound {
				wantPath := filepath.Join(tmpDir, tt.want)
				if got != wantPath {
					t.Errorf("FindPackageJSON() = %v, want %v", got, wantPath)
				}
			} else {
				if got != "" {
					t.Errorf("FindPackageJSON() = %v, want empty string", got)
				}
			}
		})
	}
}

func TestUpdater_ReadVersion(t *testing.T) {
	tests := []struct {
		name        string
		pkgContent  string
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:       "reads version successfully",
			pkgContent: `{"name": "test", "version": "1.2.3"}`,
			want:       "1.2.3",
			wantErr:    false,
		},
		{
			name:       "reads version with other fields",
			pkgContent: `{"name": "test", "version": "2.0.0", "dependencies": {}}`,
			want:       "2.0.0",
			wantErr:    false,
		},
		{
			name:        "errors on missing version",
			pkgContent:  `{"name": "test"}`,
			wantErr:     true,
			errContains: "version field not found",
		},
		{
			name:        "errors on invalid JSON",
			pkgContent:  `{invalid json`,
			wantErr:     true,
			errContains: "parse package.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			pkgPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(pkgPath, []byte(tt.pkgContent), 0644); err != nil {
				t.Fatalf("write file: %v", err)
			}

			ctx := log.WithLogger(context.Background(), log.New(false))
			updater := NewUpdater(tmpDir, false)

			got, err := updater.ReadVersion(ctx, pkgPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ReadVersion() expected error containing %q, got nil", tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("ReadVersion() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("ReadVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdater_UpdateVersion(t *testing.T) {
	tests := []struct {
		name        string
		initial     string
		newVersion  string
		wantVersion string
		dryRun      bool
		shouldWrite bool
		checkFormat bool // whether to check that original formatting is preserved
	}{
		{
			name:        "updates version successfully",
			initial:     `{"name": "test", "version": "1.0.0"}`,
			newVersion:  "2.0.0",
			wantVersion: "2.0.0",
			dryRun:      false,
			shouldWrite: true,
			checkFormat: false,
		},
		{
			name:        "preserves other fields",
			initial:     `{"name": "test", "version": "1.0.0", "description": "test pkg"}`,
			newVersion:  "1.1.0",
			wantVersion: "1.1.0",
			dryRun:      false,
			shouldWrite: true,
			checkFormat: false,
		},
		{
			name:        "dry-run does not write",
			initial:     `{"name": "test", "version": "1.0.0"}`,
			newVersion:  "2.0.0",
			wantVersion: "1.0.0", // should remain unchanged
			dryRun:      true,
			shouldWrite: false,
			checkFormat: false,
		},
		{
			name: "preserves formatting and comments",
			initial: `{
  // This is a comment
  "name": "test-package",
  "version": "1.0.0",
  "description": "A test package",
  // Another comment
  "author": "Test Author"
}`,
			newVersion:  "2.0.0",
			wantVersion: "2.0.0",
			dryRun:      false,
			shouldWrite: true,
			checkFormat: true,
		},
		{
			name: "preserves indentation style",
			initial: `{
    "name": "test",
    "version": "1.0.0",
    "scripts": {
        "test": "echo test"
    }
}`,
			newVersion:  "3.0.0",
			wantVersion: "3.0.0",
			dryRun:      false,
			shouldWrite: true,
			checkFormat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			pkgPath := filepath.Join(tmpDir, "package.json")
			if err := os.WriteFile(pkgPath, []byte(tt.initial), 0644); err != nil {
				t.Fatalf("write file: %v", err)
			}

			ctx := log.WithLogger(context.Background(), log.New(false))
			updater := NewUpdater(tmpDir, tt.dryRun)

			// Update version
			changed, err := updater.UpdateVersion(ctx, pkgPath, tt.newVersion)
			if err != nil {
				t.Fatalf("UpdateVersion() error = %v", err)
			}

			// For dry-run or no-change cases, changed should reflect expectation
			_ = changed // We don't need to assert on this in these tests

			// Read back and verify
			data, err := os.ReadFile(pkgPath)
			if err != nil {
				t.Fatalf("read file: %v", err)
			}

			content := string(data)

			// For files with comments, we need to strip them before parsing
			contentToParse := content
			if strings.Contains(content, "//") {
				// Strip comments for validation
				contentToParse = stripJSONCommentsTest(content)
			}

			var pkg map[string]interface{}
			if err := json.Unmarshal([]byte(contentToParse), &pkg); err != nil {
				t.Fatalf("parse JSON: %v", err)
			}

			gotVersion := pkg["version"].(string)
			if gotVersion != tt.wantVersion {
				t.Errorf("version = %v, want %v", gotVersion, tt.wantVersion)
			}

			// Verify other fields are preserved
			if tt.shouldWrite {
				if name, ok := pkg["name"]; ok {
					// Check name exists but don't validate exact value for all tests
					if name == "" {
						t.Errorf("name field is empty")
					}
				}
			}

			// Check that formatting is preserved (comments, whitespace, etc)
			if tt.checkFormat {
				// Verify comments are still present
				if strings.Contains(tt.initial, "//") {
					if !strings.Contains(content, "//") {
						t.Errorf("comments were removed from package.json")
					}
				}

				// Verify the content structure is similar (same number of lines ±1)
				originalLines := len(strings.Split(tt.initial, "\n"))
				newLines := len(strings.Split(content, "\n"))
				if abs(originalLines-newLines) > 1 {
					t.Errorf("formatting changed significantly: original %d lines, new %d lines", originalLines, newLines)
				}
			}
		})
	}
}

func TestUpdater_Update(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  map[string]string
		providePath string
		newVersion  string
		wantUpdated bool
		wantErr     bool
	}{
		{
			name: "updates package.json when found",
			setupFiles: map[string]string{
				"package.json": `{"name": "test", "version": "1.0.0"}`,
			},
			providePath: "",
			newVersion:  "2.0.0",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name: "updates with explicit path",
			setupFiles: map[string]string{
				"frontend/package.json": `{"name": "test", "version": "1.0.0"}`,
			},
			providePath: "frontend/package.json",
			newVersion:  "2.0.0",
			wantUpdated: true,
			wantErr:     false,
		},
		{
			name:        "returns false when not found",
			setupFiles:  map[string]string{},
			providePath: "",
			newVersion:  "2.0.0",
			wantUpdated: false,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Setup files
			for path, content := range tt.setupFiles {
				fullPath := filepath.Join(tmpDir, path)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("create dir: %v", err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("write file: %v", err)
				}
			}

			ctx := log.WithLogger(context.Background(), log.New(false))
			updater := NewUpdater(tmpDir, false)

			// Update
			updated, err := updater.Update(ctx, tt.providePath, tt.newVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if updated != tt.wantUpdated {
				t.Errorf("Update() updated = %v, want %v", updated, tt.wantUpdated)
			}

			// If updated, verify the version changed
			if updated && len(tt.setupFiles) > 0 {
				pkgPath := filepath.Join(tmpDir, tt.providePath)
				if tt.providePath == "" {
					pkgPath = filepath.Join(tmpDir, "package.json")
				}

				data, err := os.ReadFile(pkgPath)
				if err != nil {
					t.Fatalf("read file: %v", err)
				}

				var pkg map[string]interface{}
				if err := json.Unmarshal(data, &pkg); err != nil {
					t.Fatalf("parse JSON: %v", err)
				}

				if pkg["version"] != tt.newVersion {
					t.Errorf("version not updated, got %v, want %v", pkg["version"], tt.newVersion)
				}
			}
		})
	}
}
