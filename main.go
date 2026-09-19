package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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
	fmt.Printf("  -u, --update   Update tsub to the latest version\n")
	fmt.Printf("  -v, --version  Show version information\n")
	fmt.Printf("  -h, --help     Show this help message\n\n")
	fmt.Printf("Environment Variables:\n")
	fmt.Printf("  TSUB_VAULT_DIR Directory for daily markdown files (default: ~/vault/Daily)\n")
}

func runUpdate() {
	fmt.Println("tsub を最新バージョンにアップデートしています...")
	shell := "bash"
	if _, err := exec.LookPath("bash"); err != nil {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-c", "curl -fsSL https://raw.githubusercontent.com/u1e2k/tsub/main/install.sh | bash")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "アップデート中にエラーが発生しました: %v\n", err)
		os.Exit(1)
	}
}

// cursorWriter wraps stdout to reposition the hardware cursor at the end of
// every Bubble Tea flush, neutralizing Bubble Tea's default trailing ansi.CursorPosition
// and accurately positioning the cursor inside the input box for uim-fep (Anthy).
type cursorWriter struct {
	out io.Writer
}

func (w *cursorWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write(p)
	if err != nil {
		return n, err
	}
	cy, cx, visible := GetCursorPosition()
	if cy > 0 && cx > 0 {
		var seq string
		if visible {
			seq = fmt.Sprintf("\x1b[?25h\x1b[%d;%dH", cy, cx)
		} else {
			seq = fmt.Sprintf("\x1b[?25l\x1b[%d;%dH", cy, cx)
		}
		_, _ = w.out.Write([]byte(seq))
	}
	return n, nil
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-u", "--update", "update":
			runUpdate()
			return
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
	// Use cursorWriter with tea.WithOutput to position the hardware cursor after every render.
	cw := &cursorWriter{out: os.Stdout}
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithoutBracketedPaste(),
		tea.WithOutput(cw),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running tsub: %v\n", err)
		os.Exit(1)
	}
}
