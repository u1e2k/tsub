package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	storage := NewStorage()

	// Ensure today's daily file exists
	if _, err := storage.EnsureDailyFile(time.Now()); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing daily file: %v\n", err)
		os.Exit(1)
	}

	model, err := InitialModel(storage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting tsub: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running tsub: %v\n", err)
		os.Exit(1)
	}
}
