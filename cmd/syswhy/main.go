package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/VladMinzatu/syswhy/internal/adapters/host"
	"github.com/VladMinzatu/syswhy/internal/adapters/tui"
	"github.com/VladMinzatu/syswhy/internal/app"
)

func main() {
	check := app.NewHostCheck(host.New())
	if _, err := tea.NewProgram(tui.New(check)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "syswhy:", err)
		os.Exit(1)
	}
}
