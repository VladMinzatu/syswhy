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

const (
	maxEvents   = 1000 // exec events kept in memory
	defaultRows = 20   // table rows when the terminal height is unknown
	chromeLines = 5    // title, blank line, table header, blank line, footer
)

// Model is the root Bubble Tea model.
type Model struct {
	ctx     context.Context
	checker HostChecker
	tracer  ExecTracer

	checked   bool
	readiness domain.Readiness
	err       error

	stream    *execStream
	events    []domain.ExecEvent
	traceDone bool
	traceErr  error

	width, height int
}

// New returns the root model. Background work stops when ctx is done.
func New(ctx context.Context, checker HostChecker, tracer ExecTracer) Model {
	return Model{ctx: ctx, checker: checker, tracer: tracer}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		r, err := m.checker.Run(m.ctx)
		return hostCheckedMsg{readiness: r, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case hostCheckedMsg:
		m.checked, m.readiness, m.err = true, msg.readiness, msg.err
		if m.err == nil && m.readiness.Ready() {
			return m, startExec(m.ctx, m.tracer)
		}
	case execStartedMsg:
		m.stream = msg.stream
		return m, m.stream.next
	case execEventsMsg:
		m.events = append(m.events, msg.events...)
		if over := len(m.events) - maxEvents; over > 0 {
			m.events = m.events[over:]
		}
		if m.stream != nil {
			return m, m.stream.next
		}
	case execEndedMsg:
		m.traceDone, m.traceErr = true, msg.err
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
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
	switch {
	case !m.checked:
		b.WriteString("syswhy\n\nChecking host...\n")
	case m.err != nil:
		fmt.Fprintf(&b, "syswhy\n\nHost check failed: %v\n", m.err)
	case !m.readiness.Ready():
		b.WriteString("syswhy\n\n")
		writeReadiness(&b, m.readiness)
	default:
		fmt.Fprintf(&b, "syswhy · kernel %s · process execs\n\n", m.readiness.KernelRelease)
		writeExecTable(&b, m.events, m.tableRows(), m.width)
	}
	b.WriteString("\n" + m.footer() + "\n")
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (m Model) tableRows() int {
	if m.height == 0 {
		return defaultRows
	}
	return max(m.height-chromeLines, 0)
}

func (m Model) footer() string {
	parts := []string{"q quit"}
	if m.stream != nil {
		parts = append(parts, fmt.Sprintf("%d events", len(m.events)))
		if d := m.stream.dropped.Load(); d > 0 {
			parts = append(parts, fmt.Sprintf("%d dropped", d))
		}
	}
	if m.traceDone {
		if m.traceErr != nil {
			parts = append(parts, fmt.Sprintf("tracing failed: %v", m.traceErr))
		} else {
			parts = append(parts, "tracing stopped")
		}
	}
	return strings.Join(parts, " · ")
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
	b.WriteString("\nNot ready for eBPF diagnostics.\n")
}
