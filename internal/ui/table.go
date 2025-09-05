package ui

import (
	"strings"

	"sshm/internal/config"
	"sshm/internal/history"

	"github.com/charmbracelet/bubbles/table"
)

// calculateNameColumnWidth calculates the optimal width for the Name column
// based on the longest hostname, with a minimum of 8 and maximum of 40 characters
func calculateNameColumnWidth(hosts []config.SSHHost) int {
	maxLength := 8 // Minimum width to accommodate the "Name" header

	for _, host := range hosts {
		if len(host.Name) > maxLength {
			maxLength = len(host.Name)
		}
	}

	// Add some padding (2 characters) for better visual spacing
	maxLength += 2

	// Limit the maximum width to avoid extremely large columns
	if maxLength > 40 {
		maxLength = 40
	}

	return maxLength
}

// calculateTagsColumnWidth calculates the optimal width for the Tags column
// based on the longest tag string, with a minimum of 8 and maximum of 40 characters
func calculateTagsColumnWidth(hosts []config.SSHHost) int {
	maxLength := 8 // Minimum width to accommodate the "Tags" header

	for _, host := range hosts {
		// Format tags exactly as they appear in the table
		var tagsStr string
		if len(host.Tags) > 0 {
			// Add the # prefix to each tag and join them with spaces
			var formattedTags []string
			for _, tag := range host.Tags {
				formattedTags = append(formattedTags, "#"+tag)
			}
			tagsStr = strings.Join(formattedTags, " ")
		}

		if len(tagsStr) > maxLength {
			maxLength = len(tagsStr)
		}
	}

	// Add some padding (2 characters) for better visual spacing
	maxLength += 2

	// Limit the maximum width to avoid extremely large columns
	if maxLength > 40 {
		maxLength = 40
	}

	return maxLength
}

// calculateLastLoginColumnWidth calculates the optimal width for the Last Login column
// based on the longest time format, with a minimum of 12 and maximum of 20 characters
func calculateLastLoginColumnWidth(hosts []config.SSHHost, historyManager *history.HistoryManager) int {
	maxLength := 12 // Minimum width to accommodate the "Last Login" header

	if historyManager != nil {
		for _, host := range hosts {
			if lastConnect, exists := historyManager.GetLastConnectionTime(host.Name); exists {
				timeStr := formatTimeAgo(lastConnect)
				if len(timeStr) > maxLength {
					maxLength = len(timeStr)
				}
			}
		}
	}

	// Add some padding (2 characters) for better visual spacing
	maxLength += 2

	// Limit the maximum width to avoid extremely large columns
	if maxLength > 20 {
		maxLength = 20
	}

	return maxLength
}

// updateTableRows updates the table with filtered hosts
func (m *Model) updateTableRows() {
	var rows []table.Row
	hostsToShow := m.filteredHosts
	if hostsToShow == nil {
		hostsToShow = m.hosts
	}

	for _, host := range hostsToShow {
		// Format tags for display
		var tagsStr string
		if len(host.Tags) > 0 {
			// Add the # prefix to each tag and join them with spaces
			var formattedTags []string
			for _, tag := range host.Tags {
				formattedTags = append(formattedTags, "#"+tag)
			}
			tagsStr = strings.Join(formattedTags, " ")
		}

		// Format last login information
		var lastLoginStr string
		if m.historyManager != nil {
			if lastConnect, exists := m.historyManager.GetLastConnectionTime(host.Name); exists {
				lastLoginStr = formatTimeAgo(lastConnect)
			}
		}

		rows = append(rows, table.Row{
			host.Name,
			host.Hostname,
			// host.User,      // Commented to save space
			// host.Port,      // Commented to save space
			tagsStr,
			lastLoginStr,
		})
	}

	m.table.SetRows(rows)

	// Update table height and columns based on current terminal size
	m.updateTableHeight()
	m.updateTableColumns()
}

