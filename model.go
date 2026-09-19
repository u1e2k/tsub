package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
	ta.Prompt = "❯ "
	ta.FocusedStyle.Prompt = editorPromptStyle
	ta.BlurredStyle.Prompt = editorPromptDimStyle
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
	}

	return m, nil
}

// Init starts bubbletea loop.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
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
		m.height = msg.Height
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

	// Slit width: 85% to 90% of screen width, bounded sensibly
	slitWidth := int(float64(m.width) * 0.88)
	if slitWidth < 30 {
		slitWidth = m.width - 2
	}
	if slitWidth > 120 {
		slitWidth = 120
	}
	if slitWidth <= 0 {
		slitWidth = 20
	}

	// Textarea inner width (accounting for border and padding = 4 chars)
	taInnerWidth := slitWidth - 4
	if taInnerWidth < 10 {
		taInnerWidth = 10
	}
	m.textarea.SetWidth(taInnerWidth)

	// Vertical layout:
	// Header: ~2 lines (padding + content)
	// Slit Editor: ~5 lines (title 1 line + border & content 4 lines)
	// Spacers / margins: ~2 lines
	// Footer: ~2 lines
	fixedHeight := 2 + 5 + 2 + 2
	vpHeight := m.height - fixedHeight
	if vpHeight < 3 {
		vpHeight = 3
	}

	m.viewport.Width = slitWidth
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

	// 1. Header
	dateStr := m.date.Format("2006-01-02 (Mon)")
	brand := headerBrand.Render(" tsub ")
	headerInfo := fmt.Sprintf(" %s | 投稿数: %s ", dateStr, headerCountBadge.Render(fmt.Sprintf("%d", len(m.posts))))
	headerBar := headerStyle.Render(brand + headerInfo)

	// 2. Slit Editor Box
	var editorTitle string
	editorBoxStyle := editorActiveBox
	if m.mode == ModeInput {
		editorTitle = editorTitleActive.Render("💭 いまどうしてる？") + " " + editorTitleDim.Render("(Enter: 投稿 / Esc: 閲覧)")
	} else {
		editorBoxStyle = editorInactiveBox
		editorTitle = editorTitleDim.Render("💭 いまどうしてる？ (i: 投稿入力)")
	}
	editorBox := editorBoxStyle.Width(m.viewport.Width).Render(m.textarea.View())
	editorRendered := lipgloss.JoinVertical(
		lipgloss.Left,
		editorTitle,
		editorBox,
	)

	// 3. Timeline Viewport
	timelineRendered := m.viewport.View()

	// 4. Footer
	var modeBadge string
	var keyHelp string
	if m.mode == ModeInput {
		modeBadge = footerModeBadgeInput.Render("入力")
		keyHelp = fmt.Sprintf(" %s %s  %s %s  %s %s",
			footerKeyStyle.Render("Enter:"), "投稿",
			footerKeyStyle.Render("Esc:"), "閲覧モード",
			footerKeyStyle.Render("Ctrl+C/Q:"), "終了",
		)
	} else {
		modeBadge = footerModeBadgeView.Render("閲覧")
		keyHelp = fmt.Sprintf(" %s %s  %s %s  %s %s  %s %s",
			footerKeyStyle.Render("j/k:"), "スクロール",
			footerKeyStyle.Render("i:"), "入力へ戻る",
			footerKeyStyle.Render("Esc:"), "入力へ戻る",
			footerKeyStyle.Render("Ctrl+C/Q:"), "終了",
		)
	}
	footerBar := footerStyle.Render(modeBadge + " " + keyHelp)

	// Compose layout vertically
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		headerBar,
		"",
		editorRendered,
		"",
		timelineRendered,
		"",
		footerBar,
	)

	// Horizontally center the layout across the terminal width
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, content)
}
