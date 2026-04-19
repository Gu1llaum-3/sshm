package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
	keypkg "github.com/Gu1llaum-3/sshm/internal/key"
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

type editFormDeployFinishedMsg struct {
	err error
}

type editFormCancelMsg struct{}

type editFormModel struct {
	hostInputs       []textinput.Model // Support for multiple hosts
	inputs           []textinput.Model
	focusArea        int // 0=hosts, 1=properties
	focused          int
	currentTab       int // 0=General, 1=Advanced (only applies when focusArea == focusAreaProperties)
	err              string
	styles           Styles
	originalName     string
	originalHosts    []string        // Store original host names for multi-host detection
	host             *config.SSHHost // Store the original host with SourceFile
	configFile       string          // Configuration file path passed by user
	actualConfigFile string          // Actual config file to use (either configFile or host.SourceFile)
	width            int
	height           int
	keyPicker        *keyPickerModel
	keyCreate        *keyCreateFormModel
}

type editFormField struct {
	index int
	label string
}

var runEditFormDeploy = func(opts keypkg.DeployOptions) tea.Cmd {
	return runUIDeployCommand(opts, func(err error) tea.Msg {
		return editFormDeployFinishedMsg{err: err}
	})
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

	inputs := make([]textinput.Model, 10)

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

	// ProxyCommand input
	inputs[5] = textinput.New()
	inputs[5].Placeholder = "ssh -W %h:%p Jumphost"
	inputs[5].CharLimit = 200
	inputs[5].Width = 50
	inputs[5].SetValue(host.ProxyCommand)

	// Options input
	inputs[6] = textinput.New()
	inputs[6].Placeholder = "-o StrictHostKeyChecking=no"
	inputs[6].CharLimit = 200
	inputs[6].Width = 50
	if host.Options != "" {
		inputs[6].SetValue(config.FormatSSHOptionsForCommand(host.Options))
	}

	// Tags input
	inputs[7] = textinput.New()
	inputs[7].Placeholder = "production, web, database"
	inputs[7].CharLimit = 200
	inputs[7].Width = 50
	if len(host.Tags) > 0 {
		inputs[7].SetValue(strings.Join(host.Tags, ", "))
	}

	// Remote Command input
	inputs[8] = textinput.New()
	inputs[8].Placeholder = "ls -la, htop, bash"
	inputs[8].CharLimit = 300
	inputs[8].Width = 70
	inputs[8].SetValue(host.RemoteCommand)

	// RequestTTY input
	inputs[9] = textinput.New()
	inputs[9].Placeholder = "yes, no, force, auto"
	inputs[9].CharLimit = 10
	inputs[9].Width = 30
	inputs[9].SetValue(host.RequestTTY)

	return &editFormModel{
		hostInputs:       hostInputs,
		inputs:           inputs,
		focusArea:        focusAreaHosts, // Start with hosts focused for multi-host editing
		focused:          0,
		currentTab:       0, // Start on General tab
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

// getPropertiesForCurrentTab returns the property input indices for the current tab
func (m *editFormModel) getPropertiesForCurrentTab() []int {
	switch m.currentTab {
	case 0: // General
		return []int{0, 1, 2, 3, 4, 5, 7} // hostname, user, port, identity, proxyjump, proxycommand, tags
	case 1: // Advanced
		return []int{6, 8, 9} // options, remotecommand, requesttty
	default:
		return []int{0, 1, 2, 3, 4, 5, 7}
	}
}

// getFirstPropertyForTab returns the first property index for a given tab
func (m *editFormModel) getFirstPropertyForTab(tab int) int {
	properties := []int{0, 1, 2, 3, 4, 5, 7} // General tab
	if tab == 1 {
		properties = []int{6, 8, 9} // Advanced tab
	}
	if len(properties) > 0 {
		return properties[0]
	}
	return 0
}

// handleEditNavigation handles navigation in the edit form with tab support
func (m *editFormModel) handleEditNavigation(key string) tea.Cmd {
	if m.focusArea == focusAreaHosts {
		// Navigate in hosts area
		if key == "up" || key == "shift+tab" {
			m.focused--
		} else {
			m.focused++
		}

		if m.focused >= len(m.hostInputs) {
			// Move to properties area, keep current tab
			m.focusArea = focusAreaProperties
			// Keep the current tab instead of forcing it to 0
			m.focused = m.getFirstPropertyForTab(m.currentTab)
		} else if m.focused < 0 {
			m.focused = len(m.hostInputs) - 1
		}
	} else {
		// Navigate in properties area within current tab
		currentTabProperties := m.getPropertiesForCurrentTab()

		currentPos := slices.Index(currentTabProperties, m.focused)
		if currentPos < 0 {
			currentPos = 0
		}

		// Handle form submission on last field of Advanced tab
		if key == "enter" && m.currentTab == 1 && currentPos == len(currentTabProperties)-1 {
			return m.submitEditForm()
		}

		// Navigate within current tab
		if key == "up" || key == "shift+tab" {
			currentPos--
		} else {
			currentPos++
		}

		// Handle transitions between areas and tabs
		if currentPos >= len(currentTabProperties) {
			// Move to next area/tab
			if m.currentTab == 0 {
				// Move to advanced tab
				m.currentTab = 1
				m.focused = m.getFirstPropertyForTab(1)
			} else {
				// Move back to hosts area
				m.focusArea = focusAreaHosts
				m.focused = 0
			}
		} else if currentPos < 0 {
			// Move to previous area/tab
			if m.currentTab == 1 {
				// Move to general tab
				m.currentTab = 0
				properties := m.getPropertiesForCurrentTab()
				m.focused = properties[len(properties)-1]
			} else {
				// Move to hosts area
				m.focusArea = focusAreaHosts
				m.focused = len(m.hostInputs) - 1
			}
		} else {
			m.focused = currentTabProperties[currentPos]
		}
	}

	return m.updateFocus()
}

// getMinimumHeight calculates the minimum height needed to display the edit form
func (m *editFormModel) getMinimumHeight() int {
	// Title: 1 line + 2 newlines = 3
	titleLines := 3
	// Config file info: 1 line + 2 newlines = 3
	configLines := 3
	// Host Names section: title (1) + spacing (2) = 3
	hostSectionLines := 3
	// Host inputs: number of hosts * 3 lines each (reduced from 4)
	hostLines := len(m.hostInputs) * 3
	// Properties section: title (1) + spacing (2) = 3
	propertiesSectionLines := 3
	// Tabs: 1 line + 2 newlines = 3
	tabLines := 3
	// Fields in current tab
	var fieldsCount int
	if m.currentTab == 0 {
		fieldsCount = 6 // 6 fields in general tab
	} else {
		fieldsCount = 3 // 3 fields in advanced tab
	}
	// Each field: reduced from 4 to 3 lines per field
	fieldsLines := fieldsCount * 3
	// Help text: 3 lines
	helpLines := 3
	// Error message space when needed: 2 lines
	errorLines := 0 // Only count when there's actually an error
	if m.err != "" {
		errorLines = 2
	}

	return titleLines + configLines + hostSectionLines + hostLines + propertiesSectionLines + tabLines + fieldsLines + helpLines + errorLines + 1 // +1 minimal safety margin
}

func (m *editFormModel) getMinimumCompactHeight() int {
	return 10
}

// isHeightSufficient checks if the current terminal height is sufficient
func (m *editFormModel) isHeightSufficient() bool {
	return m.height >= m.getMinimumHeight()
}

// renderHeightWarning renders a warning message when height is insufficient
func (m *editFormModel) renderHeightWarning() string {
	required := m.getMinimumCompactHeight()
	current := m.height

	warning := m.styles.ErrorText.Render("⚠️  Terminal height is too small!")
	details := m.styles.FormField.Render(fmt.Sprintf("Current: %d lines, Need at least: %d lines", current, required))
	instruction := m.styles.FormHelp.Render("Compact mode is available, but this terminal is still too short.")
	instruction2 := m.styles.FormHelp.Render("Resize the terminal or cancel with Esc/Ctrl+C.")

	return warning + "\n\n" + details + "\n\n" + instruction + "\n" + instruction2
}

func (m *editFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.keyCreate != nil {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.styles = NewStyles(m.width)
			m.keyCreate.width = m.width
			m.keyCreate.height = m.height
			m.keyCreate.styles = m.styles
			return m, nil
		case keyCreateFinishedMsg:
			if msg.err != nil {
				m.keyCreate.err = msg.err.Error()
				return m, nil
			}
			m.keyCreate = nil
			m.inputs[3].SetValue(msg.path)
			m.focusArea = focusAreaProperties
			m.currentTab = 0
			m.focused = 3
			return m, m.updateFocus()
		case keyCreateCancelMsg:
			m.keyCreate = nil
			m.focusArea = focusAreaProperties
			m.currentTab = 0
			m.focused = 3
			return m, m.updateFocus()
		}

		newForm, cmd := m.keyCreate.Update(msg)
		m.keyCreate = newForm
		return m, cmd
	}

	if m.keyPicker != nil {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.styles = NewStyles(m.width)
			m.keyPicker.width = m.width
			m.keyPicker.height = m.height
			m.keyPicker.styles = m.styles
			return m, nil
		case keyPickerSelectMsg:
			m.keyPicker = nil
			m.inputs[3].SetValue(msg.path)
			m.focusArea = focusAreaProperties
			m.currentTab = 0
			m.focused = 3
			return m, m.updateFocus()
		case keyPickerCancelMsg:
			m.keyPicker = nil
			m.focusArea = focusAreaProperties
			m.currentTab = 0
			m.focused = 3
			return m, m.updateFocus()
		case keyPickerCreateMsg:
			m.keyPicker = nil
			m.keyCreate = NewKeyCreateForm(m.styles, m.width, m.height)
			return m, m.keyCreate.Init()
		}

		newPicker, cmd := m.keyPicker.Update(msg)
		m.keyPicker = newPicker
		return m, cmd
	}

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

		case "ctrl+j":
			// Switch to next tab
			m.currentTab = (m.currentTab + 1) % 2
			// If we're in hosts area, stay there. If in properties, go to the first field of the new tab
			if m.focusArea == focusAreaProperties {
				m.focused = m.getFirstPropertyForTab(m.currentTab)
			}
			return m, m.updateFocus()

		case "ctrl+k":
			// Switch to previous tab
			m.currentTab = (m.currentTab - 1 + 2) % 2
			// If we're in hosts area, stay there. If in properties, go to the first field of the new tab
			if m.focusArea == focusAreaProperties {
				m.focused = m.getFirstPropertyForTab(m.currentTab)
			}
			return m, m.updateFocus()

		case "ctrl+g":
			if m.focusArea == focusAreaProperties && m.currentTab == 0 && m.focused == 3 {
				m.keyCreate = NewKeyCreateForm(m.styles, m.width, m.height)
				return m, m.keyCreate.Init()
			}

		case "ctrl+o":
			if m.focusArea == focusAreaProperties && m.currentTab == 0 && m.focused == 3 {
				return m, m.openKeyPicker()
			}

		case "enter":
			if m.focusArea == focusAreaProperties && m.currentTab == 0 && m.focused == 3 {
				return m, m.openKeyPicker()
			}
			return m, m.handleEditNavigation(msg.String())

		case "tab", "shift+tab", "up", "down":
			return m, m.handleEditNavigation(msg.String())

		case "ctrl+a":
			// Add a new host input
			return m, m.addHostInput()

		case "ctrl+d":
			// Delete the currently focused host (if more than one exists)
			if m.focusArea == focusAreaHosts && len(m.hostInputs) > 1 {
				return m, m.deleteHostInput()
			}
			return m, m.deployAndSave()
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
	case editFormDeployFinishedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		return m, m.submitEditForm()
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
	if m.keyCreate != nil {
		return m.keyCreate.View()
	}

	if m.keyPicker != nil {
		return m.keyPicker.View()
	}

	if m.height < m.getMinimumCompactHeight() {
		return m.renderHeightWarning()
	}
	if !m.isHeightSufficient() {
		return m.renderCompactView()
	}

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

	// Render tabs for properties
	b.WriteString(m.renderEditTabs())
	b.WriteString("\n\n")

	// Render current tab content
	switch m.currentTab {
	case 0: // General
		b.WriteString(m.renderEditGeneralTab())
	case 1: // Advanced
		b.WriteString(m.renderEditAdvancedTab())
	}

	if m.err != "" {
		b.WriteString(m.styles.Error.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	// Show different help based on number of hosts
	if len(m.hostInputs) > 1 {
		b.WriteString(m.styles.FormHelp.Render("Tab/↑↓: navigate • Identity File: Enter/Ctrl+O choose key • Ctrl+G generate"))
		b.WriteString("\n")
		b.WriteString(m.styles.FormHelp.Render("Ctrl+J/K: switch tabs • Ctrl+A: add host • Ctrl+D: deploy+save"))
		b.WriteString("\n")
		b.WriteString(m.styles.FormHelp.Render("In Host Names: Ctrl+D deletes the focused alias"))
		b.WriteString("\n")
	} else {
		b.WriteString(m.styles.FormHelp.Render("Tab/↑↓: navigate • Identity File: Enter/Ctrl+O choose key • Ctrl+G generate"))
		b.WriteString("\n")
		b.WriteString(m.styles.FormHelp.Render("Ctrl+J/K: switch tabs • Ctrl+A: add host • Ctrl+D: deploy selected key then save"))
		b.WriteString("\n")
	}
	b.WriteString(m.styles.FormHelp.Render("Ctrl+S: save • Ctrl+C/Esc: cancel • * Required fields"))

	return b.String()
}

// renderEditTabs renders the tab headers for properties
func (m *editFormModel) renderEditTabs() string {
	var generalTab, advancedTab string

	if m.currentTab == 0 {
		generalTab = m.styles.FocusedLabel.Render("[ General ]")
		advancedTab = m.styles.FormField.Render("  Advanced  ")
	} else {
		generalTab = m.styles.FormField.Render("  General  ")
		advancedTab = m.styles.FocusedLabel.Render("[ Advanced ]")
	}

	return generalTab + "  " + advancedTab
}

// renderEditGeneralTab renders the general tab content for properties
func (m *editFormModel) renderEditGeneralTab() string {
	var b strings.Builder

	fields := m.generalPropertyFields()

	for _, field := range fields {
		fieldStyle := m.styles.FormField
		if m.focusArea == focusAreaProperties && m.focused == field.index {
			fieldStyle = m.styles.FocusedLabel
		}
		b.WriteString(fieldStyle.Render(field.label))
		b.WriteString("\n")
		b.WriteString(m.inputs[field.index].View())
		b.WriteString("\n")
		if field.index == 7 && m.focusArea == focusAreaProperties && m.focused == 7 {
			b.WriteString(m.styles.FormHelp.Render(`  tip: use "hidden" to hide this host from the list`))
			b.WriteString("\n")
		}
		if field.index == 3 && m.focusArea == focusAreaProperties && m.focused == 3 {
			b.WriteString(m.styles.FormHelp.Render("  tip: Enter/Ctrl+O choose a key • Ctrl+G generates one • keep typing for a custom path"))
			b.WriteString("\n")
		}
		if field.index == 0 && m.focusArea == focusAreaProperties && m.focused == 0 {
			b.WriteString(m.styles.FormHelp.Render("  tip: Ctrl+D deploys the selected key to this host and saves on success"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderEditAdvancedTab renders the advanced tab content for properties
func (m *editFormModel) renderEditAdvancedTab() string {
	var b strings.Builder

	fields := m.advancedPropertyFields()

	for _, field := range fields {
		fieldStyle := m.styles.FormField
		if m.focusArea == focusAreaProperties && m.focused == field.index {
			fieldStyle = m.styles.FocusedLabel
		}
		b.WriteString(fieldStyle.Render(field.label))
		b.WriteString("\n")
		b.WriteString(m.inputs[field.index].View())
		b.WriteString("\n\n")
	}

	return b.String()
}

func (m *editFormModel) renderCompactView() string {
	var b strings.Builder

	b.WriteString(m.styles.Header.Render("Edit SSH Host"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("Compact mode"))
	b.WriteString("\n\n")

	if m.host != nil && m.host.SourceFile != "" {
		b.WriteString(m.styles.FormHelp.Render("Config: " + formatConfigFile(m.host.SourceFile)))
		b.WriteString("\n\n")
	}

	if m.focusArea == focusAreaHosts {
		if len(m.hostInputs) == 0 {
			b.WriteString(m.styles.ErrorText.Render("No host names available."))
		} else {
			b.WriteString(m.styles.FormHelp.Render(fmt.Sprintf("Host name %d/%d", m.focused+1, len(m.hostInputs))))
			b.WriteString("\n")
			b.WriteString(m.styles.FocusedLabel.Render(fmt.Sprintf("Host Name %d *", m.focused+1)))
			b.WriteString("\n")
			b.WriteString(m.hostInputs[m.focused].View())
			b.WriteString("\n")
		}
	} else {
		b.WriteString(m.renderEditTabs())
		b.WriteString("\n\n")
		fields := m.fieldsForCurrentPropertyTab()
		currentField, currentPos := m.currentPropertyField(fields)
		b.WriteString(m.styles.FormHelp.Render(fmt.Sprintf("Property %d/%d", currentPos+1, len(fields))))
		b.WriteString("\n")
		b.WriteString(m.styles.FocusedLabel.Render(currentField.label))
		b.WriteString("\n")
		b.WriteString(m.inputs[currentField.index].View())
		b.WriteString("\n")
		if currentField.index == 7 {
			b.WriteString(m.styles.FormHelp.Render(`tip: use "hidden" to hide this host from the list`))
			b.WriteString("\n")
		}
		if currentField.index == 3 {
			b.WriteString(m.styles.FormHelp.Render("tip: Enter/Ctrl+O choose • Ctrl+G generate • type manually for a custom path"))
			b.WriteString("\n")
		}
		if currentField.index == 0 {
			b.WriteString(m.styles.FormHelp.Render("tip: Ctrl+D deploys the selected key and saves only if deploy succeeds"))
			b.WriteString("\n")
		}
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.ErrorText.Render("Error: " + m.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if len(m.hostInputs) > 1 && m.focusArea == focusAreaHosts {
		b.WriteString(m.styles.FormHelp.Render("↑/↓ or Tab move • Ctrl+A add host • Ctrl+D delete alias • Ctrl+S save • Esc cancel"))
	} else {
		b.WriteString(m.styles.FormHelp.Render("↑/↓ or Tab move • Ctrl+J/K tabs • Ctrl+A add host • Ctrl+D deploy+save • Ctrl+S save • Esc cancel"))
	}
	return b.String()
}

func (m *editFormModel) generalPropertyFields() []editFormField {
	return []editFormField{
		{0, "Hostname/IP *"},
		{1, "User"},
		{2, "Port"},
		{3, "Identity File"},
		{4, "Proxy Jump"},
		{5, "Proxy Command"},
		{7, "Tags (comma-separated)"},
	}
}

func (m *editFormModel) advancedPropertyFields() []editFormField {
	return []editFormField{
		{6, "SSH Options"},
		{8, "Remote Command"},
		{9, "Request TTY"},
	}
}

func (m *editFormModel) fieldsForCurrentPropertyTab() []editFormField {
	if m.currentTab == 1 {
		return m.advancedPropertyFields()
	}
	return m.generalPropertyFields()
}

func (m *editFormModel) currentPropertyField(fields []editFormField) (editFormField, int) {
	for i, field := range fields {
		if field.index == m.focused {
			return field, i
		}
	}
	return fields[0], 0
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
		hostname := strings.TrimSpace(m.inputs[0].Value())                                   // hostnameInput
		user := strings.TrimSpace(m.inputs[1].Value())                                       // userInput
		port := strings.TrimSpace(m.inputs[2].Value())                                       // portInput
		identity := strings.TrimSpace(m.inputs[3].Value())                                   // identityInput
		proxyJump := strings.TrimSpace(m.inputs[4].Value())                                  // proxyJumpInput
		proxyCommand := strings.TrimSpace(m.inputs[5].Value())                               // proxyCommandInput
		options := config.ParseSSHOptionsFromCommand(strings.TrimSpace(m.inputs[6].Value())) // optionsInput
		remoteCommand := strings.TrimSpace(m.inputs[8].Value())                              // remoteCommandInput
		requestTTY := strings.TrimSpace(m.inputs[9].Value())                                 // requestTTYInput

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

		tags := parseCommaList(m.inputs[7].Value()) // tagsInput

		// Create the common host configuration
		commonHost := config.SSHHost{
			Hostname:      hostname,
			User:          user,
			Port:          port,
			Identity:      identity,
			ProxyJump:     proxyJump,
			ProxyCommand:  proxyCommand,
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

func (m *editFormModel) deployAndSave() tea.Cmd {
	hostname := strings.TrimSpace(m.inputs[0].Value())
	if hostname == "" {
		m.err = "hostname/IP is required before deploy"
		return nil
	}
	if !validation.ValidateHostname(hostname) && !validation.ValidateIP(hostname) {
		m.err = "invalid hostname or IP address format"
		return nil
	}

	port := strings.TrimSpace(m.inputs[2].Value())
	if port == "" {
		port = "22"
	}
	if !validation.ValidatePort(port) {
		m.err = "port must be between 1 and 65535"
		return nil
	}

	identity := strings.TrimSpace(m.inputs[3].Value())
	if identity == "" {
		m.err = "choose or generate a key before deploy"
		return nil
	}

	user := strings.TrimSpace(m.inputs[1].Value())
	if user == "" {
		user = strings.TrimSpace(m.inputs[1].Placeholder)
	}

	m.err = ""
	return runEditFormDeploy(keypkg.DeployOptions{
		Target:       hostname,
		User:         user,
		Port:         port,
		ProxyJump:    strings.TrimSpace(m.inputs[4].Value()),
		ProxyCommand: strings.TrimSpace(m.inputs[5].Value()),
		Identity:     identity,
	})
}

func (m *editFormModel) openKeyPicker() tea.Cmd {
	contextLine := "Choose key"
	if len(m.hostInputs) > 0 {
		var names []string
		for _, input := range m.hostInputs {
			name := strings.TrimSpace(input.Value())
			if name != "" {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			if len(names) == 1 {
				contextLine = fmt.Sprintf("For %s", names[0])
			} else {
				contextLine = fmt.Sprintf("For %s (+%d)", names[0], len(names)-1)
			}
		}
	}

	picker, err := NewKeyPicker("Edit Host", contextLine, m.styles, m.width, m.height, m.configFile)
	if err != nil {
		m.err = err.Error()
		return nil
	}

	m.err = ""
	m.keyPicker = picker
	return m.keyPicker.Init()
}
