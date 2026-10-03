package main

import (
	"errors"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type inputModel struct {
	text      textarea.Model
	submitted bool
}

func (m inputModel) Init() tea.Cmd { return textarea.Blink }
func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.text.SetWidth(msg.Width - 4)
		m.text.SetHeight(max(3, msg.Height-6))
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			m.submitted = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.text, cmd = m.text.Update(msg)
	return m, cmd
}
func (m inputModel) View() string {
	return "Instructions — Ctrl+S: apply • Esc: cancel\n\n" + m.text.View() + "\n"
}
func editInput() ([]byte, error) {
	text := textarea.New()
	text.Placeholder = "Enter instructions…"
	text.CharLimit = 0
	text.Focus()
	result, err := tea.NewProgram(inputModel{text: text}).Run()
	if err != nil {
		return nil, err
	}
	m := result.(inputModel)
	if !m.submitted {
		return nil, errors.New("cancelled")
	}
	return []byte(m.text.Value() + "\n"), nil
}
