package changelog

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alexjoedt/forge/internal/run"
)

// CommitType represents the type of commit (feat, fix, etc.)
type CommitType string

const (
	TypeFeat     CommitType = "feat"
	TypeFix      CommitType = "fix"
	TypeDocs     CommitType = "docs"
	TypeStyle    CommitType = "style"
	TypeRefactor CommitType = "refactor"
	TypePerf     CommitType = "perf"
	TypeTest     CommitType = "test"
	TypeBuild    CommitType = "build"
	TypeCI       CommitType = "ci"
	TypeChore    CommitType = "chore"
	TypeOther    CommitType = "other"
)

// Commit represents a parsed git commit
type Commit struct {
	Hash      string
	ShortHash string
	Subject   string
	Body      string
	Author    string
	Date      time.Time
	Type      CommitType
	Scope     string
	Breaking  bool
	PRNumber  string
}

// Changelog represents a collection of commits grouped by type
type Changelog struct {
	FromTag   string
	ToTag     string
	FromDate  time.Time
	ToDate    time.Time
	Commits   []Commit
	ByType    map[CommitType][]Commit
}

var (
	// Conventional Commits regex: type(scope): subject
	conventionalRegex = regexp.MustCompile(`^(?P<type>\w+)(?:\((?P<scope>[^)]+)\))?(?P<breaking>!)?: (?P<subject>.+)$`)
	// PR number regex: (#123)
	prRegex = regexp.MustCompile(`\(#(\d+)\)`)
	// Breaking change markers
	breakingMarkers = []string{"BREAKING CHANGE:", "BREAKING-CHANGE:", "BREAKING:"}
)

// Parser parses git commits
type Parser struct {
	repoDir   string
	tagPrefix string
}

// NewParser creates a new parser
func NewParser(repoDir, tagPrefix string) *Parser {
	return &Parser{
		repoDir:   repoDir,
		tagPrefix: tagPrefix,
	}
}

// Parse parses git log between two commits/tags
func (p *Parser) Parse(ctx context.Context, from, to string) (*Changelog, error) {
	return Parse(ctx, p.repoDir, from, to)
}

// Parse parses git log between two commits/tags
func Parse(ctx context.Context, repoDir, from, to string) (*Changelog, error) {
	// Build git log command
	var logRange string
	if from != "" && to != "" {
		logRange = fmt.Sprintf("%s..%s", from, to)
	} else if from != "" {
		logRange = fmt.Sprintf("%s..HEAD", from)
	} else if to != "" {
		logRange = to
	} else {
		logRange = "HEAD"
	}

	// Format: hash|short|author|date|subject|body
	format := "%H|%h|%an|%aI|%s|%b"
	result := run.CmdInDir(ctx, repoDir, "git", "log", logRange, "--no-merges", fmt.Sprintf("--pretty=format:%s", format), "--date=iso")
	
	if !result.Success() {
		return nil, fmt.Errorf("git log failed: %s", result.Stderr)
	}

	output := result.Stdout
	if output == "" {
		return &Changelog{
			FromTag: from,
			ToTag:   to,
			Commits: []Commit{},
			ByType:  make(map[CommitType][]Commit),
		}, nil
	}

	// Parse commits
	commits := []Commit{}
	lines := strings.Split(output, "\n")
	
	var currentCommit *Commit
	var bodyLines []string
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		// Check if this is a new commit line (starts with hash)
		parts := strings.SplitN(line, "|", 6)
		if len(parts) == 6 {
			// Save previous commit if exists
			if currentCommit != nil {
				currentCommit.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
				commits = append(commits, *currentCommit)
				bodyLines = []string{}
			}
			
			// Parse new commit
			hash := parts[0]
			shortHash := parts[1]
			author := parts[2]
			dateStr := parts[3]
			subject := parts[4]
			body := parts[5]
			
			date, _ := time.Parse(time.RFC3339, dateStr)
			
			commit := &Commit{
				Hash:      hash,
				ShortHash: shortHash,
				Author:    author,
				Date:      date,
				Subject:   subject,
				Body:      body,
			}
			
			// Parse conventional commit format
			parseConventionalCommit(commit)
			
			// Check for breaking changes
			checkBreakingChange(commit)
			
			// Extract PR number
			extractPRNumber(commit)
			
			currentCommit = commit
			
			if body != "" {
				bodyLines = append(bodyLines, body)
			}
		} else {
			// This is a body continuation line
			if currentCommit != nil {
				bodyLines = append(bodyLines, line)
			}
		}
	}
	
	// Save last commit
	if currentCommit != nil {
		currentCommit.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
		commits = append(commits, *currentCommit)
	}

	// Group by type
	byType := make(map[CommitType][]Commit)
	for _, commit := range commits {
		byType[commit.Type] = append(byType[commit.Type], commit)
	}

	return &Changelog{
		FromTag: from,
		ToTag:   to,
		Commits: commits,
		ByType:  byType,
	}, nil
}

