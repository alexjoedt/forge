package table

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Table styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	currentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	schemeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))

	dateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	commitStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	borderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
)

// Column represents a table column
type Column struct {
	Header string
	Width  int
	Align  Align
}

// Align represents text alignment
type Align int

const (
	AlignLeft Align = iota
	AlignRight
	AlignCenter
)

// Row represents a table row with styled cells
type Row struct {
	Cells []string
}

// Table represents a formatted table
type Table struct {
	Columns []Column
	Rows    []Row
	Border  bool
}

// New creates a new table
func New(columns []Column) *Table {
	return &Table{
		Columns: columns,
		Rows:    []Row{},
		Border:  true,
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(cells ...string) {
	t.Rows = append(t.Rows, Row{Cells: cells})
}

// Render returns the formatted table as a string
func (t *Table) Render() string {
	if len(t.Columns) == 0 {
		return ""
	}

	// Calculate actual column widths based on content
	widths := make([]int, len(t.Columns))
	for i, col := range t.Columns {
		widths[i] = len(col.Header)
		for _, row := range t.Rows {
			if i < len(row.Cells) {
				// Strip ANSI codes for width calculation
				cleanCell := stripAnsi(row.Cells[i])
				if len(cleanCell) > widths[i] {
					widths[i] = len(cleanCell)
				}
			}
		}
		// Use configured width if larger
		if col.Width > widths[i] {
			widths[i] = col.Width
		}
	}

	var sb strings.Builder

	// Header
	if t.Border {
		sb.WriteString(borderStyle.Render("┌"))
		for i, width := range widths {
			sb.WriteString(borderStyle.Render(strings.Repeat("─", width+2)))
			if i < len(widths)-1 {
				sb.WriteString(borderStyle.Render("┬"))
			}
		}
		sb.WriteString(borderStyle.Render("┐"))
		sb.WriteString("\n")
	}

	// Header row
	if t.Border {
		sb.WriteString(borderStyle.Render("│ "))
	}
	for i, col := range t.Columns {
		cell := headerStyle.Render(pad(col.Header, widths[i], col.Align))
		sb.WriteString(cell)
		if i < len(t.Columns)-1 {
			if t.Border {
				sb.WriteString(borderStyle.Render(" │ "))
			} else {
				sb.WriteString("  ")
			}
		}
	}
	if t.Border {
		sb.WriteString(borderStyle.Render(" │"))
	}
	sb.WriteString("\n")

	// Header separator
	if t.Border {
		sb.WriteString(borderStyle.Render("├"))
		for i, width := range widths {
			sb.WriteString(borderStyle.Render(strings.Repeat("─", width+2)))
			if i < len(widths)-1 {
				sb.WriteString(borderStyle.Render("┼"))
			}
		}
		sb.WriteString(borderStyle.Render("┤"))
		sb.WriteString("\n")
	} else {
		for i, width := range widths {
			sb.WriteString(strings.Repeat("─", width))
			if i < len(widths)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}

	// Data rows
	for _, row := range t.Rows {
		if t.Border {
			sb.WriteString(borderStyle.Render("│ "))
		}
		for i := range t.Columns {
			var cell string
			if i < len(row.Cells) {
				cell = padStyled(row.Cells[i], widths[i], t.Columns[i].Align)
			} else {
				cell = pad("", widths[i], t.Columns[i].Align)
			}
			sb.WriteString(cell)
			if i < len(t.Columns)-1 {
				if t.Border {
					sb.WriteString(borderStyle.Render(" │ "))
				} else {
					sb.WriteString("  ")
				}
			}
		}
		if t.Border {
			sb.WriteString(borderStyle.Render(" │"))
		}
		sb.WriteString("\n")
	}

	// Bottom border
	if t.Border {
		sb.WriteString(borderStyle.Render("└"))
		for i, width := range widths {
			sb.WriteString(borderStyle.Render(strings.Repeat("─", width+2)))
			if i < len(widths)-1 {
				sb.WriteString(borderStyle.Render("┴"))
			}
		}
		sb.WriteString(borderStyle.Render("┘"))
		sb.WriteString("\n")
	}

	return sb.String()
}

// pad pads a string to the specified width with the given alignment
func pad(s string, width int, align Align) string {
	if len(s) >= width {
		return s
	}

	padding := width - len(s)
	switch align {
	case AlignRight:
		return strings.Repeat(" ", padding) + s
	case AlignCenter:
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default: // AlignLeft
		return s + strings.Repeat(" ", padding)
	}
}

// padStyled pads a styled string (with ANSI codes) to the specified width
func padStyled(s string, width int, align Align) string {
	cleanLen := len(stripAnsi(s))
	if cleanLen >= width {
		return s
	}

	padding := width - cleanLen
	switch align {
	case AlignRight:
		return strings.Repeat(" ", padding) + s
	case AlignCenter:
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	default: // AlignLeft
		return s + strings.Repeat(" ", padding)
	}
}

// stripAnsi removes ANSI escape codes from a string for length calculation
func stripAnsi(s string) string {
	// Simple ANSI stripping - remove \x1b[...m sequences
	result := []rune{}
	inEscape := false
	runes := []rune(s)
	
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			inEscape = true
			i++ // skip the '['
			continue
		}
		if inEscape {
			if runes[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result = append(result, runes[i])
	}
	
	return string(result)
}

// Style helpers for table cells
func CurrentVersion(s string) string {
	return currentStyle.Render(s)
}

func Scheme(s string) string {
	return schemeStyle.Render(s)
}

func Date(s string) string {
	return dateStyle.Render(s)
}

func Commit(s string) string {
	return commitStyle.Render(s)
}
