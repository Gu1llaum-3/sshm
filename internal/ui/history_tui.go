package ui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/history"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HistoryModel represents the TUI model for history view
type HistoryModel struct {
	table          table.Model
	connections    []history.ConnectionInfo
	searchInput    textinput.Model
	searchActive   bool
	filteredConns  []history.ConnectionInfo
	configFile     string
	currentVersion string
	styles         Styles
	width          int
	height         int
	showAddForm    bool
	addForm        *addFormModel
	selectedConn   *history.ConnectionInfo
	err            string
}

// NewHistoryModel creates a new history TUI model
func NewHistoryModel(connections []history.ConnectionInfo, configFile, currentVersion string) HistoryModel {
	styles := NewStyles(80)

	// Create search input (different placeholder than main interface)
	searchInput := textinput.New()
	searchInput.Placeholder = "Search connections..."
	searchInput.CharLimit = 50
	searchInput.Width = 25 // Same width as main interface

	m := HistoryModel{
		connections:    connections,
		filteredConns:  connections,
		searchInput:    searchInput,
		configFile:     configFile,
		currentVersion: currentVersion,
		styles:         styles,
	}

	m.updateTable()
	return m
}

// Init initializes the history model
func (m HistoryModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the history model
func (m HistoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle add form if active
	if m.showAddForm && m.addForm != nil {
		switch msg := msg.(type) {
		case addFormSubmitMsg:
			if msg.err != nil {
				m.err = msg.err.Error()
			} else {
				m.showAddForm = false
				m.addForm = nil
				// Return to main list and refresh hosts
				return m, func() tea.Msg { return refreshHostsMsg{} }
			}
		case addFormCancelMsg:
			m.showAddForm = false
			m.addForm = nil
			return m, nil
		}

		newForm, cmd := m.addForm.Update(msg)
		m.addForm = newForm
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		m.updateTable()
		return m, nil

	case tea.KeyMsg:
		// Handle search mode
		if m.searchActive {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.searchActive = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.filteredConns = m.connections
				m.updateTable()
				return m, nil
			case "enter":
				m.searchActive = false
				m.searchInput.Blur()
				return m, nil
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
				m.filterConnections()
				m.updateTable()
				return m, tea.Batch(cmds...)
			}
		}

		// Normal mode key handling
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "ctrl+l":
			// Return to main list view
			return m, func() tea.Msg { return returnToListMsg{} }

		case "enter":
			// Connect to selected host
			if len(m.filteredConns) > 0 {
				selectedIdx := m.table.Cursor()
				if selectedIdx < len(m.filteredConns) {
					conn := m.filteredConns[selectedIdx]
					return m, m.connectToHistory(conn)
				}
			}

		case "a":
			// Add manual connection to config
			if len(m.filteredConns) > 0 {
				selectedIdx := m.table.Cursor()
				if selectedIdx < len(m.filteredConns) {
					conn := m.filteredConns[selectedIdx]
					// Only allow adding manual connections to config
					if history.IsManualConnection(conn.HostName) {
						m.selectedConn = &conn
						m.showAddForm = true
						m.addForm = m.createAddFormFromConnection(conn)
						return m, m.addForm.Init()
					}
				}
			}

		case "d":
			// Delete connection from history
			if len(m.filteredConns) > 0 {
				selectedIdx := m.table.Cursor()
				if selectedIdx < len(m.filteredConns) {
					conn := m.filteredConns[selectedIdx]
					return m, m.deleteFromHistory(conn)
				}
			}

		case "/":
			// Activate search
			m.searchActive = true
			m.searchInput.Focus()
			return m, textinput.Blink
		}
	}

	// Update table
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the history TUI
func (m HistoryModel) View() string {
	if m.showAddForm && m.addForm != nil {
		return m.addForm.View()
	}

	// Build the interface components (same structure as main view)
	components := []string{}

	// Add the ASCII title
	components = append(components, m.styles.Header.Render(asciiTitle))

	// Add error message if there's one to show
	if m.err != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red color
			Background(lipgloss.Color("1")). // Dark red background
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")).
			Align(lipgloss.Center)

		components = append(components, errorStyle.Render("❌ "+m.err))
	}

	// Add the search bar with the appropriate style based on focus
	searchPrompt := "Search (/ to focus): "
	if m.searchActive {
		components = append(components, m.styles.SearchFocused.Render(searchPrompt+m.searchInput.View()))
	} else {
		components = append(components, m.styles.SearchUnfocused.Render(searchPrompt+m.searchInput.View()))
	}

	// Add the table with the appropriate style based on focus
	if m.searchActive {
		// The table is not focused, use the unfocused style
		components = append(components, m.styles.TableUnfocused.Render(m.table.View()))
	} else {
		// The table is focused, use the focused style
		components = append(components, m.styles.TableFocused.Render(m.table.View()))
	}

	// Add the help text
	var helpText string
	if !m.searchActive {
		helpText = " ↑/↓: navigate • Enter: connect • Ctrl+L: list • a: add to config (★) • d: delete • q: quit"
	} else {
		helpText = " Type to filter • Enter: validate • Tab: switch • ESC: quit"
	}
	components = append(components, m.styles.HelpText.Render(helpText))

	// Join all components vertically with appropriate spacing
	mainView := m.styles.App.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			components...,
		),
	)

	return mainView
} // updateTable updates the table with current filtered connections
func (m *HistoryModel) updateTable() {
	columns := []table.Column{
		{Title: "Host", Width: 22}, // Host name with ★ for manual connections
		{Title: "User", Width: 15},
		{Title: "Hostname", Width: 25},
		{Title: "Port", Width: 6},
		{Title: "Last Connect", Width: 20},
		{Title: "Count", Width: 6},
	}

	// Load SSH hosts to get details for configured connections
	var sshHosts []config.SSHHost
	var err error
	if m.configFile != "" {
		sshHosts, err = config.ParseSSHConfigFile(m.configFile)
	} else {
		sshHosts, err = config.ParseSSHConfig()
	}
	if err != nil {
		sshHosts = []config.SSHHost{}
	}

	// Create a map for quick lookup
	hostsMap := make(map[string]config.SSHHost)
	for _, host := range sshHosts {
		hostsMap[host.Name] = host
	}

	rows := []table.Row{}
	for _, conn := range m.filteredConns {
		var hostDisplay, user, hostname, port string

		// Parse manual connections
		if history.IsManualConnection(conn.HostName) {
			u, h, p, ok := history.ParseManualConnectionID(conn.HostName)
			if ok {
				hostDisplay = "★" // Star indicates this can be added to config
				user = u
				hostname = h
				port = p
			}
		} else {
			// For configured hosts, show the host name
			hostDisplay = conn.HostName

			if host, exists := hostsMap[conn.HostName]; exists {
				user = host.User
				hostname = host.Hostname
				port = host.Port
				if port == "" {
					port = "22"
				}
			}
		}

		lastConnect := formatTimeSince(conn.LastConnect)

		rows = append(rows, table.Row{
			hostDisplay,
			user,
			hostname,
			port,
			lastConnect,
			fmt.Sprintf("%d", conn.ConnectCount),
		})
	}

	// Calculate dynamic table height (same logic as main interface)
	tableHeight := m.calculateTableHeight(len(rows))

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(tableHeight),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(PrimaryColor)).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color(PrimaryColor)).
		Bold(false)

	t.SetStyles(s)
	m.table = t
}

