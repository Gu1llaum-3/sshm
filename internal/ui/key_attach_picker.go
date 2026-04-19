package ui

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
	keypkg "github.com/Gu1llaum-3/sshm/internal/key"

	tea "github.com/charmbracelet/bubbletea"
)

type keyAttachHostPickerModel struct {
	hosts       []config.SSHHost
	selected    int
	styles      Styles
	width       int
	height      int
	loading     bool
	err         string
	configFile  string
	title       string
	description string
	actionLabel string
}

type keyAttachHostsLoadedMsg struct {
	hosts []config.SSHHost
	err   error
}

type keyAttachHostSelectMsg struct {
	host config.SSHHost
}

type keyAttachHostCancelMsg struct{}

type keyAttachFinishedMsg struct {
	hostName string
	path     string
	err      error
}

type keyDeployFinishedMsg struct {
	hostName string
	path     string
	err      error
}

var loadAttachHostsForUI = func(configFile string) ([]config.SSHHost, error) {
	var (
		hosts []config.SSHHost
		err   error
	)

	if configFile == "" {
		hosts, err = config.ParseSSHConfig()
	} else {
		hosts, err = config.ParseSSHConfigFile(configFile)
	}
	if err != nil {
		return nil, err
	}

	slices.SortFunc(hosts, func(a, b config.SSHHost) int {
		if diff := cmp.Compare(a.Name, b.Name); diff != 0 {
			return diff
		}
		if diff := cmp.Compare(a.SourceFile, b.SourceFile); diff != 0 {
			return diff
		}
		return cmp.Compare(a.LineNumber, b.LineNumber)
	})

	return hosts, nil
}

var runAttachKeyForUI = func(host config.SSHHost, path string) tea.Cmd {
	return func() tea.Msg {
		_, err := keypkg.AttachToConcreteHost(host, path, false)
		return keyAttachFinishedMsg{
			hostName: host.Name,
			path:     path,
			err:      err,
		}
	}
}

var runDeployKeyForUI = func(host config.SSHHost, path string, configFile string) tea.Cmd {
	target := strings.TrimSpace(host.Hostname)
	if target == "" {
		target = host.Name
	}

	return runUIDeployCommand(keypkg.DeployOptions{
		Target:       target,
		User:         strings.TrimSpace(host.User),
		Port:         strings.TrimSpace(host.Port),
		ProxyJump:    strings.TrimSpace(host.ProxyJump),
		ProxyCommand: strings.TrimSpace(host.ProxyCommand),
		Identity:     path,
		ConfigPath:   configFile,
	}, func(err error) tea.Msg {
		return keyDeployFinishedMsg{
			hostName: host.Name,
			path:     path,
			err:      err,
		}
	})
}

func newKeyAttachHostPicker(styles Styles, width, height int, configFile string) *keyAttachHostPickerModel {
	return &keyAttachHostPickerModel{
		styles:      styles,
		width:       width,
		height:      height,
		loading:     true,
		configFile:  configFile,
		title:       "SSHM - Attach Key",
		description: "Choose a host to attach this key to.",
		actionLabel: "attach",
	}
}

func newKeyDeployHostPicker(styles Styles, width, height int, configFile string) *keyAttachHostPickerModel {
	return &keyAttachHostPickerModel{
		styles:      styles,
		width:       width,
		height:      height,
		loading:     true,
		configFile:  configFile,
		title:       "SSHM - Deploy Key",
		description: "Choose a host to copy this public key to.",
		actionLabel: "deploy",
	}
}

func (m *keyAttachHostPickerModel) Init() tea.Cmd {
	return loadAttachHostsCmd(m.configFile)
}

func (m *keyAttachHostPickerModel) Update(msg tea.Msg) (*keyAttachHostPickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		return m, nil
	case keyAttachHostsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.hosts = nil
			m.selected = 0
			return m, nil
		}
		m.err = ""
		m.hosts = msg.hosts
		if len(m.hosts) == 0 {
			m.selected = 0
			return m, nil
		}
		if m.selected >= len(m.hosts) {
			m.selected = len(m.hosts) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q", "left":
			return m, func() tea.Msg { return keyAttachHostCancelMsg{} }
		case "up", "k":
			if !m.loading && m.selected > 0 {
				m.selected--
			}
			return m, nil
		case "down", "j":
			if !m.loading && m.selected < len(m.hosts)-1 {
				m.selected++
			}
			return m, nil
		case "r":
			m.loading = true
			m.err = ""
			return m, loadAttachHostsCmd(m.configFile)
		case "enter":
			if m.loading || len(m.hosts) == 0 {
				return m, nil
			}
			return m, func() tea.Msg { return keyAttachHostSelectMsg{host: m.hosts[m.selected]} }
		}
	}
	return m, nil
}

