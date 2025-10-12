package ui

import (
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

	inputs := make([]textinput.Model, 10) // Increased from 9 to 10 for RequestTTY

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
		styles:     styles,
		width:      width,
		height:     height,
		configFile: configFile,
	}
}

const (
	nameInput = iota
	hostnameInput
	userInput
	portInput
	identityInput
	proxyJumpInput
	optionsInput
	tagsInput
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

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Handle form submission
			if s == "enter" && m.focused == len(m.inputs)-1 {
				return m, m.submitForm()
			}

			// Cycle inputs
			if s == "up" || s == "shift+tab" {
				m.focused--
			} else {
				m.focused++
			}

			if m.focused > len(m.inputs)-1 {
				m.focused = 0
			} else if m.focused < 0 {
				m.focused = len(m.inputs) - 1
			}

			for i := range m.inputs {
				if i == m.focused {
					cmds = append(cmds, m.inputs[i].Focus())
					continue
				}
				m.inputs[i].Blur()
			}

			return m, tea.Batch(cmds...)
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

func (m *addFormModel) View() string {
	if m.success {
		return ""
	}

	var b strings.Builder

	b.WriteString(m.styles.FormTitle.Render("Add SSH Host Configuration"))
	b.WriteString("\n\n")

	fields := []string{
		"Host Name *",
		"Hostname/IP *",
		"User",
		"Port",
		"Identity File",
		"ProxyJump",
		"SSH Options",
		"Tags (comma-separated)",
		"Remote Command",
		"Request TTY",
	}

	for i, field := range fields {
		b.WriteString(m.styles.FormField.Render(field))
		b.WriteString("\n")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(m.styles.Error.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(m.styles.FormHelp.Render("Tab/Shift+Tab: navigate • Enter on last field: submit • Ctrl+S: save • Ctrl+C/Esc: cancel"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("* Required fields"))

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
