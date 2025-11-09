package version

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Scheme represents the versioning scheme (semver or calver).
type Scheme string

const (
	SchemeSemVer Scheme = "semver"
	SchemeCalVer Scheme = "calver"
)

// BumpType indicates which part of a semver to increment.
type BumpType string

const (
	BumpMajor BumpType = "major"
	BumpMinor BumpType = "minor"
	BumpPatch BumpType = "patch"
)

// Version represents a parsed version tag.
type Version struct {
	Scheme Scheme
	Raw    string

	// SemVer fields
	Major int
	Minor int
	Patch int
	Pre   string
	Meta  string

	// CalVer fields
	CalVerDate     string // e.g., "2025.10.02"
	CalVerSequence int    // optional sequence number for same-day releases
}

// ParseSemVer parses a semantic version string (without prefix).
// Format: MAJOR.MINOR.PATCH[-PRERELEASE][+METADATA]
func ParseSemVer(s string) (*Version, error) {
	v := &Version{
		Scheme: SchemeSemVer,
		Raw:    s,
	}

	// Split off metadata
	parts := strings.SplitN(s, "+", 2)
	if len(parts) == 2 {
		v.Meta = parts[1]
		s = parts[0]
	}

	// Split off prerelease
	parts = strings.SplitN(s, "-", 2)
	if len(parts) == 2 {
		v.Pre = parts[1]
		s = parts[0]
	}

	// Parse major.minor.patch
	versionParts := strings.Split(s, ".")
	if len(versionParts) != 3 {
		return nil, fmt.Errorf("invalid semver format: %s", s)
	}

	var err error
	v.Major, err = strconv.Atoi(versionParts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %w", err)
	}

	v.Minor, err = strconv.Atoi(versionParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %w", err)
	}

	v.Patch, err = strconv.Atoi(versionParts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %w", err)
	}

	return v, nil
}

// ParseCalVer parses a calendar version string (without prefix).
// Format: YYYY.MM.DD[.SEQUENCE][-PRERELEASE][+METADATA]
//
//	or: YYYY.WW[.SEQUENCE][-PRERELEASE][+METADATA]
//
// Supports both date-based (3 parts: year.month.day) and week-based (2 parts: year.week) formats.
func ParseCalVer(s string) (*Version, error) {
	v := &Version{
		Scheme: SchemeCalVer,
		Raw:    s,
	}

	// Split off metadata
	parts := strings.SplitN(s, "+", 2)
	if len(parts) == 2 {
		v.Meta = parts[1]
		s = parts[0]
	}

	// Split off prerelease
	parts = strings.SplitN(s, "-", 2)
	if len(parts) == 2 {
		v.Pre = parts[1]
		s = parts[0]
	}

	// Parse date and optional sequence
	dateParts := strings.Split(s, ".")
	if len(dateParts) < 2 {
		return nil, fmt.Errorf("invalid calver format: %s", s)
	}

	// Determine if this is a week-based (2 parts) or date-based (3 parts) format
	// Week-based: YYYY.WW[.SEQUENCE] - 2 or 3 parts
	// Date-based: YYYY.MM.DD[.SEQUENCE] - 3 or 4 parts
	// We detect by checking if we have exactly 2 or 3 parts (week) vs 3 or 4 parts (date)
	// The heuristic: if we have 2 parts, it's week-based without sequence
	//                if we have 3 parts, check if the 3rd part looks like a sequence (small number)
	//                if we have 4 parts, it's date-based with sequence

	if len(dateParts) == 2 {
		// YYYY.WW (week format without sequence)
		// Only accept if second part is > 12 (definitely a week, not a month)
		weekNum, err := strconv.Atoi(dateParts[1])
		if err != nil || weekNum <= 12 {
			return nil, fmt.Errorf("invalid calver format: %s (ambiguous 2-part format - use 3 parts for dates or week > 12)", s)
		}
		v.CalVerDate = strings.Join(dateParts, ".")
	} else if len(dateParts) == 3 {
		// Could be either:
		// 1. YYYY.MM.DD (date format without sequence)
		// 2. YYYY.WW.SEQUENCE (week format with sequence)
		// Heuristic: if the second part is <= 53 (max ISO week), and third part is a small number,
		// it's likely a week format with sequence. Otherwise, it's a date format.
		secondPart, err1 := strconv.Atoi(dateParts[1])
		thirdPart, err2 := strconv.Atoi(dateParts[2])

		if err1 == nil && err2 == nil && secondPart <= 53 && thirdPart <= 31 {
			// Ambiguous case: could be either YYYY.MM.DD or YYYY.WW.SEQUENCE
			// Use heuristic: if third part is <= 12, more likely to be a sequence number
			// This isn't perfect but handles most cases
			// Better approach: weeks are 01-53, months are 01-12, days are 01-31
			// If second part is > 12, it's definitely a week number
			if secondPart > 12 {
				// Definitely week format (weeks 13-53)
				v.CalVerDate = strings.Join(dateParts[:2], ".")
				v.CalVerSequence = thirdPart
			} else {
				// Could be either, assume date format (YYYY.MM.DD)
				// This is the safest default for backward compatibility
				v.CalVerDate = strings.Join(dateParts[:3], ".")
			}
		} else {
			// Invalid numbers or format, assume date format
			v.CalVerDate = strings.Join(dateParts[:3], ".")
		}
	} else if len(dateParts) == 4 {
		// YYYY.MM.DD.SEQUENCE (date format with sequence)
		v.CalVerDate = strings.Join(dateParts[:3], ".")
		seq, err := strconv.Atoi(dateParts[3])
		if err != nil {
			return nil, fmt.Errorf("invalid calver sequence: %w", err)
		}
		v.CalVerSequence = seq
	} else {
		return nil, fmt.Errorf("invalid calver format: %s", s)
	}

	return v, nil
}

