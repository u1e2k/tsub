package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	cursorMutex         sync.Mutex
	globalCursorY       int
	globalCursorX       int
	globalCursorVisible bool
)

// SetCursorPosition sets the target hardware cursor position and visibility.
func SetCursorPosition(y, x int, visible bool) {
	cursorMutex.Lock()
	defer cursorMutex.Unlock()
	globalCursorY = y
	globalCursorX = x
	globalCursorVisible = visible
}

// GetCursorPosition returns the target hardware cursor position and visibility.
func GetCursorPosition() (y, x int, visible bool) {
	cursorMutex.Lock()
	defer cursorMutex.Unlock()
	return globalCursorY, globalCursorX, globalCursorVisible
}

// Mode represents the current interaction mode.
type Mode int

const (
	ModeInput Mode = iota
	ModeView
)

// Model represents the tsub TUI state.
type Model struct {
	storage    *Storage
	posts      []Post
	mode       Mode
	textarea   textarea.Model
	viewport   viewport.Model
	width      int
	height     int
	date       time.Time
	statusMsg  string
	err        error
	ready      bool
}

// InitialModel initializes the application model.
func InitialModel(storage *Storage) (Model, error) {
	today := time.Now()
	posts, err := storage.LoadTodayPosts(today)
	if err != nil {
		return Model{}, fmt.Errorf("failed to load today's posts: %w", err)
	}

	ta := textarea.New()
	ta.Placeholder = ""
	ta.Focus()
	ta.Prompt = ""
	ta.FocusedStyle.Prompt = lipgloss.NewStyle()
	ta.BlurredStyle.Prompt = lipgloss.NewStyle()
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.SetHeight(2)

	vp := viewport.New(80, 10)

	m := Model{
		storage:  storage,
		posts:    posts,
		mode:     ModeInput,
		textarea: ta,
		viewport: vp,
		date:     today,
		width:    80,
		height:   21, // 24 - 3 default bottom margin
		ready:    true,
	}
	m.resizeComponents()

	return m, nil
}

// Init starts bubbletea loop.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.WindowSize())
}

