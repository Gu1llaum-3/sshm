package ui

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/validation"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type addFormModel struct {
	inputs     []textinput.Model
	focused    int
	currentTab int // 0 = General, 1 = Advanced
	err        string
	styles     Styles
	success    bool
	width      int
	height     int
	configFile string
}

// NewAddForm creates a new add form model
func NewAddForm(hostname string, styles Styles, width, height int, configFile string) *addFormModel {
	// Get current user for default
	currentUser, _ := user.Current()
	defaultUser := "root"
	if currentUser != nil {
		defaultUser = currentUser.Username
	}

	// Find default identity file
	homeDir, _ := os.UserHomeDir()
	defaultIdentity := filepath.Join(homeDir, ".ssh", "id_rsa")

	// Check for other common key types
	keyTypes := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
	for _, keyType := range keyTypes {
		keyPath := filepath.Join(homeDir, ".ssh", keyType)
		if _, err := os.Stat(keyPath); err == nil {
			defaultIdentity = keyPath
			break
		}
	}

	inputs := make([]textinput.Model, 11)

	// Name input
	inputs[nameInput] = textinput.New()
	inputs[nameInput].Placeholder = "server-name"
	inputs[nameInput].Focus()
	inputs[nameInput].CharLimit = 50
	inputs[nameInput].Width = 30
	if hostname != "" {
		inputs[nameInput].SetValue(hostname)
	}

	// Hostname input
	inputs[hostnameInput] = textinput.New()
	inputs[hostnameInput].Placeholder = "192.168.1.100 or example.com"
	inputs[hostnameInput].CharLimit = 100
	inputs[hostnameInput].Width = 30

	// User input
	inputs[userInput] = textinput.New()
	inputs[userInput].Placeholder = defaultUser
	inputs[userInput].CharLimit = 50
	inputs[userInput].Width = 30

	// Port input
	inputs[portInput] = textinput.New()
	inputs[portInput].Placeholder = "22"
	inputs[portInput].CharLimit = 5
	inputs[portInput].Width = 30

	// Identity input
	inputs[identityInput] = textinput.New()
	inputs[identityInput].Placeholder = defaultIdentity
	inputs[identityInput].CharLimit = 200
	inputs[identityInput].Width = 50

	// ProxyJump input
	inputs[proxyJumpInput] = textinput.New()
	inputs[proxyJumpInput].Placeholder = "user@jump-host:port or existing-host-name"
	inputs[proxyJumpInput].CharLimit = 200
	inputs[proxyJumpInput].Width = 50

	// ProxyCommand input
	inputs[proxyCommandInput] = textinput.New()
	inputs[proxyCommandInput].Placeholder = "ssh -W %h:%p Jumphost"
	inputs[proxyCommandInput].CharLimit = 200
	inputs[proxyCommandInput].Width = 50

	// SSH Options input
	inputs[optionsInput] = textinput.New()
	inputs[optionsInput].Placeholder = "-o Compression=yes -o ServerAliveInterval=60"
	inputs[optionsInput].CharLimit = 500
	inputs[optionsInput].Width = 70

	// Tags input
	inputs[tagsInput] = textinput.New()
	inputs[tagsInput].Placeholder = "production, web, database"
	inputs[tagsInput].CharLimit = 200
	inputs[tagsInput].Width = 50

	// Remote Command input
	inputs[remoteCommandInput] = textinput.New()
	inputs[remoteCommandInput].Placeholder = "ls -la, htop, bash"
	inputs[remoteCommandInput].CharLimit = 300
	inputs[remoteCommandInput].Width = 70

	// RequestTTY input
	inputs[requestTTYInput] = textinput.New()
	inputs[requestTTYInput].Placeholder = "yes, no, force, auto"
	inputs[requestTTYInput].CharLimit = 10
	inputs[requestTTYInput].Width = 30

	return &addFormModel{
		inputs:     inputs,
		focused:    nameInput,
		currentTab: tabGeneral, // Start on General tab
		styles:     styles,
		width:      width,
		height:     height,
		configFile: configFile,
	}
}

const (
	tabGeneral = iota
	tabAdvanced
)

const (
	nameInput = iota
	hostnameInput
	userInput
	portInput
	identityInput
	proxyJumpInput
	proxyCommandInput
	tagsInput
	// Advanced tab inputs
	optionsInput
	remoteCommandInput
	requestTTYInput
)

// Messages for communication with parent model
type addFormSubmitMsg struct {
	hostname string
	err      error
}

type addFormCancelMsg struct{}

func (m *addFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *addFormModel) Update(msg tea.Msg) (*addFormModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg { return addFormCancelMsg{} }

		case "ctrl+s":
			// Allow submission from any field with Ctrl+S (Save)
			return m, m.submitForm()

		case "ctrl+j":
			// Switch to next tab
			m.currentTab = (m.currentTab + 1) % 2
			m.focused = m.getFirstInputForTab(m.currentTab)
			return m, m.updateFocus()

		case "ctrl+k":
			// Switch to previous tab
			m.currentTab = (m.currentTab - 1 + 2) % 2
			m.focused = m.getFirstInputForTab(m.currentTab)
			return m, m.updateFocus()

		case "tab", "shift+tab", "enter", "up", "down":
			return m, m.handleNavigation(msg.String())
		}

	case addFormSubmitMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.success = true
			m.err = ""
			// Don't quit here, let parent handle the success
		}
		return m, nil
	}

	// Update inputs
	cmd := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmd[i] = m.inputs[i].Update(msg)
	}
	cmds = append(cmds, cmd...)

	return m, tea.Batch(cmds...)
}

