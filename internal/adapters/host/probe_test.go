package host

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

func file(s string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(s)}
}

func fullHost() fstest.MapFS {
	return fstest.MapFS{
		"proc/sys/kernel/osrelease": file("6.8.0-generic\n"),
		"proc/self/status":          file("Name:\tsyswhy\nCapEff:\t000001ffffffffff\n"),
		"proc/self/mounts": file("sysfs /sys sysfs rw 0 0\n" +
			"bpf /sys/fs/bpf bpf rw,mode=700 0 0\n" +
			"debugfs /sys/kernel/debug debugfs rw 0 0\n"),
		"sys/kernel/btf/vmlinux":              file(""),
		"sys/kernel/tracing/available_events": file(""),
	}
}

func TestProbeFullHost(t *testing.T) {
	got, err := NewFS(fullHost()).Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := domain.HostInfo{
		KernelRelease: "6.8.0-generic",
		BTF:           true,
		BPFFS:         true,
		DebugFS:       true,
		TraceFS:       true,
		EffectiveCaps: 0x1ffffffffff,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestProbeBareHost(t *testing.T) {
	fsys := fstest.MapFS{
		"proc/sys/kernel/osrelease": file("6.8.0\n"),
		"proc/self/status":          file("CapEff:\t00000000a80425fb\n"),
		"proc/self/mounts":          file("sysfs /sys sysfs rw 0 0\n"),
	}
	got, err := NewFS(fsys).Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.BTF || got.BPFFS || got.DebugFS || got.TraceFS {
		t.Errorf("expected nothing available, got %+v", got)
	}
	if got.EffectiveCaps != 0xa80425fb {
		t.Errorf("caps = %x", got.EffectiveCaps)
	}
}

func TestProbeTraceFSUnderDebugFS(t *testing.T) {
	fsys := fullHost()
	delete(fsys, "sys/kernel/tracing/available_events")
	fsys["sys/kernel/debug/tracing/available_events"] = file("")
	got, err := NewFS(fsys).Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got.TraceFS {
		t.Error("expected tracefs via debugfs")
	}
}

func TestProbeBPFFSWrongType(t *testing.T) {
	fsys := fullHost()
	fsys["proc/self/mounts"] = file("tmpfs /sys/fs/bpf tmpfs rw 0 0\n")
	got, err := NewFS(fsys).Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.BPFFS {
		t.Error("tmpfs at /sys/fs/bpf must not count as bpffs")
	}
}

func TestProbeErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(fstest.MapFS)
	}{
		{"missing osrelease", func(f fstest.MapFS) { delete(f, "proc/sys/kernel/osrelease") }},
		{"missing status", func(f fstest.MapFS) { delete(f, "proc/self/status") }},
		{"missing CapEff", func(f fstest.MapFS) { f["proc/self/status"] = file("Name:\tx\n") }},
		{"bad CapEff", func(f fstest.MapFS) { f["proc/self/status"] = file("CapEff:\tzz\n") }},
		{"missing mounts", func(f fstest.MapFS) { delete(f, "proc/self/mounts") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fullHost()
			tt.mutate(fsys)
			if _, err := NewFS(fsys).Probe(context.Background()); err == nil {
				t.Error("expected error")
			}
		})
	}
}
