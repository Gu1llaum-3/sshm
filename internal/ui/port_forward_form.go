package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/history"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Input field indices for port forward form
const (
	pfTypeInput = iota
	pfLocalPortInput
	pfRemoteHostInput
	pfRemotePortInput
	pfBindAddressInput
)

type portForwardModel struct {
	inputs         []textinput.Model
	focused        int
	forwardType    PortForwardType
	hostName       string
	err            string
	styles         Styles
	width          int
	height         int
	configFile     string
	historyManager *history.HistoryManager
}

// portForwardSubmitMsg is sent when the port forward form is submitted
type portForwardSubmitMsg struct {
	err     error
	sshArgs []string
}

// portForwardCancelMsg is sent when the port forward form is cancelled
type portForwardCancelMsg struct{}

// NewPortForwardForm creates a new port forward form model
func NewPortForwardForm(hostName string, styles Styles, width, height int, configFile string, historyManager *history.HistoryManager) *portForwardModel {
	inputs := make([]textinput.Model, 5)

	// Forward type input (display only, controlled by arrow keys)
	inputs[pfTypeInput] = textinput.New()
	inputs[pfTypeInput].Placeholder = "Use ←/→ to change forward type"
	inputs[pfTypeInput].Focus()
	inputs[pfTypeInput].Width = 40

	// Local port input
	inputs[pfLocalPortInput] = textinput.New()
	inputs[pfLocalPortInput].Placeholder = "8080"
	inputs[pfLocalPortInput].CharLimit = 5
	inputs[pfLocalPortInput].Width = 20

	// Remote host input
	inputs[pfRemoteHostInput] = textinput.New()
	inputs[pfRemoteHostInput].Placeholder = "localhost"
	inputs[pfRemoteHostInput].CharLimit = 100
	inputs[pfRemoteHostInput].Width = 30
	inputs[pfRemoteHostInput].SetValue("localhost")

	// Remote port input
	inputs[pfRemotePortInput] = textinput.New()
	inputs[pfRemotePortInput].Placeholder = "80"
	inputs[pfRemotePortInput].CharLimit = 5
	inputs[pfRemotePortInput].Width = 20

	// Bind address input (optional)
	inputs[pfBindAddressInput] = textinput.New()
	inputs[pfBindAddressInput].Placeholder = "127.0.0.1 (optional)"
	inputs[pfBindAddressInput].CharLimit = 50
	inputs[pfBindAddressInput].Width = 30

	pf := &portForwardModel{
		inputs:         inputs,
		focused:        0,
		forwardType:    LocalForward,
		hostName:       hostName,
		styles:         styles,
		width:          width,
		height:         height,
		configFile:     configFile,
		historyManager: historyManager,
	}

	// Load previous port forwarding configuration if available
	pf.loadPreviousConfig()

	// Initialize input visibility
	pf.updateInputVisibility()

	return pf
}

func (m *portForwardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *portForwardModel) Update(msg tea.Msg) (*portForwardModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, func() tea.Msg { return portForwardCancelMsg{} }

		case "enter":
			nextField := m.getNextValidField(m.focused)
			if nextField != -1 {
				// Move to next valid input
				m.inputs[m.focused].Blur()
				m.focused = nextField
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			} else {
				// Submit form
				return m, m.submitForm()
			}

		case "shift+tab", "up":
			prevField := m.getPrevValidField(m.focused)
			if prevField != -1 {
				m.inputs[m.focused].Blur()
				m.focused = prevField
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}

		case "tab", "down":
			nextField := m.getNextValidField(m.focused)
			if nextField != -1 {
				m.inputs[m.focused].Blur()
				m.focused = nextField
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}

		case "left", "right":
			if m.focused == pfTypeInput {
				// Change forward type
				if msg.String() == "left" {
					if m.forwardType > 0 {
						m.forwardType--
					} else {
						m.forwardType = DynamicForward
					}
				} else {
					if m.forwardType < DynamicForward {
						m.forwardType++
					} else {
						m.forwardType = LocalForward
					}
				}
				m.inputs[pfTypeInput].SetValue(m.forwardType.String())
				m.updateInputVisibility()

				// Ensure focused field is valid for the new type
				validFields := m.getValidFields()
				validFocus := false
				for _, field := range validFields {
					if field == m.focused {
						validFocus = true
						break
					}
				}
				if !validFocus && len(validFields) > 0 {
					m.inputs[m.focused].Blur()
					m.focused = validFields[0]
					m.inputs[m.focused].Focus()
				}

				return m, nil
			}
		}
	}

	// Update the focused input
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m *portForwardModel) updateInputVisibility() {
	// Reset all inputs visibility
	for i := range m.inputs {
		if i != pfTypeInput {
			m.inputs[i].Placeholder = ""
		}
	}

	switch m.forwardType {
	case LocalForward:
		m.inputs[pfLocalPortInput].Placeholder = "Local port (e.g., 8080)"
		m.inputs[pfRemoteHostInput].Placeholder = "Remote host (e.g., localhost)"
		m.inputs[pfRemotePortInput].Placeholder = "Remote port (e.g., 80)"
		m.inputs[pfBindAddressInput].Placeholder = "Bind address (optional, default: 127.0.0.1)"
	case RemoteForward:
		m.inputs[pfLocalPortInput].Placeholder = "Remote port (e.g., 8080)"
		m.inputs[pfRemoteHostInput].Placeholder = "Local host (e.g., localhost)"
		m.inputs[pfRemotePortInput].Placeholder = "Local port (e.g., 80)"
		m.inputs[pfBindAddressInput].Placeholder = "Bind address (optional)"
	case DynamicForward:
		m.inputs[pfLocalPortInput].Placeholder = "SOCKS port (e.g., 1080)"
		m.inputs[pfRemoteHostInput].Placeholder = ""
		m.inputs[pfRemotePortInput].Placeholder = ""
		m.inputs[pfBindAddressInput].Placeholder = "Bind address (optional, default: 127.0.0.1)"
	}
}