func (m *keyAttachHostPickerModel) View() string {
	var b strings.Builder
	modalWidth := clampModalWidth(m.width, 92)
	const listIndent = "  "

	b.WriteString(m.styles.Header.Render(listIndent + m.title))
	b.WriteString("\n\n")
	b.WriteString(m.styles.FormHelp.Render(listIndent + m.description))
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(m.styles.ErrorText.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	if m.loading {
		b.WriteString(m.styles.FormField.Render("Loading SSH hosts..."))
		b.WriteString("\n")
		b.WriteString(m.styles.FormHelp.Render("Scanning SSH config."))
		b.WriteString("\n\n")
		b.WriteString(m.styles.FormHelp.Render("esc/← back"))
		return renderKeyPickerModal(m.width, m.height, m.styles, "Key details", b.String(), modalWidth)
	}

	if len(m.hosts) == 0 {
		b.WriteString(m.styles.FormField.Render("No SSH hosts found."))
		b.WriteString("\n\n")
		b.WriteString(m.styles.FormHelp.Render("r refresh • esc/← back"))
		return renderKeyPickerModal(m.width, m.height, m.styles, "Key details", b.String(), modalWidth)
	}

	nameWidth, hostWidth, fileWidth := attachHostColumnWidths(m.hosts, modalWidth)
	header := fmt.Sprintf("%s%-*s  %-*s  %-*s",
		listIndent,
		nameWidth, "Host",
		hostWidth, "Hostname",
		fileWidth, "Config",
	)
	b.WriteString(m.styles.FocusedLabel.Render(header))
	b.WriteString("\n\n")

	for i, host := range m.hosts {
		row := fmt.Sprintf("%-*s  %-*s  %-*s",
			nameWidth, truncateText(host.Name, nameWidth),
			hostWidth, truncateText(emptyFallback(host.Hostname, "-"), hostWidth),
			fileWidth, truncateText(formatConfigFile(host.SourceFile), fileWidth),
		)
		if i == m.selected {
			b.WriteString(m.styles.Selected.Render("> " + row))
		} else {
			b.WriteString(listIndent + row)
		}
		b.WriteString("\n")
	}

	selected := m.hosts[m.selected]
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Host", selected.Name, modalWidth)))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Hostname", emptyFallback(selected.Hostname, "-"), modalWidth)))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Config", formatConfigFile(selected.SourceFile), modalWidth)))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Line", fmt.Sprintf("%d", selected.LineNumber), modalWidth)))
	b.WriteString("\n\n")
	b.WriteString(m.styles.FormHelp.Render(listIndent + "j/k or ↑/↓ select • enter " + m.actionLabel + " • r refresh • esc/← back"))

	return renderKeyPickerModal(m.width, m.height, m.styles, "Key details", b.String(), modalWidth)
}

func loadAttachHostsCmd(configFile string) tea.Cmd {
	return func() tea.Msg {
		hosts, err := loadAttachHostsForUI(configFile)
		return keyAttachHostsLoadedMsg{hosts: hosts, err: err}
	}
}

func attachHostColumnWidths(hosts []config.SSHHost, modalWidth int) (name int, hostname int, file int) {
	name = len("Host")
	hostname = len("Hostname")
	file = len("Config")

	for _, host := range hosts {
		name = max(name, len(host.Name))
		hostname = max(hostname, len(emptyFallback(host.Hostname, "-")))
		file = max(file, len(formatConfigFile(host.SourceFile)))
	}

	totalPadding := 10
	available := max(20, modalWidth-totalPadding)
	name = min(name, max(10, available/5))
	hostname = min(hostname, max(12, available/4))
	file = min(file, max(14, available-name-hostname-6))
	return name, hostname, file
}
