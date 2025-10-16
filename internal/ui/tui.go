package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/connectivity"
	"github.com/Gu1llaum-3/sshm/internal/history"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NewModel creates a new TUI model with the given SSH hosts
func NewModel(hosts []config.SSHHost, configFile string, searchMode bool, currentVersion string) Model {
	// Load application configuration
	appConfig, err := config.LoadAppConfig()
	if err != nil {
		// Log the error but continue with default configuration
		fmt.Printf("Warning: Could not load application config: %v, using defaults\n", err)
		defaultConfig := config.GetDefaultAppConfig()
		appConfig = &defaultConfig
	}

	// Initialize the history manager
	historyManager, err := history.NewHistoryManager()
	if err != nil {
		// Log the error but continue without the history functionality
		fmt.Printf("Warning: Could not initialize history manager: %v\n", err)
		historyManager = nil
	}

	// Create initial styles (will be updated on first WindowSizeMsg)
	styles := NewStyles(80) // Default width

	// Initialize ping manager with 5 second timeout
	pingManager := connectivity.NewPingManager(5 * time.Second)

	// Create the model with default sorting by name
	m := Model{
		hosts:          hosts,
		historyManager: historyManager,
		pingManager:    pingManager,
		sortMode:       SortByName,
		configFile:     configFile,
		currentVersion: currentVersion,
		appConfig:      appConfig,
		styles:         styles,
		width:          80,
		height:         24,
		ready:          false,
		viewMode:       ViewList,
		searchMode:     searchMode,
	}

	// Sort hosts according to the default sort mode
	sortedHosts := m.sortHosts(hosts)

	// Create the search input
	ti := textinput.New()
	ti.Placeholder = "Search hosts or tags..."
	ti.CharLimit = 50
	ti.Width = 25
	if searchMode {
		ti.Focus()
	}

	// Use dynamic column width calculation (will fallback to static if width not available)
	nameWidth, hostnameWidth, tagsWidth, lastLoginWidth := m.calculateDynamicColumnWidths(sortedHosts)

	// Create table columns
	columns := []table.Column{
		{Title: "Name", Width: nameWidth},
		{Title: "Hostname", Width: hostnameWidth},
		// {Title: "User", Width: 12},                  // Commented to save space
		// {Title: "Port", Width: 6},                   // Commented to save space
		{Title: "Tags", Width: tagsWidth},
		{Title: "Last Login", Width: lastLoginWidth},
	}

	// Convert hosts to table rows
	var rows []table.Row
	for _, host := range sortedHosts {
		// Get ping status indicator
		statusIndicator := m.getPingStatusIndicator(host.Name)

		// Format tags for display
		var tagsStr string
		if len(host.Tags) > 0 {
			// Add the # prefix to each tag and join them with spaces
			var formattedTags []string
			for _, tag := range host.Tags {
				formattedTags = append(formattedTags, "#"+tag)
			}
			tagsStr = strings.Join(formattedTags, " ")
		}

		// Format last login information
		var lastLoginStr string
		if historyManager != nil {
			if lastConnect, exists := historyManager.GetLastConnectionTime(host.Name); exists {
				lastLoginStr = formatTimeAgo(lastConnect)
			}
		}

		rows = append(rows, table.Row{
			statusIndicator + " " + host.Name,
			host.Hostname,
			// host.User,        // Commented to save space
			// host.Port,        // Commented to save space
			tagsStr,
			lastLoginStr,
		})
	}

	// Create the table with initial height (will be updated on first WindowSizeMsg)
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10), // Initial height, will be recalculated dynamically
	)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(SecondaryColor)).
		BorderBottom(true).
		Bold(false)
	s.Selected = m.styles.Selected

	t.SetStyles(s)

	// Update the model with the table and other properties
	m.table = t
	m.searchInput = ti
	m.filteredHosts = sortedHosts

	// Initialize table styles based on initial focus state
	m.updateTableStyles()

	// The table height will be properly set on the first WindowSizeMsg
	// when m.ready becomes true and actual terminal dimensions are known

	return m
}

// RunInteractiveMode starts the interactive TUI interface
func RunInteractiveMode(hosts []config.SSHHost, configFile string, searchMode bool, currentVersion string) error {
	m := NewModel(hosts, configFile, searchMode, currentVersion)

	// Start the application in alt screen mode for clean output
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
