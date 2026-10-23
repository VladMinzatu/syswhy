package tui

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

// ExecTracer streams exec events from the host.
type ExecTracer interface {
	TraceExec(ctx context.Context, emit func(domain.ExecEvent)) error
}

const (
	streamBuffer = 1024 // events buffered between tracer and UI
	maxBatch     = 256  // events delivered to the UI per message
)

// execStream bridges a blocking ExecTracer to Bubble Tea messages. Events
// that arrive while the buffer is full are dropped and counted, so a slow UI
// never stalls the tracer.
type execStream struct {
	events  chan domain.ExecEvent
	err     error // set before events is closed
	dropped atomic.Int64
}

type execStartedMsg struct{ stream *execStream }

type execEventsMsg struct{ events []domain.ExecEvent }

type execEndedMsg struct{ err error }

func startExec(ctx context.Context, tracer ExecTracer) tea.Cmd {
	return func() tea.Msg {
		s := &execStream{events: make(chan domain.ExecEvent, streamBuffer)}
		go func() {
			s.err = tracer.TraceExec(ctx, func(e domain.ExecEvent) {
				select {
				case s.events <- e:
				default:
					s.dropped.Add(1)
				}
			})
			close(s.events)
		}()
		return execStartedMsg{stream: s}
	}
}

// next blocks for the next event, then batches whatever else is buffered.
func (s *execStream) next() tea.Msg {
	e, ok := <-s.events
	if !ok {
		return execEndedMsg{err: s.err}
	}
	batch := []domain.ExecEvent{e}
	for len(batch) < maxBatch {
		select {
		case e, ok := <-s.events:
			if !ok {
				return execEventsMsg{events: batch}
			}
			batch = append(batch, e)
		default:
			return execEventsMsg{events: batch}
		}
	}
	return execEventsMsg{events: batch}
}

const execHeader = "TIME          PID     PPID    UID    COMM             ARGS"

func execRow(e domain.ExecEvent) string {
	return fmt.Sprintf("%-12s  %-6d  %-6d  %-5d  %-15s  %s",
		e.Time.Format("15:04:05.000"), e.PID, e.PPID, e.UID, truncate(e.Comm, 15), e.CommandLine())
}

// truncate cuts s to at most n runes.
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func writeExecTable(b *strings.Builder, events []domain.ExecEvent, rows, width int) {
	if len(events) > rows {
		events = events[len(events)-rows:]
	}
	line := func(s string) {
		if width > 0 {
			s = truncate(s, width)
		}
		b.WriteString(s + "\n")
	}
	line(execHeader)
	for _, e := range events {
		line(execRow(e))
	}
}
