package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/config"
	keypkg "github.com/Gu1llaum-3/sshm/internal/key"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var loadKeyInventoryForUI = func(ctx context.Context, configFile string) ([]keypkg.InventoryItem, error) {
	return keypkg.Inventory(ctx, keypkg.ExecRunner{}, configFile)
}

type keyPickerSelectMsg struct {
	path string
}

type keyPickerCancelMsg struct{}

type keyPickerCreateMsg struct{}

type keyInventoryLoadedMsg struct {
	items []keypkg.InventoryItem
	err   error
}

type keyCreateFinishedMsg struct {
	path string
	err  error
}

type keyCreateCancelMsg struct{}

type keysViewCloseMsg struct{}

type keyPublicKeyLoadedMsg struct {
	content string
	err     error
}

type keyPublicKeyCopiedMsg struct {
	err error
}

type keyPathCopiedMsg struct {
	err error
}

type keyPathRevealedMsg struct {
	path string
	err  error
}

type keyDeleteFinishedMsg struct {
	path string
	err  error
}

type keyAgentAddedMsg struct {
	path string
	err  error
}

type keyOpenHostInfoMsg struct {
	hostName   string
	configFile string
}

type keyOpenHostEditMsg struct {
	hostName   string
	configFile string
}

type keyPickerMode int

const (
	keyPickerModeSelect keyPickerMode = iota
	keyPickerModeBrowse
)

type keyPickerModel struct {
	items      []keypkg.InventoryItem
	selected   int
	styles     Styles
	width      int
	height     int
	backLabel  string
	title      string
	configFile string
	err        string
	loading    bool
	mode       keyPickerMode
}

type keysViewModel struct {
	browser       *keyPickerModel
	keyCreate     *keyCreateFormModel
	attachPicker  *keyAttachHostPickerModel
	deployPicker  *keyAttachHostPickerModel
	styles        Styles
	width         int
	height        int
	configFile    string
	inspect       bool
	showPublic    bool
	publicKey     string
	status        string
	statusErr     bool
	hostRefs      bool
	refSelected   int
	deleteDialog  bool
	deleteBlocked bool
}

var readPublicKeyFile = os.ReadFile

var copyToClipboard = func(text string) error {
	return copyTextToClipboard(text)
}

var revealPathInManager = func(path string) error {
	return revealPath(path)
}

