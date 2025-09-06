package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type helpModel struct {
	styles Styles
	width  int
	height int
}

// helpCloseMsg is sent when the help window is closed
type helpCloseMsg struct{}

// NewHelpForm creates a new help form model
func NewHelpForm(styles Styles, width, height int) *helpModel {
	return &helpModel{
		styles: styles,
		width:  width,
		height: height,
	}
}

func (m *helpModel) Init() tea.Cmd {
	return nil
}

func (m *helpModel) Update(msg tea.Msg) (*helpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "h", "enter", "ctrl+c":
			return m, func() tea.Msg { return helpCloseMsg{} }
		}
	}
	return m, nil
}

func (m *helpModel) View() string {
	// Title
	title := m.styles.Header.Render("📖 SSHM - Help & Commands")

	// Create horizontal sections with compact layout
	line1 := lipgloss.JoinHorizontal(lipgloss.Center,
		m.styles.FocusedLabel.Render("🧭 ↑/↓/j/k"),
		"  ",
		m.styles.HelpText.Render("navigate"),
		"    ",
		m.styles.FocusedLabel.Render("⏎"),
		"  ",
		m.styles.HelpText.Render("connect"),
		"    ",
		m.styles.FocusedLabel.Render("a/e/d"),
		"  ",
		m.styles.HelpText.Render("add/edit/delete"),
	)

	line2 := lipgloss.JoinHorizontal(lipgloss.Center,
		m.styles.FocusedLabel.Render("Tab"),
		"  ",
		m.styles.HelpText.Render("switch focus"),
		"    ",
		m.styles.FocusedLabel.Render("p"),
		"  ",
		m.styles.HelpText.Render("ping all"),
		"    ",
		m.styles.FocusedLabel.Render("f"),
		"  ",
		m.styles.HelpText.Render("port forward"),
		"    ",
		m.styles.FocusedLabel.Render("s/r/n"),
		"  ",
		m.styles.HelpText.Render("sort modes"),
	)

	line3 := lipgloss.JoinHorizontal(lipgloss.Center,
		m.styles.FocusedLabel.Render("/"),
		"  ",
		m.styles.HelpText.Render("search"),
		"    ",
		m.styles.FocusedLabel.Render("h"),
		"  ",
		m.styles.HelpText.Render("help"),
		"    ",
		m.styles.FocusedLabel.Render("q/ESC"),
		"  ",
		m.styles.HelpText.Render("quit"),
	)

	// Create the main content
	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		line1,
		"",
		line2,
		"",
		line3,
		"",
		m.styles.HelpText.Render("Press ESC, h, q or Enter to close"),
	)

	// Center the help window
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.styles.FormContainer.Render(content),
	)
}
