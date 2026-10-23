package app

import (
	"context"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

// ExecTracer streams exec events from the host.
type ExecTracer interface {
	// TraceExec calls emit for each event until ctx is done or tracing fails.
	// It returns nil when stopped by ctx. emit must not block for long.
	TraceExec(ctx context.Context, emit func(domain.ExecEvent)) error
}
