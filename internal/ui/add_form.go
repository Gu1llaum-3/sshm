package ui

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
	keypkg "github.com/Gu1llaum-3/sshm/internal/key"
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
	keyPicker  *keyPickerModel
	keyCreate  *keyCreateFormModel
}

type addFormField struct {
	index int
	label string
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
	optionsInput
	tagsInput
	// Advanced tab inputs
	remoteCommandInput
	requestTTYInput
)

var (
	addGeneralInputs  = [...]int{nameInput, hostnameInput, userInput, portInput, identityInput, proxyJumpInput, proxyCommandInput, tagsInput}
	addAdvancedInputs = [...]int{optionsInput, remoteCommandInput, requestTTYInput}
)

// Messages for communication with parent model
type addFormSubmitMsg struct {
	hostname string
	err      error
}

type addFormDeployFinishedMsg struct {
	err error
}

type addFormCancelMsg struct{}

var runAddFormDeploy = func(opts keypkg.DeployOptions) tea.Cmd {
	return runUIDeployCommand(opts, func(err error) tea.Msg {
		return addFormDeployFinishedMsg{err: err}
	})
}

func (m *addFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *addFormModel) Update(msg tea.Msg) (*addFormModel, tea.Cmd) {
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
			m.inputs[identityInput].SetValue(msg.path)
			m.focused = identityInput
			m.currentTab = tabGeneral
			return m, m.updateFocus()
		case keyCreateCancelMsg:
			m.keyCreate = nil
			m.focused = identityInput
			m.currentTab = tabGeneral
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
			m.inputs[identityInput].SetValue(msg.path)
			m.focused = identityInput
			m.currentTab = tabGeneral
			return m, m.updateFocus()
		case keyPickerCancelMsg:
			m.keyPicker = nil
			m.focused = identityInput
			m.currentTab = tabGeneral
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
		m.styles = NewStyles(m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg { return addFormCancelMsg{} }

		case "ctrl+s":
			// Allow submission from any field with Ctrl+S (Save)
			return m, m.submitForm()

		case "ctrl+d":
			return m, m.deployAndSave()

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

		case "ctrl+g":
			if m.currentTab == tabGeneral && m.focused == identityInput {
				m.keyCreate = NewKeyCreateForm(m.styles, m.width, m.height)
				return m, m.keyCreate.Init()
			}

		case "ctrl+o":
			if m.currentTab == tabGeneral && m.focused == identityInput {
				return m, m.openKeyPicker()
			}

		case "enter":
			if m.currentTab == tabGeneral && m.focused == identityInput {
				return m, m.openKeyPicker()
			}
			return m, m.handleNavigation(msg.String())

		case "tab", "shift+tab", "up", "down":
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
	case addFormDeployFinishedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		return m, m.submitForm()
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
		return addGeneralInputs[:]
	case tabAdvanced:
		return addAdvancedInputs[:]
	default:
		return addGeneralInputs[:]
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

	currentPos := slices.Index(currentTabInputs, m.focused)
	if currentPos < 0 {
		currentPos = 0
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

	// Handle transitions between tabs
	if currentPos >= len(currentTabInputs) {
		// Move to next tab
		if m.currentTab == tabGeneral {
			// Move to advanced tab
			m.currentTab = tabAdvanced
			m.focused = m.getFirstInputForTab(tabAdvanced)
			return m.updateFocus()
		} else {
			// Wrap around to first field of current tab
			currentPos = 0
		}
	} else if currentPos < 0 {
		// Move to previous tab
		if m.currentTab == tabAdvanced {
			// Move to general tab
			m.currentTab = tabGeneral
			currentTabInputs = m.getInputsForCurrentTab()
			currentPos = len(currentTabInputs) - 1
		} else {
			// Wrap around to last field of current tab
			currentPos = len(currentTabInputs) - 1
		}
	}

	m.focused = currentTabInputs[currentPos]
	return m.updateFocus()
}

func (m *addFormModel) View() string {
	if m.success {
		return ""
	}

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
	b.WriteString(m.styles.FormHelp.Render("Identity File: Enter/Ctrl+O choose key • Ctrl+G generate • Enter on last field: submit"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("Ctrl+S: save • Ctrl+D: deploy selected key then save • Esc: cancel"))
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

func (m *addFormModel) getMinimumCompactHeight() int {
	return 10
}

// isHeightSufficient checks if the current terminal height is sufficient
func (m *addFormModel) isHeightSufficient() bool {
	return m.height >= m.getMinimumHeight()
}

// renderHeightWarning renders a warning message when height is insufficient
func (m *addFormModel) renderHeightWarning() string {
	required := m.getMinimumCompactHeight()
	current := m.height

	warning := m.styles.ErrorText.Render("⚠️  Terminal height is too small!")
	details := m.styles.FormField.Render(fmt.Sprintf("Current: %d lines, Need at least: %d lines", current, required))
	instruction := m.styles.FormHelp.Render("Compact mode is available, but this terminal is still too short.")
	instruction2 := m.styles.FormHelp.Render("Resize the terminal or cancel with Esc/Ctrl+C.")

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

	fields := m.generalTabFields()

	for _, field := range fields {
		fieldStyle := m.styles.FormField
		if m.focused == field.index {
			fieldStyle = m.styles.FocusedLabel
		}
		b.WriteString(fieldStyle.Render(field.label))
		b.WriteString("\n")
		b.WriteString(m.inputs[field.index].View())
		b.WriteString("\n")
		if field.index == tagsInput && m.focused == tagsInput {
			b.WriteString(m.styles.FormHelp.Render(`  tip: use "hidden" to hide this host from the list`))
			b.WriteString("\n")
		}
		if field.index == identityInput && m.focused == identityInput {
			b.WriteString(m.styles.FormHelp.Render("  tip: Enter/Ctrl+O choose a key • Ctrl+G generates one • keep typing for a custom path"))
			b.WriteString("\n")
		}
		if field.index == hostnameInput && m.focused == hostnameInput {
			b.WriteString(m.styles.FormHelp.Render("  tip: Ctrl+D deploys the selected key to this host and saves on success"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderAdvancedTab renders the advanced tab content
func (m *addFormModel) renderAdvancedTab() string {
	var b strings.Builder

	fields := m.advancedTabFields()

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

func (m *addFormModel) renderCompactView() string {
	var b strings.Builder

	b.WriteString(m.styles.FormTitle.Render("Add SSH Host Configuration"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("Compact mode"))
	b.WriteString("\n\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	fields := m.fieldsForCurrentTab()
	currentField, currentPos := m.currentField(fields)

	b.WriteString(m.styles.FormHelp.Render(fmt.Sprintf("Field %d/%d", currentPos+1, len(fields))))
	b.WriteString("\n")
	fieldStyle := m.styles.FocusedLabel
	b.WriteString(fieldStyle.Render(currentField.label))
	b.WriteString("\n")
	b.WriteString(m.inputs[currentField.index].View())
	b.WriteString("\n")

	if currentField.index == tagsInput {
		b.WriteString(m.styles.FormHelp.Render(`tip: use "hidden" to hide this host from the list`))
		b.WriteString("\n")
	}
	if currentField.index == identityInput {
		b.WriteString(m.styles.FormHelp.Render("tip: Enter/Ctrl+O choose • Ctrl+G generate • type manually for a custom path"))
		b.WriteString("\n")
	}
	if currentField.index == hostnameInput {
		b.WriteString(m.styles.FormHelp.Render("tip: Ctrl+D deploys the selected key and saves only if deploy succeeds"))
		b.WriteString("\n")
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.ErrorText.Render("Error: " + m.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("↑/↓ or Tab move • Ctrl+S save • Ctrl+D deploy+save • Ctrl+J/K tabs • Enter submit on last field • Esc cancel"))
	return b.String()
}

func (m *addFormModel) generalTabFields() []addFormField {
	return []addFormField{
		{nameInput, "Host Name *"},
		{hostnameInput, "Hostname/IP *"},
		{userInput, "User"},
		{portInput, "Port"},
		{identityInput, "Identity File"},
		{proxyJumpInput, "ProxyJump"},
		{proxyCommandInput, "ProxyCommand"},
		{tagsInput, "Tags (comma-separated)"},
	}
}

func (m *addFormModel) advancedTabFields() []addFormField {
	return []addFormField{
		{optionsInput, "SSH Options"},
		{remoteCommandInput, "Remote Command"},
		{requestTTYInput, "Request TTY"},
	}
}

func (m *addFormModel) fieldsForCurrentTab() []addFormField {
	if m.currentTab == tabAdvanced {
		return m.advancedTabFields()
	}
	return m.generalTabFields()
}

func (m *addFormModel) currentField(fields []addFormField) (addFormField, int) {
	for i, field := range fields {
		if field.index == m.focused {
			return field, i
		}
	}
	return fields[0], 0
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

		tags := parseCommaList(m.inputs[tagsInput].Value())

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

func (m *addFormModel) deployAndSave() tea.Cmd {
	target := strings.TrimSpace(m.inputs[hostnameInput].Value())
	if target == "" {
		m.err = "hostname/IP is required before deploy"
		return nil
	}
	if !validation.ValidateHostname(target) && !validation.ValidateIP(target) {
		m.err = "invalid hostname or IP address format"
		return nil
	}

	port := strings.TrimSpace(m.inputs[portInput].Value())
	if port == "" {
		port = "22"
	}
	if !validation.ValidatePort(port) {
		m.err = "port must be between 1 and 65535"
		return nil
	}

	identity := strings.TrimSpace(m.inputs[identityInput].Value())
	if identity == "" {
		m.err = "choose or generate a key before deploy"
		return nil
	}

	user := strings.TrimSpace(m.inputs[userInput].Value())
	if user == "" {
		user = strings.TrimSpace(m.inputs[userInput].Placeholder)
	}

	m.err = ""
	return runAddFormDeploy(keypkg.DeployOptions{
		Target:       target,
		User:         user,
		Port:         port,
		ProxyJump:    strings.TrimSpace(m.inputs[proxyJumpInput].Value()),
		ProxyCommand: strings.TrimSpace(m.inputs[proxyCommandInput].Value()),
		Identity:     identity,
	})
}

func (m *addFormModel) openKeyPicker() tea.Cmd {
	contextLine := "For new host"
	name := strings.TrimSpace(m.inputs[nameInput].Value())
	if name != "" {
		contextLine = fmt.Sprintf("For %s", name)
	}

	picker, err := NewKeyPicker("Add Host", contextLine, m.styles, m.width, m.height, m.configFile)
	if err != nil {
		m.err = err.Error()
		return nil
	}

	m.err = ""
	m.keyPicker = picker
	return m.keyPicker.Init()
}

func parseCommaList(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}
