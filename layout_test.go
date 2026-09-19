package main

import (
	"strings"
	"testing"
)

// TestCursorPositionSequence verifies that View() appends the correct hardware
// cursor position ANSI escape sequence in Input mode and hides the cursor in View mode,
// resolving uim-fep (Anthy) preedit rendering issues.
func TestCursorPositionSequence(t *testing.T) {
	storage := NewStorage()
	m, err := InitialModel(storage)
	if err != nil {
		t.Fatalf("failed to init model: %v", err)
	}

	m.width = 100
	m.height = 30
	m.resizeComponents()
	m.ready = true

	// ModeInput with empty text
	viewInput := m.View()
	// Should end with ANSI cursor position sequence: \x1b[?25h\x1b[5;10H
	expectedSeq := "\x1b[?25h\x1b[5;10H"
	if !strings.HasSuffix(viewInput, expectedSeq) {
		t.Errorf("expected view to end with %q, but got suffix %q", expectedSeq, viewInput[len(viewInput)-len(expectedSeq):])
	}

	// ModeInput with text
	m.textarea.SetValue("こんにちは")
	viewInputText := m.View()
	// "こんにちは" is 10 columns wide, so 10 + 10 = 20
	expectedSeqText := "\x1b[?25h\x1b[5;20H"
	if !strings.HasSuffix(viewInputText, expectedSeqText) {
		t.Errorf("expected view to end with %q, got suffix %q", expectedSeqText, viewInputText[len(viewInputText)-len(expectedSeqText):])
	}

	// ModeInput with mixed fullwidth and halfwidth text: "aあbい" -> 1 + 2 + 1 + 2 = 6 cols
	m.textarea.SetValue("aあbい")
	viewMixed := m.View()
	// base col 10 + 6 = 16
	expectedMixed := "\x1b[?25h\x1b[5;16H"
	if !strings.HasSuffix(viewMixed, expectedMixed) {
		t.Errorf("expected view to end with %q, got suffix %q", expectedMixed, viewMixed[len(viewMixed)-len(expectedMixed):])
	}

	// ModeView
	m.mode = ModeView
	viewModeView := m.View()
	expectedSeqView := "\x1b[?25l"
	if !strings.HasSuffix(viewModeView, expectedSeqView) {
		t.Errorf("expected view to end with %q, got suffix %q", expectedSeqView, viewModeView[len(viewModeView)-len(expectedSeqView):])
	}

	// Narrow width test (m.width <= boxWidth)
	m.mode = ModeInput
	m.width = 40
	m.resizeComponents()
	viewNarrow := m.View()
	if !strings.Contains(viewNarrow, "\x1b[?25h\x1b[") {
		t.Errorf("narrow view should contain cursor sequence, got %q", viewNarrow)
	}
}