var deleteKeyFiles = func(privatePath string, publicPath string) error {
	if err := os.Remove(privatePath); err != nil {
		return err
	}
	if strings.TrimSpace(publicPath) == "" {
		return nil
	}
	if err := os.Remove(publicPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func NewKeyPicker(backLabel string, title string, styles Styles, width, height int, configFile string) (*keyPickerModel, error) {
	return &keyPickerModel{
		styles:     styles,
		width:      width,
		height:     height,
		backLabel:  backLabel,
		title:      title,
		configFile: configFile,
		loading:    true,
		mode:       keyPickerModeSelect,
	}, nil
}

func NewKeyBrowser(backLabel string, styles Styles, width, height int, configFile string) *keyPickerModel {
	return &keyPickerModel{
		styles:     styles,
		width:      width,
		height:     height,
		backLabel:  backLabel,
		title:      "SSH keys and config references.",
		configFile: configFile,
		loading:    true,
		mode:       keyPickerModeBrowse,
	}
}

func NewKeysView(styles Styles, width, height int, configFile string) *keysViewModel {
	return &keysViewModel{
		browser:    NewKeyBrowser("Main screen", styles, width, height, configFile),
		styles:     styles,
		width:      width,
		height:     height,
		configFile: configFile,
	}
}

func (m *keysViewModel) Init() tea.Cmd {
	if m.browser == nil {
		m.browser = NewKeyBrowser("Main screen", m.styles, m.width, m.height, m.configFile)
	}
	return m.browser.Init()
}

func (m *keysViewModel) Update(msg tea.Msg) (*keysViewModel, tea.Cmd) {
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
			m.browser = NewKeyBrowser("Main screen", m.styles, m.width, m.height, m.configFile)
			m.browser.title = fmt.Sprintf("Generated %s. SSH keys and config references.", filepath.Base(msg.path))
			return m, m.browser.Init()
		case keyCreateCancelMsg:
			m.keyCreate = nil
			if m.browser == nil {
				m.browser = NewKeyBrowser("Main screen", m.styles, m.width, m.height, m.configFile)
			}
			return m, nil
		}

		newForm, cmd := m.keyCreate.Update(msg)
		m.keyCreate = newForm
		return m, cmd
	}

	if m.browser == nil {
		m.browser = NewKeyBrowser("Main screen", m.styles, m.width, m.height, m.configFile)
	}

	switch msg := msg.(type) {
	case keyAttachFinishedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			m.attachPicker = nil
			return m, nil
		}
		m.status = fmt.Sprintf("Attached %s to %s.", filepath.Base(msg.path), msg.hostName)
		m.statusErr = false
		m.attachPicker = nil
		return m, loadKeyInventoryCmd(m.configFile)
	case keyDeployFinishedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			m.deployPicker = nil
			return m, nil
		}
		m.status = fmt.Sprintf("Deployed %s to %s.", filepath.Base(msg.path), msg.hostName)
		m.statusErr = false
		m.deployPicker = nil
		return m, nil
	case keyPublicKeyLoadedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.publicKey = msg.content
		m.showPublic = true
		m.status = "Public key shown."
		m.statusErr = false
		return m, nil
	case keyPublicKeyCopiedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = "Copied public key."
		m.statusErr = false
		return m, nil
	case keyPathCopiedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = "Copied key path."
		m.statusErr = false
		return m, nil
	case keyPathRevealedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf("Revealed %s in file manager.", filepath.Base(msg.path))
		m.statusErr = false
		return m, nil
	case keyDeleteFinishedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf("Deleted %s.", filepath.Base(msg.path))
		m.statusErr = false
		m.inspect = false
		m.showPublic = false
		m.publicKey = ""
		m.deleteDialog = false
		m.deleteBlocked = false
		m.browser = NewKeyBrowser("Main screen", m.styles, m.width, m.height, m.configFile)
		return m, m.browser.Init()
	case keyAgentAddedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.status = fmt.Sprintf("Added %s to ssh-agent.", filepath.Base(msg.path))
		m.statusErr = false
		return m, nil
	}

	if m.deleteDialog {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "ctrl+c", "esc", "left", "q":
				m.deleteDialog = false
				m.deleteBlocked = false
				return m, nil
			case "enter":
				if m.deleteBlocked {
					m.deleteDialog = false
					m.deleteBlocked = false
					return m, nil
				}
				item := m.selectedItem()
				if item == nil {
					m.deleteDialog = false
					return m, nil
				}
				return m, deleteKeyCmd(item.Path, item.PublicKeyPath)
			}
		}
		return m, nil
	}

	if m.attachPicker != nil {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.styles = NewStyles(m.width)
			m.browser.width = m.width
			m.browser.height = m.height
			m.browser.styles = m.styles
			m.attachPicker.width = m.width
			m.attachPicker.height = m.height
			m.attachPicker.styles = m.styles
			return m, nil
		case keyAttachHostSelectMsg:
			item := m.selectedItem()
			if item == nil {
				m.attachPicker = nil
				return m, nil
			}
			return m, runAttachKeyForUI(msg.host, item.Path)
		case keyAttachHostCancelMsg:
			m.attachPicker = nil
			return m, nil
		}

		newPicker, cmd := m.attachPicker.Update(msg)
		m.attachPicker = newPicker
		return m, cmd
	}

	if m.deployPicker != nil {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.styles = NewStyles(m.width)
			m.browser.width = m.width
			m.browser.height = m.height
			m.browser.styles = m.styles
			m.deployPicker.width = m.width
			m.deployPicker.height = m.height
			m.deployPicker.styles = m.styles
			return m, nil
		case keyAttachHostSelectMsg:
			item := m.selectedItem()
			if item == nil {
				m.deployPicker = nil
				return m, nil
			}
			return m, runDeployKeyForUI(msg.host, item.Path, m.configFile)
		case keyAttachHostCancelMsg:
			m.deployPicker = nil
			return m, nil
		}

		newPicker, cmd := m.deployPicker.Update(msg)
		m.deployPicker = newPicker
		return m, cmd
	}

	if m.hostRefs {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.styles = NewStyles(m.width)
			m.browser.width = m.width
			m.browser.height = m.height
			m.browser.styles = m.styles
			return m, nil
		case tea.KeyMsg:
			item := m.selectedItem()
			if item == nil {
				m.hostRefs = false
				return m, nil
			}

			switch msg.String() {
			case "ctrl+c", "esc", "q", "left":
				m.hostRefs = false
				return m, nil
			case "up", "k":
				if m.refSelected > 0 {
					m.refSelected--
				}
				return m, nil
			case "down", "j":
				if m.refSelected < len(item.References)-1 {
					m.refSelected++
				}
				return m, nil
			case "enter":
				if len(item.References) == 0 {
					return m, nil
				}
				ref := item.References[m.refSelected]
				return m, func() tea.Msg {
					return keyOpenHostInfoMsg{hostName: ref.Host, configFile: ref.SourceFile}
				}
			case "e":
				if len(item.References) == 0 {
					return m, nil
				}
				ref := item.References[m.refSelected]
				return m, func() tea.Msg {
					return keyOpenHostEditMsg{hostName: ref.Host, configFile: ref.SourceFile}
				}
			}
		}
		newBrowser, cmd := m.browser.Update(msg)
		m.browser = newBrowser
		return m, cmd
	}

	if m.inspect {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			m.styles = NewStyles(m.width)
			m.browser.width = m.width
			m.browser.height = m.height
			m.browser.styles = m.styles
			return m, nil
		case tea.KeyMsg:
			item := m.selectedItem()
			if item == nil {
				m.inspect = false
				return m, nil
			}

			switch msg.String() {
			case "ctrl+c", "esc", "q", "left":
				m.showPublic = false
				m.publicKey = ""
				m.status = ""
				m.statusErr = false
				m.inspect = false
				return m, nil
			case "h":
				if len(item.References) == 0 {
					m.status = "No config references for this key."
					m.statusErr = false
					return m, nil
				}
				m.hostRefs = true
				m.refSelected = 0
				return m, nil
			case "p":
				if m.showPublic {
					m.showPublic = false
					return m, nil
				}
				return m, loadPublicKeyCmd(item.PublicKeyPath)
			case "c":
				return m, copyPublicKeyCmd(item.PublicKeyPath)
			case "y":
				return m, copyPathCmd(item.Path)
			case "o":
				return m, revealPathCmd(item.Path)
			case "a":
				return m, addKeyToAgentCmd(item.Path)
			case "t":
				m.attachPicker = newKeyAttachHostPicker(m.styles, m.width, m.height, m.configFile)
				return m, m.attachPicker.Init()
			case "v":
				m.deployPicker = newKeyDeployHostPicker(m.styles, m.width, m.height, m.configFile)
				return m, m.deployPicker.Init()
			case "d":
				m.deleteDialog = true
				m.deleteBlocked = !item.CanDelete
				return m, nil
			}
		}
		newBrowser, cmd := m.browser.Update(msg)
		m.browser = newBrowser
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		m.browser.width = m.width
		m.browser.height = m.height
		m.browser.styles = m.styles
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "left", "q":
			return m, func() tea.Msg { return keysViewCloseMsg{} }
		case "enter":
			if len(m.browser.items) == 0 || m.browser.loading {
				return m, nil
			}
			m.inspect = true
			m.showPublic = false
			m.publicKey = ""
			m.status = ""
			m.statusErr = false
			return m, nil
		}
	case keyPickerCancelMsg:
		return m, func() tea.Msg { return keysViewCloseMsg{} }
	case keyPickerCreateMsg:
		m.keyCreate = NewKeyCreateForm(m.styles, m.width, m.height)
		return m, m.keyCreate.Init()
	}

	newBrowser, cmd := m.browser.Update(msg)
	m.browser = newBrowser
	return m, cmd
}

