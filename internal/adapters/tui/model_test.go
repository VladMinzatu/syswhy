package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

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
			_, cmd := New().Update(tt.key)
			if cmd == nil {
				t.Fatal("expected a command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("expected tea.QuitMsg")
			}
		})
	}
}

func TestOtherKeysDoNotQuit(t *testing.T) {
	_, cmd := New().Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if cmd != nil {
		t.Fatalf("expected no command, got %T", cmd())
	}
}

func TestView(t *testing.T) {
	v := New().View()
	for _, want := range []string{"syswhy", "q quit"} {
		if !strings.Contains(v.Content, want) {
			t.Errorf("view missing %q:\n%s", want, v.Content)
		}
	}
	if !v.AltScreen {
		t.Error("view should use the alternate screen")
	}
}
