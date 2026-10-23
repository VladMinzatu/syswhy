package ig

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/inspektor-gadget/inspektor-gadget/pkg/datasource"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators"

	"github.com/VladMinzatu/syswhy/internal/domain"
)

// subscriberPriority runs our subscribers after IG's own operators (e.g.
// formatters), which use lower priorities.
const subscriberPriority = 50000

// argsSeparator joins argv entries in trace_exec's "args" field.
const argsSeparator = " "

// TraceExec streams exec events using the trace_exec gadget.
func (r *Runtime) TraceExec(ctx context.Context, emit func(domain.ExecEvent)) error {
	return r.run(ctx, "trace_exec", func(gctx operators.GadgetContext) error {
		ds, ok := gctx.GetDataSources()["exec"]
		if !ok {
			return errors.New("trace_exec has no exec data source")
		}
		dec, err := newExecDecoder(ds)
		if err != nil {
			return err
		}
		return ds.Subscribe(func(_ datasource.DataSource, data datasource.Data) error {
			e, err := dec.decode(data)
			if err != nil {
				return err
			}
			emit(e)
			return nil
		}, subscriberPriority)
	})
}

// execDecoder maps trace_exec records to domain events.
type execDecoder struct {
	timestamp, pid, ppid, uid, comm, parentComm, args datasource.FieldAccessor
}

func newExecDecoder(ds datasource.DataSource) (*execDecoder, error) {
	var errs []error
	field := func(name string) datasource.FieldAccessor {
		f := ds.GetField(name)
		if f == nil {
			errs = append(errs, fmt.Errorf("trace_exec has no field %q", name))
		}
		return f
	}
	d := &execDecoder{
		timestamp:  field("timestamp_raw"),
		pid:        field("proc.pid"),
		ppid:       field("proc.parent.pid"),
		uid:        field("proc.creds.uid"),
		comm:       field("proc.comm"),
		parentComm: field("proc.parent.comm"),
		args:       field("args"),
	}
	return d, errors.Join(errs...)
}

func (d *execDecoder) decode(data datasource.Data) (domain.ExecEvent, error) {
	var errs []error
	u32 := func(f datasource.FieldAccessor) int {
		v, err := f.Uint32(data)
		errs = append(errs, err)
		return int(v)
	}
	str := func(f datasource.FieldAccessor) string {
		v, err := f.String(data)
		errs = append(errs, err)
		return v
	}
	ts, err := d.timestamp.Uint64(data)
	errs = append(errs, err)
	e := domain.ExecEvent{
		Time:       time.Unix(0, int64(ts)), // nanoseconds since the epoch, after formatters
		PID:        u32(d.pid),
		PPID:       u32(d.ppid),
		UID:        u32(d.uid),
		Comm:       str(d.comm),
		ParentComm: str(d.parentComm),
	}
	if args := str(d.args); args != "" {
		e.Args = strings.Split(args, argsSeparator)
	}
	if err := errors.Join(errs...); err != nil {
		return domain.ExecEvent{}, fmt.Errorf("decoding exec event: %w", err)
	}
	return e, nil
}