// Update handles user inputs and window resize events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlQ:
			return m, tea.Quit

		case tea.KeyEsc:
			// Toggle between Input and View mode
			if m.mode == ModeInput {
				m.mode = ModeView
				m.textarea.Blur()
			} else {
				m.mode = ModeInput
				m.textarea.Focus()
			}
			return m, nil

		case tea.KeyEnter:
			// If in Input mode, Enter submits the post
			if m.mode == ModeInput {
				val := strings.TrimSpace(m.textarea.Value())
				if val != "" {
					now := time.Now()
					newPost, err := m.storage.AppendPost(now, val)
					if err != nil {
						m.err = err
					} else {
						// Prepend to posts (newest first)
						m.posts = append([]Post{newPost}, m.posts...)
						m.textarea.Reset()
						m.updateViewportContent()
						m.viewport.GotoTop()
						m.statusMsg = "投稿しました"
					}
				}
				return m, nil
			}

		case tea.KeyRunes:
			// In View mode, check for navigation / mode switch keys
			if m.mode == ModeView {
				switch string(msg.Runes) {
				case "i":
					m.mode = ModeInput
					m.textarea.Focus()
					return m, nil
				case "j":
					m.viewport.LineDown(1)
					return m, nil
				case "k":
					m.viewport.LineUp(1)
					return m, nil
				case "d":
					m.viewport.HalfViewDown()
					return m, nil
				case "u":
					m.viewport.HalfViewUp()
					return m, nil
				case "g":
					m.viewport.GotoTop()
					return m, nil
				case "G":
					m.viewport.GotoBottom()
					return m, nil
				}
			}
		}

		// In View mode, pass up/down/pgup/pgdown to viewport
		if m.mode == ModeView {
			switch msg.Type {
			case tea.KeyUp:
				m.viewport.LineUp(1)
				return m, nil
			case tea.KeyDown:
				m.viewport.LineDown(1)
				return m, nil
			case tea.KeyPgUp:
				m.viewport.HalfViewUp()
				return m, nil
			case tea.KeyPgDown:
				m.viewport.HalfViewDown()
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		if m.width < 0 {
			m.width = 0
		}
		// Reserve bottom lines for uim-fep (status line + preedit line + safe margin).
		// Default to 3 lines (configurable via TSUB_BOTTOM_MARGIN env var).
		bottomMargin := 3
		if val := os.Getenv("TSUB_BOTTOM_MARGIN"); val != "" {
			if n, err := strconv.Atoi(val); err == nil && n >= 0 {
				bottomMargin = n
			}
		}
		m.height = msg.Height - bottomMargin
		if m.height < 5 {
			m.height = 5
		}
		m.resizeComponents()
		m.ready = true
		return m, nil
	}

	if m.mode == ModeInput {
		m.textarea, tiCmd = m.textarea.Update(msg)
		cmds = append(cmds, tiCmd)
	} else {
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

// resizeComponents dynamically recalculates layout sizes based on terminal window.
func (m *Model) resizeComponents() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	// Input card width: ~70% of terminal width (clamped between 40 and m.width)
	cardWidth := int(float64(m.width) * 0.7)
	if cardWidth < 40 {
		cardWidth = 40
	}
	if cardWidth > m.width {
		cardWidth = m.width
	}

	frameSize := editorActiveBox.GetHorizontalFrameSize()
	taInnerWidth := cardWidth - frameSize
	if taInnerWidth < 10 {
		taInnerWidth = 10
	}
	m.textarea.SetWidth(taInnerWidth)

	// Adapt textarea height based on available screen height:
	// Use 1 line for compact screens (m.height < 14), 2 lines for normal screens.
	taHeight := 2
	if m.height < 14 {
		taHeight = 1
	}
	m.textarea.SetHeight(taHeight)

	cardLines := taHeight + 2 // border top + content + border bottom

	spacers := 0
	if m.height >= 16 {
		spacers++
	}
	if m.height >= 13 {
		spacers++
	}
	if m.height >= 15 {
		spacers++
	}

	fixedLines := 1 /* header */ + cardLines + spacers
	vpHeight := m.height - fixedLines
	if vpHeight < 1 {
		vpHeight = 1
	}

	// Timeline viewport spans full terminal width
	m.viewport.Width = m.width
	m.viewport.Height = vpHeight
	m.updateViewportContent()
}

// updateViewportContent formats the timeline entries and sets viewport content.
func (m *Model) updateViewportContent() {
	if len(m.posts) == 0 {
		m.viewport.SetContent(timelineEmptyNotice.Render("まだ投稿はありません。今日の思考を吐き出してみましょう。"))
		return
	}

	var sb strings.Builder
	for i, post := range m.posts {
		line := RenderPostLine(post, m.viewport.Width)
		sb.WriteString(line)
		if i < len(m.posts)-1 {
			sb.WriteString("\n\n") // Separator between posts
		}
	}
	m.viewport.SetContent(sb.String())
}

// View renders the entire TUI screen.
func (m Model) View() string {
	if !m.ready {
		return "起動中..."
	}

	// 1. Header (Full Width) with Mode & Key Help integrated
	dateStr := m.date.Format("2006-01-02 (Mon)")
	brand := headerBrand.Render(" tsub ")
	headerInfo := fmt.Sprintf(" %s | %s件 ", dateStr, headerCountBadge.Render(fmt.Sprintf("%d", len(m.posts))))

	var modeBadge string
	var keyHelp string
	if m.mode == ModeInput {
		modeBadge = footerModeBadgeInput.Render("入力") + " " + headerImeBadge.Render("Anあ")
		keyHelp = fmt.Sprintf(" %s %s  %s %s  %s %s",
			footerKeyStyle.Render("Enter:"), "投稿",
			footerKeyStyle.Render("Esc:"), "閲覧",
			footerKeyStyle.Render("Ctrl+C:"), "終了",
		)
	} else {
		modeBadge = footerModeBadgeView.Render("閲覧")
		keyHelp = fmt.Sprintf(" %s %s  %s %s  %s %s",
			footerKeyStyle.Render("j/k:"), "移動",
			footerKeyStyle.Render("i/Esc:"), "入力",
			footerKeyStyle.Render("Ctrl+C:"), "終了",
		)
	}

	headerFrame := headerStyle.GetHorizontalFrameSize()
	headerInnerWidth := m.width - headerFrame
	if headerInnerWidth < 0 {
		headerInnerWidth = 0
	}

	leftContent := brand + headerInfo
	rightContent := modeBadge + " " + keyHelp
	leftWidth := lipgloss.Width(leftContent)
	rightWidth := lipgloss.Width(rightContent)
	badgeWidth := lipgloss.Width(modeBadge)

	var headerContent string
	if leftWidth+rightWidth <= headerInnerWidth {
		gap := headerInnerWidth - leftWidth - rightWidth
		headerContent = leftContent + strings.Repeat(" ", gap) + rightContent
	} else if leftWidth+badgeWidth <= headerInnerWidth {
		gap := headerInnerWidth - leftWidth - badgeWidth
		headerContent = leftContent + strings.Repeat(" ", gap) + modeBadge
	} else {
		headerContent = leftContent
	}
	headerBar := headerStyle.Width(m.width).Render(headerContent)

	// 2. Editor Box (Input Card - ~70% width, centered)
	cardWidth := int(float64(m.width) * 0.7)
	if cardWidth < 40 {
		cardWidth = 40
	}
	if cardWidth > m.width {
		cardWidth = m.width
	}

	editorBoxStyle := editorActiveBox
	if m.mode != ModeInput {
		editorBoxStyle = editorInactiveBox
	}

	editorFrame := editorBoxStyle.GetHorizontalFrameSize()
	editorInnerWidth := cardWidth - editorFrame
	if editorInnerWidth < 10 {
		editorInnerWidth = 10
	}
	cardView := editorBoxStyle.Width(editorInnerWidth).Render(m.textarea.View())
	editorRendered := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, cardView)

	// 3. Timeline Viewport (Full Width)
	timelineRendered := m.viewport.View()

	// Compose layout vertically with adaptive spacers based on height.
	var sections []string
	sections = append(sections, headerBar)
	if m.height >= 16 {
		sections = append(sections, "")
	}
	sections = append(sections, editorRendered)
	if m.height >= 13 {
		sections = append(sections, "")
	}
	sections = append(sections, timelineRendered)
	if m.height >= 15 {
		sections = append(sections, "")
	}

	fullView := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Park hardware cursor on the dedicated preedit line in the bottom margin (Row m.height + 1).
	// This lets uim-fep (Anthy) display the in-progress preedit text cleanly at the bottom,
	// completely avoiding collisions with any UI element!
	parkRow := m.height + 1
	if m.mode == ModeInput {
		SetCursorPosition(parkRow, 1, true)
		return fullView + fmt.Sprintf("\x1b[?25h\x1b[%d;1H", parkRow)
	}

	SetCursorPosition(parkRow, 1, false)
	return fullView + fmt.Sprintf("\x1b[?25l\x1b[%d;1H", parkRow)
}
