package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/history"
	tea "github.com/charmbracelet/bubbletea"
)

// press sends the keys to the model, in order, and returns the updated model
// with the command returned by the last key
func press(t *testing.T, m Model, keys ...tea.KeyMsg) (Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, key := range keys {
		var updated tea.Model
		updated, cmd = m.Update(key)
		m = updated.(Model)
	}
	return m, cmd
}

func typed(text string) []tea.KeyMsg {
	var keys []tea.KeyMsg
	for _, r := range text {
		keys = append(keys, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return keys
}

var slash = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")}

// AC-1: in search mode, Up/Down and Ctrl+P/Ctrl+N move the selection without
// changing the search text or leaving the search
func TestSearchKeysMoveTheSelection_AC1(t *testing.T) {
	m, _ := press(t, createTestModel(), append([]tea.KeyMsg{slash}, typed("server")...)...)
	if len(m.filteredHosts) < 3 {
		t.Fatalf("expected several matches for %q, got %d", "server", len(m.filteredHosts))
	}

	steps := []struct {
		key    tea.KeyMsg
		cursor int
	}{
		{tea.KeyMsg{Type: tea.KeyDown}, 1},
		{tea.KeyMsg{Type: tea.KeyCtrlN}, 2},
		{tea.KeyMsg{Type: tea.KeyUp}, 1},
		{tea.KeyMsg{Type: tea.KeyCtrlP}, 0},
	}
	for _, step := range steps {
		m, _ = press(t, m, step.key)
		if got := m.table.Cursor(); got != step.cursor {
			t.Errorf("after %s: cursor = %d, want %d", step.key, got, step.cursor)
		}
		if !m.searchMode || !m.searchInput.Focused() {
			t.Errorf("after %s: left the search", step.key)
		}
		if got := m.searchInput.Value(); got != "server" {
			t.Errorf("after %s: search text = %q, want %q", step.key, got, "server")
		}
	}
}

// AC-3: every change of the filter text puts the selection back on the first row
func TestFilterChangeSelectsFirstResult_AC3(t *testing.T) {
	m, _ := press(t, createTestModel(), append([]tea.KeyMsg{slash}, typed("server")...)...)
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyDown})
	if m.table.Cursor() != 2 {
		t.Fatalf("setup: cursor = %d, want 2", m.table.Cursor())
	}

	m, _ = press(t, m, typed("1")...)
	if got := m.table.Cursor(); got != 0 {
		t.Errorf("after typing: cursor = %d, want 0", got)
	}

	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	if got := m.table.Cursor(); got != 0 {
		t.Errorf("after deleting a character: cursor = %d, want 0", got)
	}
}

// withHistory gives the model a history manager stored under a temporary HOME
func withHistory(t *testing.T, m Model) Model {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	historyManager, err := history.NewHistoryManager()
	if err != nil {
		t.Fatalf("NewHistoryManager() error = %v", err)
	}
	m.historyManager = historyManager
	return m
}

// AC-2: Enter in search mode connects to the selected host, like Enter in the table
func TestEnterInSearchConnectsToSelectedHost_AC2(t *testing.T) {
	m := withHistory(t, createTestModel())
	m, _ = press(t, m, append([]tea.KeyMsg{slash}, typed("db")...)...)

	m, cmd := press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter in search returned no command, want the ssh command")
	}
	if got := m.historyManager.GetConnectionCount("db-server"); got != 1 {
		t.Errorf("connections recorded for db-server = %d, want 1", got)
	}
}

// AC-2: with no result, Enter in search does nothing and the focus stays in the search
func TestEnterInSearchWithoutResultDoesNothing_AC2(t *testing.T) {
	m := withHistory(t, createTestModel())
	m, _ = press(t, m, append([]tea.KeyMsg{slash}, typed("nomatch")...)...)
	if len(m.filteredHosts) != 0 {
		t.Fatalf("setup: %d matches, want 0", len(m.filteredHosts))
	}

	m, cmd := press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Enter without result returned a command, want none")
	}
	if !m.searchMode || !m.searchInput.Focused() {
		t.Error("Enter without result left the search")
	}
}

