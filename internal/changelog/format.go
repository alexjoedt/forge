package changelog

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Format represents the output format
type Format string

const (
	MarkdownFormat Format = "markdown"
	JSONFormat     Format = "json"
	PlainFormat    Format = "plain"
)

// FormatMarkdown formats the changelog as Markdown
func FormatMarkdown(cl *Changelog) string {
	var sb strings.Builder

	// Header
	if cl.ToTag != "" {
		sb.WriteString(fmt.Sprintf("# %s", cl.ToTag))
		if cl.FromTag != "" {
			sb.WriteString(fmt.Sprintf(" (%s...%s)", cl.FromTag, cl.ToTag))
		}
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("# Changelog\n\n")
	}

	// Date range
	if !cl.ToDate.IsZero() {
		sb.WriteString(fmt.Sprintf("*%s*\n\n", cl.ToDate.Format("2006-01-02")))
	}

	// Breaking changes first
	breakingChanges := []Commit{}
	for _, commit := range cl.Commits {
		if commit.Breaking {
			breakingChanges = append(breakingChanges, commit)
		}
	}

	if len(breakingChanges) > 0 {
		sb.WriteString("## ⚠ BREAKING CHANGES\n\n")
		for _, commit := range breakingChanges {
			sb.WriteString(formatMarkdownCommit(&commit))
		}
		sb.WriteString("\n")
	}

	// Sort types by priority
	types := make([]CommitType, 0, len(cl.ByType))
	for t := range cl.ByType {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return GetTypePriority(types[i]) < GetTypePriority(types[j])
	})

	// Group commits by type
	for _, t := range types {
		commits := cl.ByType[t]
		if len(commits) == 0 {
			continue
		}

		// Skip if type is "other" and there are no commits
		if t == TypeOther && len(commits) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s\n\n", GetTypeTitle(t)))

		for _, commit := range commits {
			// Skip breaking changes as they're already listed
			if commit.Breaking {
				continue
			}
			sb.WriteString(formatMarkdownCommit(&commit))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func formatMarkdownCommit(c *Commit) string {
	var sb strings.Builder

	sb.WriteString("* ")

	// Scope
	if c.Scope != "" {
		sb.WriteString(fmt.Sprintf("**%s:** ", c.Scope))
	}

	// Subject (remove conventional commit prefix if present)
	subject := c.Subject
	if c.Type != TypeOther {
		// Remove "type(scope): " or "type: " prefix
		parts := strings.SplitN(subject, ": ", 2)
		if len(parts) == 2 {
			subject = parts[1]
		}
	}
	sb.WriteString(subject)

	// Commit hash
	sb.WriteString(fmt.Sprintf(" ([%s](commit/%s))", c.ShortHash, c.Hash))

	// PR number
	if c.PRNumber != "" {
		sb.WriteString(fmt.Sprintf(" [#%s](pull/%s)", c.PRNumber, c.PRNumber))
	}

	sb.WriteString("\n")

	return sb.String()
}

// FormatPlain formats the changelog as plain text
func FormatPlain(cl *Changelog) string {
	var sb strings.Builder

	// Header
	if cl.ToTag != "" {
		sb.WriteString(fmt.Sprintf("%s", cl.ToTag))
		if cl.FromTag != "" {
			sb.WriteString(fmt.Sprintf(" (%s...%s)", cl.FromTag, cl.ToTag))
		}
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat("=", 50))
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("Changelog\n")
		sb.WriteString(strings.Repeat("=", 50))
		sb.WriteString("\n\n")
	}

	// Date
	if !cl.ToDate.IsZero() {
		sb.WriteString(fmt.Sprintf("Date: %s\n\n", cl.ToDate.Format("2006-01-02")))
	}

	// Breaking changes first
	breakingChanges := []Commit{}
	for _, commit := range cl.Commits {
		if commit.Breaking {
			breakingChanges = append(breakingChanges, commit)
		}
	}

	if len(breakingChanges) > 0 {
		sb.WriteString("⚠ BREAKING CHANGES\n")
		sb.WriteString(strings.Repeat("-", 50))
		sb.WriteString("\n\n")
		for _, commit := range breakingChanges {
			sb.WriteString(formatPlainCommit(&commit))
		}
		sb.WriteString("\n")
	}

	// Sort types by priority
	types := make([]CommitType, 0, len(cl.ByType))
	for t := range cl.ByType {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		return GetTypePriority(types[i]) < GetTypePriority(types[j])
	})

	// Group commits by type
	for _, t := range types {
		commits := cl.ByType[t]
		if len(commits) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("%s\n", GetTypeTitle(t)))
		sb.WriteString(strings.Repeat("-", 50))
		sb.WriteString("\n\n")

		for _, commit := range commits {
			// Skip breaking changes as they are already listed
			if commit.Breaking {
				continue
			}
			sb.WriteString(formatPlainCommit(&commit))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func formatPlainCommit(c *Commit) string {
	var sb strings.Builder

	sb.WriteString("  * ")

	// Scope
	if c.Scope != "" {
		sb.WriteString(fmt.Sprintf("[%s] ", c.Scope))
	}

	// Subject
	subject := c.Subject
	if c.Type != TypeOther {
		parts := strings.SplitN(subject, ": ", 2)
		if len(parts) == 2 {
			subject = parts[1]
		}
	}
	sb.WriteString(subject)

	// Commit hash
	sb.WriteString(fmt.Sprintf(" (%s)", c.ShortHash))

	// PR number
	if c.PRNumber != "" {
		sb.WriteString(fmt.Sprintf(" #%s", c.PRNumber))
	}

	sb.WriteString("\n")

	return sb.String()
}

// FormatJSON formats the changelog as JSON
func FormatJSON(cl *Changelog) (string, error) {
	// Create JSON-friendly structure
	type JSONCommit struct {
		Hash      string    `json:"hash"`
		ShortHash string    `json:"short_hash"`
		Subject   string    `json:"subject"`
		Author    string    `json:"author"`
		Date      time.Time `json:"date"`
		Type      string    `json:"type"`
		Scope     string    `json:"scope,omitempty"`
		Breaking  bool      `json:"breaking,omitempty"`
		PRNumber  string    `json:"pr_number,omitempty"`
	}

	type JSONChangelog struct {
		FromTag  string                  `json:"from_tag,omitempty"`
		ToTag    string                  `json:"to_tag,omitempty"`
		FromDate time.Time               `json:"from_date,omitempty"`
		ToDate   time.Time               `json:"to_date,omitempty"`
		Commits  []JSONCommit            `json:"commits"`
		ByType   map[string][]JSONCommit `json:"by_type"`
	}

	jsonCL := JSONChangelog{
		FromTag:  cl.FromTag,
		ToTag:    cl.ToTag,
		FromDate: cl.FromDate,
		ToDate:   cl.ToDate,
		Commits:  make([]JSONCommit, 0, len(cl.Commits)),
		ByType:   make(map[string][]JSONCommit),
	}

	// Convert commits
	for _, c := range cl.Commits {
		jc := JSONCommit{
			Hash:      c.Hash,
			ShortHash: c.ShortHash,
			Subject:   c.Subject,
			Author:    c.Author,
			Date:      c.Date,
			Type:      string(c.Type),
			Scope:     c.Scope,
			Breaking:  c.Breaking,
			PRNumber:  c.PRNumber,
		}
		jsonCL.Commits = append(jsonCL.Commits, jc)
	}

	// Convert by type
	for t, commits := range cl.ByType {
		typeStr := string(t)
		jsonCL.ByType[typeStr] = make([]JSONCommit, 0, len(commits))
		for _, c := range commits {
			jc := JSONCommit{
				Hash:      c.Hash,
				ShortHash: c.ShortHash,
				Subject:   c.Subject,
				Author:    c.Author,
				Date:      c.Date,
				Type:      string(c.Type),
				Scope:     c.Scope,
				Breaking:  c.Breaking,
				PRNumber:  c.PRNumber,
			}
			jsonCL.ByType[typeStr] = append(jsonCL.ByType[typeStr], jc)
		}
	}

	data, err := json.MarshalIndent(jsonCL, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal JSON: %w", err)
	}

	return string(data), nil
}
