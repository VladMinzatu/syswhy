package domain

import (
	"strings"
	"time"
)

// ExecEvent is a process executing a new program.
type ExecEvent struct {
	Time       time.Time
	PID        int
	PPID       int
	UID        int
	Comm       string
	ParentComm string
	Args       []string // argv, including argv[0]
}

// CommandLine renders Args as a single space-separated line.
func (e ExecEvent) CommandLine() string {
	return strings.Join(e.Args, " ")
}