func withKeyBindings(m Model, disableEscQuit bool) Model {
	appConfig := config.GetDefaultAppConfig()
	appConfig.KeyBindings.DisableEscQuit = disableEscQuit
	m.appConfig = &appConfig
	return m
}

func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

var esc = tea.KeyMsg{Type: tea.KeyEsc}

// AC-4: Esc in search clears the text first, and quits only once the field is empty
func TestEscInSearchClearsThenQuits_AC4(t *testing.T) {
	m := withKeyBindings(createTestModel(), false)
	m, _ = press(t, m, append([]tea.KeyMsg{slash}, typed("db")...)...)

	m, cmd := press(t, m, esc)
	if quits(cmd) {
		t.Fatal("Esc with text quit, want the text cleared")
	}
	if got := m.searchInput.Value(); got != "" {
		t.Errorf("search text after Esc = %q, want empty", got)
	}
	if len(m.filteredHosts) != len(m.hosts) {
		t.Errorf("hosts listed after Esc = %d, want all %d", len(m.filteredHosts), len(m.hosts))
	}
	if !m.searchMode || !m.searchInput.Focused() {
		t.Error("Esc with text left the search")
	}

	if _, cmd = press(t, m, esc); !quits(cmd) {
		t.Error("Esc with an empty field did not quit")
	}
}

// AC-4: with disable_esc_quit, Esc on an empty field does nothing; Ctrl+C still quits
func TestEscWithDisableEscQuitAndCtrlC_AC4(t *testing.T) {
	m := withKeyBindings(createTestModel(), true)
	m, _ = press(t, m, slash)

	m, cmd := press(t, m, esc)
	if quits(cmd) {
		t.Error("Esc quit despite disable_esc_quit")
	}
	if !m.searchMode {
		t.Error("Esc on an empty field left the search")
	}

	if _, cmd = press(t, m, tea.KeyMsg{Type: tea.KeyCtrlC}); !quits(cmd) {
		t.Error("Ctrl+C in search did not quit")
	}
}

// AC-5: Tab still switches between search and table, and letters are typed in
// the search but act as shortcuts in the table
func TestTabAndShortcutsAreUnchanged_AC5(t *testing.T) {
	m := withKeyBindings(createTestModel(), false)
	m, _ = press(t, m, slash)

	m, cmd := press(t, m, typed("q")...)
	if quits(cmd) || m.searchInput.Value() != "q" {
		t.Errorf("q in search: quit=%v text=%q, want it typed", quits(cmd), m.searchInput.Value())
	}

	m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if m.searchMode || !m.table.Focused() {
		t.Fatal("Tab did not give the focus to the table")
	}
	if _, cmd = press(t, m, typed("q")...); !quits(cmd) {
		t.Error("q in the table did not quit")
	}
}

// AC-6: the search has the focus at startup, unless focus_search_on_start is
// false in config.json and -s is not passed
func TestStartupFocus_AC6(t *testing.T) {
	for _, tt := range []struct {
		name       string
		configJSON string
		searchFlag bool
		wantSearch bool
	}{
		{"default", "", false, true},
		{"option false", `{"focus_search_on_start": false}`, false, false},
		{"option false with -s", `{"focus_search_on_start": false}`, true, true},
		{"option true", `{"focus_search_on_start": true}`, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
			if tt.configJSON != "" {
				path, err := config.GetAppConfigPath()
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(tt.configJSON), 0644); err != nil {
					t.Fatal(err)
				}
			}

			m := NewModel(createTestModel().hosts, "", tt.searchFlag, "dev", true)
			if m.searchMode != tt.wantSearch || m.searchInput.Focused() != tt.wantSearch {
				t.Errorf("searchMode=%v input focused=%v, want search focus %v",
					m.searchMode, m.searchInput.Focused(), tt.wantSearch)
			}
			if m.table.Focused() == tt.wantSearch {
				t.Errorf("table focused = %v, want %v", m.table.Focused(), !tt.wantSearch)
			}
		})
	}
}
