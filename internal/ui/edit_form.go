package ui

import (
	"fmt"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/validation"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	focusAreaHosts = iota
	focusAreaProperties
)

type editFormSubmitMsg struct {
	hostname string
	err      error
}

type editFormCancelMsg struct{}

type editFormModel struct {
	hostInputs       []textinput.Model // Support for multiple hosts
	inputs           []textinput.Model
	focusArea        int // 0=hosts, 1=properties
	focused          int
	err              string
	styles           Styles
	originalName     string
	originalHosts    []string        // Store original host names for multi-host detection
	host             *config.SSHHost // Store the original host with SourceFile
	configFile       string          // Configuration file path passed by user
	actualConfigFile string          // Actual config file to use (either configFile or host.SourceFile)
	width            int
	height           int
}

// NewEditForm creates a new edit form model that supports both single and multi-host editing
func NewEditForm(hostName string, styles Styles, width, height int, configFile string) (*editFormModel, error) {
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

	// Check if this host is part of a multi-host declaration
	var actualConfigFile string
	var hostNames []string
	var isMulti bool

	if configFile != "" {
		actualConfigFile = configFile
	} else {
		actualConfigFile = host.SourceFile
	}

	if actualConfigFile != "" {
		isMulti, hostNames, err = config.IsPartOfMultiHostDeclaration(hostName, actualConfigFile)
		if err != nil {
			// If we can't determine multi-host status, treat as single host
			isMulti = false
			hostNames = []string{hostName}
		}
	}

	if !isMulti {
		hostNames = []string{hostName}
	}

	// Create host inputs
	hostInputs := make([]textinput.Model, len(hostNames))
	for i, name := range hostNames {
		hostInputs[i] = textinput.New()
		hostInputs[i].Placeholder = "host-name"
		hostInputs[i].SetValue(name)
		if i == 0 {
			hostInputs[i].Focus()
		}
	}

	inputs := make([]textinput.Model, 9) // Increased from 8 to 9 for RequestTTY

	// Hostname input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "192.168.1.100 or example.com"
	inputs[0].CharLimit = 100
	inputs[0].Width = 30
	inputs[0].SetValue(host.Hostname)

	// User input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "root"
	inputs[1].CharLimit = 50
	inputs[1].Width = 30
	inputs[1].SetValue(host.User)

	// Port input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "22"
	inputs[2].CharLimit = 5
	inputs[2].Width = 30
	inputs[2].SetValue(host.Port)

	// Identity input
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "~/.ssh/id_rsa"
	inputs[3].CharLimit = 200
	inputs[3].Width = 50
	inputs[3].SetValue(host.Identity)

	// ProxyJump input
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "jump-server"
	inputs[4].CharLimit = 100
	inputs[4].Width = 30
	inputs[4].SetValue(host.ProxyJump)

	// Options input
	inputs[5] = textinput.New()
	inputs[5].Placeholder = "-o StrictHostKeyChecking=no"
	inputs[5].CharLimit = 200
	inputs[5].Width = 50
	if host.Options != "" {
		inputs[5].SetValue(config.FormatSSHOptionsForCommand(host.Options))
	}

	// Tags input
	inputs[6] = textinput.New()
	inputs[6].Placeholder = "production, web, database"
	inputs[6].CharLimit = 200
	inputs[6].Width = 50
	if len(host.Tags) > 0 {
		inputs[6].SetValue(strings.Join(host.Tags, ", "))
	}

	// Remote Command input
	inputs[7] = textinput.New()
	inputs[7].Placeholder = "ls -la, htop, bash"
	inputs[7].CharLimit = 300
	inputs[7].Width = 70
	inputs[7].SetValue(host.RemoteCommand)

	// RequestTTY input
	inputs[8] = textinput.New()
	inputs[8].Placeholder = "yes, no, force, auto"
	inputs[8].CharLimit = 10
	inputs[8].Width = 30
	inputs[8].SetValue(host.RequestTTY)

	return &editFormModel{
		hostInputs:       hostInputs,
		inputs:           inputs,
		focusArea:        focusAreaHosts, // Start with hosts focused for multi-host editing
		focused:          0,
		originalName:     hostName,
		originalHosts:    hostNames,
		host:             host,
		configFile:       configFile,
		actualConfigFile: actualConfigFile,
		styles:           styles,
		width:            width,
		height:           height,
	}, nil
}

func (m *editFormModel) Init() tea.Cmd {
	return textinput.Blink
}

// addHostInput adds a new empty host input
func (m *editFormModel) addHostInput() tea.Cmd {
	newInput := textinput.New()
	newInput.Placeholder = "host-name"
	newInput.Focus()

	// Unfocus current input regardless of which area we're in
	if m.focusArea == focusAreaHosts && m.focused < len(m.hostInputs) {
		m.hostInputs[m.focused].Blur()
	} else if m.focusArea == focusAreaProperties && m.focused < len(m.inputs) {
		m.inputs[m.focused].Blur()
	}

	m.hostInputs = append(m.hostInputs, newInput)

	// Move focus to the new host input
	m.focusArea = focusAreaHosts
	m.focused = len(m.hostInputs) - 1

	return textinput.Blink
}