func (m *keysViewModel) View() string {
	if m.keyCreate != nil {
		return m.keyCreate.View()
	}
	if m.browser == nil {
		m.browser = NewKeyBrowser("Main screen", m.styles, m.width, m.height, m.configFile)
	}
	if m.deleteDialog {
		return m.renderDeleteDialog()
	}
	if m.attachPicker != nil {
		return m.attachPicker.View()
	}
	if m.deployPicker != nil {
		return m.deployPicker.View()
	}
	if m.hostRefs {
		return m.renderHostRefsView()
	}
	if m.inspect {
		return m.renderInspectView()
	}
	return m.browser.View()
}

func (m *keysViewModel) selectedItem() *keypkg.InventoryItem {
	if m.browser == nil || m.browser.loading || len(m.browser.items) == 0 {
		return nil
	}
	if m.browser.selected < 0 || m.browser.selected >= len(m.browser.items) {
		return nil
	}
	return &m.browser.items[m.browser.selected]
}

func (m *keysViewModel) renderInspectView() string {
	item := m.selectedItem()
	if item == nil {
		return m.browser.View()
	}

	var b strings.Builder
	modalWidth := clampModalWidth(m.width, 92)
	const listIndent = "  "

	b.WriteString(m.styles.Header.Render(listIndent + "SSHM - Keys"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.FormHelp.Render(listIndent + "Key details"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.FocusedLabel.Render(listIndent + filepath.Base(item.Path)))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Path", displayPath(item.Path), modalWidth)))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Algorithm", emptyFallback(item.Algorithm, "-"), modalWidth)))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Perm", item.Permissions, modalWidth)))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Fingerprint", item.Fingerprint, modalWidth)))
	b.WriteString("\n")
	if item.PublicKeyPath != "" {
		b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Public key", displayPath(item.PublicKeyPath), modalWidth)))
	} else {
		b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Public key", "missing", modalWidth)))
	}
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Config refs", summarizeDeclaredRefs(item.References), modalWidth)))

	if m.showPublic {
		b.WriteString("\n\n")
		b.WriteString(m.styles.FocusedLabel.Render(listIndent + "Public key contents"))
		b.WriteString("\n")
		b.WriteString(m.styles.FormHelp.Render(listIndent + strings.TrimSpace(m.publicKey)))
	}

	if m.status != "" {
		b.WriteString("\n\n")
		if m.statusErr {
			b.WriteString(m.styles.ErrorText.Render("Error: " + m.status))
		} else {
			b.WriteString(m.styles.FormHelp.Render(listIndent + m.status))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(m.styles.FormHelp.Render(listIndent + "a add agent • p show/hide public key • c copy public key • y copy path"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(listIndent + "o reveal path • h config refs • t attach to host • v deploy to host"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(listIndent + "d delete"))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(listIndent + "esc/← back"))

	return renderKeyPickerModal(m.width, m.height, m.styles, "Main screen", b.String(), modalWidth)
}

func (m *keysViewModel) renderHostRefsView() string {
	item := m.selectedItem()
	if item == nil {
		return m.browser.View()
	}

	var b strings.Builder
	modalWidth := clampModalWidth(m.width, 92)
	const listIndent = "  "

	b.WriteString(m.styles.Header.Render(listIndent + "SSHM - Keys"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.FormHelp.Render(listIndent + "Config references for " + filepath.Base(item.Path)))
	b.WriteString("\n\n")

	for i, ref := range item.References {
		row := ref.Host
		if strings.TrimSpace(ref.SourceFile) != "" {
			row += "  " + formatConfigFile(ref.SourceFile)
		}
		if i == m.refSelected {
			b.WriteString(m.styles.Selected.Render("> " + row))
		} else {
			b.WriteString(listIndent + row)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(listIndent + "enter open host info • e edit host • esc/← back"))
	return renderKeyPickerModal(m.width, m.height, m.styles, "Key details", b.String(), modalWidth)
}

func (m *keysViewModel) renderDeleteDialog() string {
	item := m.selectedItem()
	if item == nil {
		return m.browser.View()
	}

	var b strings.Builder
	b.WriteString(m.styles.Header.Render("Delete key"))
	b.WriteString("\n\n")

	if m.deleteBlocked {
		b.WriteString(m.styles.ErrorText.Render("Can't delete key"))
		b.WriteString("\n\n")
		if len(item.References) > 0 {
			b.WriteString(m.styles.FormHelp.Render("This key is explicitly referenced by these SSH hosts:"))
			for _, ref := range item.References {
				b.WriteString("\n")
				b.WriteString(m.styles.FormHelp.Render("  - " + ref.Host))
			}
		} else {
			b.WriteString(m.styles.FormHelp.Render("Delete is blocked by the current safety rules."))
		}
		b.WriteString("\n\n")
		b.WriteString(m.styles.FormHelp.Render("Enter/Esc/← back"))
		return renderKeyModal(m.width, m.height, m.styles, b.String(), 72)
	}

	b.WriteString(m.styles.FormHelp.Render("Delete key " + displayPath(item.Path) + "?"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.FormHelp.Render("No explicit IdentityFile entries point to this key."))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("SSHM can't verify inherited or external SSH config."))
	b.WriteString("\n\n")
	b.WriteString(m.styles.FormHelp.Render("Private: " + displayPath(item.Path)))
	if item.PublicKeyPath != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.FormHelp.Render("Public:  " + displayPath(item.PublicKeyPath)))
	}
	b.WriteString("\n\n")
	b.WriteString(m.styles.FormHelp.Render("Enter delete • Esc/← cancel"))
	return renderKeyModal(m.width, m.height, m.styles, b.String(), 72)
}

func (m *keyPickerModel) Init() tea.Cmd {
	return loadKeyInventoryCmd(m.configFile)
}

func (m *keyPickerModel) Update(msg tea.Msg) (*keyPickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		return m, nil

	case keyInventoryLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.items = nil
			m.selected = 0
			return m, nil
		}
		m.err = ""
		m.items = msg.items
		if len(m.items) == 0 {
			m.selected = 0
			return m, nil
		}
		if m.selected >= len(m.items) {
			m.selected = len(m.items) - 1
		}
		if m.selected < 0 {
			m.selected = 0
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q", "left":
			return m, func() tea.Msg { return keyPickerCancelMsg{} }
		case "enter":
			if m.loading || len(m.items) == 0 || m.mode == keyPickerModeBrowse {
				return m, nil
			}
			path := m.items[m.selected].Path
			return m, func() tea.Msg { return keyPickerSelectMsg{path: path} }
		case "g", "ctrl+g":
			return m, func() tea.Msg { return keyPickerCreateMsg{} }
		case "r":
			m.loading = true
			m.err = ""
			return m, loadKeyInventoryCmd(m.configFile)
		case "up", "k":
			if m.loading {
				return m, nil
			}
			if m.selected > 0 {
				m.selected--
			}
			return m, nil
		case "down", "j":
			if m.loading {
				return m, nil
			}
			if m.selected < len(m.items)-1 {
				m.selected++
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *keyPickerModel) View() string {
	var b strings.Builder
	modalWidth := clampModalWidth(m.width, 92)
	const listIndent = "  "

	b.WriteString(m.styles.Header.Render(listIndent + "SSHM - Keys"))
	b.WriteString("\n\n")
	if strings.TrimSpace(m.title) != "" {
		b.WriteString(m.styles.FormHelp.Render(listIndent + m.title))
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(m.styles.ErrorText.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	if m.loading {
		b.WriteString(m.styles.FormField.Render("Loading local SSH keys..."))
		b.WriteString("\n")
		b.WriteString(m.styles.FormHelp.Render("Scanning ~/.ssh and SSH config references."))
		b.WriteString("\n\n")
		b.WriteString(m.styles.FormHelp.Render("g generate • esc/← back"))
		return renderKeyPickerModal(m.width, m.height, m.styles, m.backLabel, b.String(), modalWidth)
	}

	if len(m.items) == 0 {
		b.WriteString(m.styles.FormField.Render("No local SSH keys found."))
		b.WriteString("\n")
		if m.mode == keyPickerModeBrowse {
			b.WriteString(m.styles.FormHelp.Render("Press g to generate a key. Esc or ← returns to hosts."))
		} else {
			b.WriteString(m.styles.FormHelp.Render("Press g to generate a key, or Esc to return and enter a path."))
		}
		b.WriteString("\n\n")
		b.WriteString(m.styles.FormHelp.Render("g generate • esc/← back"))
		return renderKeyPickerModal(m.width, m.height, m.styles, m.backLabel, b.String(), modalWidth)
	}

	nameWidth, algoWidth, permWidth, refsWidth := keyTableColumnWidths(m.items, modalWidth)
	header := fmt.Sprintf("%s%-*s        %-*s  %-*s  %-*s",
		listIndent,
		nameWidth, "Name",
		algoWidth, "Algo",
		permWidth, "Perm",
		refsWidth, "Config",
	)
	b.WriteString(m.styles.FocusedLabel.Render(header))
	b.WriteString("\n\n")

	for i, item := range m.items {
		row := fmt.Sprintf("%-*s        %-*s  %-*s  %-*s",
			nameWidth, truncateText(filepath.Base(item.Path), nameWidth),
			algoWidth, emptyFallback(item.Algorithm, "-"),
			permWidth, item.Permissions,
			refsWidth, truncateText(summarizeDeclaredRefCount(item.References), refsWidth),
		)
		if i == m.selected {
			b.WriteString(m.styles.Selected.Render("> " + row))
		} else {
			b.WriteString(listIndent + row)
		}
		b.WriteString("\n")
		if i == m.selected {
			b.WriteString(m.styles.FormHelp.Render(listIndent + truncateMiddle(displayPath(item.Path), modalWidth-6)))
			b.WriteString("\n")
		}
	}

	selected := m.items[m.selected]
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Path", displayPath(selected.Path), modalWidth)))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Fingerprint", selected.Fingerprint, modalWidth)))
	b.WriteString("\n")
	if selected.PublicKeyPath != "" {
		b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Public key", displayPath(selected.PublicKeyPath), modalWidth)))
	} else {
		b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Public key", "missing", modalWidth)))
	}
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render(renderKeyDetailLine("Config refs", summarizeDeclaredRefs(selected.References), modalWidth)))
	b.WriteString("\n\n")
	if m.mode == keyPickerModeBrowse {
		b.WriteString(m.styles.FormHelp.Render(listIndent + "j/k or ↑/↓ select • g generate • r refresh • esc/← back"))
	} else {
		b.WriteString(m.styles.FormHelp.Render(listIndent + "j/k or ↑/↓ select • enter choose • g generate • r refresh • esc/← back"))
	}
	return renderKeyPickerModal(m.width, m.height, m.styles, m.backLabel, b.String(), modalWidth)
}

type keyCreateFormModel struct {
	inputs           []textinput.Model
	focused          int
	currentTab       int
	algorithmOptions []string
	algorithmIndex   int
	styles           Styles
	width            int
	height           int
	err              string
}

const (
	keyCreateNameInput = iota
	keyCreateCommentInput
	keyCreatePathInput
	keyCreateAlgorithmInput
)

const (
	keyCreateTabGeneral = iota
	keyCreateTabAdvanced
)

var (
	keyCreateGeneralInputs  = [...]int{keyCreateNameInput, keyCreateCommentInput, keyCreatePathInput}
	keyCreateAdvancedInputs = [...]int{keyCreateAlgorithmInput}
)

func NewKeyCreateForm(styles Styles, width, height int) *keyCreateFormModel {
	defaultPath, err := config.GetSSHDirectory()
	if err != nil {
		defaultPath = "~/.ssh"
	}

	defaultComment := suggestedKeyComment()
	algorithms := keypkg.AllowedGenerateAlgorithms()
	defaultAlgorithm, _ := keypkg.NormalizeGenerateAlgorithm("")
	algorithmIndex := 0
	for i, algorithm := range algorithms {
		if algorithm == defaultAlgorithm {
			algorithmIndex = i
			break
		}
	}

	inputs := make([]textinput.Model, 3)

	inputs[keyCreateNameInput] = textinput.New()
	inputs[keyCreateNameInput].Placeholder = "id_ed25519_label"
	inputs[keyCreateNameInput].Focus()
	inputs[keyCreateNameInput].CharLimit = 120
	inputs[keyCreateNameInput].Width = 40

	inputs[keyCreateCommentInput] = textinput.New()
	inputs[keyCreateCommentInput].Placeholder = defaultComment
	inputs[keyCreateCommentInput].CharLimit = 200
	inputs[keyCreateCommentInput].Width = 60

	inputs[keyCreatePathInput] = textinput.New()
	inputs[keyCreatePathInput].SetValue(defaultPath)
	inputs[keyCreatePathInput].CharLimit = 240
	inputs[keyCreatePathInput].Width = 60

	return &keyCreateFormModel{
		inputs:           inputs,
		focused:          keyCreateNameInput,
		currentTab:       keyCreateTabGeneral,
		algorithmOptions: algorithms,
		algorithmIndex:   algorithmIndex,
		styles:           styles,
		width:            width,
		height:           height,
	}
}

func (m *keyCreateFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *keyCreateFormModel) Update(msg tea.Msg) (*keyCreateFormModel, tea.Cmd) {
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
			return m, func() tea.Msg { return keyCreateCancelMsg{} }
		case "ctrl+s":
			return m, m.submit()
		case "ctrl+j":
			m.currentTab = keyCreateTabAdvanced
			m.focused = keyCreateAlgorithmInput
			return m, m.updateFocus()
		case "ctrl+k":
			m.currentTab = keyCreateTabGeneral
			m.focused = keyCreateNameInput
			return m, m.updateFocus()
		case "left", "h":
			if m.currentTab == keyCreateTabAdvanced && m.focused == keyCreateAlgorithmInput {
				m.moveAlgorithmSelection(-1)
				return m, nil
			}
		case "right", "l":
			if m.currentTab == keyCreateTabAdvanced && m.focused == keyCreateAlgorithmInput {
				m.moveAlgorithmSelection(1)
				return m, nil
			}
		case "tab", "enter", "down", "shift+tab", "up":
			return m, m.handleNavigation(msg.String())
		}
	}

	if m.currentTab == keyCreateTabGeneral {
		cmd := make([]tea.Cmd, len(m.inputs))
		for i := range m.inputs {
			if !m.isInputVisible(i) {
				continue
			}
			m.inputs[i], cmd[i] = m.inputs[i].Update(msg)
		}
		cmds = append(cmds, cmd...)
	}

	return m, tea.Batch(cmds...)
}

