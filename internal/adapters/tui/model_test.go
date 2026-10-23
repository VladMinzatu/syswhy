package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

type fakeChecker struct {
	readiness domain.Readiness
	err       error
}

func (f fakeChecker) Run(context.Context) (domain.Readiness, error) {
	return f.readiness, f.err
}

// fakeTracer emits its events, then returns err (or waits for ctx if err is nil).
type fakeTracer struct {
	events []domain.ExecEvent
	err    error
}

func (f fakeTracer) TraceExec(ctx context.Context, emit func(domain.ExecEvent)) error {
	for _, e := range f.events {
		emit(e)
	}
	if f.err != nil {
		return f.err
	}
	<-ctx.Done()
	return nil
}

var ready = domain.Readiness{
	KernelRelease: "6.8.0",
	Checks:        []domain.Check{{Name: "BTF type information", OK: true}},
}

func event(pid int, args ...string) domain.ExecEvent {
	return domain.ExecEvent{
		Time: time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC),
		PID:  pid, PPID: 1, Comm: args[0], Args: args,
	}
}

// step applies msg and runs the resulting command, as the Bubble Tea runtime
// would, returning the updated model and the command's message.
func step(t *testing.T, m Model, msg tea.Msg) (Model, tea.Msg) {
	t.Helper()
	next, cmd := m.Update(msg)
	if cmd == nil {
		return next.(Model), nil
	}
	return next.(Model), cmd()
}

func TestQuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"q", tea.KeyPressMsg{Code: 'q', Text: "q"}},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, msg := step(t, Model{}, tt.key)
			if _, ok := msg.(tea.QuitMsg); !ok {
				t.Fatalf("expected tea.QuitMsg, got %T", msg)
			}
		})
	}
}

func TestOtherKeysDoNotQuit(t *testing.T) {
	_, cmd := Model{}.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if cmd != nil {
		t.Fatalf("expected no command, got %T", cmd())
	}
}

func TestViewBeforeCheck(t *testing.T) {
	v := New(context.Background(), fakeChecker{}, fakeTracer{}).View()
	if !strings.Contains(v.Content, "Checking host") {
		t.Errorf("unexpected view %q", v.Content)
	}
	if !v.AltScreen {
		t.Error("view should use the alternate screen")
	}
}

func TestNotReadyShowsChecksAndDoesNotTrace(t *testing.T) {
	r := domain.Readiness{
		KernelRelease: "6.8.0",
		Checks: []domain.Check{
			{Name: "BTF type information", OK: true},
			{Name: "BPF filesystem", Detail: "mount it"},
		},
	}
	m := New(context.Background(), fakeChecker{readiness: r}, fakeTracer{})
	m, msg := step(t, m, m.Init()())
	if msg != nil {
		t.Fatalf("expected no tracing, got %T", msg)
	}
	content := m.View().Content
	for _, want := range []string{"Kernel 6.8.0", "✓ BTF type information", "✗ BPF filesystem: mount it", "Not ready"} {
		if !strings.Contains(content, want) {
			t.Errorf("view missing %q:\n%s", want, content)
		}
	}
}

func TestViewShowsCheckError(t *testing.T) {
	m := New(context.Background(), fakeChecker{err: errors.New("no /proc")}, fakeTracer{})
	m, _ = step(t, m, m.Init()())
	if content := m.View().Content; !strings.Contains(content, "Host check failed: no /proc") {
		t.Errorf("view should show error:\n%s", content)
	}
}

func TestReadyHostTracesExecs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracer := fakeTracer{events: []domain.ExecEvent{event(42, "ls", "-l", "/tmp")}}
	m := New(ctx, fakeChecker{readiness: ready}, tracer)

	m, started := step(t, m, m.Init()())
	m, events := step(t, m, started)
	// Don't run the follow-up command: it waits for more events, and the
	// fake tracer sends no more.
	next, _ := m.Update(events)
	m = next.(Model)

	content := m.View().Content
	for _, want := range []string{"kernel 6.8.0", "PID", "42", "ls -l /tmp", "1 events"} {
		if !strings.Contains(content, want) {
			t.Errorf("view missing %q:\n%s", want, content)
		}
	}
}

func TestTraceFailureShown(t *testing.T) {
	m := New(context.Background(), fakeChecker{readiness: ready}, fakeTracer{err: errors.New("boom")})
	m, started := step(t, m, m.Init()())
	m, ended := step(t, m, started)
	m, _ = step(t, m, ended)
	if content := m.View().Content; !strings.Contains(content, "tracing failed: boom") {
		t.Errorf("view should show trace error:\n%s", content)
	}
}

func TestEventsCappedAtMax(t *testing.T) {
	var batch []domain.ExecEvent
	for i := range maxEvents + 10 {
		batch = append(batch, event(i, "true"))
	}
	m, _ := step(t, Model{}, execEventsMsg{events: batch})
	if len(m.events) != maxEvents {
		t.Fatalf("kept %d events, want %d", len(m.events), maxEvents)
	}
	if m.events[0].PID != 10 {
		t.Errorf("oldest kept PID = %d, want 10", m.events[0].PID)
	}
}

func TestTableShowsNewestRowsThatFit(t *testing.T) {
	m := Model{checked: true, readiness: ready}
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 80, Height: chromeLines + 2})
	m, _ = step(t, m, execEventsMsg{events: []domain.ExecEvent{
		event(1, "first"), event(2, "second"), event(3, "third"),
	}})
	content := m.View().Content
	if strings.Contains(content, "first") {
		t.Errorf("oldest row should be scrolled off:\n%s", content)
	}
	for _, want := range []string{"second", "third"} {
		if !strings.Contains(content, want) {
			t.Errorf("view missing %q:\n%s", want, content)
		}
	}
}

func TestRowsTruncatedToWidth(t *testing.T) {
	m := Model{checked: true, readiness: ready}
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 40, Height: 30})
	m, _ = step(t, m, execEventsMsg{events: []domain.ExecEvent{event(1, "sh", strings.Repeat("x", 100))}})
	for i, line := range strings.Split(m.View().Content, "\n") {
		if strings.HasPrefix(line, "syswhy") || strings.Contains(line, "quit") {
			continue // title and footer are not table rows
		}
		if n := len([]rune(line)); n > 40 {
			t.Errorf("line %d has %d runes, want <= 40: %q", i, n, line)
		}
	}
}

func TestFooterShowsDropped(t *testing.T) {
	s := &execStream{}
	s.dropped.Add(7)
	m := Model{checked: true, readiness: ready, stream: s}
	if content := m.View().Content; !strings.Contains(content, "7 dropped") {
		t.Errorf("footer should show dropped count:\n%s", content)
	}
}

func ExampleModel_footer() {
	fmt.Println(Model{traceDone: true}.footer())
	// Output: q quit · tracing stopped
}
