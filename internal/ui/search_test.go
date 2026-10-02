package ui

import (
	"testing"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// createTestModel creates a model with test data for testing
func createTestModel() Model {
	hosts := []config.SSHHost{
		{Name: "server1", Hostname: "server1.example.com", User: "user1"},
		{Name: "server2", Hostname: "server2.example.com", User: "user2"},
		{Name: "server3", Hostname: "server3.example.com", User: "user3"},
		{Name: "web-server", Hostname: "web.example.com", User: "webuser"},
		{Name: "db-server", Hostname: "db.example.com", User: "dbuser"},
	}

	m := Model{
		allHosts:      hosts,
		hosts:         hosts,
		filteredHosts: hosts,
		searchInput:   textinput.New(),
		table:         table.New(),
		searchMode:    false,
		ready:         true,
		width:         80,
		height:        24,
		styles:        NewStyles(80),
	}

	// Initialize table with test data
	m.updateTableColumns()
	m.updateTableRows()

	return m
}

func TestSearchModeToggle(t *testing.T) {
	m := createTestModel()

	// Initially should not be in search mode
	if m.searchMode {
		t.Error("Model should not start in search mode")
	}

	// Simulate pressing "/" to enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	// Should now be in search mode
	if !m.searchMode {
		t.Error("Model should be in search mode after pressing '/'")
	}

	// The search input should be focused
	if !m.searchInput.Focused() {
		t.Error("Search input should be focused in search mode")
	}
}

func TestSearchFiltering(t *testing.T) {
	m := createTestModel()

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	// Type "server" in search
	for _, char := range "server" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		newModel, _ := m.Update(keyMsg)
		m = newModel.(Model)
	}

	// Should filter to only hosts containing "server"
	expectedHosts := []string{"server1", "server2", "server3", "web-server", "db-server"}
	if len(m.filteredHosts) != len(expectedHosts) {
		t.Errorf("Expected %d filtered hosts, got %d", len(expectedHosts), len(m.filteredHosts))
	}

	// Check that all filtered hosts contain "server"
	for _, host := range m.filteredHosts {
		found := false
		for _, expected := range expectedHosts {
			if host.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Unexpected host in filtered results: %s", host.Name)
		}
	}
}

func TestSearchFilteringSpecific(t *testing.T) {
	m := createTestModel()

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	// Type "web" in search
	for _, char := range "web" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		newModel, _ := m.Update(keyMsg)
		m = newModel.(Model)
	}

	// Should filter to only hosts containing "web"
	if len(m.filteredHosts) != 1 {
		t.Errorf("Expected 1 filtered host, got %d", len(m.filteredHosts))
	}

	if len(m.filteredHosts) > 0 && m.filteredHosts[0].Name != "web-server" {
		t.Errorf("Expected 'web-server', got '%s'", m.filteredHosts[0].Name)
	}
}

func TestSearchClearReturnToOriginal(t *testing.T) {
	m := createTestModel()
	originalHostCount := len(m.hosts)

	// Enter search mode and type something
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	// Type "web" in search
	for _, char := range "web" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		newModel, _ := m.Update(keyMsg)
		m = newModel.(Model)
	}

	// Should have filtered results
	if len(m.filteredHosts) >= originalHostCount {
		t.Error("Search should have filtered down the results")
	}

	// Clear the search by simulating backspace
	for i := 0; i < 3; i++ { // "web" is 3 characters
		keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
		newModel, _ := m.Update(keyMsg)
		m = newModel.(Model)
	}

	// Should return to all hosts
	if len(m.filteredHosts) != originalHostCount {
		t.Errorf("Expected %d hosts after clearing search, got %d", originalHostCount, len(m.filteredHosts))
	}
}

func TestCursorPositionAfterFiltering(t *testing.T) {
	m := createTestModel()

	// Move cursor down to position 2 (third item)
	m.table.SetCursor(2)
	initialCursor := m.table.Cursor()

	if initialCursor != 2 {
		t.Errorf("Expected cursor at position 2, got %d", initialCursor)
	}

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	// Type "web" - this will filter to only 1 result
	for _, char := range "web" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		newModel, _ := m.Update(keyMsg)
		m = newModel.(Model)
	}

	// Cursor should be reset to 0 since filtered results has only 1 item
	// and cursor position 2 would be out of bounds
	if len(m.filteredHosts) == 1 && m.table.Cursor() != 0 {
		t.Errorf("Expected cursor to be reset to 0 when filtered results are smaller, got %d", m.table.Cursor())
	}
}

func TestTabSwitchBetweenSearchAndTable(t *testing.T) {
	m := createTestModel()

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	if !m.searchMode {
		t.Error("Should be in search mode")
	}

	// Press Tab to switch to table
	keyMsg = tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if m.searchMode {
		t.Error("Should not be in search mode after Tab")
	}

	// Press Tab again to switch back to search
	keyMsg = tea.KeyMsg{Type: tea.KeyTab}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if !m.searchMode {
		t.Error("Should be in search mode after second Tab")
	}
}

