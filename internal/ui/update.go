package ui

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/connectivity"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Messages for SSH ping functionality
type (
	pingResultMsg *connectivity.HostPingResult
)

// startPingAllCmd creates a command to ping all hosts concurrently
func (m Model) startPingAllCmd() tea.Cmd {
	if m.pingManager == nil {
		return nil
	}

	return tea.Batch(
		// Create individual ping commands for each host
		func() tea.Cmd {
			var cmds []tea.Cmd
			for _, host := range m.hosts {
				cmds = append(cmds, pingSingleHostCmd(m.pingManager, host))
			}
			return tea.Batch(cmds...)
		}(),
	)
}

// listenForPingResultsCmd is no longer needed since we use individual ping commands

// pingSingleHostCmd creates a command to ping a single host
func pingSingleHostCmd(pingManager *connectivity.PingManager, host config.SSHHost) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		result := pingManager.PingHost(ctx, host)
		return pingResultMsg(result)
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		// Ping is now optional - use 'p' key to start ping
	)
}

// Update handles model updates
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle different message types
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update terminal size and recalculate styles
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		m.ready = true

		// Update table height and columns based on new window size
		m.updateTableHeight()
		m.updateTableColumns()

		// Update sub-forms if they exist
		if m.addForm != nil {
			m.addForm.width = m.width
			m.addForm.height = m.height
			m.addForm.styles = m.styles
		}
		if m.editForm != nil {
			m.editForm.width = m.width
			m.editForm.height = m.height
			m.editForm.styles = m.styles
		}
		if m.moveForm != nil {
			m.moveForm.width = m.width
			m.moveForm.height = m.height
			m.moveForm.styles = m.styles
		}
		if m.infoForm != nil {
			m.infoForm.width = m.width
			m.infoForm.height = m.height
			m.infoForm.styles = m.styles
		}
		if m.portForwardForm != nil {
			m.portForwardForm.width = m.width
			m.portForwardForm.height = m.height
			m.portForwardForm.styles = m.styles
		}
		if m.helpForm != nil {
			m.helpForm.width = m.width
			m.helpForm.height = m.height
			m.helpForm.styles = m.styles
		}
		if m.fileSelectorForm != nil {
			m.fileSelectorForm.width = m.width
			m.fileSelectorForm.height = m.height
			m.fileSelectorForm.styles = m.styles
		}
		return m, nil

	case pingResultMsg:
		// Handle ping result - update table display
		if msg != nil {
			// Update the table to reflect the new ping status
			m.updateTableRows()
		}
		return m, nil

	case addFormSubmitMsg:
		if msg.err != nil {
			// Show error in form
			if m.addForm != nil {
				m.addForm.err = msg.err.Error()
			}
			return m, nil
		} else {
			// Success: refresh hosts and return to list view
			var hosts []config.SSHHost
			var err error

			if m.configFile != "" {
				hosts, err = config.ParseSSHConfigFile(m.configFile)
			} else {
				hosts, err = config.ParseSSHConfig()
			}

			if err != nil {
				return m, tea.Quit
			}
			m.hosts = m.sortHosts(hosts)

			// Reapply search filter if there is one active
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.hosts
			}

			m.updateTableRows()
			m.viewMode = ViewList
			m.addForm = nil
			m.table.Focus()
			return m, nil
		}

	case addFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.addForm = nil
		m.table.Focus()
		return m, nil

	case editFormSubmitMsg:
		if msg.err != nil {
			// Show error in form
			if m.editForm != nil {
				m.editForm.err = msg.err.Error()
			}
			return m, nil
		} else {
			// Success: refresh hosts and return to list view
			var hosts []config.SSHHost
			var err error

			if m.configFile != "" {
				hosts, err = config.ParseSSHConfigFile(m.configFile)
			} else {
				hosts, err = config.ParseSSHConfig()
			}

			if err != nil {
				return m, tea.Quit
			}
			m.hosts = m.sortHosts(hosts)

			// Reapply search filter if there is one active
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.hosts
			}

			m.updateTableRows()
			m.viewMode = ViewList
			m.editForm = nil
			m.table.Focus()
			return m, nil
		}

	case editFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.editForm = nil
		m.table.Focus()
		return m, nil

	case moveFormSubmitMsg:
		if msg.err != nil {
			// En cas d'erreur, on pourrait afficher une notification ou retourner à la liste
			// Pour l'instant, on retourne simplement à la liste
			m.viewMode = ViewList
			m.moveForm = nil
			m.table.Focus()
			return m, nil
		} else {
			// Success: refresh hosts and return to list view
			var hosts []config.SSHHost
			var err error

			if m.configFile != "" {
				hosts, err = config.ParseSSHConfigFile(m.configFile)
			} else {
				hosts, err = config.ParseSSHConfig()
			}

			if err != nil {
				return m, tea.Quit
			}
			m.hosts = m.sortHosts(hosts)

			// Reapply search filter if there is one active
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.hosts
			}

			m.updateTableRows()
			m.viewMode = ViewList
			m.moveForm = nil
			m.table.Focus()
			return m, nil
		}

	case moveFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.moveForm = nil
		m.table.Focus()
		return m, nil

	case infoFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.infoForm = nil
		m.table.Focus()
		return m, nil

	case fileSelectorMsg:
		if msg.cancelled {
			// Cancel: return to list view
			m.viewMode = ViewList
			m.fileSelectorForm = nil
			m.table.Focus()
			return m, nil
		} else {
			// File selected: proceed to add form with selected file
			m.addForm = NewAddForm("", m.styles, m.width, m.height, msg.selectedFile)
			m.viewMode = ViewAdd
			m.fileSelectorForm = nil
			return m, textinput.Blink
		}

	case infoFormEditMsg:
		// Switch from info to edit mode
		editForm, err := NewEditForm(msg.hostName, m.styles, m.width, m.height, m.configFile)
		if err != nil {
			// Handle error - could show in UI, for now just go back to list
			m.viewMode = ViewList
			m.infoForm = nil
			m.table.Focus()
			return m, nil
		}
		m.editForm = editForm
		m.infoForm = nil
		m.viewMode = ViewEdit
		return m, textinput.Blink

	case portForwardSubmitMsg:
		if msg.err != nil {
			// Show error in form
			if m.portForwardForm != nil {
				m.portForwardForm.err = msg.err.Error()
			}
			return m, nil
		} else {
			// Success: execute SSH command with port forwarding
			if len(msg.sshArgs) > 0 {
				sshCmd := exec.Command("ssh", msg.sshArgs...)

				// Record the connection in history
				if m.historyManager != nil && m.portForwardForm != nil {
					err := m.historyManager.RecordConnection(m.portForwardForm.hostName)
					if err != nil {
						fmt.Printf("Warning: Could not record connection history: %v\n", err)
					}
				}

				return m, tea.ExecProcess(sshCmd, func(err error) tea.Msg {
					return tea.Quit()
				})
			}

			// If no SSH args, just return to list view
			m.viewMode = ViewList
			m.portForwardForm = nil
			m.table.Focus()
			return m, nil
		}

	case portForwardCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.portForwardForm = nil
		m.table.Focus()
		return m, nil

	case helpCloseMsg:
		// Close help: return to list view
		m.viewMode = ViewList
		m.helpForm = nil
		m.table.Focus()
		return m, nil

	case tea.KeyMsg:
		// Handle view-specific key presses
		switch m.viewMode {
		case ViewAdd:
			if m.addForm != nil {
				var newForm *addFormModel
				newForm, cmd = m.addForm.Update(msg)
				m.addForm = newForm
				return m, cmd
			}
		case ViewEdit:
			if m.editForm != nil {
				var newForm *editFormModel
				newForm, cmd = m.editForm.Update(msg)
				m.editForm = newForm
				return m, cmd
			}
		case ViewMove:
			if m.moveForm != nil {
				var newForm *moveFormModel
				newForm, cmd = m.moveForm.Update(msg)
				m.moveForm = newForm
				return m, cmd
			}
		case ViewInfo:
			if m.infoForm != nil {
				var newForm *infoFormModel
				newForm, cmd = m.infoForm.Update(msg)
				m.infoForm = newForm
				return m, cmd
			}
		case ViewPortForward:
			if m.portForwardForm != nil {
				var newForm *portForwardModel
				newForm, cmd = m.portForwardForm.Update(msg)
				m.portForwardForm = newForm
				return m, cmd
			}
		case ViewHelp:
			if m.helpForm != nil {
				var newForm *helpModel
				newForm, cmd = m.helpForm.Update(msg)
				m.helpForm = newForm
				return m, cmd
			}
		case ViewFileSelector:
			if m.fileSelectorForm != nil {
				var newForm *fileSelectorModel
				newForm, cmd = m.fileSelectorForm.Update(msg)
				m.fileSelectorForm = newForm
				return m, cmd
			}
		case ViewList:
			// Handle list view keys
			return m.handleListViewKeys(msg)
		}
	}

	return m, cmd
}

