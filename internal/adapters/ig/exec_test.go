package ig

import (
	"strings"
	"testing"
	"time"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/datasource"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-service/api"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

// execFields mirrors the trace_exec fields the decoder reads.
var execFields = map[string]api.Kind{
	"timestamp_raw":    api.Kind_Uint64,
	"proc.pid":         api.Kind_Uint32,
	"proc.parent.pid":  api.Kind_Uint32,
	"proc.creds.uid":   api.Kind_Uint32,
	"proc.comm":        api.Kind_CString,
	"proc.parent.comm": api.Kind_CString,
	"args":             api.Kind_CString,
}

func execSource(t *testing.T, skip string) datasource.DataSource {
	t.Helper()
	ds, err := datasource.New(datasource.TypeSingle, "exec")
	if err != nil {
		t.Fatal(err)
	}
	for name, kind := range execFields {
		if name == skip {
			continue
		}
		if _, err := ds.AddField(name, kind); err != nil {
			t.Fatal(err)
		}
	}
	return ds
}

func TestDecodeExec(t *testing.T) {
	ds := execSource(t, "")
	p, err := ds.NewPacketSingle()
	if err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 10, 23, 12, 0, 0, 123456789, time.UTC)
	put := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	put(ds.GetField("timestamp_raw").PutUint64(p, uint64(when.UnixNano())))
	put(ds.GetField("proc.pid").PutUint32(p, 42))
	put(ds.GetField("proc.parent.pid").PutUint32(p, 7))
	put(ds.GetField("proc.creds.uid").PutUint32(p, 1000))
	put(ds.GetField("proc.comm").PutString(p, "ls"))
	put(ds.GetField("proc.parent.comm").PutString(p, "sh"))
	put(ds.GetField("args").PutString(p, strings.Join([]string{"ls", "-l", "my dir"}, argsSeparator)))

	dec, err := newExecDecoder(ds)
	if err != nil {
		t.Fatal(err)
	}
	got, err := dec.decode(p)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.ExecEvent{
		Time: when, PID: 42, PPID: 7, UID: 1000, Comm: "ls", ParentComm: "sh",
		Args: []string{"ls", "-l", "my dir"},
	}
	if !got.Time.Equal(want.Time) {
		t.Errorf("Time = %v, want %v", got.Time, want.Time)
	}
	got.Time = want.Time
	if got.CommandLine() != want.CommandLine() || got.PID != want.PID || got.PPID != want.PPID ||
		got.UID != want.UID || got.Comm != want.Comm || got.ParentComm != want.ParentComm || len(got.Args) != 3 {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestDecodeExecEmptyArgs(t *testing.T) {
	ds := execSource(t, "")
	p, err := ds.NewPacketSingle()
	if err != nil {
		t.Fatal(err)
	}
	dec, err := newExecDecoder(ds)
	if err != nil {
		t.Fatal(err)
	}
	got, err := dec.decode(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Args != nil {
		t.Errorf("Args = %q, want nil", got.Args)
	}
}

func TestExecDecoderRequiresFields(t *testing.T) {
	for name := range execFields {
		t.Run(name, func(t *testing.T) {
			_, err := newExecDecoder(execSource(t, name))
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("err = %v, want missing %q", err, name)
			}
		})
	}
}