func TestEnterExitsSearchMode(t *testing.T) {
	m := createTestModel()

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	if !m.searchMode {
		t.Error("Should be in search mode")
	}

	// Press Enter to exit search mode
	keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(Model)

	if m.searchMode {
		t.Error("Should not be in search mode after Enter")
	}
}

func TestSearchModeDoesNotTriggerOnEmptyInput(t *testing.T) {
	m := createTestModel()
	originalHostCount := len(m.hosts)

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	// At this point, filteredHosts should still be the same as the original hosts
	// because entering search mode should not trigger filtering with empty input
	if len(m.filteredHosts) != originalHostCount {
		t.Errorf("Expected %d hosts when entering search mode, got %d", originalHostCount, len(m.filteredHosts))
	}
}

func TestSearchByHostname(t *testing.T) {
	m := createTestModel()

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	// Search by hostname part "example.com"
	searchTerm := "example.com"
	for _, char := range searchTerm {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		newModel, _ := m.Update(keyMsg)
		m = newModel.(Model)
	}

	// All hosts should match since they all have "example.com" in hostname
	if len(m.filteredHosts) != len(m.hosts) {
		t.Errorf("Expected all %d hosts to match hostname search, got %d", len(m.hosts), len(m.filteredHosts))
	}
}

func TestSearchByUser(t *testing.T) {
	m := createTestModel()

	// Enter search mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(Model)

	// Search by user "user1"
	searchTerm := "user1"
	for _, char := range searchTerm {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		newModel, _ := m.Update(keyMsg)
		m = newModel.(Model)
	}

	// Only server1 should match
	if len(m.filteredHosts) != 1 {
		t.Errorf("Expected 1 host to match user search, got %d", len(m.filteredHosts))
	}

	if len(m.filteredHosts) > 0 && m.filteredHosts[0].Name != "server1" {
		t.Errorf("Expected 'server1' to match user search, got '%s'", m.filteredHosts[0].Name)
	}
}

func TestSourceFileFilterKeybindings(t *testing.T) {
	hosts := []config.SSHHost{
		{Name: "a", SourceFile: "/x/main"},
		{Name: "b", SourceFile: "/x/work.conf"},
		{Name: "c", SourceFile: "/x/work.conf"},
	}
	m := Model{
		allHosts: hosts, hosts: hosts, filteredHosts: hosts,
		searchInput: textinput.New(), table: table.New(),
		ready: true, width: 80, height: 24, styles: NewStyles(80),
	}
	m.updateTableColumns()
	m.updateTableRows()

	// Simulate selecting a file via the selector message path.
	m.fileSelectorPurpose = purposeFilterHosts
	newModel, _ := m.Update(fileSelectorMsg{selectedFile: "/x/work.conf"})
	m = newModel.(Model)

	if m.selectedSourceFile != "/x/work.conf" {
		t.Fatalf("selectedSourceFile = %q, want %q", m.selectedSourceFile, "/x/work.conf")
	}
	if len(m.filteredHosts) != 2 {
		t.Fatalf("expected 2 filtered hosts, got %d", len(m.filteredHosts))
	}

	// Press "C" to clear.
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	m = newModel.(Model)

	if m.selectedSourceFile != "" {
		t.Fatalf("after C, selectedSourceFile = %q, want empty", m.selectedSourceFile)
	}
	if len(m.filteredHosts) != 3 {
		t.Fatalf("after C, expected 3 hosts, got %d", len(m.filteredHosts))
	}
}

func TestFilterComposesWithSearchAndHidden(t *testing.T) {
	hosts := []config.SSHHost{
		{Name: "web-main", SourceFile: "/x/main"},
		{Name: "db-main", SourceFile: "/x/main"},
		{Name: "web-work", SourceFile: "/x/work.conf"},
		{Name: "db-work", SourceFile: "/x/work.conf"},
		{Name: "secret-work", SourceFile: "/x/work.conf", Tags: []string{"hidden"}},
	}
	m := Model{
		allHosts: hosts, searchInput: textinput.New(), table: table.New(),
		ready: true, width: 80, height: 24, styles: NewStyles(80),
		selectedSourceFile: "/x/work.conf",
	}
	m.updateTableColumns()
	m.rebuildFilteredHosts()

	// File filter alone: hidden excluded by default, 2 remaining.
	if len(m.filteredHosts) != 2 {
		t.Fatalf("file filter alone: expected 2 hosts, got %d", len(m.filteredHosts))
	}

	// File filter + search "web" -> 1 host.
	m.searchInput.SetValue("web")
	m.rebuildFilteredHosts()
	if len(m.filteredHosts) != 1 || m.filteredHosts[0].Name != "web-work" {
		t.Fatalf("file+search: got %+v", m.filteredHosts)
	}

	// Toggle hidden ON, clear search -> 3 hosts from work.conf.
	m.searchInput.SetValue("")
	m.showHidden = true
	m.rebuildFilteredHosts()
	if len(m.filteredHosts) != 3 {
		t.Fatalf("file+hidden: expected 3 hosts, got %d", len(m.filteredHosts))
	}
}