func (m *keyCreateFormModel) handleNavigation(key string) tea.Cmd {
	switch key {
	case "enter":
		if m.currentTab == keyCreateTabAdvanced {
			return m.submit()
		}
		return m.moveForward(true)
	case "down", "tab":
		return m.moveForward(false)
	case "up", "shift+tab":
		return m.moveBackward()
	default:
		return nil
	}
}

func (m *keyCreateFormModel) isInputVisible(index int) bool {
	return slices.Contains(m.visibleInputIndices(), index)
}

func (m *keyCreateFormModel) visibleInputIndices() []int {
	if m.currentTab == keyCreateTabAdvanced {
		return keyCreateAdvancedInputs[:]
	}
	return keyCreateGeneralInputs[:]
}

func (m *keyCreateFormModel) moveForward(submitOnLast bool) tea.Cmd {
	if m.currentTab == keyCreateTabAdvanced {
		if submitOnLast {
			return m.submit()
		}
		return nil
	}

	inputs := keyCreateGeneralInputs[:]
	currentPos := slices.Index(inputs, m.focused)
	if currentPos < 0 {
		currentPos = 0
	}

	if currentPos == len(inputs)-1 {
		if submitOnLast {
			return m.submit()
		}
		m.currentTab = keyCreateTabAdvanced
		m.focused = keyCreateAlgorithmInput
		return m.updateFocus()
	}

	m.focused = inputs[currentPos+1]
	return m.updateFocus()
}

