package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// UI theme colors
var (
	colorSky       = lipgloss.Color("#38BDF8") // Cyan / Twitter-like sky blue
	colorSkyDim    = lipgloss.Color("#0284C7")
	colorBorderDim = lipgloss.Color("#475569") // Slate-600
	colorTextDim   = lipgloss.Color("#94A3B8") // Slate-400
	colorWhite     = lipgloss.Color("#F8FAFC") // Slate-50
	colorTime      = lipgloss.Color("#38BDF8") // Accent time badge
	colorHeaderBg  = lipgloss.Color("#1E293B") // Slate-800
	colorSuccess   = lipgloss.Color("#34D399") // Emerald-400
)

// Styles
var (
	// Header styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorHeaderBg).
			Padding(0, 1)

	headerBrand = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSky).
			Background(colorHeaderBg)

	headerCountBadge = lipgloss.NewStyle().
				Foreground(colorSky).
				Bold(true)

	// Editor styles
	editorActiveBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSky).
			Padding(0, 1)

	editorInactiveBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorderDim).
				Padding(0, 1)

	editorPromptStyle = lipgloss.NewStyle().
				Foreground(colorSky).
				Bold(true)

	// Timeline styles
	timelineItemTime = lipgloss.NewStyle().
				Foreground(colorTime).
				Bold(true)

	timelineItemBullet = lipgloss.NewStyle().
				Foreground(colorBorderDim)

	timelineEmptyNotice = lipgloss.NewStyle().
				Foreground(colorTextDim).
				Italic(true).
				Padding(1, 2)

	// Footer styles
	footerStyle = lipgloss.NewStyle().
			Foreground(colorTextDim).
			Padding(0, 1)

	footerModeBadgeInput = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite).
				Background(colorSkyDim).
				Padding(0, 1)

	footerModeBadgeView = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorWhite).
				Background(lipgloss.Color("#6366F1")). // Indigo
				Padding(0, 1)

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(colorSky).
			Bold(true)
)

// RenderPostLine formats a single post for timeline viewport.
// HH:MM content with proper line wrapping respecting East Asian Width.
func RenderPostLine(p Post, width int) string {
	timeBadge := timelineItemTime.Render(p.TimeStr)
	prefix := timeBadge + " "
	prefixWidth := runewidth.StringWidth(p.TimeStr) + 1

	contentWidth := width - prefixWidth
	if contentWidth < 20 {
		contentWidth = 20
	}

	lines := strings.Split(p.Content, "\n")
	var formatted strings.Builder

	indent := strings.Repeat(" ", prefixWidth)

	for i, line := range lines {
		// Wrap line using lipgloss with awareness of width
		wrapped := lipgloss.NewStyle().Width(contentWidth).Render(line)
		wrappedLines := strings.Split(wrapped, "\n")

		for j, wl := range wrappedLines {
			if i == 0 && j == 0 {
				formatted.WriteString(prefix)
				formatted.WriteString(wl)
			} else {
				formatted.WriteString("\n")
				formatted.WriteString(indent)
				formatted.WriteString(wl)
			}
		}
	}

	return formatted.String()
}
