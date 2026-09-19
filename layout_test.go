package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestUimFepMarginAndLayoutHeight verifies that WindowSizeMsg allocates msg.Height - 3
// to leave the bottom rows free for uim-fep (Anthy) status and preedit lines,
// and that renderedHeight strictly equals expectedHeight on both standard and compact screens.
func TestUimFepMarginAndLayoutHeight(t *testing.T) {
	storage := NewStorage()
	m, err := InitialModel(storage)
	if err != nil {
		t.Fatalf("failed to init model: %v", err)
	}

	// Case 1: Standard screen (30 rows)
	terminalHeight := 30
	terminalWidth := 100
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: terminalWidth, Height: terminalHeight})
	m = updatedModel.(Model)

	expectedHeight := terminalHeight - 3
	if m.height != expectedHeight {
		t.Fatalf("expected m.height = %d (msg.Height - 3), got %d", expectedHeight, m.height)
	}

	view := m.View()
	renderedHeight := lipgloss.Height(view)
	if renderedHeight != expectedHeight {
		t.Errorf("expected rendered view height to be %d, got %d", expectedHeight, renderedHeight)
	}

	lines := strings.Split(view, "\n")
	if len(lines) != expectedHeight {
		t.Errorf("expected %d lines, got %d", expectedHeight, len(lines))
	}
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, "Enter:") {
		t.Errorf("expected footer with 'Enter:' on the last rendered line, got: %q", lastLine)
	}

	// Case 2: Compact screen (13 rows, e.g. Raspberry Pi handheld LCD)
	compactModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 13})
	mCompact := compactModel.(Model)

	expectedCompactHeight := 13 - 3 // 10 lines
	if mCompact.height != expectedCompactHeight {
		t.Fatalf("expected compact height %d, got %d", expectedCompactHeight, mCompact.height)
	}

	compactView := mCompact.View()
	compactRenderedHeight := lipgloss.Height(compactView)
	if compactRenderedHeight != expectedCompactHeight {
		t.Errorf("expected compact rendered height %d, got %d", expectedCompactHeight, compactRenderedHeight)
	}
}

// TestNoHardwareCursorEscapeSequences verifies that ANSI cursor position sequences
// are completely removed from View() in both Input and View modes.
func TestNoHardwareCursorEscapeSequences(t *testing.T) {
	storage := NewStorage()
	m, err := InitialModel(storage)
	if err != nil {
		t.Fatalf("failed to init model: %v", err)
	}

	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updatedModel.(Model)

	// In Input mode
	m.mode = ModeInput
	m.textarea.SetValue("テストメッセージ")
	viewInput := m.View()
	if strings.Contains(viewInput, "\x1b[?25h") || strings.Contains(viewInput, "\x1b[?25l") {
		t.Errorf("view should not contain cursor visibility escape sequences")
	}
	if strings.Contains(viewInput, ";") && strings.Contains(viewInput, "H") {
		// Check for cursor movement sequences like \x1b[...H
		for _, line := range strings.Split(viewInput, "\n") {
			if strings.Contains(line, "\x1b[") && strings.HasSuffix(strings.TrimSpace(line), "H") {
				t.Errorf("view should not contain cursor positioning escape sequences: %q", line)
			}
		}
	}

	// In View mode
	m.mode = ModeView
	viewModeView := m.View()
	if strings.HasSuffix(viewModeView, "\x1b[?25l") {
		t.Errorf("view mode should not append \\x1b[?25l")
	}
}

// TestEmptyPrompt verifies that textarea.Prompt is empty (no unwanted "> " or extra prompt characters).
func TestEmptyPrompt(t *testing.T) {
	storage := NewStorage()
	m, err := InitialModel(storage)
	if err != nil {
		t.Fatalf("failed to init model: %v", err)
	}

	if m.textarea.Prompt != "" {
		t.Errorf("expected prompt to be \"\", got %q", m.textarea.Prompt)
	}
}

// TestFullWidthAndCenteredCard verifies that header, timeline, and footer span the full terminal width (横幅目一杯),
// while the editor card is sized to ~70% and centered, with no right-edge cut-off or overflow.
func TestFullWidthAndCenteredCard(t *testing.T) {
	storage := NewStorage()
	m, err := InitialModel(storage)
	if err != nil {
		t.Fatalf("failed to init model: %v", err)
	}

	// Case 1: 100 cols
	terminalWidth := 100
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: terminalWidth, Height: 30})
	m = updatedModel.(Model)

	if m.width != terminalWidth {
		t.Errorf("expected width to be %d, got %d", terminalWidth, m.width)
	}

	// Timeline viewport spans full width
	if m.viewport.Width != terminalWidth {
		t.Errorf("expected timeline viewport width to be %d, got %d", terminalWidth, m.viewport.Width)
	}

	// Card width is 70% of 100 = 70 cols
	cardWidth := 70
	expectedTaWidth := cardWidth - editorActiveBox.GetHorizontalFrameSize() // 70 - 4 = 66
	if m.textarea.Width() != expectedTaWidth {
		t.Errorf("expected textarea text width to be %d, got %d", expectedTaWidth, m.textarea.Width())
	}

	// Verify no line in the rendered view exceeds terminalWidth
	view := m.View()
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w > terminalWidth {
			t.Errorf("line %d has width %d, exceeding terminal width %d: %q", i, w, terminalWidth, line)
		}
	}

	// Verify full width lines exist (header and footer reach terminalWidth)
	fullWidthCount := 0
	for _, line := range lines {
		if lipgloss.Width(line) == terminalWidth {
			fullWidthCount++
		}
	}
	if fullWidthCount == 0 {
		t.Errorf("expected layout to contain lines of full terminal width %d", terminalWidth)
	}

	// Verify centered card: lines containing the editor card border have leading margins (15 spaces)
	hasCenteredCard := false
	for _, line := range lines {
		if strings.HasPrefix(line, "               ╭") || strings.HasPrefix(line, "               │") || strings.HasPrefix(line, "               ┃") || strings.HasPrefix(line, "               ┎") {
			hasCenteredCard = true
			break
		}
	}
	if !hasCenteredCard {
		t.Errorf("expected editor card to be centered with 15 leading spaces on 100-col terminal")
	}

	// Case 2: Standard terminal (80 cols)
	updatedModel80, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m80 := updatedModel80.(Model)
	if m80.width != 80 || m80.viewport.Width != 80 {
		t.Errorf("expected 80 cols full width for timeline, got m.width=%d, m.viewport.Width=%d", m80.width, m80.viewport.Width)
	}
	view80 := m80.View()
	for i, line := range strings.Split(view80, "\n") {
		w := lipgloss.Width(line)
		if w > 80 {
			t.Errorf("80-col mode: line %d has width %d, exceeding 80: %q", i, w, line)
		}
	}
}
