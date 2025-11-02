package output

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type contextKey string

const outputKey contextKey = "output"

// Format represents the output format type
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Manager handles output formatting
type Manager struct {
	format Format
}

// New creates a new output manager
func New(format Format) *Manager {
	return &Manager{
		format: format,
	}
}

// IsJSON returns true if the output format is JSON
func (m *Manager) IsJSON() bool {
	return m.format == FormatJSON
}

// Print outputs the result in the appropriate format
func (m *Manager) Print(result interface{}) error {
	if m.format == FormatJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	// For text mode, let the caller handle the output
	return nil
}

// WithManager adds the output manager to the context
func WithManager(ctx context.Context, manager *Manager) context.Context {
	return context.WithValue(ctx, outputKey, manager)
}

// FromContext retrieves the output manager from the context
func FromContext(ctx context.Context) *Manager {
	if manager, ok := ctx.Value(outputKey).(*Manager); ok {
		return manager
	}
	return New(FormatText)
}

// TagResult represents the result of a bump command (creates a git tag)
type TagResult struct {
	Tag     string `json:"tag"`
	Pushed  bool   `json:"pushed"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
}

// VersionResult represents the result of a version command
type VersionResult struct {
	Version string `json:"version"`
	Scheme  string `json:"scheme"`
	Commit  string `json:"commit"`
	Dirty   bool   `json:"dirty,omitempty"`
}

// VersionHistoryEntry represents a single version in the history
type VersionHistoryEntry struct {
	Version string `json:"version"`
	Tag     string `json:"tag"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Message string `json:"message,omitempty"`
}

// VersionHistoryResult represents the result of a version history command
type VersionHistoryResult struct {
	Versions []VersionHistoryEntry `json:"versions"`
	Count    int                   `json:"count"`
}

// VersionTagResult represents the result of querying a specific tag
type VersionTagResult struct {
	Tag     string `json:"tag"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Message string `json:"message,omitempty"`
	Exists  bool   `json:"exists"`
}

// BuildResult represents the result of a build command
type BuildResult struct {
	Version     string   `json:"version"`
	Commit      string   `json:"commit"`
	ShortCommit string   `json:"short_commit"`
	Date        string   `json:"date"`
	OutputDir   string   `json:"output_dir"`
	Targets     []string `json:"targets"`
	Binaries    []string `json:"binaries,omitempty"`
	Message     string   `json:"message,omitempty"`
}

// ImageResult represents the result of an image command
type ImageResult struct {
	Version     string   `json:"version"`
	Commit      string   `json:"commit"`
	ShortCommit string   `json:"short_commit"`
	Repository  string   `json:"repository"`
	Tags        []string `json:"tags"`
	Platforms   []string `json:"platforms"`
	Pushed      bool     `json:"pushed"`
	Message     string   `json:"message,omitempty"`
}

// InitResult represents the result of an init command
type InitResult struct {
	OutputPath string `json:"output_path"`
	Created    bool   `json:"created"`
	Message    string `json:"message,omitempty"`
}

// ErrorResult represents an error result
type ErrorResult struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// PrintError outputs an error in the appropriate format
func (m *Manager) PrintError(err error, message string) {
	if m.format == FormatJSON {
		result := ErrorResult{
			Error:   err.Error(),
			Message: message,
		}
		encoder := json.NewEncoder(os.Stderr)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(result)
	} else {
		if message != "" {
			fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
	}
}