func (m *keyCreateFormModel) moveBackward() tea.Cmd {
	if m.currentTab == keyCreateTabAdvanced {
		m.currentTab = keyCreateTabGeneral
		m.focused = keyCreatePathInput
		return m.updateFocus()
	}

	inputs := keyCreateGeneralInputs[:]
	currentPos := slices.Index(inputs, m.focused)
	if currentPos < 0 {
		currentPos = 0
	}
	if currentPos == 0 {
		return nil
	}

	m.focused = inputs[currentPos-1]
	return m.updateFocus()
}

func (m *keyCreateFormModel) moveAlgorithmSelection(delta int) {
	if len(m.algorithmOptions) == 0 {
		return
	}
	m.algorithmIndex += delta
	if m.algorithmIndex < 0 {
		m.algorithmIndex = len(m.algorithmOptions) - 1
	}
	if m.algorithmIndex >= len(m.algorithmOptions) {
		m.algorithmIndex = 0
	}
}

func (m *keyCreateFormModel) renderTabs() string {
	var generalTab, advancedTab string

	if m.currentTab == keyCreateTabGeneral {
		generalTab = m.styles.FocusedLabel.Render("[ General ]")
		advancedTab = m.styles.FormField.Render("  Advanced  ")
	} else {
		generalTab = m.styles.FormField.Render("  General  ")
		advancedTab = m.styles.FocusedLabel.Render("[ Advanced ]")
	}

	return generalTab + "  " + advancedTab
}

