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
	title := m.styles.Header.Render("📖 SSHM - Commands")

	// Create two columns of commands for better visual organization
	leftColumn := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.FocusedLabel.Render("Navigation & Connection"),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("⏎  "),
			m.styles.HelpText.Render("connect to selected host")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("i  "),
			m.styles.HelpText.Render("show host information")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("/  "),
			m.styles.HelpText.Render("search hosts")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("Tab "),
			m.styles.HelpText.Render("switch focus")),
		"",
		m.styles.FocusedLabel.Render("Host Management"),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("a  "),
			m.styles.HelpText.Render("add new host")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("e  "),
			m.styles.HelpText.Render("edit selected host")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("m  "),
			m.styles.HelpText.Render("move host to another config")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("d  "),
			m.styles.HelpText.Render("delete selected host")),
	)

	rightColumn := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.FocusedLabel.Render("Advanced Features"),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("p  "),
			m.styles.HelpText.Render("ping all hosts")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("f  "),
			m.styles.HelpText.Render("setup port forwarding")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("s  "),
			m.styles.HelpText.Render("cycle sort modes")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("n  "),
			m.styles.HelpText.Render("sort by name")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("r  "),
			m.styles.HelpText.Render("sort by recent connection")),
		"",
		m.styles.FocusedLabel.Render("System"),
		"",
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("h  "),
			m.styles.HelpText.Render("show this help")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("q  "),
			m.styles.HelpText.Render("quit application")),
		lipgloss.JoinHorizontal(lipgloss.Left,
			m.styles.FocusedLabel.Render("ESC "),
			m.styles.HelpText.Render("exit current view")),
	)

	// Join the two columns side by side
	columns := lipgloss.JoinHorizontal(lipgloss.Top,
		leftColumn,
		"    ", // spacing between columns
		rightColumn,
	)

	// Create the main content
	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		columns,
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
