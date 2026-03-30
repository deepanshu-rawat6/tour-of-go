// Command tui is a terminal dashboard for the distributed-scheduler.
// Uses Bubble Tea (Elm architecture) for the TUI.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"tour_of_go/projects/cli-tui/internal/client"
	"tour_of_go/projects/cli-tui/internal/tui"
)

func main() {
	schedulerURL := os.Getenv("SCHEDULER_URL")
	if schedulerURL == "" {
		schedulerURL = "http://localhost:8080"
	}

	sched := client.New(schedulerURL)
	model := tui.NewModel(sched)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