// updateTableHeight dynamically adjusts table height based on terminal size
func (m *Model) updateTableHeight() {
	if !m.ready {
		return
	}

	// Calculate dynamic table height based on terminal size
	// Layout breakdown:
	// - ASCII title: 5 lines (1 empty + 4 text lines)
	// - Search bar: 1 line
	// - Help text: 1 line
	// - App margins/spacing: 3 lines
	// - Safety margin: 3 lines (to ensure UI elements are always visible)
	// Total reserved: 12 lines minimum to preserve essential UI elements
	reservedHeight := 12
	availableHeight := m.height - reservedHeight
	hostCount := len(m.table.Rows())

	// Minimum height should be at least 3 rows for basic usability
	// Even in very small terminals, we want to show at least header + 2 hosts
	minTableHeight := 4 // 1 header + 3 data rows minimum
	maxTableHeight := availableHeight
	if maxTableHeight < minTableHeight {
		maxTableHeight = minTableHeight
	}

	tableHeight := 1 // header
	dataRowsNeeded := hostCount
	maxDataRows := maxTableHeight - 1 // subtract 1 for header

	if dataRowsNeeded <= maxDataRows {
		// We have enough space for all hosts
		tableHeight += dataRowsNeeded
	} else {
		// We need to limit to available space
		tableHeight += maxDataRows
	}

	// Update table height
	m.table.SetHeight(tableHeight)
}

// updateTableColumns dynamically adjusts table column widths based on terminal size
func (m *Model) updateTableColumns() {
	if !m.ready {
		return
	}

	hostsToShow := m.filteredHosts
	if hostsToShow == nil {
		hostsToShow = m.hosts
	}

	// Calculate base column widths
	nameWidth := calculateNameColumnWidth(hostsToShow)
	tagsWidth := calculateTagsColumnWidth(hostsToShow)
	lastLoginWidth := calculateLastLoginColumnWidth(hostsToShow, m.historyManager)

	// Fixed column widths
	hostnameWidth := 25
	// userWidth := 12      // Commented to save space
	// portWidth := 6       // Commented to save space

	// Calculate total width needed for all columns
	totalFixedWidth := hostnameWidth // + userWidth + portWidth  // Commented columns
	totalVariableWidth := nameWidth + tagsWidth + lastLoginWidth
	totalWidth := totalFixedWidth + totalVariableWidth

	// Available width (accounting for table borders and padding)
	availableWidth := m.width - 4 // 4 chars for borders and padding

	// If the table is too wide, scale down the variable columns proportionally
	if totalWidth > availableWidth {
		excessWidth := totalWidth - availableWidth
		variableColumnsWidth := totalVariableWidth

		if variableColumnsWidth > 0 {
			// Reduce variable columns proportionally
			nameReduction := (excessWidth * nameWidth) / variableColumnsWidth
			tagsReduction := (excessWidth * tagsWidth) / variableColumnsWidth
			lastLoginReduction := excessWidth - nameReduction - tagsReduction

			nameWidth = max(8, nameWidth-nameReduction)
			tagsWidth = max(8, tagsWidth-tagsReduction)
			lastLoginWidth = max(10, lastLoginWidth-lastLoginReduction)
		}
	}

	// Create new columns with updated widths and sort indicators
	nameTitle := "Name"
	lastLoginTitle := "Last Login"

	// Add sort indicators based on current sort mode
	switch m.sortMode {
	case SortByName:
		nameTitle += " ↓"
	case SortByLastUsed:
		lastLoginTitle += " ↓"
	}

	columns := []table.Column{
		{Title: nameTitle, Width: nameWidth},
		{Title: "Hostname", Width: hostnameWidth},
		// {Title: "User", Width: userWidth},      // Commented to save space
		// {Title: "Port", Width: portWidth},      // Commented to save space
		{Title: "Tags", Width: tagsWidth},
		{Title: lastLoginTitle, Width: lastLoginWidth},
	}

	m.table.SetColumns(columns)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
