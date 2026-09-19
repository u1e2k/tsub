package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Version is the current version of tsub (injected at build time via -ldflags).
var Version = "dev"

func printHelp() {
	fmt.Printf("tsub - TUI micro-blogging tool for daily notes\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  tsub [options]\n\n")
	fmt.Printf("Options:\n")
	fmt.Printf("  -v, --version  Show version information\n")
	fmt.Printf("  -h, --help     Show this help message\n\n")
	fmt.Printf("Environment Variables:\n")
	fmt.Printf("  TSUB_VAULT_DIR Directory for daily markdown files (default: ~/vault/Daily)\n")
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version", "version":
			fmt.Printf("tsub version %s\n", Version)
			return
		case "-h", "--help", "help":
			printHelp()
			return
		}
	}

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
