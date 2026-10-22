// Package tui is the driving adapter: a Bubble Tea front end for syswhy.
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

// HostChecker assesses the host's readiness for eBPF diagnostics.
type HostChecker interface {
	Run(ctx context.Context) (domain.Readiness, error)
}

type hostCheckedMsg struct {
	readiness domain.Readiness
	err       error
}

// Model is the root Bubble Tea model.
type Model struct {
	checker   HostChecker
	checked   bool
	readiness domain.Readiness
	err       error
}

// New returns the root model.
func New(checker HostChecker) Model {
	return Model{checker: checker}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		r, err := m.checker.Run(context.Background())
		return hostCheckedMsg{readiness: r, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case hostCheckedMsg:
		m.checked, m.readiness, m.err = true, msg.readiness, msg.err
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	var b strings.Builder
	b.WriteString("syswhy\n\n")
	switch {
	case !m.checked:
		b.WriteString("Checking host...\n")
	case m.err != nil:
		fmt.Fprintf(&b, "Host check failed: %v\n", m.err)
	default:
		writeReadiness(&b, m.readiness)
	}
	b.WriteString("\nPress q to quit.\n")
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func writeReadiness(b *strings.Builder, r domain.Readiness) {
	fmt.Fprintf(b, "Kernel %s\n\n", r.KernelRelease)
	for _, c := range r.Checks {
		if c.OK {
			fmt.Fprintf(b, "  ✓ %s\n", c.Name)
		} else {
			fmt.Fprintf(b, "  ✗ %s: %s\n", c.Name, c.Detail)
		}
	}
	if r.Ready() {
		b.WriteString("\nReady for eBPF diagnostics.\n")
	} else {
		b.WriteString("\nNot ready for eBPF diagnostics.\n")
	}
}
