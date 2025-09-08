package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the complete user interface
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Handle different view modes
	switch m.viewMode {
	case ViewAdd:
		if m.addForm != nil {
			return m.addForm.View()
		}
	case ViewEdit:
		if m.editForm != nil {
			return m.editForm.View()
		}
	case ViewMove:
		if m.moveForm != nil {
			return m.moveForm.View()
		}
	case ViewInfo:
		if m.infoForm != nil {
			return m.infoForm.View()
		}
	case ViewPortForward:
		if m.portForwardForm != nil {
			return m.portForwardForm.View()
		}
	case ViewHelp:
		if m.helpForm != nil {
			return m.helpForm.View()
		}
	case ViewFileSelector:
		if m.fileSelectorForm != nil {
			return m.fileSelectorForm.View()
		}
	case ViewList:
		return m.renderListView()
	}

	return m.renderListView()
}

// renderListView renders the main list interface
func (m Model) renderListView() string {
	// Build the interface components
	components := []string{}

	// Add the ASCII title
	components = append(components, m.styles.Header.Render(asciiTitle))

	// Add the search bar with the appropriate style based on focus
	searchPrompt := "Search (/ to focus): "
	if m.searchMode {
		components = append(components, m.styles.SearchFocused.Render(searchPrompt+m.searchInput.View()))
	} else {
		components = append(components, m.styles.SearchUnfocused.Render(searchPrompt+m.searchInput.View()))
	}

	// Add the table with the appropriate style based on focus
	if m.searchMode {
		// The table is not focused, use the unfocused style
		components = append(components, m.styles.TableUnfocused.Render(m.table.View()))
	} else {
		// The table is focused, use the focused style with the primary color
		components = append(components, m.styles.TableFocused.Render(m.table.View()))
	}

	// Add the help text
	var helpText string
	if !m.searchMode {
		helpText = " ↑/↓: navigate • Enter: connect • p: ping all • i: info • h: help • q: quit"
	} else {
		helpText = " Type to filter • Enter: validate • Tab: switch • ESC: quit"
	}
	components = append(components, m.styles.HelpText.Render(helpText))

	// Join all components vertically with appropriate spacing
	mainView := m.styles.App.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			components...,
		),
	)

	// If in delete mode, overlay the confirmation dialog
	if m.deleteMode {
		// Combine the main view with the confirmation dialog overlay
		confirmation := m.renderDeleteConfirmation()

		// Center the confirmation dialog on the screen
		centeredConfirmation := lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			confirmation,
		)

		return centeredConfirmation
	}

	return mainView
}

// renderDeleteConfirmation renders a clean delete confirmation dialog
func (m Model) renderDeleteConfirmation() string {
	// Remove emojis (uncertain width depending on terminal) to stabilize the frame
	title := "DELETE SSH HOST"
	question := fmt.Sprintf("Are you sure you want to delete host '%s'?", m.deleteHost)
	action := "This action cannot be undone."
	help := "Enter: confirm • Esc: cancel"

	// Individual styles (do not affect width via internal centering)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	questionStyle := lipgloss.NewStyle()
	actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	lines := []string{
		titleStyle.Render(title),
		"",
		questionStyle.Render(question),
		"",
		actionStyle.Render(action),
		"",
		helpStyle.Render(help),
	}

	// Compute the real maximum width (ANSI-safe via lipgloss.Width)
	maxw := 0
	for _, ln := range lines {
		w := lipgloss.Width(ln)
		if w > maxw {
			maxw = w
		}
	}
	// Minimal width for aesthetics
	if maxw < 40 {
		maxw = 40
	}

	// Build the raw text block (without centering) then apply the container style
	raw := strings.Join(lines, "\n")

	// Container style: wider horizontal padding, stable border
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		PaddingTop(1).PaddingBottom(1).PaddingLeft(2).PaddingRight(2).
		Width(maxw + 4) // +4 = internal margin (2 spaces of left/right padding)

	return box.Render(raw)
}
