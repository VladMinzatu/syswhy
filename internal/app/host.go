// Package app holds syswhy's use cases and the ports they depend on.
package app

import (
	"context"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

// HostProbe gathers facts about the host.
type HostProbe interface {
	Probe(ctx context.Context) (domain.HostInfo, error)
}

// HostCheck assesses whether the host can run eBPF diagnostics.
type HostCheck struct {
	probe HostProbe
}

func NewHostCheck(probe HostProbe) HostCheck {
	return HostCheck{probe: probe}
}

func (c HostCheck) Run(ctx context.Context) (domain.Readiness, error) {
	info, err := c.probe.Probe(ctx)
	if err != nil {
		return domain.Readiness{}, err
	}
	return domain.Assess(info), nil
}
