package ui

import (
	"strings"

	"sshm/internal/config"
	"sshm/internal/history"

	"github.com/charmbracelet/bubbles/table"
)

// calculateDynamicColumnWidths calculates optimal column widths based on terminal width
// and content length, ensuring all content fits when possible
func (m *Model) calculateDynamicColumnWidths(hosts []config.SSHHost) (int, int, int, int) {
	if m.width <= 0 {
		// Fallback to static widths if terminal width is not available
		return calculateNameColumnWidth(hosts), 25, calculateTagsColumnWidth(hosts), calculateLastLoginColumnWidth(hosts, m.historyManager)
	}

	// Calculate content lengths
	maxNameLength := 8       // Minimum for "Name" header + status indicator
	maxHostnameLength := 8   // Minimum for "Hostname" header
	maxTagsLength := 8       // Minimum for "Tags" header
	maxLastLoginLength := 12 // Minimum for "Last Login" header

	for _, host := range hosts {
		// Name column includes status indicator (2 chars) + space (1 char) + name
		nameLength := 3 + len(host.Name)
		if nameLength > maxNameLength {
			maxNameLength = nameLength
		}

		if len(host.Hostname) > maxHostnameLength {
			maxHostnameLength = len(host.Hostname)
		}

		// Calculate tags string length
		var tagsStr string
		if len(host.Tags) > 0 {
			var formattedTags []string
			for _, tag := range host.Tags {
				formattedTags = append(formattedTags, "#"+tag)
			}
			tagsStr = strings.Join(formattedTags, " ")
		}
		if len(tagsStr) > maxTagsLength {
			maxTagsLength = len(tagsStr)
		}

		// Calculate last login length
		if m.historyManager != nil {
			if lastConnect, exists := m.historyManager.GetLastConnectionTime(host.Name); exists {
				timeStr := formatTimeAgo(lastConnect)
				if len(timeStr) > maxLastLoginLength {
					maxLastLoginLength = len(timeStr)
				}
			}
		}
	}

	// Add padding to each column
	maxNameLength += 2
	maxHostnameLength += 2
	maxTagsLength += 2
	maxLastLoginLength += 2

	// Calculate available width (minus borders and separators)
	// Table has borders (2 chars) + column separators (3 chars between 4 columns)
	availableWidth := m.width - 5

	totalNeededWidth := maxNameLength + maxHostnameLength + maxTagsLength + maxLastLoginLength

	if totalNeededWidth <= availableWidth {
		// Everything fits perfectly
		return maxNameLength, maxHostnameLength, maxTagsLength, maxLastLoginLength
	}

	// Need to adjust widths - prioritize columns by importance
	// Priority: Name > Hostname > Last Login > Tags

	// Calculate minimum widths
	minNameWidth := 15 // Enough for status + short name
	minHostnameWidth := 15
	minLastLoginWidth := 12
	minTagsWidth := 10

	remainingWidth := availableWidth

	// Allocate minimum widths first
	nameWidth := minNameWidth
	hostnameWidth := minHostnameWidth
	lastLoginWidth := minLastLoginWidth
	tagsWidth := minTagsWidth

	remainingWidth -= (nameWidth + hostnameWidth + lastLoginWidth + tagsWidth)

	// Distribute remaining space proportionally
	if remainingWidth > 0 {
		// Calculate how much each column wants beyond minimum
		nameWant := maxNameLength - minNameWidth
		hostnameWant := maxHostnameLength - minHostnameWidth
		lastLoginWant := maxLastLoginLength - minLastLoginWidth
		tagsWant := maxTagsLength - minTagsWidth

		totalWant := nameWant + hostnameWant + lastLoginWant + tagsWant

		if totalWant > 0 {
			// Distribute proportionally
			nameExtra := (nameWant * remainingWidth) / totalWant
			hostnameExtra := (hostnameWant * remainingWidth) / totalWant
			lastLoginExtra := (lastLoginWant * remainingWidth) / totalWant
			tagsExtra := remainingWidth - nameExtra - hostnameExtra - lastLoginExtra

			nameWidth += nameExtra
			hostnameWidth += hostnameExtra
			lastLoginWidth += lastLoginExtra
			tagsWidth += tagsExtra
		}
	}

	return nameWidth, hostnameWidth, tagsWidth, lastLoginWidth
}

// updateTableRows updates the table with filtered hosts
func (m *Model) updateTableRows() {
	var rows []table.Row
	hostsToShow := m.filteredHosts
	if hostsToShow == nil {
		hostsToShow = m.hosts
	}

	for _, host := range hostsToShow {
		// Get ping status indicator
		statusIndicator := m.getPingStatusIndicator(host.Name)

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
			statusIndicator + " " + host.Name,
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
	// Total reserved: 13 lines minimum to preserve essential UI elements
	reservedHeight := 13
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

	// Add one extra line to prevent the last host from being hidden
	// This compensates for table rendering quirks in bubble tea
	tableHeight += 1

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

	// Use dynamic column width calculation
	nameWidth, hostnameWidth, tagsWidth, lastLoginWidth := m.calculateDynamicColumnWidths(hostsToShow)

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

// Legacy functions for compatibility

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
