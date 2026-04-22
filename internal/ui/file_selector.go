package ui

import (
	"fmt"
	"path/filepath"
	"github.com/Gu1llaum-3/sshm/internal/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type fileSelectorModel struct {
	files        []string // Chemins absolus des fichiers
	displayNames []string // Noms d'affichage conviviaux
	selected     int
	styles       Styles
	width        int
	height       int
	title        string
}

type fileSelectorMsg struct {
	selectedFile string
	cancelled    bool
}

// NewFileSelector creates a new file selector for choosing config files
func NewFileSelector(title string, styles Styles, width, height int) (*fileSelectorModel, error) {
	files, err := config.GetAllConfigFiles()
	if err != nil {
		return nil, err
	}

	return newFileSelectorFromFiles(title, styles, width, height, files)
}

// NewFileSelectorFromBase creates a new file selector starting from a specific base config file
func NewFileSelectorFromBase(title string, styles Styles, width, height int, baseConfigFile string) (*fileSelectorModel, error) {
	var files []string
	var err error

	if baseConfigFile != "" {
		files, err = config.GetAllConfigFilesFromBase(baseConfigFile)
	} else {
		files, err = config.GetAllConfigFiles()
	}

	if err != nil {
		return nil, err
	}

	return newFileSelectorFromFiles(title, styles, width, height, files)
}

// NewFileSelectorWithAll creates a file selector listing every config file
// reachable from baseConfigFile, with a synthetic leading entry whose file
// value is the empty string and whose display name is allLabel (e.g.
// "[All files]"). Selecting that entry makes the selector return "" through
// the normal fileSelectorMsg path, which callers can interpret as "no
// specific file".
func NewFileSelectorWithAll(title string, styles Styles, width, height int,
	baseConfigFile string, allLabel string) (*fileSelectorModel, error) {

	var files []string
	var err error
	if baseConfigFile != "" {
		files, err = config.GetAllConfigFilesFromBase(baseConfigFile)
	} else {
		files, err = config.GetAllConfigFiles()
	}
	if err != nil {
		return nil, err
	}

	m, err := newFileSelectorFromFiles(title, styles, width, height, files)
	if err != nil {
		return nil, err
	}

	// Prepend the synthetic entry. files[0] == "" is the signal used by
	// callers to detect "no file selected / clear filter".
	m.files = append([]string{""}, m.files...)
	m.displayNames = append([]string{allLabel}, m.displayNames...)
	return m, nil
}

// newFileSelectorFromFiles creates a file selector from a list of files
func newFileSelectorFromFiles(title string, styles Styles, width, height int, files []string) (*fileSelectorModel, error) {

	// Convert absolute paths to more user-friendly names
	var displayNames []string
	homeDir, _ := config.GetSSHDirectory()

	for _, file := range files {
		// Check if it's the main config file
		mainConfig, _ := config.GetDefaultSSHConfigPath()
		if file == mainConfig {
			displayNames = append(displayNames, "Main SSH Config (~/.ssh/config)")
		} else {
			// Try to make path relative to home/.ssh/
			if strings.HasPrefix(file, homeDir) {
				relPath, err := filepath.Rel(homeDir, file)
				if err == nil {
					displayNames = append(displayNames, fmt.Sprintf("~/.ssh/%s", relPath))
				} else {
					displayNames = append(displayNames, file)
				}
			} else {
				displayNames = append(displayNames, file)
			}
		}
	}

	return &fileSelectorModel{
		files:        files,
		displayNames: displayNames,
		selected:     0,
		styles:       styles,
		width:        width,
		height:       height,
		title:        title,
	}, nil
}

func (m *fileSelectorModel) Init() tea.Cmd {
	return nil
}

func (m *fileSelectorModel) Update(msg tea.Msg) (*fileSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg {
				return fileSelectorMsg{cancelled: true}
			}

		case "enter":
			selectedFile := ""
			if m.selected < len(m.files) {
				selectedFile = m.files[m.selected]
			}
			return m, func() tea.Msg {
				return fileSelectorMsg{selectedFile: selectedFile}
			}

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}

		case "down", "j":
			if m.selected < len(m.files)-1 {
				m.selected++
			}
		}
	}

	return m, nil
}

func (m *fileSelectorModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.FormTitle.Render(m.title))
	b.WriteString("\n\n")

	if len(m.files) == 0 {
		b.WriteString(m.styles.Error.Render("No SSH config files found."))
		b.WriteString("\n\n")
		b.WriteString(m.styles.FormHelp.Render("Esc: cancel"))
		return b.String()
	}

	for i, displayName := range m.displayNames {
		if i == m.selected {
			b.WriteString(m.styles.Selected.Render(fmt.Sprintf("▶ %s", displayName)))
		} else {
			b.WriteString(fmt.Sprintf("  %s", displayName))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("↑/↓: navigate • Enter: select • Esc: cancel"))

	return b.String()
}
