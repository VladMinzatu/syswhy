package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/VladMinzatu/syswhy/internal/adapters/demo"
	"github.com/VladMinzatu/syswhy/internal/adapters/host"
	"github.com/VladMinzatu/syswhy/internal/adapters/ig"
	"github.com/VladMinzatu/syswhy/internal/adapters/tui"
	"github.com/VladMinzatu/syswhy/internal/app"
)

func main() {
	demoMode := flag.Bool("demo", false, "show synthetic events instead of tracing the host")
	logFile := flag.String("log", "", "write logs to this file")
	flag.Parse()

	if err := run(*demoMode, *logFile); err != nil {
		fmt.Fprintln(os.Stderr, "syswhy:", err)
		os.Exit(1)
	}
}

func run(demoMode bool, logFile string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Logs must not go to the terminal: they would corrupt the TUI. The
	// default logger is replaced too, for libraries that use it.
	log := slog.New(slog.DiscardHandler)
	if logFile != "" {
		f, err := os.Create(logFile)
		if err != nil {
			return err
		}
		defer f.Close()
		log = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	slog.SetDefault(log)

	var tracer tui.ExecTracer = ig.New(log)
	if demoMode {
		tracer = demo.ExecTracer{Interval: 300 * time.Millisecond}
	}
	check := app.NewHostCheck(host.New())
	_, err := tea.NewProgram(tui.New(ctx, check, tracer)).Run()
	return err
}
