// Package ig runs Inspektor Gadget (IG) gadgets embedded in the binary. It is
// the only package that imports IG.
package ig

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"strings"
	"sync"

	orasoci "oras.land/oras-go/v2/content/oci"

	gadgetcontext "github.com/inspektor-gadget/inspektor-gadget/pkg/gadget-context"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/logger"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators"
	_ "github.com/inspektor-gadget/inspektor-gadget/pkg/operators/ebpf"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators/formatters"
	ocihandler "github.com/inspektor-gadget/inspektor-gadget/pkg/operators/oci-handler"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/operators/simple"
	_ "github.com/inspektor-gadget/inspektor-gadget/pkg/operators/wasm"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/params"
	"github.com/inspektor-gadget/inspektor-gadget/pkg/runtime/local"
)

// The OCI layout of the gadgets is generated at build time (see Dockerfile);
// the checked-in directory holds only a placeholder.
//
//go:embed all:oci
var embedded embed.FS

// dataOperators are the IG data operators run with every gadget. The
// formatters operator turns kernel timestamps into wall-clock time.
var dataOperators = []operators.DataOperator{ocihandler.OciHandler, formatters.FormattersOperator}

// Runtime runs embedded gadgets.
type Runtime struct {
	gadgets fs.FS
	logger  logger.Logger
}

// New returns a Runtime over the embedded gadgets. IG's logs, including
// those through its global logger, go to log.
func New(log *slog.Logger) *Runtime {
	gadgets, _ := fs.Sub(embedded, "oci") // cannot fail: "oci" is a valid path
	redirectGlobal(log)
	return &Runtime{gadgets: gadgets, logger: newLogger(log)}
}

var (
	initOnce sync.Once
	initOps  []operators.DataOperator
	initErr  error
)

// initOperators initializes the data operators with their default global
// params. IG allows this only once per process.
func initOperators() ([]operators.DataOperator, error) {
	initOnce.Do(func() {
		for _, op := range dataOperators {
			var descs params.ParamDescs
			for _, p := range op.GlobalParams() {
				descs = append(descs, &params.ParamDesc{Key: p.Key, DefaultValue: p.DefaultValue, TypeHint: params.TypeHint(p.TypeHint)})
			}
			if err := op.Init(descs.ToParams()); err != nil {
				initErr = fmt.Errorf("initializing IG operator %q: %w", op.Name(), err)
				return
			}
			initOps = append(initOps, op)
		}
	})
	return initOps, initErr
}

// run runs the named gadget until ctx is done. onInit is called once the
// gadget's data sources exist, to subscribe to them.
func (r *Runtime) run(ctx context.Context, name string, onInit func(operators.GadgetContext) error) error {
	ref, err := imageRef(r.gadgets, name)
	if err != nil {
		return err
	}
	ops, err := initOperators()
	if err != nil {
		return err
	}
	target, err := orasoci.NewFromFS(ctx, r.gadgets)
	if err != nil {
		return fmt.Errorf("opening embedded gadgets: %w", err)
	}
	ops = append(slices.Clone(ops), simple.New("syswhy", simple.OnInit(onInit)))
	gctx := gadgetcontext.New(ctx, ref,
		gadgetcontext.WithDataOperators(ops...),
		gadgetcontext.WithOrasReadonlyTarget(target),
		gadgetcontext.WithLogger(r.logger),
	)
	if err := checkFilesystems(statfsType); err != nil {
		return err
	}
	rt := local.New()
	if err := rt.Init(nil); err != nil {
		return fmt.Errorf("initializing IG runtime: %w", err)
	}
	defer rt.Close()
	if err := rt.RunGadget(gctx, nil, nil); err != nil {
		return fmt.Errorf("running gadget %s: %w", name, err)
	}
	return nil
}

// imageRef finds the full reference of the gadget called name (e.g.
// "trace_exec" -> "ghcr.io/inspektor-gadget/gadget/trace_exec:v0.57.0") in an
// OCI layout, so code doesn't hardcode registries or versions.
func imageRef(layout fs.FS, name string) (string, error) {
	b, err := fs.ReadFile(layout, "index.json")
	if err != nil {
		return "", fmt.Errorf("reading embedded gadgets (built without them?): %w", err)
	}
	var index struct {
		Manifests []struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"manifests"`
	}
	if err := json.Unmarshal(b, &index); err != nil {
		return "", fmt.Errorf("parsing gadget index: %w", err)
	}
	for _, m := range index.Manifests {
		ref := m.Annotations["org.opencontainers.image.ref.name"]
		repo, _, ok := strings.Cut(ref[strings.LastIndex(ref, "/")+1:], ":")
		if ok && repo == name {
			return ref, nil
		}
	}
	return "", errors.New("gadget " + name + " is not embedded")
}