func (m *portForwardModel) View() string {
	var sections []string

	// Title
	title := m.styles.Header.Render("🔗 Port Forwarding Setup")
	sections = append(sections, title)

	// Host info
	hostInfo := fmt.Sprintf("Host: %s", m.hostName)
	sections = append(sections, m.styles.HelpText.Render(hostInfo))

	// Error message
	if m.err != "" {
		sections = append(sections, m.styles.Error.Render("Error: "+m.err))
	}

	// Form fields
	var fields []string

	// Forward type
	typeLabel := "Forward Type:"
	if m.focused == pfTypeInput {
		typeLabel = m.styles.FocusedLabel.Render(typeLabel)
	} else {
		typeLabel = m.styles.Label.Render(typeLabel)
	}
	fields = append(fields, typeLabel)
	fields = append(fields, m.inputs[pfTypeInput].View())
	fields = append(fields, m.styles.HelpText.Render("Use ←/→ to change type"))

	switch m.forwardType {
	case LocalForward:
		fields = append(fields, "")
		fields = append(fields, m.styles.HelpText.Render("Local forwarding: ssh -L [bind_address:]local_port:remote_host:remote_port"))
		fields = append(fields, "")

		// Local port
		localPortLabel := "Local Port:"
		if m.focused == pfLocalPortInput {
			localPortLabel = m.styles.FocusedLabel.Render(localPortLabel)
		} else {
			localPortLabel = m.styles.Label.Render(localPortLabel)
		}
		fields = append(fields, localPortLabel)
		fields = append(fields, m.inputs[pfLocalPortInput].View())

		// Remote host
		remoteHostLabel := "Remote Host:"
		if m.focused == pfRemoteHostInput {
			remoteHostLabel = m.styles.FocusedLabel.Render(remoteHostLabel)
		} else {
			remoteHostLabel = m.styles.Label.Render(remoteHostLabel)
		}
		fields = append(fields, remoteHostLabel)
		fields = append(fields, m.inputs[pfRemoteHostInput].View())

		// Remote port
		remotePortLabel := "Remote Port:"
		if m.focused == pfRemotePortInput {
			remotePortLabel = m.styles.FocusedLabel.Render(remotePortLabel)
		} else {
			remotePortLabel = m.styles.Label.Render(remotePortLabel)
		}
		fields = append(fields, remotePortLabel)
		fields = append(fields, m.inputs[pfRemotePortInput].View())

	case RemoteForward:
		fields = append(fields, "")
		fields = append(fields, m.styles.HelpText.Render("Remote forwarding: ssh -R [bind_address:]remote_port:local_host:local_port"))
		fields = append(fields, "")

		// Remote port
		remotePortLabel := "Remote Port:"
		if m.focused == pfLocalPortInput {
			remotePortLabel = m.styles.FocusedLabel.Render(remotePortLabel)
		} else {
			remotePortLabel = m.styles.Label.Render(remotePortLabel)
		}
		fields = append(fields, remotePortLabel)
		fields = append(fields, m.inputs[pfLocalPortInput].View())

		// Local host
		localHostLabel := "Local Host:"
		if m.focused == pfRemoteHostInput {
			localHostLabel = m.styles.FocusedLabel.Render(localHostLabel)
		} else {
			localHostLabel = m.styles.Label.Render(localHostLabel)
		}
		fields = append(fields, localHostLabel)
		fields = append(fields, m.inputs[pfRemoteHostInput].View())

		// Local port
		localPortLabel := "Local Port:"
		if m.focused == pfRemotePortInput {
			localPortLabel = m.styles.FocusedLabel.Render(localPortLabel)
		} else {
			localPortLabel = m.styles.Label.Render(localPortLabel)
		}
		fields = append(fields, localPortLabel)
		fields = append(fields, m.inputs[pfRemotePortInput].View())

	case DynamicForward:
		fields = append(fields, "")
		fields = append(fields, m.styles.HelpText.Render("Dynamic forwarding (SOCKS proxy): ssh -D [bind_address:]port"))
		fields = append(fields, "")

		// SOCKS port
		socksPortLabel := "SOCKS Port:"
		if m.focused == pfLocalPortInput {
			socksPortLabel = m.styles.FocusedLabel.Render(socksPortLabel)
		} else {
			socksPortLabel = m.styles.Label.Render(socksPortLabel)
		}
		fields = append(fields, socksPortLabel)
		fields = append(fields, m.inputs[pfLocalPortInput].View())
	}

	// Bind address (for all types)
	fields = append(fields, "")
	bindLabel := "Bind Address (optional):"
	if m.focused == pfBindAddressInput {
		bindLabel = m.styles.FocusedLabel.Render(bindLabel)
	} else {
		bindLabel = m.styles.Label.Render(bindLabel)
	}
	fields = append(fields, bindLabel)
	fields = append(fields, m.inputs[pfBindAddressInput].View())

	// Join form fields
	formContent := lipgloss.JoinVertical(lipgloss.Left, fields...)
	sections = append(sections, formContent)

	// Help text
	helpText := " Tab/↓: next field • Shift+Tab/↑: previous field • Enter: connect • Esc: cancel"
	sections = append(sections, m.styles.HelpText.Render(helpText))

	// Join all sections
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Center the form
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.styles.FormContainer.Render(content),
	)
}

