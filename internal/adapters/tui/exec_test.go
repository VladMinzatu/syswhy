package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

func started(t *testing.T, ctx context.Context, tracer ExecTracer) *execStream {
	t.Helper()
	msg, ok := startExec(ctx, tracer)().(execStartedMsg)
	if !ok {
		t.Fatal("startExec did not return execStartedMsg")
	}
	return msg.stream
}

func TestStreamBatchesBufferedEvents(t *testing.T) {
	// The tracer returns after emitting, so all events are buffered before
	// the stream closes.
	tracer := fakeTracer{events: []domain.ExecEvent{event(1, "a"), event(2, "b"), event(3, "c")}, err: errors.New("done")}
	s := started(t, context.Background(), tracer)

	var got []domain.ExecEvent
	for {
		switch msg := s.next().(type) {
		case execEventsMsg:
			got = append(got, msg.events...)
			continue
		case execEndedMsg:
			if msg.err == nil || msg.err.Error() != "done" {
				t.Errorf("ended with %v, want done", msg.err)
			}
		default:
			t.Fatalf("unexpected message %T", msg)
		}
		break
	}
	if len(got) != 3 || got[0].PID != 1 || got[2].PID != 3 {
		t.Errorf("got events %+v", got)
	}
}

// burstTracer emits n events, signals emitted, then returns.
type burstTracer struct {
	n       int
	emitted chan struct{}
}

func (b burstTracer) TraceExec(_ context.Context, emit func(domain.ExecEvent)) error {
	for i := range b.n {
		emit(event(i, "x"))
	}
	close(b.emitted)
	return nil
}

func TestStreamDropsWhenBufferFull(t *testing.T) {
	tracer := burstTracer{n: streamBuffer + 5, emitted: make(chan struct{})}
	s := started(t, context.Background(), tracer)
	<-tracer.emitted // read only after the whole burst, so overflow is deterministic

	n := 0
	for {
		msg, ok := s.next().(execEventsMsg)
		if !ok {
			break
		}
		n += len(msg.events)
	}
	if n != streamBuffer || s.dropped.Load() != 5 {
		t.Errorf("delivered %d, dropped %d; want %d and 5", n, s.dropped.Load(), streamBuffer)
	}
}

func TestStreamEndsCleanlyOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := started(t, ctx, fakeTracer{})
	cancel()
	msg, ok := s.next().(execEndedMsg)
	if !ok || msg.err != nil {
		t.Errorf("got %#v, want clean execEndedMsg", msg)
	}
}