// deleteHostInput removes the currently focused host input
func (m *editFormModel) deleteHostInput() tea.Cmd {
	if len(m.hostInputs) <= 1 || m.focusArea != focusAreaHosts {
		return nil // Can't delete if only one host or not in host area
	}

	// Remove the focused host input
	m.hostInputs = append(m.hostInputs[:m.focused], m.hostInputs[m.focused+1:]...)

	// Adjust focus
	if m.focused >= len(m.hostInputs) {
		m.focused = len(m.hostInputs) - 1
	}

	// Focus the new current input
	if len(m.hostInputs) > 0 {
		m.hostInputs[m.focused].Focus()
	}

	return nil
}

// updateFocus updates the focus state based on current area and index
func (m *editFormModel) updateFocus() tea.Cmd {
	// Blur all inputs first
	for i := range m.hostInputs {
		m.hostInputs[i].Blur()
	}
	for i := range m.inputs {
		m.inputs[i].Blur()
	}

	// Focus the appropriate input
	if m.focusArea == focusAreaHosts {
		if m.focused < len(m.hostInputs) {
			m.hostInputs[m.focused].Focus()
		}
	} else {
		if m.focused < len(m.inputs) {
			m.inputs[m.focused].Focus()
		}
	}

	return textinput.Blink
}

func (m *editFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.err = ""
			return m, func() tea.Msg { return editFormCancelMsg{} }

		case "ctrl+s":
			// Allow submission from any field with Ctrl+S (Save)
			return m, m.submitEditForm()

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Handle form submission
			totalFields := len(m.hostInputs) + len(m.inputs)
			currentGlobalIndex := m.focused
			if m.focusArea == focusAreaProperties {
				currentGlobalIndex = len(m.hostInputs) + m.focused
			}

			if s == "enter" && currentGlobalIndex == totalFields-1 {
				return m, m.submitEditForm()
			}

			// Cycle inputs
			if s == "up" || s == "shift+tab" {
				currentGlobalIndex--
			} else {
				currentGlobalIndex++
			}

			if currentGlobalIndex >= totalFields {
				currentGlobalIndex = 0
			} else if currentGlobalIndex < 0 {
				currentGlobalIndex = totalFields - 1
			}

			// Update focus area and focused index based on global index
			if currentGlobalIndex < len(m.hostInputs) {
				m.focusArea = focusAreaHosts
				m.focused = currentGlobalIndex
			} else {
				m.focusArea = focusAreaProperties
				m.focused = currentGlobalIndex - len(m.hostInputs)
			}

			return m, m.updateFocus()

		case "ctrl+a":
			// Add a new host input
			return m, m.addHostInput()

		case "ctrl+d":
			// Delete the currently focused host (if more than one exists)
			if m.focusArea == focusAreaHosts && len(m.hostInputs) > 1 {
				return m, m.deleteHostInput()
			}
		}

	case editFormSubmitMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			// Success: let the wrapper handle this
			// In TUI mode, this will be handled by the parent
			// In standalone mode, the wrapper will quit
		}
		return m, nil
	}

	// Update host inputs
	hostCmd := make([]tea.Cmd, len(m.hostInputs))
	for i := range m.hostInputs {
		m.hostInputs[i], hostCmd[i] = m.hostInputs[i].Update(msg)
	}
	cmds = append(cmds, hostCmd...)

	// Update property inputs
	propCmd := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], propCmd[i] = m.inputs[i].Update(msg)
	}
	cmds = append(cmds, propCmd...)

	return m, tea.Batch(cmds...)
}

