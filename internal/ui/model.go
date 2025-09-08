package ui

import (
	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/connectivity"
	"github.com/Gu1llaum-3/sshm/internal/history"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// SortMode defines the available sorting modes
type SortMode int

const (
	SortByName SortMode = iota
	SortByLastUsed
)

func (s SortMode) String() string {
	switch s {
	case SortByName:
		return "Name (A-Z)"
	case SortByLastUsed:
		return "Last Login"
	default:
		return "Name (A-Z)"
	}
}

// ViewMode defines the current view state
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewAdd
	ViewEdit
	ViewMove
	ViewInfo
	ViewPortForward
	ViewHelp
	ViewFileSelector
)

// PortForwardType defines the type of port forwarding
type PortForwardType int

const (
	LocalForward PortForwardType = iota
	RemoteForward
	DynamicForward
)

func (p PortForwardType) String() string {
	switch p {
	case LocalForward:
		return "Local (-L)"
	case RemoteForward:
		return "Remote (-R)"
	case DynamicForward:
		return "Dynamic (-D)"
	default:
		return "Local (-L)"
	}
}

// Model represents the state of the user interface
type Model struct {
	table          table.Model
	searchInput    textinput.Model
	hosts          []config.SSHHost
	filteredHosts  []config.SSHHost
	searchMode     bool
	deleteMode     bool
	deleteHost     string
	historyManager *history.HistoryManager
	pingManager    *connectivity.PingManager
	sortMode       SortMode
	configFile     string // Path to the SSH config file

	// View management
	viewMode         ViewMode
	addForm          *addFormModel
	editForm         *editFormModel
	moveForm         *moveFormModel
	infoForm         *infoFormModel
	portForwardForm  *portForwardModel
	helpForm         *helpModel
	fileSelectorForm *fileSelectorModel

	// Terminal size and styles
	width  int
	height int
	styles Styles
	ready  bool
}

// updateTableStyles updates the table header border color based on focus state
func (m *Model) updateTableStyles() {
	s := table.DefaultStyles()
	s.Selected = m.styles.Selected

	if m.searchMode {
		// When in search mode, use secondary color for table header
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(SecondaryColor)).
			BorderBottom(true).
			Bold(false)
	} else {
		// When table is focused, use primary color for table header
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(PrimaryColor)).
			BorderBottom(true).
			Bold(false)
	}

	m.table.SetStyles(s)
}
