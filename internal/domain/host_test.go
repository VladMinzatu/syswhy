package domain

import "testing"

func caps(bits ...uint) uint64 {
	var m uint64
	for _, b := range bits {
		m |= 1 << b
	}
	return m
}

func readyHost() HostInfo {
	return HostInfo{
		KernelRelease: "6.8.0",
		BTF:           true,
		BPFFS:         true,
		TraceFS:       true,
		EffectiveCaps: caps(CapSysAdmin),
	}
}

func TestAssessReadyHost(t *testing.T) {
	r := Assess(readyHost())
	if !r.Ready() {
		t.Fatalf("expected ready, got %+v", r.Checks)
	}
	if r.KernelRelease != "6.8.0" {
		t.Errorf("kernel release = %q", r.KernelRelease)
	}
}

func TestAssessFailingCheck(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*HostInfo)
		failed string
	}{
		{"no BTF", func(h *HostInfo) { h.BTF = false }, "BTF type information"},
		{"no bpffs", func(h *HostInfo) { h.BPFFS = false }, "BPF filesystem"},
		{"no tracefs", func(h *HostInfo) { h.TraceFS = false }, "tracefs"},
		{"no caps", func(h *HostInfo) { h.EffectiveCaps = 0 }, "eBPF privileges"},
		{"CAP_BPF without CAP_PERFMON", func(h *HostInfo) { h.EffectiveCaps = caps(CapBPF) }, "eBPF privileges"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := readyHost()
			tt.mutate(&h)
			r := Assess(h)
			if r.Ready() {
				t.Fatal("expected not ready")
			}
			for _, c := range r.Checks {
				if c.Name == tt.failed {
					if c.OK || c.Detail == "" {
						t.Errorf("check %q: OK=%v detail=%q", c.Name, c.OK, c.Detail)
					}
				} else if !c.OK {
					t.Errorf("unexpected failing check %q", c.Name)
				}
			}
		})
	}
}

func TestAssessBPFAndPerfmonSuffice(t *testing.T) {
	h := readyHost()
	h.EffectiveCaps = caps(CapBPF, CapPerfmon)
	if r := Assess(h); !r.Ready() {
		t.Fatalf("expected ready, got %+v", r.Checks)
	}
}
