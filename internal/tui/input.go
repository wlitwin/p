package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type inputModel struct {
	value    string
	prompt   string
	done     bool
	quitting bool
}

func (m inputModel) Init() tea.Cmd {
	return nil
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		case "backspace":
			if len(m.value) > 0 {
				m.value = m.value[:len(m.value)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.value += msg.String()
			}
		}
	}
	return m, nil
}

func (m inputModel) View() string {
	if m.done || m.quitting {
		return ""
	}
	cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("█")
	return dimStyle.Render(m.prompt) + " " + m.value + cursor + "\n"
}

func Input(prompt string) (string, error) {
	m := inputModel{prompt: prompt}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(inputModel)
	if result.quitting {
		return "", fmt.Errorf("cancelled")
	}
	return result.value, nil
}

func Confirm(prompt string) (bool, error) {
	m := inputModel{prompt: prompt + " [Y/n]"}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	result := finalModel.(inputModel)
	if result.quitting {
		return false, nil
	}
	v := result.value
	return v == "" || v == "y" || v == "Y" || v == "yes", nil
}
