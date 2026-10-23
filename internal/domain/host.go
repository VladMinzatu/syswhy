// Package domain holds syswhy's core types and logic, free of I/O.
package domain

// Linux capability bit numbers relevant to eBPF.
const (
	CapSysAdmin = 21
	CapPerfmon  = 38
	CapBPF      = 39
)

// HostInfo are the facts about a host that determine whether eBPF-based
// diagnostics can run on it.
type HostInfo struct {
	KernelRelease string
	BTF           bool   // kernel BTF type information is available
	BPFFS         bool   // the BPF filesystem is mounted at /sys/fs/bpf
	DebugFS       bool   // debugfs is mounted at /sys/kernel/debug
	TraceFS       bool   // tracefs is reachable
	EffectiveCaps uint64 // effective capability bitmask of this process
}

// HasCap reports whether capability bit c is in the effective set.
func (h HostInfo) HasCap(c uint) bool {
	return h.EffectiveCaps&(1<<c) != 0
}

// Check is the outcome of a single readiness requirement.
type Check struct {
	Name   string
	OK     bool
	Detail string // how to fix it; empty when OK
}

// Readiness is the assessment of a host for eBPF diagnostics.
type Readiness struct {
	KernelRelease string
	Checks        []Check
}

// Ready reports whether all checks passed.
func (r Readiness) Ready() bool {
	for _, c := range r.Checks {
		if !c.OK {
			return false
		}
	}
	return true
}

// Assess evaluates host facts against the requirements for eBPF diagnostics.
func Assess(h HostInfo) Readiness {
	return Readiness{
		KernelRelease: h.KernelRelease,
		Checks: []Check{
			check("BTF type information", h.BTF,
				"kernel lacks /sys/kernel/btf/vmlinux (needs CONFIG_DEBUG_INFO_BTF)"),
			check("BPF filesystem", h.BPFFS,
				"mount it: -v /sys/fs/bpf:/sys/fs/bpf"),
			check("debugfs", h.DebugFS,
				"mount it: -v /sys/kernel/debug:/sys/kernel/debug"),
			check("tracefs", h.TraceFS,
				"mount it: -v /sys/kernel/tracing:/sys/kernel/tracing"),
			check("eBPF privileges", h.HasCap(CapSysAdmin) || (h.HasCap(CapBPF) && h.HasCap(CapPerfmon)),
				"needs CAP_SYS_ADMIN, or CAP_BPF and CAP_PERFMON: run with --privileged"),
		},
	}
}

func check(name string, ok bool, fix string) Check {
	if ok {
		return Check{Name: name, OK: true}
	}
	return Check{Name: name, Detail: fix}
}