func (m *editFormModel) View() string {
	var b strings.Builder

	if m.err != "" {
		b.WriteString(m.styles.Error.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(m.styles.Header.Render("Edit SSH Host"))
	b.WriteString("\n\n")

	if m.host != nil && m.host.SourceFile != "" {
		labelStyle := m.styles.FormField
		pathStyle := m.styles.FormField
		configInfo := labelStyle.Render("Config file: ") + pathStyle.Render(formatConfigFile(m.host.SourceFile))
		b.WriteString(configInfo)
	}

	b.WriteString("\n\n")

	// Host Names Section
	b.WriteString(m.styles.FormTitle.Render("Host Names"))
	b.WriteString("\n\n")

	for i, hostInput := range m.hostInputs {
		hostStyle := m.styles.FormField
		if m.focusArea == focusAreaHosts && m.focused == i {
			hostStyle = m.styles.FocusedLabel
		}
		b.WriteString(hostStyle.Render(fmt.Sprintf("Host Name %d *", i+1)))
		b.WriteString("\n")
		b.WriteString(hostInput.View())
		b.WriteString("\n\n")
	}

	// Properties Section
	b.WriteString(m.styles.FormTitle.Render("Common Properties"))
	b.WriteString("\n\n")

	fields := []string{
		"Hostname/IP *",
		"User",
		"Port",
		"Identity File",
		"Proxy Jump",
		"SSH Options",
		"Tags (comma-separated)",
		"Remote Command",
		"Request TTY",
	}

	for i, field := range fields {
		fieldStyle := m.styles.FormField
		if m.focusArea == focusAreaProperties && m.focused == i {
			fieldStyle = m.styles.FocusedLabel
		}
		b.WriteString(fieldStyle.Render(field))
		b.WriteString("\n")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(m.styles.Error.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	// Show different help based on number of hosts
	if len(m.hostInputs) > 1 {
		b.WriteString(m.styles.FormHelp.Render("Tab/↑↓/Enter: navigate • Ctrl+A: add host • Ctrl+D: delete host"))
		b.WriteString("\n")
	} else {
		b.WriteString(m.styles.FormHelp.Render("Tab/↑↓/Enter: navigate • Ctrl+A: add host"))
		b.WriteString("\n")
	}
	b.WriteString(m.styles.FormHelp.Render("Ctrl+S: save • Ctrl+C/Esc: cancel • * Required fields"))

	return b.String()
}

// Standalone wrapper for edit form
type standaloneEditForm struct {
	*editFormModel
}

func (m standaloneEditForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editFormSubmitMsg:
		if msg.err != nil {
			m.editFormModel.err = msg.err.Error()
			return m, nil
		} else {
			// Success: quit the program
			return m, tea.Quit
		}
	case editFormCancelMsg:
		return m, tea.Quit
	}

	newForm, cmd := m.editFormModel.Update(msg)
	m.editFormModel = newForm.(*editFormModel)
	return m, cmd
}

// RunEditForm runs the edit form as a standalone program
func RunEditForm(hostName string, configFile string) error {
	styles := NewStyles(80) // Default width
	editForm, err := NewEditForm(hostName, styles, 80, 24, configFile)
	if err != nil {
		return err
	}

	m := standaloneEditForm{editForm}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func (m *editFormModel) submitEditForm() tea.Cmd {
	return func() tea.Msg {
		// Collect host names
		var hostNames []string
		for _, input := range m.hostInputs {
			name := strings.TrimSpace(input.Value())
			if name != "" {
				hostNames = append(hostNames, name)
			}
		}

		if len(hostNames) == 0 {
			return editFormSubmitMsg{err: fmt.Errorf("at least one host name is required")}
		}

		// Get property values using direct indices
		hostname := strings.TrimSpace(m.inputs[0].Value())      // hostnameInput
		user := strings.TrimSpace(m.inputs[1].Value())          // userInput
		port := strings.TrimSpace(m.inputs[2].Value())          // portInput
		identity := strings.TrimSpace(m.inputs[3].Value())      // identityInput
		proxyJump := strings.TrimSpace(m.inputs[4].Value())     // proxyJumpInput
		options := strings.TrimSpace(m.inputs[5].Value())       // optionsInput
		remoteCommand := strings.TrimSpace(m.inputs[7].Value()) // remoteCommandInput
		requestTTY := strings.TrimSpace(m.inputs[8].Value())    // requestTTYInput

		// Set defaults
		if port == "" {
			port = "22"
		}

		// Validate hostname
		if hostname == "" {
			return editFormSubmitMsg{err: fmt.Errorf("hostname is required")}
		}

		// Validate all host names
		for _, hostName := range hostNames {
			if err := validation.ValidateHost(hostName, hostname, port, identity); err != nil {
				return editFormSubmitMsg{err: err}
			}
		}

		// Parse tags
		tagsStr := strings.TrimSpace(m.inputs[6].Value()) // tagsInput
		var tags []string
		if tagsStr != "" {
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		// Create the common host configuration
		commonHost := config.SSHHost{
			Hostname:      hostname,
			User:          user,
			Port:          port,
			Identity:      identity,
			ProxyJump:     proxyJump,
			Options:       options,
			RemoteCommand: remoteCommand,
			RequestTTY:    requestTTY,
			Tags:          tags,
		}

		var err error
		if len(hostNames) == 1 && len(m.originalHosts) == 1 {
			// Single host editing
			commonHost.Name = hostNames[0]
			if m.actualConfigFile != "" {
				err = config.UpdateSSHHostInFile(m.originalName, commonHost, m.actualConfigFile)
			} else {
				err = config.UpdateSSHHost(m.originalName, commonHost)
			}
		} else {
			// Multi-host editing or conversion from single to multi
			err = config.UpdateMultiHostBlock(m.originalHosts, hostNames, commonHost, m.actualConfigFile)
		}

		return editFormSubmitMsg{hostname: hostNames[0], err: err}
	}
}
