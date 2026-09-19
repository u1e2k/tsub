package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// Explicitly set dark background to prevent lipgloss/termenv from sending
	// terminal capability query sequences (OSC 11 / CSI 6 n) before AltScreen.
	lipgloss.SetHasDarkBackground(true)

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

	// Disable bracketed paste to prevent sending unsupported \x1b[?2004h sequences
	// on framebuffer/console and uim-fep environments.
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithoutBracketedPaste(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running tsub: %v\n", err)
		os.Exit(1)
	}
}
