package ig

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"testing/fstest"
)

const index = `{"manifests": [
	{"annotations": {"org.opencontainers.image.ref.name": "ghcr.io/inspektor-gadget/gadget/trace_exec:v0.57.0"}},
	{"annotations": {"org.opencontainers.image.ref.name": "sha256-9182278eb6a1"}},
	{"annotations": {"org.opencontainers.image.ref.name": "ghcr.io/syswhy/trace_exec_extra:dev"}},
	{}
]}`

func layout(index string) fstest.MapFS {
	return fstest.MapFS{"index.json": &fstest.MapFile{Data: []byte(index)}}
}

func TestImageRef(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{"trace_exec", "ghcr.io/inspektor-gadget/gadget/trace_exec:v0.57.0"},
		{"trace_exec_extra", "ghcr.io/syswhy/trace_exec_extra:dev"},
	}
	for _, tt := range tests {
		got, err := imageRef(layout(index), tt.name)
		if err != nil || got != tt.want {
			t.Errorf("imageRef(%q) = %q, %v; want %q", tt.name, got, err, tt.want)
		}
	}
}

func TestImageRefErrors(t *testing.T) {
	tests := []struct {
		name   string
		layout fstest.MapFS
		want   string
	}{
		{"not embedded", layout(index), "not embedded"},
		{"no layout", fstest.MapFS{}, "built without them"},
		{"bad index", layout("{"), "parsing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := imageRef(tt.layout, "top_process")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestTraceExecWithoutEmbeddedGadgets(t *testing.T) {
	// Unit tests build with the placeholder layout only.
	err := New(slog.New(slog.DiscardHandler)).TraceExec(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "built without them") {
		t.Errorf("err = %v, want missing gadgets error", err)
	}
}