func (m *portForwardModel) submitForm() tea.Cmd {
	return func() tea.Msg {
		// Validate inputs
		localPort := strings.TrimSpace(m.inputs[pfLocalPortInput].Value())
		if localPort == "" {
			return portForwardSubmitMsg{err: fmt.Errorf("port is required"), sshArgs: nil}
		}

		// Validate port number
		if _, err := strconv.Atoi(localPort); err != nil {
			return portForwardSubmitMsg{err: fmt.Errorf("invalid port number"), sshArgs: nil}
		}

		// Get form values for saving to history
		remoteHost := strings.TrimSpace(m.inputs[pfRemoteHostInput].Value())
		remotePort := strings.TrimSpace(m.inputs[pfRemotePortInput].Value())
		bindAddress := strings.TrimSpace(m.inputs[pfBindAddressInput].Value())

		// Build SSH command with port forwarding
		var sshArgs []string

		// Add config file if specified
		if m.configFile != "" {
			sshArgs = append(sshArgs, "-F", m.configFile)
		}

		// Add forwarding arguments
		var forwardTypeStr string
		switch m.forwardType {
		case LocalForward:
			forwardTypeStr = "local"
			if remoteHost == "" {
				remoteHost = "localhost"
			}
			if remotePort == "" {
				return portForwardSubmitMsg{err: fmt.Errorf("remote port is required for local forwarding"), sshArgs: nil}
			}

			// Validate remote port
			if _, err := strconv.Atoi(remotePort); err != nil {
				return portForwardSubmitMsg{err: fmt.Errorf("invalid remote port number"), sshArgs: nil}
			}

			// Build -L argument
			var forwardArg string
			if bindAddress != "" {
				forwardArg = fmt.Sprintf("%s:%s:%s:%s", bindAddress, localPort, remoteHost, remotePort)
			} else {
				forwardArg = fmt.Sprintf("%s:%s:%s", localPort, remoteHost, remotePort)
			}
			sshArgs = append(sshArgs, "-L", forwardArg)

		case RemoteForward:
			forwardTypeStr = "remote"
			if remoteHost == "" {
				remoteHost = "localhost"
			}
			if remotePort == "" {
				return portForwardSubmitMsg{err: fmt.Errorf("local port is required for remote forwarding"), sshArgs: nil}
			}

			// Validate local port
			if _, err := strconv.Atoi(remotePort); err != nil {
				return portForwardSubmitMsg{err: fmt.Errorf("invalid local port number"), sshArgs: nil}
			}

			// Build -R argument (note: localPort is actually the remote port in this context)
			var forwardArg string
			if bindAddress != "" {
				forwardArg = fmt.Sprintf("%s:%s:%s:%s", bindAddress, localPort, remoteHost, remotePort)
			} else {
				forwardArg = fmt.Sprintf("%s:%s:%s", localPort, remoteHost, remotePort)
			}
			sshArgs = append(sshArgs, "-R", forwardArg)

		case DynamicForward:
			forwardTypeStr = "dynamic"
			// Build -D argument
			var forwardArg string
			if bindAddress != "" {
				forwardArg = fmt.Sprintf("%s:%s", bindAddress, localPort)
			} else {
				forwardArg = localPort
			}
			sshArgs = append(sshArgs, "-D", forwardArg)
		}

		// Save port forwarding configuration to history
		if m.historyManager != nil {
			if err := m.historyManager.RecordPortForwarding(
				m.hostName,
				forwardTypeStr,
				localPort,
				remoteHost,
				remotePort,
				bindAddress,
			); err != nil {
				// Log the error but don't fail the connection
				// In a production environment, you might want to handle this differently
			}
		}

		// Add hostname
		sshArgs = append(sshArgs, m.hostName)

		// Return success with the SSH command to execute
		return portForwardSubmitMsg{err: nil, sshArgs: sshArgs}
	}
}

