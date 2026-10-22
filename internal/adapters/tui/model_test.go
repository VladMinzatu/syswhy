package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

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

// checked runs the model's Init command and feeds the result back, as the
// Bubble Tea runtime would.
func checked(t *testing.T, c HostChecker) Model {
	t.Helper()
	m := New(c)
	next, _ := m.Update(m.Init()())
	return next.(Model)
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
			_, cmd := New(fakeChecker{}).Update(tt.key)
			if cmd == nil {
				t.Fatal("expected a command, got nil")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("expected tea.QuitMsg, got %T", cmd())
			}
		})
	}
}

func TestOtherKeysDoNotQuit(t *testing.T) {
	_, cmd := New(fakeChecker{}).Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if cmd != nil {
		t.Fatalf("expected no command, got %T", cmd())
	}
}

func TestViewBeforeCheck(t *testing.T) {
	v := New(fakeChecker{}).View()
	if !strings.Contains(v.Content, "syswhy") || !strings.Contains(v.Content, "Checking host") {
		t.Errorf("unexpected view %q", v.Content)
	}
	if !v.AltScreen {
		t.Error("view should use the alternate screen")
	}
}

func TestViewShowsReadiness(t *testing.T) {
	r := domain.Readiness{
		KernelRelease: "6.8.0",
		Checks: []domain.Check{
			{Name: "BTF type information", OK: true},
			{Name: "BPF filesystem", Detail: "mount it"},
		},
	}
	content := checked(t, fakeChecker{readiness: r}).View().Content
	for _, want := range []string{"Kernel 6.8.0", "✓ BTF type information", "✗ BPF filesystem: mount it", "Not ready"} {
		if !strings.Contains(content, want) {
			t.Errorf("view missing %q:\n%s", want, content)
		}
	}
}

func TestViewShowsReady(t *testing.T) {
	r := domain.Readiness{Checks: []domain.Check{{Name: "BTF type information", OK: true}}}
	content := checked(t, fakeChecker{readiness: r}).View().Content
	if !strings.Contains(content, "Ready for eBPF diagnostics") {
		t.Errorf("view should report ready:\n%s", content)
	}
}

func TestViewShowsCheckError(t *testing.T) {
	content := checked(t, fakeChecker{err: errors.New("no /proc")}).View().Content
	if !strings.Contains(content, "Host check failed: no /proc") {
		t.Errorf("view should show error:\n%s", content)
	}
}
