package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestUimFepMarginAndLayoutHeight verifies that WindowSizeMsg allocates msg.Height - 2
// to leave the bottom 2 rows free for uim-fep (Anthy) status and preedit lines.
func TestUimFepMarginAndLayoutHeight(t *testing.T) {
	storage := NewStorage()
	m, err := InitialModel(storage)
	if err != nil {
		t.Fatalf("failed to init model: %v", err)
	}

	terminalHeight := 30
	terminalWidth := 100
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: terminalWidth, Height: terminalHeight})
	m = updatedModel.(Model)

	expectedHeight := terminalHeight - 2
	if m.height != expectedHeight {
		t.Fatalf("expected m.height = %d (msg.Height - 2), got %d", expectedHeight, m.height)
	}

	view := m.View()
	renderedHeight := lipgloss.Height(view)
	if renderedHeight != expectedHeight {
		t.Errorf("expected rendered view height to be %d, got %d", expectedHeight, renderedHeight)
	}

	// Verify footer is at the bottom of the rendered view (row H - 2, i.e. 3rd line from bottom)
	lines := strings.Split(view, "\n")
	if len(lines) != expectedHeight {
		t.Errorf("expected %d lines, got %d", expectedHeight, len(lines))
	}
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, "Enter:") {
		t.Errorf("expected footer with 'Enter:' on the last rendered line, got: %q", lastLine)
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

// TestFullWidthAndNoOverflow verifies that the layout utilizes the full terminal width (横幅目一杯)
// and that all components (header, editor box with border/padding, timeline, footer)
// fit precisely within the window width without right-edge cut-off or overflow.
func TestFullWidthAndNoOverflow(t *testing.T) {
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

	if m.viewport.Width != terminalWidth {
		t.Errorf("expected viewport width to be %d, got %d", terminalWidth, m.viewport.Width)
	}

	// Textarea inner width is terminalWidth - editorActiveBox.GetHorizontalFrameSize() (4)
	expectedTaWidth := terminalWidth - editorActiveBox.GetHorizontalFrameSize()
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

	// Verify full width lines exist (header, card border, footer all reach terminalWidth)
	fullWidthCount := 0
	for _, line := range lines {
		if lipgloss.Width(line) == terminalWidth {
			fullWidthCount++
		}
	}
	if fullWidthCount == 0 {
		t.Errorf("expected layout to contain lines of full terminal width %d", terminalWidth)
	}

	// Case 2: Standard terminal (80 cols)
	updatedModel80, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m80 := updatedModel80.(Model)
	if m80.width != 80 || m80.viewport.Width != 80 {
		t.Errorf("expected 80 cols full width, got m.width=%d, m.viewport.Width=%d", m80.width, m80.viewport.Width)
	}
	view80 := m80.View()
	for i, line := range strings.Split(view80, "\n") {
		w := lipgloss.Width(line)
		if w > 80 {
			t.Errorf("80-col mode: line %d has width %d, exceeding 80: %q", i, w, line)
		}
	}
}
