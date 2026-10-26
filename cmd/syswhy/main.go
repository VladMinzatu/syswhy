package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/VladMinzatu/syswhy/internal/adapters/tui"
)

func main() {
	if _, err := tea.NewProgram(tui.New()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "syswhy:", err)
		os.Exit(1)
	}
}
