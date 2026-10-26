// Package tui is the driving adapter: a Bubble Tea front end for syswhy.
package tui

import (
	tea "charm.land/bubbletea/v2"
)

// Model is the root Bubble Tea model.
type Model struct{}

// New returns the root model.
func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	v := tea.NewView("syswhy\n\nNo tools yet.\n\nq quit\n")
	v.AltScreen = true
	return v
}
