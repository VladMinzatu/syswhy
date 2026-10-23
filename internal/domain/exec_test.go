package domain

import "testing"

func TestCommandLine(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{nil, ""},
		{[]string{"ls"}, "ls"},
		{[]string{"ls", "-l", "/tmp"}, "ls -l /tmp"},
	}
	for _, tt := range tests {
		if got := (ExecEvent{Args: tt.args}).CommandLine(); got != tt.want {
			t.Errorf("CommandLine(%q) = %q, want %q", tt.args, got, tt.want)
		}
	}
}