// getValidFields returns the list of valid field indices for the current forward type
func (m *portForwardModel) getValidFields() []int {
	switch m.forwardType {
	case LocalForward:
		return []int{pfTypeInput, pfLocalPortInput, pfRemoteHostInput, pfRemotePortInput, pfBindAddressInput}
	case RemoteForward:
		return []int{pfTypeInput, pfLocalPortInput, pfRemoteHostInput, pfRemotePortInput, pfBindAddressInput}
	case DynamicForward:
		return []int{pfTypeInput, pfLocalPortInput, pfBindAddressInput}
	default:
		return []int{pfTypeInput, pfLocalPortInput, pfRemoteHostInput, pfRemotePortInput, pfBindAddressInput}
	}
}

// getNextValidField returns the next valid field index, or -1 if none
func (m *portForwardModel) getNextValidField(currentField int) int {
	validFields := m.getValidFields()

	for i, field := range validFields {
		if field == currentField && i < len(validFields)-1 {
			return validFields[i+1]
		}
	}
	return -1
}

// getPrevValidField returns the previous valid field index, or -1 if none
func (m *portForwardModel) getPrevValidField(currentField int) int {
	validFields := m.getValidFields()

	for i, field := range validFields {
		if field == currentField && i > 0 {
			return validFields[i-1]
		}
	}
	return -1
}

// loadPreviousConfig loads the previous port forwarding configuration for this host
func (m *portForwardModel) loadPreviousConfig() {
	if m.historyManager == nil {
		m.inputs[pfTypeInput].SetValue("Local (-L)")
		return
	}

	config := m.historyManager.GetPortForwardingConfig(m.hostName)
	if config == nil {
		m.inputs[pfTypeInput].SetValue("Local (-L)")
		return
	}

	// Set forward type based on saved configuration
	switch config.Type {
	case "local":
		m.forwardType = LocalForward
	case "remote":
		m.forwardType = RemoteForward
	case "dynamic":
		m.forwardType = DynamicForward
	default:
		m.forwardType = LocalForward
	}
	m.inputs[pfTypeInput].SetValue(m.forwardType.String())

	// Set values from saved configuration
	if config.LocalPort != "" {
		m.inputs[pfLocalPortInput].SetValue(config.LocalPort)
	}
	if config.RemoteHost != "" {
		m.inputs[pfRemoteHostInput].SetValue(config.RemoteHost)
	} else if m.forwardType != DynamicForward {
		// Default to localhost for local and remote forwarding if not set
		m.inputs[pfRemoteHostInput].SetValue("localhost")
	}
	if config.RemotePort != "" {
		m.inputs[pfRemotePortInput].SetValue(config.RemotePort)
	}
	if config.BindAddress != "" {
		m.inputs[pfBindAddressInput].SetValue(config.BindAddress)
	}
}