// parseConventionalCommit parses the subject line for Conventional Commits format
func parseConventionalCommit(commit *Commit) {
	matches := conventionalRegex.FindStringSubmatch(commit.Subject)
	if matches == nil {
		commit.Type = TypeOther
		return
	}

	// Extract named groups
	result := make(map[string]string)
	for i, name := range conventionalRegex.SubexpNames() {
		if i != 0 && name != "" && i < len(matches) {
			result[name] = matches[i]
		}
	}

	// Set type
	typeStr := strings.ToLower(result["type"])
	commit.Type = CommitType(typeStr)
	
	// Validate type
	validTypes := []CommitType{
		TypeFeat, TypeFix, TypeDocs, TypeStyle, TypeRefactor,
		TypePerf, TypeTest, TypeBuild, TypeCI, TypeChore,
	}
	
	isValid := false
	for _, t := range validTypes {
		if commit.Type == t {
			isValid = true
			break
		}
	}
	
	if !isValid {
		commit.Type = TypeOther
	}

	// Set scope
	commit.Scope = result["scope"]

	// Set breaking change from !
	if result["breaking"] == "!" {
		commit.Breaking = true
	}
}

// checkBreakingChange checks the commit body for breaking change markers
func checkBreakingChange(commit *Commit) {
	bodyLower := strings.ToLower(commit.Body)
	for _, marker := range breakingMarkers {
		if strings.Contains(bodyLower, strings.ToLower(marker)) {
			commit.Breaking = true
			return
		}
	}
}

// extractPRNumber extracts PR number from subject or body
func extractPRNumber(commit *Commit) {
	// Check subject first
	matches := prRegex.FindStringSubmatch(commit.Subject)
	if len(matches) > 1 {
		commit.PRNumber = matches[1]
		return
	}

	// Check body
	matches = prRegex.FindStringSubmatch(commit.Body)
	if len(matches) > 1 {
		commit.PRNumber = matches[1]
	}
}

// GetTypeTitle returns a human-readable title for a commit type
func GetTypeTitle(t CommitType) string {
	switch t {
	case TypeFeat:
		return "Features"
	case TypeFix:
		return "Bug Fixes"
	case TypeDocs:
		return "Documentation"
	case TypeStyle:
		return "Code Style"
	case TypeRefactor:
		return "Code Refactoring"
	case TypePerf:
		return "Performance Improvements"
	case TypeTest:
		return "Tests"
	case TypeBuild:
		return "Build System"
	case TypeCI:
		return "Continuous Integration"
	case TypeChore:
		return "Chores"
	default:
		return "Other Changes"
	}
}

// GetTypePriority returns the display priority for a commit type (lower = higher priority)
func GetTypePriority(t CommitType) int {
	switch t {
	case TypeFeat:
		return 1
	case TypeFix:
		return 2
	case TypePerf:
		return 3
	case TypeRefactor:
		return 4
	case TypeDocs:
		return 5
	case TypeTest:
		return 6
	case TypeBuild:
		return 7
	case TypeCI:
		return 8
	case TypeStyle:
		return 9
	case TypeChore:
		return 10
	default:
		return 99
	}
}