func (m Model) handleListViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc", "ctrl+c":
		if m.deleteMode {
			// Exit delete mode
			m.deleteMode = false
			m.deleteHost = ""
			m.table.Focus()
			return m, nil
		}
		return m, tea.Quit
	case "q":
		if !m.searchMode && !m.deleteMode {
			return m, tea.Quit
		}
	case "/", "ctrl+f":
		if !m.searchMode && !m.deleteMode {
			// Enter search mode
			m.searchMode = true
			m.updateTableStyles()
			m.table.Blur()
			m.searchInput.Focus()
			return m, textinput.Blink
		}
	case "tab":
		if !m.deleteMode {
			// Switch focus between search input and table
			if m.searchMode {
				// Switch from search to table
				m.searchMode = false
				m.updateTableStyles()
				m.searchInput.Blur()
				m.table.Focus()
			} else {
				// Switch from table to search
				m.searchMode = true
				m.updateTableStyles()
				m.table.Blur()
				m.searchInput.Focus()
				return m, textinput.Blink
			}
			return m, nil
		}
	case "enter":
		if m.searchMode {
			// Validate search and return to table mode to allow commands
			m.searchMode = false
			m.updateTableStyles()
			m.searchInput.Blur()
			m.table.Focus()
			return m, nil
		} else if m.deleteMode {
			// Confirm deletion
			var err error
			if m.configFile != "" {
				err = config.DeleteSSHHostFromFile(m.deleteHost, m.configFile)
			} else {
				err = config.DeleteSSHHost(m.deleteHost)
			}
			if err != nil {
				// Could display an error message here
				m.deleteMode = false
				m.deleteHost = ""
				m.table.Focus()
				return m, nil
			}
			// Refresh the hosts list
			var hosts []config.SSHHost
			var parseErr error

			if m.configFile != "" {
				hosts, parseErr = config.ParseSSHConfigFile(m.configFile)
			} else {
				hosts, parseErr = config.ParseSSHConfig()
			}

			if parseErr != nil {
				// Could display an error message here
				m.deleteMode = false
				m.deleteHost = ""
				m.table.Focus()
				return m, nil
			}
			m.hosts = m.sortHosts(hosts)

			// Reapply search filter if there is one active
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.hosts
			}

			m.updateTableRows()
			m.deleteMode = false
			m.deleteHost = ""
			m.table.Focus()
			return m, nil
		} else {
			// Connect to the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				hostName := extractHostNameFromTableRow(selected[0]) // Extract hostname from first column

				// Record the connection in history
				if m.historyManager != nil {
					err := m.historyManager.RecordConnection(hostName)
					if err != nil {
						// Log the error but don't prevent the connection
						fmt.Printf("Warning: Could not record connection history: %v\n", err)
					}
				}

				// Build the SSH command with the appropriate config file
				var sshCmd *exec.Cmd
				if m.configFile != "" {
					sshCmd = exec.Command("ssh", "-F", m.configFile, hostName)
				} else {
					sshCmd = exec.Command("ssh", hostName)
				}

				return m, tea.ExecProcess(sshCmd, func(err error) tea.Msg {
					return tea.Quit()
				})
			}
		}
	case "e":
		if !m.searchMode && !m.deleteMode {
			// Edit the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				hostName := extractHostNameFromTableRow(selected[0]) // Extract hostname from first column
				editForm, err := NewEditForm(hostName, m.styles, m.width, m.height, m.configFile)
				if err != nil {
					// Handle error - could show in UI
					return m, nil
				}
				m.editForm = editForm
				m.viewMode = ViewEdit
				return m, textinput.Blink
			}
		}
	case "m":
		if !m.searchMode && !m.deleteMode {
			// Move the selected host to another config file
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				hostName := extractHostNameFromTableRow(selected[0]) // Extract hostname from first column
				moveForm, err := NewMoveForm(hostName, m.styles, m.width, m.height, m.configFile)
				if err != nil {
					// Handle error - could show in UI, e.g., no other config files available
					return m, nil
				}
				m.moveForm = moveForm
				m.viewMode = ViewMove
				return m, textinput.Blink
			}
		}
	case "i":
		if !m.searchMode && !m.deleteMode {
			// Show info for the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				hostName := extractHostNameFromTableRow(selected[0]) // Extract hostname from first column
				infoForm, err := NewInfoForm(hostName, m.styles, m.width, m.height, m.configFile)
				if err != nil {
					// Handle error - could show in UI
					return m, nil
				}
				m.infoForm = infoForm
				m.viewMode = ViewInfo
				return m, nil
			}
		}
	case "a":
		if !m.searchMode && !m.deleteMode {
			// Check if there are multiple config files starting from the current base config
			var configFiles []string
			var err error

			if m.configFile != "" {
				// Use the specified config file as base
				configFiles, err = config.GetAllConfigFilesFromBase(m.configFile)
			} else {
				// Use the default config file as base
				configFiles, err = config.GetAllConfigFiles()
			}

			if err != nil || len(configFiles) <= 1 {
				// Only one config file (or error), go directly to add form
				var configFile string
				if len(configFiles) == 1 {
					configFile = configFiles[0]
				} else {
					configFile = m.configFile
				}
				m.addForm = NewAddForm("", m.styles, m.width, m.height, configFile)
				m.viewMode = ViewAdd
			} else {
				// Multiple config files, show file selector
				fileSelectorForm, err := NewFileSelectorFromBase("Select config file to add host to:", m.styles, m.width, m.height, m.configFile)
				if err != nil {
					// Fallback to default behavior if file selector fails
					m.addForm = NewAddForm("", m.styles, m.width, m.height, m.configFile)
					m.viewMode = ViewAdd
				} else {
					m.fileSelectorForm = fileSelectorForm
					m.viewMode = ViewFileSelector
				}
			}
			return m, textinput.Blink
		}
	case "d":
		if !m.searchMode && !m.deleteMode {
			// Delete the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				hostName := extractHostNameFromTableRow(selected[0]) // Extract hostname from first column
				m.deleteMode = true
				m.deleteHost = hostName
				m.table.Blur()
				return m, nil
			}
		}
	case "p":
		if !m.searchMode && !m.deleteMode {
			// Ping all hosts
			return m, m.startPingAllCmd()
		}
	case "f":
		if !m.searchMode && !m.deleteMode {
			// Port forwarding for the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				hostName := extractHostNameFromTableRow(selected[0]) // Extract hostname from first column
				m.portForwardForm = NewPortForwardForm(hostName, m.styles, m.width, m.height, m.configFile)
				m.viewMode = ViewPortForward
				return m, textinput.Blink
			}
		}
	case "h":
		if !m.searchMode && !m.deleteMode {
			// Show help
			m.helpForm = NewHelpForm(m.styles, m.width, m.height)
			m.viewMode = ViewHelp
			return m, nil
		}
	case "s":
		if !m.searchMode && !m.deleteMode {
			// Cycle through sort modes (only 2 modes now)
			m.sortMode = (m.sortMode + 1) % 2
			// Re-apply the current filter with the new sort mode
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.sortHosts(m.hosts)
			}
			m.updateTableRows()
			return m, nil
		}
	case "r":
		if !m.searchMode && !m.deleteMode {
			// Switch to sort by recent (last used)
			m.sortMode = SortByLastUsed
			// Re-apply the current filter with the new sort mode
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.sortHosts(m.hosts)
			}
			m.updateTableRows()
			return m, nil
		}
	case "n":
		if !m.searchMode && !m.deleteMode {
			// Switch to sort by name
			m.sortMode = SortByName
			// Re-apply the current filter with the new sort mode
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.sortHosts(m.hosts)
			}
			m.updateTableRows()
			return m, nil
		}
	}

	// Update the appropriate component based on mode
	if m.searchMode {
		oldValue := m.searchInput.Value()
		m.searchInput, cmd = m.searchInput.Update(msg)
		// Update filtered hosts only if the search value has changed
		if m.searchInput.Value() != oldValue {
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.sortHosts(m.hosts)
			}
			m.updateTableRows()
		}
	} else {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}
