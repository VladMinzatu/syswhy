// Package demo provides synthetic event sources, for running the TUI without
// a real tracer.
package demo

import (
	"context"
	"time"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

var commands = [][]string{
	{"ls", "-l", "/tmp"},
	{"cat", "/etc/hostname"},
	{"git", "status"},
	{"sh", "-c", "sleep 1"},
	{"curl", "-s", "https://example.com"},
}

// ExecTracer emits a synthetic exec event every Interval.
type ExecTracer struct {
	Interval time.Duration
}

func (d ExecTracer) TraceExec(ctx context.Context, emit func(domain.ExecEvent)) error {
	ticker := time.NewTicker(d.Interval)
	defer ticker.Stop()
	for pid := 1000; ; pid++ {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			args := commands[pid%len(commands)]
			emit(domain.ExecEvent{
				Time:       now,
				PID:        pid,
				PPID:       1,
				Comm:       args[0],
				ParentComm: "demo",
				Args:       args,
			})
		}
	}
}
