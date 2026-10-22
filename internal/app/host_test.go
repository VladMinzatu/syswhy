package app

import (
	"context"
	"errors"
	"testing"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

type fakeProbe struct {
	info domain.HostInfo
	err  error
}

func (f fakeProbe) Probe(context.Context) (domain.HostInfo, error) {
	return f.info, f.err
}

func TestHostCheckAssessesProbedFacts(t *testing.T) {
	probe := fakeProbe{info: domain.HostInfo{KernelRelease: "6.8.0", BTF: true}}
	r, err := NewHostCheck(probe).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.KernelRelease != "6.8.0" || len(r.Checks) == 0 || r.Ready() {
		t.Errorf("unexpected readiness %+v", r)
	}
}

func TestHostCheckPropagatesProbeError(t *testing.T) {
	want := errors.New("boom")
	_, err := NewHostCheck(fakeProbe{err: want}).Run(context.Background())
	if !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
}
