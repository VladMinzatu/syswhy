// Package host probes the local Linux host via /proc and /sys.
package host

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

// Probe reads host facts from a filesystem rooted at "/".
type Probe struct {
	fsys fs.FS
}

// New returns a Probe over the real root filesystem.
func New() Probe {
	return NewFS(os.DirFS("/"))
}

// NewFS returns a Probe over fsys, which must be laid out like "/".
func NewFS(fsys fs.FS) Probe {
	return Probe{fsys: fsys}
}

func (p Probe) Probe(context.Context) (domain.HostInfo, error) {
	release, err := fs.ReadFile(p.fsys, "proc/sys/kernel/osrelease")
	if err != nil {
		return domain.HostInfo{}, fmt.Errorf("reading kernel release: %w", err)
	}
	caps, err := p.effectiveCaps()
	if err != nil {
		return domain.HostInfo{}, err
	}
	bpffs, err := p.mounted("/sys/fs/bpf", "bpf")
	if err != nil {
		return domain.HostInfo{}, err
	}
	debugfs, err := p.mounted("/sys/kernel/debug", "debugfs")
	if err != nil {
		return domain.HostInfo{}, err
	}
	return domain.HostInfo{
		KernelRelease: strings.TrimSpace(string(release)),
		BTF:           p.exists("sys/kernel/btf/vmlinux"),
		BPFFS:         bpffs,
		DebugFS:       debugfs,
		// Stat rather than the mount table: tracefs under debugfs is automounted on access.
		TraceFS: p.exists("sys/kernel/tracing/available_events") ||
			p.exists("sys/kernel/debug/tracing/available_events"),
		EffectiveCaps: caps,
	}, nil
}

func (p Probe) exists(name string) bool {
	_, err := fs.Stat(p.fsys, name)
	return err == nil
}

func (p Probe) effectiveCaps() (uint64, error) {
	status, err := fs.ReadFile(p.fsys, "proc/self/status")
	if err != nil {
		return 0, fmt.Errorf("reading process status: %w", err)
	}
	s := bufio.NewScanner(bytes.NewReader(status))
	for s.Scan() {
		if v, ok := strings.CutPrefix(s.Text(), "CapEff:"); ok {
			caps, err := strconv.ParseUint(strings.TrimSpace(v), 16, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing CapEff: %w", err)
			}
			return caps, nil
		}
	}
	return 0, fmt.Errorf("CapEff not found in process status")
}

func (p Probe) mounted(target, fstype string) (bool, error) {
	mounts, err := fs.ReadFile(p.fsys, "proc/self/mounts")
	if err != nil {
		return false, fmt.Errorf("reading mounts: %w", err)
	}
	s := bufio.NewScanner(bytes.NewReader(mounts))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) >= 3 && f[1] == target && f[2] == fstype {
			return true, nil
		}
	}
	return false, nil
}