// calculateTableHeight calculates the appropriate height for the table based on terminal size
func (m *HistoryModel) calculateTableHeight(rowCount int) int {
	// Calculate dynamic table height based on terminal size
	// Layout breakdown (same as main interface):
	// - ASCII title: 5 lines (1 empty + 4 text lines)
	// - Search bar: 1 line
	// - Help text: 1 line
	// - App margins/spacing: 3 lines
	// - Safety margin: 3 lines
	// Total reserved: 13 lines
	reservedHeight := 13
	availableHeight := m.height - reservedHeight

	// Add 1 if there's an error message showing
	if m.err != "" {
		availableHeight -= 3 // Error box takes about 3 lines
	}

	// Minimum height should be at least 3 rows for basic usability
	minTableHeight := 4 // 1 header + 3 data rows minimum
	maxTableHeight := availableHeight
	if maxTableHeight < minTableHeight {
		maxTableHeight = minTableHeight
	}

	tableHeight := 1 // header
	dataRowsNeeded := rowCount
	maxDataRows := maxTableHeight - 1 // subtract 1 for header

	if dataRowsNeeded <= maxDataRows {
		// We have enough space for all connections
		tableHeight += dataRowsNeeded
	} else {
		// We need to limit to available space
		tableHeight += maxDataRows
	}

	// Add one extra line to prevent the last row from being hidden
	tableHeight += 1

	return tableHeight
}

