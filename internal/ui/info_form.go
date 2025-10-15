package ui

import (
	"fmt"
	"github.com/Gu1llaum-3/sshm/internal/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type infoFormModel struct {
	host       *config.SSHHost
	styles     Styles
	width      int
	height     int
	configFile string
	hostName   string
}

// Messages for communication with parent model
type infoFormEditMsg struct {
	hostName string
}

type infoFormCancelMsg struct{}

// NewInfoForm creates a new info form model for displaying host details in read-only mode
func NewInfoForm(hostName string, styles Styles, width, height int, configFile string) (*infoFormModel, error) {
	// Get the existing host configuration
	var host *config.SSHHost
	var err error

	if configFile != "" {
		host, err = config.GetSSHHostFromFile(hostName, configFile)
	} else {
		host, err = config.GetSSHHost(hostName)
	}

	if err != nil {
		return nil, err
	}

	return &infoFormModel{
		host:       host,
		hostName:   hostName,
		configFile: configFile,
		styles:     styles,
		width:      width,
		height:     height,
	}, nil
}

func (m *infoFormModel) Init() tea.Cmd {
	return nil
}

func (m *infoFormModel) Update(msg tea.Msg) (*infoFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, func() tea.Msg { return infoFormCancelMsg{} }

		case "e", "enter":
			// Switch to edit mode
			return m, func() tea.Msg { return infoFormEditMsg{hostName: m.hostName} }
		}
	}

	return m, nil
}

func (m *infoFormModel) View() string {
	var b strings.Builder

	// Title
	title := fmt.Sprintf("SSH Host Information: %s", m.host.Name)
	b.WriteString(m.styles.FormTitle.Render(title))
	b.WriteString("\n\n")

	// Create info sections with consistent formatting
	sections := []struct {
		label string
		value string
	}{
		{"Host Name", m.host.Name},
		{"Config File", formatConfigFile(m.host.SourceFile)},
		{"Hostname/IP", m.host.Hostname},
		{"User", formatOptionalValue(m.host.User)},
		{"Port", formatOptionalValue(m.host.Port)},
		{"Identity File", formatOptionalValue(m.host.Identity)},
		{"ProxyJump", formatOptionalValue(m.host.ProxyJump)},
		{"ProxyCommand", formatOptionalValue(m.host.ProxyCommand)},
		{"SSH Options", formatSSHOptions(m.host.Options)},
		{"Tags", formatTags(m.host.Tags)},
	}

	// Render each section
	for _, section := range sections {
		// Label style
		labelStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")). // Bright blue
			Width(15).
			AlignHorizontal(lipgloss.Right)

		// Value style
		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")) // White

		// If value is empty or default, use a muted style
		if section.value == "Not set" || section.value == "22" && section.label == "Port" {
			valueStyle = valueStyle.Foreground(lipgloss.Color("243")) // Gray
		}

		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			labelStyle.Render(section.label+":"),
			" ",
			valueStyle.Render(section.value),
		)
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Action instructions
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	b.WriteString(helpStyle.Render("Actions:"))
	b.WriteString("\n")

	actionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("120")). // Green
		Bold(true)

	b.WriteString("  ")
	b.WriteString(actionStyle.Render("e/Enter"))
	b.WriteString(helpStyle.Render(" - Switch to edit mode"))
	b.WriteString("\n")

	b.WriteString("  ")
	b.WriteString(actionStyle.Render("q/Esc"))
	b.WriteString(helpStyle.Render(" - Return to host list"))

	// Wrap in a border for better visual separation
	content := b.String()

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1).
		Margin(1)

	// Center the info window
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		borderStyle.Render(content),
	)
}

// Helper functions for formatting values

func formatOptionalValue(value string) string {
	if value == "" {
		return "Not set"
	}
	return value
}

func formatSSHOptions(options string) string {
	if options == "" {
		return "Not set"
	}
	return options
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "Not set"
	}
	return strings.Join(tags, ", ")
}

// Standalone wrapper for info form (for testing or standalone use)
type standaloneInfoForm struct {
	*infoFormModel
}

func (m standaloneInfoForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case infoFormCancelMsg:
		return m, tea.Quit
	case infoFormEditMsg:
		// For standalone mode, just quit - parent should handle edit transition
		return m, tea.Quit
	}

	newForm, cmd := m.infoFormModel.Update(msg)
	m.infoFormModel = newForm
	return m, cmd
}

// RunInfoForm provides a standalone info form for testing
func RunInfoForm(hostName string, configFile string) error {
	styles := NewStyles(80)
	infoForm, err := NewInfoForm(hostName, styles, 80, 24, configFile)
	if err != nil {
		return err
	}
	m := standaloneInfoForm{infoForm}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
