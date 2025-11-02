package interactive

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// IsInteractive checks if stdin is a TTY (terminal)
func IsInteractive() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// BumpType represents the type of version bump
type BumpType string

const (
	BumpPatch BumpType = "patch"
	BumpMinor BumpType = "minor"
	BumpMajor BumpType = "major"
)

// BumpChoice represents a selection choice for version bumping
type BumpChoice struct {
	Type        BumpType
	Description string
	Preview     string
}

// SelectionResult holds the result of a selection prompt
type SelectionResult struct {
	Choice   BumpChoice
	Canceled bool
}

// selectionModel is the Bubble Tea model for selection prompts
type selectionModel struct {
	choices  []BumpChoice
	cursor   int
	selected *BumpChoice
	canceled bool
	title    string
}

func (m selectionModel) Init() tea.Cmd {
	return nil
}

func (m selectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.canceled = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", " ":
			m.selected = &m.choices[m.cursor]
			return m, tea.Quit
		}
	}

	return m, nil
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			MarginBottom(1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	previewStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1)
)

func (m selectionModel) View() string {
	s := titleStyle.Render(m.title) + "\n\n"

	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("❯ ")
		}

		line := fmt.Sprintf("%s%s", cursor, choice.Type)
		if choice.Preview != "" {
			line += previewStyle.Render(fmt.Sprintf(" (%s)", choice.Preview))
		}

		if choice.Description != "" {
			line += " " + descStyle.Render("- "+choice.Description)
		}

		s += line + "\n"
	}

	s += "\n" + helpStyle.Render("↑/↓ or k/j: navigate • enter/space: select • q/esc: cancel")

	return s
}

// PromptBumpType shows an interactive selection for version bump type
func PromptBumpType(currentVersion string, choices []BumpChoice) (*BumpChoice, error) {
	if !IsInteractive() {
		return nil, fmt.Errorf("not running in an interactive terminal")
	}

	title := fmt.Sprintf("Select version bump type (current: %s)", currentVersion)
	
	m := selectionModel{
		choices: choices,
		cursor:  0,
		title:   title,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running prompt: %w", err)
	}

	result := finalModel.(selectionModel)
	if result.canceled {
		return nil, fmt.Errorf("selection canceled")
	}

	return result.selected, nil
}

// confirmationModel is the Bubble Tea model for yes/no confirmation
type confirmationModel struct {
	question string
	preview  string
	yes      bool
	answered bool
	canceled bool
}

func (m confirmationModel) Init() tea.Cmd {
	return nil
}

func (m confirmationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.canceled = true
			return m, tea.Quit

		case "y", "Y":
			m.yes = true
			m.answered = true
			return m, tea.Quit

		case "n", "N":
			m.yes = false
			m.answered = true
			return m, tea.Quit

		case "enter":
			// Default to yes on enter
			m.yes = true
			m.answered = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m confirmationModel) View() string {
	s := titleStyle.Render(m.question) + "\n\n"

	if m.preview != "" {
		s += previewStyle.Render(m.preview) + "\n\n"
	}

	s += "  " + selectedStyle.Render("[Y]es") + " / " + descStyle.Render("[N]o")
	s += "\n\n" + helpStyle.Render("y: yes • n: no • enter: confirm (yes) • q/esc: cancel")

	return s
}

// PromptConfirmation shows a yes/no confirmation prompt
func PromptConfirmation(question string, preview string) (bool, error) {
	if !IsInteractive() {
		return false, fmt.Errorf("not running in an interactive terminal")
	}

	m := confirmationModel{
		question: question,
		preview:  preview,
		yes:      true, // Default to yes
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false, fmt.Errorf("error running prompt: %w", err)
	}

	result := finalModel.(confirmationModel)
	if result.canceled {
		return false, fmt.Errorf("confirmation canceled")
	}

	return result.yes, nil
}
