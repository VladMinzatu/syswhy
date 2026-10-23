package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/VladMinzatu/syswhy/internal/adapters/demo"
	"github.com/VladMinzatu/syswhy/internal/adapters/host"
	"github.com/VladMinzatu/syswhy/internal/adapters/tui"
	"github.com/VladMinzatu/syswhy/internal/app"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	check := app.NewHostCheck(host.New())
	tracer := demo.ExecTracer{Interval: 300 * time.Millisecond}
	if _, err := tea.NewProgram(tui.New(ctx, check, tracer)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "syswhy:", err)
		os.Exit(1)
	}
}