// BumpSemVer increments the version according to the bump type.
func (v *Version) BumpSemVer(bump BumpType) *Version {
	next := &Version{
		Scheme: SchemeSemVer,
		Major:  v.Major,
		Minor:  v.Minor,
		Patch:  v.Patch,
	}

	switch bump {
	case BumpMajor:
		next.Major++
		next.Minor = 0
		next.Patch = 0
	case BumpMinor:
		next.Minor++
		next.Patch = 0
	case BumpPatch:
		next.Patch++
	}

	return next
}

// FormatCalVer formats a time using the given format string, with support for ISO week numbers.
// Supports special format codes:
// - "YYYY" or "2006" for 4-digit year
// - "WW" for ISO week number (01-53)
// All other format codes are passed to time.Format().
// Examples:
// - "2006.01.02" -> "2025.10.03" (year.month.day)
// - "2006.WW" -> "2025.40" (year.week)
func FormatCalVer(t time.Time, format string) string {
	// Check if format contains week number placeholder
	if strings.Contains(format, "WW") {
		year, week := t.ISOWeek()
		// Replace WW with the week number (zero-padded to 2 digits)
		weekStr := fmt.Sprintf("%02d", week)
		result := strings.ReplaceAll(format, "WW", weekStr)
		// Also handle year replacement if YYYY or 2006 is present
		result = strings.ReplaceAll(result, "2006", fmt.Sprintf("%d", year))
		result = strings.ReplaceAll(result, "YYYY", fmt.Sprintf("%d", year))
		return result
	}
	// No special week formatting, use standard Go time formatting
	return t.UTC().Format(format)
}

// NextCalVer generates the next calendar version using the given format and current time.
// If the current version is for the same period (e.g., same day/week), it increments the sequence number.
// Supports special format codes via FormatCalVer (e.g., "WW" for ISO week number).
//
// For week-based formats (containing "WW"), build numbers start at 1 for clarity.
// For date-based formats, build numbers start at 0 (omitted) for the first release.
func NextCalVer(current *Version, format string, now time.Time) *Version {
	nowStr := FormatCalVer(now.UTC(), format)
	isWeekFormat := strings.Contains(format, "WW")

	next := &Version{
		Scheme:         SchemeCalVer,
		CalVerDate:     nowStr,
		CalVerSequence: 0,
	}

	// If current version exists and is for the same period, increment sequence
	if current != nil && current.CalVerDate == nowStr {
		next.CalVerSequence = current.CalVerSequence + 1
	} else if isWeekFormat {
		// For week-based formats, always start with build number 1 (for new weeks or no previous version)
		next.CalVerSequence = 1
	}

	return next
}

// String returns the string representation of the version (without prefix).
func (v *Version) String() string {
	var s string

	switch v.Scheme {
	case SchemeSemVer:
		s = fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	case SchemeCalVer:
		s = v.CalVerDate
		if v.CalVerSequence > 0 {
			s = fmt.Sprintf("%s.%d", s, v.CalVerSequence)
		}
	}

	if v.Pre != "" {
		s = fmt.Sprintf("%s-%s", s, v.Pre)
	}

	if v.Meta != "" {
		s = fmt.Sprintf("%s+%s", s, v.Meta)
	}

	return s
}

