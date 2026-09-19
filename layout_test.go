package main

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestCursorWriter verifies that cursorWriter appends the desired hardware cursor
// positioning ANSI escape sequence after Bubble Tea's output, ensuring the cursor
// is parked in the bottom margin for uim-fep (visible in ModeInput, hidden in ModeView).
func TestCursorWriter(t *testing.T) {
	var buf bytes.Buffer
	cw := &cursorWriter{out: &buf}

	// Test ModeInput (visible, parked in bottom margin)
	SetCursorPosition(28, 1, true)
	_, _ = cw.Write([]byte("rendered frame\x1b[27;H"))
	got := buf.String()
	expectedSeq := "\x1b[?25h\x1b[28;1H"
	if !strings.HasSuffix(got, expectedSeq) {
		t.Errorf("expected output to end with %q, got %q", expectedSeq, got)
	}

	// Test ModeView (hidden, parked in bottom margin)
	buf.Reset()
	SetCursorPosition(28, 1, false)
	_, _ = cw.Write([]byte("rendered frame\x1b[27;H"))
	gotView := buf.String()
	expectedViewSeq := "\x1b[?25l\x1b[28;1H"
	if !strings.HasSuffix(gotView, expectedViewSeq) {
		t.Errorf("expected view mode output to end with %q, got %q", expectedViewSeq, gotView)
	}
}

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
	lines := strings.Split(view, "\n")
	if renderedHeight != expectedHeight {
		t.Errorf("expected rendered view height to be %d, got %d", expectedHeight, renderedHeight)
	}
	if len(lines) != expectedHeight {
		t.Errorf("expected %d lines, got %d", expectedHeight, len(lines))
	}
	firstLine := lines[0]
	if !strings.Contains(firstLine, "Enter:") {
		t.Errorf("expected header with 'Enter:' on the first line, got: %q", firstLine)
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

// TestHardwareCursorPositioning verifies that ANSI cursor position sequences
// are emitted in View() to park the hardware cursor on the dedicated preedit line
// in the bottom margin (Row m.height + 1) in ModeInput, and park/hide the cursor in ModeView.
func TestHardwareCursorPositioning(t *testing.T) {
	storage := NewStorage()
	m, err := InitialModel(storage)
	if err != nil {
		t.Fatalf("failed to init model: %v", err)
	}

	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updatedModel.(Model)

	// In Input mode: cursor should be visible and parked on the preedit row in bottom margin (Row 28, Col 1)
	m.mode = ModeInput
	viewInputEmpty := m.View()
	expectedEmptySeq := "\x1b[?25h\x1b[28;1H"
	if !strings.HasSuffix(viewInputEmpty, expectedEmptySeq) {
		t.Errorf("expected view to end with %q, got suffix %q", expectedEmptySeq, viewInputEmpty[len(viewInputEmpty)-len(expectedEmptySeq):])
	}

	// In View mode: cursor hidden and parked on margin line below footer
	m.mode = ModeView
	viewModeView := m.View()
	expectedParkSeq := "\x1b[?25l\x1b[28;1H"
	if !strings.HasSuffix(viewModeView, expectedParkSeq) {
		t.Errorf("expected view mode to end with %q, got suffix %q", expectedParkSeq, viewModeView[len(viewModeView)-len(expectedParkSeq):])
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
		if strings.HasPrefix(line, "               ") && (strings.Contains(line, "┎") || strings.Contains(line, "┃")) {
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
