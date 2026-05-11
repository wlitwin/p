package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	normalStyle   = lipgloss.NewStyle()
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// PickerItem represents a selectable option in the interactive picker TUI.
type PickerItem struct {
	Label string
	Desc  string
}

type pickerModel struct {
	items    []PickerItem
	cursor   int
	chosen   int
	quitting bool
	title    string
}

func (m pickerModel) Init() tea.Cmd {
	return nil
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.chosen = -1
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.cursor
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	s := dimStyle.Render(m.title) + "\n\n"

	for i, item := range m.items {
		cursor := "  "
		style := normalStyle
		if m.cursor == i {
			cursor = "> "
			style = selectedStyle
		}

		line := style.Render(cursor + item.Label)
		if item.Desc != "" {
			line += dimStyle.Render("  " + item.Desc)
		}
		s += line + "\n"
	}

	s += "\n" + dimStyle.Render("↑/↓ to move, enter to select, esc to cancel")
	return s
}

// Pick displays an interactive list picker and returns the selected index.
// Returns an error if the user cancels.
func Pick(title string, items []PickerItem) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("no items to pick from")
	}

	m := pickerModel{
		items:  items,
		cursor: 0,
		chosen: -1,
		title:  title,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return -1, err
	}

	result := finalModel.(pickerModel)
	if result.chosen == -1 {
		return -1, fmt.Errorf("cancelled")
	}

	return result.chosen, nil
}