func (m *keyCreateFormModel) renderFields(fields []struct {
	index int
	label string
}) string {
	var b strings.Builder

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

func (m *keyCreateFormModel) renderAlgorithmPicker() string {
	var options []string
	for i, algorithm := range m.algorithmOptions {
		label := strings.ToUpper(algorithm)
		if i == m.algorithmIndex {
			options = append(options, m.styles.Selected.Render(" "+label+" "))
			continue
		}
		options = append(options, m.styles.FormField.Render(" "+label+" "))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, options...)
}

func (m *keyCreateFormModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Header.Render("Generate SSH key"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.FormHelp.Render("Runs local ssh-keygen. SSHM never stores private keys or passphrases."))
	b.WriteString("\n\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	if m.currentTab == keyCreateTabGeneral {
		b.WriteString(m.renderFields([]struct {
			index int
			label string
		}{
			{keyCreateNameInput, "Key name *"},
			{keyCreateCommentInput, "Comment"},
			{keyCreatePathInput, "Directory"},
		}))
	} else {
		fieldStyle := m.styles.FormField
		if m.focused == keyCreateAlgorithmInput {
			fieldStyle = m.styles.FocusedLabel
		}
		b.WriteString(fieldStyle.Render("Algorithm"))
		b.WriteString("\n")
		b.WriteString(m.renderAlgorithmPicker())
		b.WriteString("\n")
		b.WriteString(m.styles.FormHelp.Render("←/→ choose • ↑ General"))
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(m.styles.ErrorText.Render("Error: " + m.err))
		b.WriteString("\n\n")
	}

	b.WriteString(m.styles.FormHelp.Render("Default: ed25519."))
	b.WriteString("\n")
	b.WriteString(m.styles.FormHelp.Render("tab move • ctrl+j/k tabs • ↓ Advanced • ↑ General • ctrl+s generate • esc back"))
	return renderKeyModal(m.width, m.height, m.styles, b.String(), 92)
}

func (m *keyCreateFormModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if !m.isInputVisible(i) {
			m.inputs[i].Blur()
			continue
		}
		if i == m.focused && i != keyCreateAlgorithmInput {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *keyCreateFormModel) submit() tea.Cmd {
	name := strings.TrimSpace(m.inputs[keyCreateNameInput].Value())
	comment := strings.TrimSpace(m.inputs[keyCreateCommentInput].Value())
	directory := strings.TrimSpace(m.inputs[keyCreatePathInput].Value())
	algorithm := ""
	if len(m.algorithmOptions) > 0 && m.algorithmIndex >= 0 && m.algorithmIndex < len(m.algorithmOptions) {
		algorithm = m.algorithmOptions[m.algorithmIndex]
	}

	if comment == "" {
		comment = m.inputs[keyCreateCommentInput].Placeholder
	}

	algorithm, err := keypkg.NormalizeGenerateAlgorithm(algorithm)
	if err != nil {
		m.err = err.Error()
		return nil
	}

	result, args, err := keypkg.BuildGenerateArgs(keypkg.GenerateOptions{
		Name:      name,
		Algorithm: algorithm,
		Comment:   comment,
		Directory: directory,
		KDFRounds: 100,
	})
	if err != nil {
		m.err = err.Error()
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(result.PrivateKeyPath), 0700); err != nil {
		m.err = fmt.Sprintf("create key directory: %v", err)
		return nil
	}

	m.err = ""

	cmd := exec.Command("ssh-keygen", args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return keyCreateFinishedMsg{
			path: result.PrivateKeyPath,
			err:  err,
		}
	})
}

func suggestedKeyComment() string {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Sprintf("user@host (%s)", time.Now().Format("2006-01-02"))
	}

	hostName, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostName) == "" {
		hostName = "host"
	}

	return fmt.Sprintf("%s@%s (%s)", currentUser.Username, hostName, time.Now().Format("2006-01-02"))
}

func loadPublicKeyCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(path) == "" {
			return keyPublicKeyLoadedMsg{err: fmt.Errorf("public key file is missing")}
		}
		content, err := readPublicKeyFile(path)
		if err != nil {
			return keyPublicKeyLoadedMsg{err: err}
		}
		return keyPublicKeyLoadedMsg{content: string(content)}
	}
}

func copyPublicKeyCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(path) == "" {
			return keyPublicKeyCopiedMsg{err: fmt.Errorf("public key file is missing")}
		}
		content, err := readPublicKeyFile(path)
		if err != nil {
			return keyPublicKeyCopiedMsg{err: err}
		}
		if err := copyToClipboard(string(content)); err != nil {
			return keyPublicKeyCopiedMsg{err: err}
		}
		return keyPublicKeyCopiedMsg{}
	}
}

func copyPathCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(path) == "" {
			return keyPathCopiedMsg{err: fmt.Errorf("key path is missing")}
		}
		return keyPathCopiedMsg{err: copyToClipboard(path)}
	}
}

func revealPathCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(path) == "" {
			return keyPathRevealedMsg{err: fmt.Errorf("key path is missing")}
		}
		return keyPathRevealedMsg{
			path: path,
			err:  revealPathInManager(path),
		}
	}
}

func deleteKeyCmd(privatePath string, publicPath string) tea.Cmd {
	return func() tea.Msg {
		err := deleteKeyFiles(privatePath, publicPath)
		return keyDeleteFinishedMsg{
			path: privatePath,
			err:  err,
		}
	}
}

