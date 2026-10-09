package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type contextKey string

const outputKey contextKey = "output"

// Format represents the output format type.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Manager handles output formatting.
type Manager struct {
	format Format
	out    io.Writer
	errOut io.Writer
}

// New creates a new output manager writing results to out and errors to errOut.
func New(format Format, out, errOut io.Writer) *Manager {
	return &Manager{
		format: format,
		out:    out,
		errOut: errOut,
	}
}

// Writer returns the writer for regular command output.
func (m *Manager) Writer() io.Writer {
	return m.out
}

// IsJSON returns true if the output format is JSON.
func (m *Manager) IsJSON() bool {
	return m.format == FormatJSON
}

// Print outputs the result in the appropriate format.
func (m *Manager) Print(result any) error {
	if m.format == FormatJSON {
		encoder := json.NewEncoder(m.out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("encode JSON output: %w", err)
		}
		return nil
	}
	// For text mode, let the caller handle the output.
	return nil
}

// WithManager adds the output manager to the context.
func WithManager(ctx context.Context, manager *Manager) context.Context {
	return context.WithValue(ctx, outputKey, manager)
}

// FromContext retrieves the output manager from the context.
func FromContext(ctx context.Context) *Manager {
	if manager, ok := ctx.Value(outputKey).(*Manager); ok {
		return manager
	}
	return New(FormatText, os.Stdout, os.Stderr)
}

// TagResult represents the result of a bump command (creates a git tag).
type TagResult struct {
	Tag     string `json:"tag"`
	Pushed  bool   `json:"pushed"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
}

// VersionResult represents the result of a version command.
type VersionResult struct {
	Version string `json:"version"`
	Scheme  string `json:"scheme"`
	Commit  string `json:"commit"`
	Dirty   bool   `json:"dirty,omitempty"`
}

// VersionHistoryEntry represents a single version in the history.
type VersionHistoryEntry struct {
	Version string `json:"version"`
	Tag     string `json:"tag"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Message string `json:"message,omitempty"`
}

// VersionHistoryResult represents the result of a version history command.
type VersionHistoryResult struct {
	Versions []VersionHistoryEntry `json:"versions"`
	Count    int                   `json:"count"`
}

// VersionTagResult represents the result of querying a specific tag.
type VersionTagResult struct {
	Tag     string `json:"tag"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Message string `json:"message,omitempty"`
	Exists  bool   `json:"exists"`
}

// InitResult represents the result of an init command.
type InitResult struct {
	OutputPath string `json:"output_path"`
	Created    bool   `json:"created"`
	Message    string `json:"message,omitempty"`
}

// RetagResult represents the result of a retag command.
type RetagResult struct {
	Tag        string `json:"tag"`
	FromCommit string `json:"from_commit"`
	ToCommit   string `json:"to_commit"`
	Pushed     bool   `json:"pushed"`
	Message    string `json:"message,omitempty"`
}

// ErrorResult represents an error result.
type ErrorResult struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// PrintError outputs an error in the appropriate format.
func (m *Manager) PrintError(err error, message string) {
	if m.format == FormatJSON {
		result := ErrorResult{
			Error:   err.Error(),
			Message: message,
		}
		encoder := json.NewEncoder(m.errOut)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(result)
	} else {
		if message != "" {
			fmt.Fprintf(m.errOut, "%s: %v\n", message, err)
		} else {
			fmt.Fprintf(m.errOut, "error: %v\n", err)
		}
	}
}