// filterConnections filters connections based on search input
func (m *HistoryModel) filterConnections() {
	searchTerm := strings.ToLower(m.searchInput.Value())
	if searchTerm == "" {
		m.filteredConns = m.connections
		return
	}

	m.filteredConns = []history.ConnectionInfo{}
	for _, conn := range m.connections {
		// Search in hostname
		if strings.Contains(strings.ToLower(conn.HostName), searchTerm) {
			m.filteredConns = append(m.filteredConns, conn)
			continue
		}

		// For manual connections, search in parsed fields
		if history.IsManualConnection(conn.HostName) {
			user, hostname, _, ok := history.ParseManualConnectionID(conn.HostName)
			if ok {
				if strings.Contains(strings.ToLower(user), searchTerm) ||
					strings.Contains(strings.ToLower(hostname), searchTerm) {
					m.filteredConns = append(m.filteredConns, conn)
				}
			}
		}
	}
}

// connectToHistory connects to a host from history
func (m HistoryModel) connectToHistory(conn history.ConnectionInfo) tea.Cmd {
	var sshArgs []string

	if history.IsManualConnection(conn.HostName) {
		// Manual connection
		user, hostname, port, ok := history.ParseManualConnectionID(conn.HostName)
		if !ok {
			return nil
		}

		if port != "" && port != "22" {
			sshArgs = append(sshArgs, "-p", port)
		}

		if user != "" {
			sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", user, hostname))
		} else {
			sshArgs = append(sshArgs, hostname)
		}
	} else {
		// Configured host
		if m.configFile != "" {
			sshArgs = append(sshArgs, "-F", m.configFile)
		}
		sshArgs = append(sshArgs, conn.HostName)
	}

	// Execute SSH using tea.ExecProcess for proper terminal handling
	sshCmd := exec.Command("ssh", sshArgs...)
	return tea.ExecProcess(sshCmd, func(err error) tea.Msg {
		return tea.Quit()
	})
}

// deleteFromHistory removes a connection from history
func (m HistoryModel) deleteFromHistory(conn history.ConnectionInfo) tea.Cmd {
	return func() tea.Msg {
		historyManager, err := history.NewHistoryManager()
		if err != nil {
			return tea.Quit
		}

		// Remove from history
		// This would need a new method in history manager
		// For now, just quit
		_ = historyManager

		return tea.Quit
	}
}

// createAddFormFromConnection creates an add form pre-filled with connection details
func (m *HistoryModel) createAddFormFromConnection(conn history.ConnectionInfo) *addFormModel {
	user, hostname, port, ok := history.ParseManualConnectionID(conn.HostName)
	if !ok {
		return nil
	}

	// Create form with empty name (user will choose)
	form := NewAddForm("", m.styles, m.width, m.height, m.configFile)

	// Pre-fill the form with connection details
	form.inputs[hostnameInput].SetValue(hostname)
	form.inputs[userInput].SetValue(user)
	if port != "22" && port != "" {
		form.inputs[portInput].SetValue(port)
	}

	// Leave name field empty for user to choose
	// form.inputs[nameInput].SetValue("")  // Already empty by default

	return form
}

// formatTimeSince formats a time duration in human-readable format
func formatTimeSince(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 30*24*time.Hour:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		if months < 12 {
			return fmt.Sprintf("%d months ago", months)
		}
		years := months / 12
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}