// getFirstInputForTab returns the first input index for a given tab
func (m *addFormModel) getFirstInputForTab(tab int) int {
	switch tab {
	case tabGeneral:
		return nameInput
	case tabAdvanced:
		return optionsInput
	default:
		return nameInput
	}
}

// getInputsForCurrentTab returns the input indices for the current tab
func (m *addFormModel) getInputsForCurrentTab() []int {
	switch m.currentTab {
	case tabGeneral:
		return []int{nameInput, hostnameInput, userInput, portInput, identityInput, proxyJumpInput, proxyCommandInput, tagsInput}
	case tabAdvanced:
		return []int{optionsInput, remoteCommandInput, requestTTYInput}
	default:
		return []int{nameInput, hostnameInput, userInput, portInput, identityInput, proxyJumpInput, proxyCommandInput, tagsInput}
	}
}

// updateFocus updates focus for inputs
func (m *addFormModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.focused {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

// handleNavigation handles tab/arrow navigation within the current tab
func (m *addFormModel) handleNavigation(key string) tea.Cmd {
	currentTabInputs := m.getInputsForCurrentTab()

	// Find current position within the tab
	currentPos := 0
	for i, input := range currentTabInputs {
		if input == m.focused {
			currentPos = i
			break
		}
	}

	// Handle form submission on last field of Advanced tab
	if key == "enter" && m.currentTab == tabAdvanced && currentPos == len(currentTabInputs)-1 {
		return m.submitForm()
	}

	// Navigate within current tab
	if key == "up" || key == "shift+tab" {
		currentPos--
	} else {
		currentPos++
	}

	// Wrap around within current tab
	if currentPos >= len(currentTabInputs) {
		currentPos = 0
	} else if currentPos < 0 {
		currentPos = len(currentTabInputs) - 1
	}

	m.focused = currentTabInputs[currentPos]
	return m.updateFocus()
}

func (m *addFormModel) View() string {
	if m.success {
		return ""
	}

	// Check if terminal height is sufficient
	if !m.isHeightSufficient() {
		return m.renderHeightWarning()
	}

	var b strings.Builder

	b.WriteString(m.styles.FormTitle.Render("Add SSH Host Configuration"))
	b.WriteString("\n\n")

	// Render tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Render current tab content
	switch m.currentTab {
	case tabGeneral:
		b.WriteString(m.renderGeneralTab())
	case tabAdvanced:
		b.WriteString(m.renderAdvancedTab())
	}

	if m.err != "" {
		b.WriteString(m.styles.Error.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	// Help text
	b.WriteString(m.styles.FormHelp.Render("Tab/Shift+Tab: navigate • Ctrl+J/K: switch tabs"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("Enter on last field: submit • Ctrl+S: save • Ctrl+C/Esc: cancel"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("* Required fields"))

	return b.String()
}

// getMinimumHeight calculates the minimum height needed to display the form
func (m *addFormModel) getMinimumHeight() int {
	// Title: 1 line + 2 newlines = 3
	titleLines := 3
	// Tabs: 1 line + 2 newlines = 3
	tabLines := 3
	// Fields in current tab
	var fieldsCount int
	if m.currentTab == tabGeneral {
		fieldsCount = 7 // 7 fields in general tab
	} else {
		fieldsCount = 3 // 3 fields in advanced tab
	}
	// Each field: label (1) + input (1) + spacing (2) = 4 lines per field, but let's be more conservative
	fieldsLines := fieldsCount * 3 // Reduced from 4 to 3
	// Help text: 3 lines
	helpLines := 3
	// Error message space when needed: 2 lines
	errorLines := 0 // Only count when there's actually an error
	if m.err != "" {
		errorLines = 2
	}

	return titleLines + tabLines + fieldsLines + helpLines + errorLines + 1 // +1 minimal safety margin
}

// isHeightSufficient checks if the current terminal height is sufficient
func (m *addFormModel) isHeightSufficient() bool {
	return m.height >= m.getMinimumHeight()
}

// renderHeightWarning renders a warning message when height is insufficient
func (m *addFormModel) renderHeightWarning() string {
	required := m.getMinimumHeight()
	current := m.height

	warning := m.styles.ErrorText.Render("⚠️  Terminal height is too small!")
	details := m.styles.FormField.Render(fmt.Sprintf("Current: %d lines, Required: %d lines", current, required))
	instruction := m.styles.FormHelp.Render("Please resize your terminal window and try again.")
	instruction2 := m.styles.FormHelp.Render("Press Ctrl+C to cancel or resize terminal window.")

	return warning + "\n\n" + details + "\n\n" + instruction + "\n" + instruction2
}

// renderTabs renders the tab headers
func (m *addFormModel) renderTabs() string {
	var generalTab, advancedTab string

	if m.currentTab == tabGeneral {
		generalTab = m.styles.FocusedLabel.Render("[ General ]")
		advancedTab = m.styles.FormField.Render("  Advanced  ")
	} else {
		generalTab = m.styles.FormField.Render("  General  ")
		advancedTab = m.styles.FocusedLabel.Render("[ Advanced ]")
	}

	return generalTab + "  " + advancedTab
}

// renderGeneralTab renders the general tab content
func (m *addFormModel) renderGeneralTab() string {
	var b strings.Builder

	fields := []struct {
		index int
		label string
	}{
		{nameInput, "Host Name *"},
		{hostnameInput, "Hostname/IP *"},
		{userInput, "User"},
		{portInput, "Port"},
		{identityInput, "Identity File"},
		{proxyJumpInput, "ProxyJump"},
		{proxyCommandInput, "ProxyCommand"},
		{tagsInput, "Tags (comma-separated)"},
	}

	for _, field := range fields {
		fieldStyle := m.styles.FormField
		if m.focused == field.index {
			fieldStyle = m.styles.FocusedLabel
		}
		b.WriteString(fieldStyle.Render(field.label))
		b.WriteString("\n")
		b.WriteString(m.inputs[field.index].View())
		b.WriteString("\n\n")
	}

	return b.String()
}

// renderAdvancedTab renders the advanced tab content
func (m *addFormModel) renderAdvancedTab() string {
	var b strings.Builder

	fields := []struct {
		index int
		label string
	}{
		{optionsInput, "SSH Options"},
		{remoteCommandInput, "Remote Command"},
		{requestTTYInput, "Request TTY"},
	}

	for _, field := range fields {
		fieldStyle := m.styles.FormField
		if m.focused == field.index {
			fieldStyle = m.styles.FocusedLabel
		}
		b.WriteString(fieldStyle.Render(field.label))
		b.WriteString("\n")
		b.WriteString(m.inputs[field.index].View())
		b.WriteString("\n\n")
	}

	return b.String()
}

// Standalone wrapper for add form
type standaloneAddForm struct {
	*addFormModel
}

func (m standaloneAddForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case addFormSubmitMsg:
		if msg.err != nil {
			m.addFormModel.err = msg.err.Error()
		} else {
			m.addFormModel.success = true
			return m, tea.Quit
		}
		return m, nil
	case addFormCancelMsg:
		return m, tea.Quit
	}

	newForm, cmd := m.addFormModel.Update(msg)
	m.addFormModel = newForm
	return m, cmd
}

// RunAddForm provides backward compatibility for standalone add form
func RunAddForm(hostname string, configFile string) error {
	styles := NewStyles(80)
	addForm := NewAddForm(hostname, styles, 80, 24, configFile)
	m := standaloneAddForm{addForm}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m *addFormModel) submitForm() tea.Cmd {
	return func() tea.Msg {
		// Get values
		name := strings.TrimSpace(m.inputs[nameInput].Value())
		hostname := strings.TrimSpace(m.inputs[hostnameInput].Value())
		user := strings.TrimSpace(m.inputs[userInput].Value())
		port := strings.TrimSpace(m.inputs[portInput].Value())
		identity := strings.TrimSpace(m.inputs[identityInput].Value())
		proxyJump := strings.TrimSpace(m.inputs[proxyJumpInput].Value())
		proxyCommand := strings.TrimSpace(m.inputs[proxyCommandInput].Value())
		options := strings.TrimSpace(m.inputs[optionsInput].Value())
		remoteCommand := strings.TrimSpace(m.inputs[remoteCommandInput].Value())
		requestTTY := strings.TrimSpace(m.inputs[requestTTYInput].Value())

		// Set defaults
		if user == "" {
			user = m.inputs[userInput].Placeholder
		}
		if port == "" {
			port = "22"
		}
		// Do not auto-fill identity with placeholder if left empty; keep it empty so it's optional

		// Validate all fields
		if err := validation.ValidateHost(name, hostname, port, identity); err != nil {
			return addFormSubmitMsg{err: err}
		}

		tagsStr := strings.TrimSpace(m.inputs[tagsInput].Value())
		var tags []string
		if tagsStr != "" {
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		// Create host configuration
		host := config.SSHHost{
			Name:          name,
			Hostname:      hostname,
			User:          user,
			Port:          port,
			Identity:      identity,
			ProxyJump:     proxyJump,
			ProxyCommand:  proxyCommand,
			Options:       config.ParseSSHOptionsFromCommand(options),
			RemoteCommand: remoteCommand,
			RequestTTY:    requestTTY,
			Tags:          tags,
		}

		// Add to config
		var err error
		if m.configFile != "" {
			err = config.AddSSHHostToFile(host, m.configFile)
		} else {
			err = config.AddSSHHost(host)
		}
		return addFormSubmitMsg{hostname: name, err: err}
	}
}
