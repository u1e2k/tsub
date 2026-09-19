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

// TestPlainPrompt verifies that textarea.Prompt is a plain single space with no escape codes.
func TestPlainPrompt(t *testing.T) {
	storage := NewStorage()
	m, err := InitialModel(storage)
	if err != nil {
		t.Fatalf("failed to init model: %v", err)
	}

	if m.textarea.Prompt != " " {
		t.Errorf("expected prompt to be \" \", got %q", m.textarea.Prompt)
	}

	// Render prompt style to verify it has no ANSI escape codes
	renderedFocused := m.textarea.FocusedStyle.Prompt.Render(m.textarea.Prompt)
	if strings.Contains(renderedFocused, "\x1b") {
		t.Errorf("focused prompt should not contain ANSI escape codes, got %q", renderedFocused)
	}
	renderedBlurred := m.textarea.BlurredStyle.Prompt.Render(m.textarea.Prompt)
	if strings.Contains(renderedBlurred, "\x1b") {
		t.Errorf("blurred prompt should not contain ANSI escape codes, got %q", renderedBlurred)
	}
}
