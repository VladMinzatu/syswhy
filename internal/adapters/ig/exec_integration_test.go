//go:build integration

package ig

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

// TestIntegrationTraceExec runs trace_exec from the embedded gadgets against
// the real kernel (see SYSWHY_IT_ENV in the host adapter's integration test).
// Privileged, it must observe an exec of ours; unprivileged, it must fail
// rather than hang.
func TestIntegrationTraceExec(t *testing.T) {
	var log bytes.Buffer
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("IG log:\n%s", log.String())
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	events := make(chan domain.ExecEvent, 1024)
	done := make(chan error, 1)
	go func() {
		done <- New(slog.New(slog.NewTextHandler(&log, nil))).TraceExec(ctx, func(e domain.ExecEvent) {
			select {
			case events <- e:
			default:
			}
		})
	}()

	switch env := os.Getenv("SYSWHY_IT_ENV"); env {
	case "unprivileged":
		select {
		case err := <-done:
			if err == nil {
				t.Fatal("expected an error without privileges")
			}
			t.Logf("failed as expected: %v", err)
		case <-ctx.Done():
			t.Fatal("tracing hung without privileges")
		}
		return
	case "privileged":
	default:
		t.Fatalf("SYSWHY_IT_ENV must be privileged or unprivileged, got %q", env)
	}

	// Exec a uniquely tagged command until the tracer reports it: the gadget
	// takes a moment to attach.
	marker := fmt.Sprintf("syswhy-it-%d", time.Now().UnixNano())
	want := []string{"/busybox/true", marker}
	pids := map[int]bool{}
	start := time.Now()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	var got domain.ExecEvent
wait:
	for {
		select {
		case <-ticker.C:
			cmd := exec.Command(want[0], want[1])
			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
			pids[cmd.Process.Pid] = true
		case e := <-events:
			if slices.Equal(e.Args, want) {
				got = e
				break wait
			}
		case err := <-done:
			t.Fatalf("tracing stopped early: %v", err)
		case <-ctx.Done():
			t.Fatal("never saw our exec")
		}
	}

	// eBPF reports PIDs in the kernel's initial PID namespace. They match ours
	// only if we run in it, which --pid=host gives on a regular Linux host
	// but not on e.g. OrbStack, where Docker's host namespace is nested.
	if initialPIDNamespace(t) {
		if !pids[got.PID] {
			t.Errorf("PID %d is not one of our execs %v", got.PID, pids)
		}
		if got.PPID != os.Getpid() {
			t.Errorf("PPID = %d, want %d", got.PPID, os.Getpid())
		}
	} else {
		t.Log("NOT checking PIDs: not in the kernel's initial PID namespace")
	}
	if got.Comm != "true" || got.UID != 0 {
		t.Errorf("Comm = %q, UID = %d; want true, 0", got.Comm, got.UID)
	}
	if got.Time.Before(start.Add(-time.Second)) || got.Time.After(time.Now().Add(time.Second)) {
		t.Errorf("Time = %v, want between %v and now", got.Time, start)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("tracing stopped with %v, want clean stop", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("tracing did not stop after cancel")
	}
}

// initialPIDNamespace reports whether we run in the kernel's initial PID
// namespace, which always has inode PROC_PID_INIT_INO.
func initialPIDNamespace(t *testing.T) bool {
	ns, err := os.Readlink("/proc/self/ns/pid")
	if err != nil {
		t.Fatal(err)
	}
	return ns == "pid:[4026531836]"
}
