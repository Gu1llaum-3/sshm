package ui

import (
	"fmt"

	"github.com/Gu1llaum-3/sshm/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

type moveFormModel struct {
	fileSelector *fileSelectorModel
	hostName     string
	configFile   string
	width        int
	height       int
	styles       Styles
	state        moveFormState
}

type moveFormState int

const (
	moveFormSelectingFile moveFormState = iota
	moveFormProcessing
)

type moveFormSubmitMsg struct {
	hostName   string
	targetFile string
	err        error
}

type moveFormCancelMsg struct{}

// NewMoveForm creates a new move form for moving a host to another config file
func NewMoveForm(hostName string, styles Styles, width, height int, configFile string) (*moveFormModel, error) {
	// Get all config files except the one containing the current host
	files, err := config.GetConfigFilesExcludingCurrent(hostName, configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get config files: %v", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no other config files available to move host to")
	}

	// Create a custom file selector for move operation
	fileSelector, err := newFileSelectorFromFiles(
		fmt.Sprintf("Select destination config file for host '%s':", hostName),
		styles,
		width,
		height,
		files,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create file selector: %v", err)
	}

	return &moveFormModel{
		fileSelector: fileSelector,
		hostName:     hostName,
		configFile:   configFile,
		width:        width,
		height:       height,
		styles:       styles,
		state:        moveFormSelectingFile,
	}, nil
}

func (m *moveFormModel) Init() tea.Cmd {
	return m.fileSelector.Init()
}

func (m *moveFormModel) Update(msg tea.Msg) (*moveFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		if m.fileSelector != nil {
			m.fileSelector.width = m.width
			m.fileSelector.height = m.height
			m.fileSelector.styles = m.styles
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case moveFormSelectingFile:
			switch msg.String() {
			case "enter":
				if m.fileSelector != nil && len(m.fileSelector.files) > 0 {
					selectedFile := m.fileSelector.files[m.fileSelector.selected]
					m.state = moveFormProcessing
					return m, m.submitMove(selectedFile)
				}
			case "esc", "q":
				return m, func() tea.Msg { return moveFormCancelMsg{} }
			default:
				// Forward other keys to file selector
				if m.fileSelector != nil {
					newFileSelector, cmd := m.fileSelector.Update(msg)
					m.fileSelector = newFileSelector
					return m, cmd
				}
			}
		case moveFormProcessing:
			// Dans cet état, on attend le résultat de l'opération
			// Le résultat sera géré par le modèle principal
			switch msg.String() {
			case "esc", "q":
				return m, func() tea.Msg { return moveFormCancelMsg{} }
			}
		}
	}

	return m, nil
}

func (m *moveFormModel) View() string {
	switch m.state {
	case moveFormSelectingFile:
		if m.fileSelector != nil {
			return m.fileSelector.View()
		}
		return "Loading..."

	case moveFormProcessing:
		return m.styles.FormTitle.Render("Moving host...") + "\n\n" +
			m.styles.HelpText.Render(fmt.Sprintf("Moving host '%s' to selected config file...", m.hostName))

	default:
		return "Unknown state"
	}
}

func (m *moveFormModel) submitMove(targetFile string) tea.Cmd {
	return func() tea.Msg {
		err := config.MoveHostToFile(m.hostName, targetFile)
		return moveFormSubmitMsg{
			hostName:   m.hostName,
			targetFile: targetFile,
			err:        err,
		}
	}
}

// Standalone move form for CLI usage
type standaloneMoveForm struct {
	moveFormModel *moveFormModel
}

func (m standaloneMoveForm) Init() tea.Cmd {
	return m.moveFormModel.Init()
}

func (m standaloneMoveForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case moveFormCancelMsg:
		return m, tea.Quit
	case moveFormSubmitMsg:
		// En mode standalone, on quitte après le déplacement (succès ou erreur)
		return m, tea.Quit
	}

	newForm, cmd := m.moveFormModel.Update(msg)
	m.moveFormModel = newForm
	return m, cmd
}

func (m standaloneMoveForm) View() string {
	return m.moveFormModel.View()
}

// RunMoveForm provides backward compatibility for standalone move form
func RunMoveForm(hostName string, configFile string) error {
	styles := NewStyles(80)
	moveForm, err := NewMoveForm(hostName, styles, 80, 24, configFile)
	if err != nil {
		return err
	}
	m := standaloneMoveForm{moveForm}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