// WithPrerelease returns a new version with the prerelease identifier set.
func (v *Version) WithPrerelease(pre string) *Version {
	next := *v
	next.Pre = pre
	return &next
}

// WithMetadata returns a new version with the build metadata set.
func (v *Version) WithMetadata(meta string) *Version {
	next := *v
	next.Meta = meta
	return &next
}

// StripPrefix removes a prefix (e.g., "v") from a version string.
func StripPrefix(tag, prefix string) string {
	return strings.TrimPrefix(tag, prefix)
}

// WithPrefix adds a prefix (e.g., "v") to a version string.
func WithPrefix(version, prefix string) string {
	return prefix + version
}

// ============================================================================
// Hotfix Version Functions
// ============================================================================

// IsHotfixVersion checks if version string has hotfix suffix.
// Examples: "v1.0.0-hotfix.1", "2025.11.09-hotfix.2", "api/v1.0.0-patch.1"
func IsHotfixVersion(tag string) bool {
	// Match pattern: anything-suffix.number
	// Common suffixes: hotfix, patch, fix
	parts := strings.Split(tag, "-")
	if len(parts) < 2 {
		return false
	}

	// Check if last part matches suffix.number pattern
	lastPart := parts[len(parts)-1]
	dotParts := strings.Split(lastPart, ".")
	if len(dotParts) != 2 {
		return false
	}

	// Check if second part is a number
	_, err := strconv.Atoi(dotParts[1])
	return err == nil
}

// ParseHotfixVersion parses versions like "v1.0.0-hotfix.3".
// Returns base version, suffix, and sequence number.
func ParseHotfixVersion(tag string) (*Version, string, int, error) {
	if !IsHotfixVersion(tag) {
		return nil, "", 0, fmt.Errorf("not a hotfix version: %s", tag)
	}

	// Split by last hyphen to separate base from suffix
	lastHyphen := strings.LastIndex(tag, "-")
	if lastHyphen == -1 {
		return nil, "", 0, fmt.Errorf("invalid hotfix format: %s", tag)
	}

	baseStr := tag[:lastHyphen]
	suffixPart := tag[lastHyphen+1:]

	// Parse suffix and sequence: "hotfix.3"
	dotIndex := strings.Index(suffixPart, ".")
	if dotIndex == -1 {
		return nil, "", 0, fmt.Errorf("invalid hotfix format: %s", tag)
	}

	suffix := suffixPart[:dotIndex]
	seqStr := suffixPart[dotIndex+1:]

	seq, err := strconv.Atoi(seqStr)
	if err != nil {
		return nil, "", 0, fmt.Errorf("invalid sequence number: %s", seqStr)
	}

	// Try to parse base as SemVer or CalVer
	// We need to determine which scheme to use
	// Try SemVer first (most common)
	var baseVersion *Version

	// Remove any prefix to parse the version
	// For now, we'll try both schemes
	if strings.Count(baseStr, ".") == 2 {
		// Likely SemVer (3 parts) or CalVer date format
		baseVersion, err = ParseSemVer(baseStr)
		if err != nil {
			// Try CalVer
			baseVersion, err = ParseCalVer(baseStr)
			if err != nil {
				return nil, "", 0, fmt.Errorf("failed to parse base version %q: %w", baseStr, err)
			}
		}
	} else {
		// Likely CalVer with week format or other
		baseVersion, err = ParseCalVer(baseStr)
		if err != nil {
			// Try SemVer as fallback
			baseVersion, err = ParseSemVer(baseStr)
			if err != nil {
				return nil, "", 0, fmt.Errorf("failed to parse base version %q: %w", baseStr, err)
			}
		}
	}

	return baseVersion, suffix, seq, nil
}

// IncrementHotfixSequence bumps the hotfix sequence number.
// "v1.0.0-hotfix.2" → "v1.0.0-hotfix.3"
func IncrementHotfixSequence(tag string) (string, error) {
	_, suffix, seq, err := ParseHotfixVersion(tag)
	if err != nil {
		return "", err
	}

	// Find the base tag
	lastHyphen := strings.LastIndex(tag, "-")
	baseTag := tag[:lastHyphen]

	// Return incremented version
	return fmt.Sprintf("%s-%s.%d", baseTag, suffix, seq+1), nil
}