func addKeyToAgentCmd(path string) tea.Cmd {
	normalized, args, err := keypkg.BuildAddArgs(path)
	if err != nil {
		return func() tea.Msg { return keyAgentAddedMsg{err: err} }
	}

	cmd := exec.Command("ssh-add", args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return keyAgentAddedMsg{
			path: normalized,
			err:  err,
		}
	})
}

func summarizeDeclaredRefs(refs []keypkg.Reference) string {
	if len(refs) == 0 {
		return "none"
	}

	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		names = append(names, ref.Host)
	}

	if len(names) <= 3 {
		return strings.Join(names, ", ")
	}

	return strings.Join(names[:3], ", ") + fmt.Sprintf(" +%d more", len(names)-3)
}

func summarizeDeclaredRefCount(refs []keypkg.Reference) string {
	switch len(refs) {
	case 0:
		return "none"
	case 1:
		return "1 host"
	default:
		return fmt.Sprintf("%d hosts", len(refs))
	}
}

func emptyFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func copyTextToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("cmd", "/c", "clip")
	default:
		switch {
		case commandExists("wl-copy"):
			cmd = exec.Command("wl-copy")
		case commandExists("xclip"):
			cmd = exec.Command("xclip", "-selection", "clipboard")
		case commandExists("xsel"):
			cmd = exec.Command("xsel", "--clipboard", "--input")
		default:
			return fmt.Errorf("no clipboard command available")
		}
	}

	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			return fmt.Errorf("copy to clipboard: %w", err)
		}
		return fmt.Errorf("copy to clipboard: %s", message)
	}
	return nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func revealPath(path string) error {
	cleanPath := filepath.Clean(path)
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", cleanPath)
	case "windows":
		cmd = exec.Command("explorer.exe", "/select,", cleanPath)
	default:
		if !commandExists("xdg-open") {
			return fmt.Errorf("no file manager command available")
		}
		cmd = exec.Command("xdg-open", filepath.Dir(cleanPath))
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			return fmt.Errorf("reveal path: %w", err)
		}
		return fmt.Errorf("reveal path: %s", message)
	}
	return nil
}

func renderKeyModal(width int, height int, styles Styles, content string, maxWidth int) string {
	container := styles.FormContainer.Width(clampModalWidth(width, maxWidth)).Render(content)
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		container,
	)
}

func renderKeyPickerModal(width int, height int, styles Styles, backLabel string, content string, modalWidth int) string {
	container := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(PrimaryColor)).
		Padding(1, 1).
		Width(modalWidth).
		Render(content)

	if strings.TrimSpace(backLabel) != "" {
		back := lipgloss.NewStyle().
			MarginLeft(2).
			Render(styles.FormHelp.Render("← back to " + backLabel))
		container = lipgloss.JoinVertical(lipgloss.Left, back, container)
	}

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		container,
	)
}

func clampModalWidth(screenWidth int, maxWidth int) int {
	if screenWidth <= 0 {
		return maxWidth
	}

	width := screenWidth - 6
	if width < 48 {
		width = 48
	}
	if width > maxWidth {
		width = maxWidth
	}
	return width
}

func keyTableColumnWidths(items []keypkg.InventoryItem, modalWidth int) (name int, algo int, perm int, refs int) {
	const totalColumnGaps = 12

	contentWidth := modalWidth - 6
	if contentWidth < 48 {
		contentWidth = 48
	}

	algo = 7
	perm = 4
	refs = 10
	name = 18
	for _, item := range items {
		width := len([]rune(filepath.Base(item.Path)))
		if width > name {
			name = width
		}
	}

	nameMax := contentWidth - algo - perm - refs - totalColumnGaps
	if nameMax < 18 {
		nameMax = 18
	}
	if nameMax > 28 {
		nameMax = 28
	}
	if name > nameMax {
		name = nameMax
	}
	return name, algo, perm, refs
}

func renderKeyDetailLine(label string, value string, modalWidth int) string {
	const (
		listIndent       = "  "
		detailLabelWidth = 13
	)

	valueWidth := modalWidth - len(listIndent) - detailLabelWidth - 3
	if valueWidth < 12 {
		valueWidth = 12
	}

	return fmt.Sprintf("%s%-*s %s",
		listIndent,
		detailLabelWidth, label,
		truncateMiddle(value, valueWidth),
	)
}

func truncateText(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func truncateMiddle(value string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 3 {
		return string(runes[:width])
	}

	head := (width - 1) / 2
	tail := width - 1 - head
	return string(runes[:head]) + "…" + string(runes[len(runes)-tail:])
}

func displayPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	clean := filepath.Clean(path)
	if clean == homeDir {
		return "~"
	}
	homePrefix := homeDir + string(os.PathSeparator)
	if rest, ok := strings.CutPrefix(clean, homePrefix); ok {
		return "~" + string(os.PathSeparator) + rest
	}
	return clean
}

func loadKeyInventoryCmd(configFile string) tea.Cmd {
	return func() tea.Msg {
		items, err := loadKeyInventoryForUI(context.Background(), configFile)
		return keyInventoryLoadedMsg{
			items: items,
			err:   err,
		}
	}
}
