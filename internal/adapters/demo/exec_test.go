package demo

import (
	"context"
	"testing"
	"time"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

func TestExecTracerEmitsUntilCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var got []domain.ExecEvent
	err := ExecTracer{Interval: time.Millisecond}.TraceExec(ctx, func(e domain.ExecEvent) {
		got = append(got, e)
		if len(got) == 3 {
			cancel()
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d events, want 3", len(got))
	}
	if got[0].PID == got[1].PID || got[0].Comm == "" || len(got[0].Args) == 0 {
		t.Errorf("unexpected events %+v", got)
	}
}
