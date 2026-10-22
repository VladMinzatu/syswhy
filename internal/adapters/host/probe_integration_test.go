//go:build integration

package host

import (
	"context"
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

// TestIntegrationProbe runs the probe against the real kernel. The container
// environment it runs in is given by SYSWHY_IT_ENV (see `make test-integration`):
//
//   - privileged:   run with the flags `make run` uses; everything is available
//   - unprivileged: plain container; only kernel-wide facts (BTF) are available
//
// Integration hosts are assumed to be supported hosts, i.e. to have BTF.
func TestIntegrationProbe(t *testing.T) {
	env := os.Getenv("SYSWHY_IT_ENV")
	if env != "privileged" && env != "unprivileged" {
		t.Fatalf("SYSWHY_IT_ENV must be privileged or unprivileged, got %q", env)
	}
	privileged := env == "privileged"

	got, err := New().Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var uts unix.Utsname
	if err := unix.Uname(&uts); err != nil {
		t.Fatal(err)
	}
	if want := unix.ByteSliceToString(uts.Release[:]); got.KernelRelease != want {
		t.Errorf("KernelRelease = %q, uname says %q", got.KernelRelease, want)
	}

	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var data [2]unix.CapUserData
	if err := unix.Capget(&hdr, &data[0]); err != nil {
		t.Fatal(err)
	}
	if want := uint64(data[1].Effective)<<32 | uint64(data[0].Effective); got.EffectiveCaps != want {
		t.Errorf("EffectiveCaps = %#x, capget says %#x", got.EffectiveCaps, want)
	}

	if !got.BTF {
		t.Error("BTF = false, want true")
	}
	if got.BPFFS != privileged {
		t.Errorf("BPFFS = %v, want %v", got.BPFFS, privileged)
	}
	if got.TraceFS != privileged {
		t.Errorf("TraceFS = %v, want %v", got.TraceFS, privileged)
	}
}
