package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gu1llaum-3/sshm/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestApplySourceFileFilter(t *testing.T) {
	hosts := []config.SSHHost{
		{Name: "a", SourceFile: "/home/u/.ssh/config"},
		{Name: "b", SourceFile: "/home/u/.ssh/work.conf"},
		{Name: "c", SourceFile: "/home/u/.ssh/work.conf"},
		{Name: "d", SourceFile: "/home/u/.ssh/perso.conf"},
	}

	t.Run("empty selected returns all", func(t *testing.T) {
		got := applySourceFileFilter(hosts, "")
		if len(got) != len(hosts) {
			t.Fatalf("expected %d hosts, got %d", len(hosts), len(got))
		}
	})

	t.Run("matching path filters to that file", func(t *testing.T) {
		got := applySourceFileFilter(hosts, "/home/u/.ssh/work.conf")
		if len(got) != 2 {
			t.Fatalf("expected 2 hosts, got %d", len(got))
		}
		for _, h := range got {
			if h.SourceFile != "/home/u/.ssh/work.conf" {
				t.Errorf("unexpected host %q with SourceFile %q", h.Name, h.SourceFile)
			}
		}
	})

	t.Run("non-matching path returns empty", func(t *testing.T) {
		got := applySourceFileFilter(hosts, "/nowhere.conf")
		if len(got) != 0 {
			t.Fatalf("expected 0 hosts, got %d", len(got))
		}
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		got := applySourceFileFilter(nil, "/home/u/.ssh/work.conf")
		if len(got) != 0 {
			t.Fatalf("expected 0 hosts, got %d", len(got))
		}
	})
}

func TestFileSelectorWithAllPrependsSynthetic(t *testing.T) {
	// We can't easily mock GetAllConfigFilesFromBase, so drive the helper
	// directly through newFileSelectorFromFiles then apply the same
	// prepending logic and assert the invariants a caller relies on.
	files := []string{"/a.conf", "/b.conf"}
	styles := NewStyles(80)
	m, err := newFileSelectorFromFiles("t", styles, 80, 24, files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.files = append([]string{""}, m.files...)
	m.displayNames = append([]string{"[All files]"}, m.displayNames...)

	if len(m.files) != 3 || m.files[0] != "" {
		t.Fatalf("expected synthetic entry at index 0, got %v", m.files)
	}
	if m.displayNames[0] != "[All files]" {
		t.Fatalf("expected display name '[All files]', got %q", m.displayNames[0])
	}
}

func TestSelectSourceFileWithNoVisibleHosts(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
	}{
		{"include-and-wildcard-only", "Include other.conf\nHost *\n  User default\n"},
		{"all-hosts-hidden", "Include other.conf\n# Tags: hidden\nHost secret\n  HostName secret.example.com\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			selected := filepath.Join(dir, "config")
			for path, data := range map[string]string{
				selected:                         tc.content,
				filepath.Join(dir, "other.conf"): "Host other\n  HostName other.example.com\n",
			} {
				if err := os.WriteFile(path, []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			hosts, err := config.ParseSSHConfigFile(selected)
			if err != nil {
				t.Fatal(err)
			}
			m := createTestModel()
			m.allHosts = hosts
			m.fileSelectorPurpose = purposeFilterHosts
			updated, _ := m.Update(fileSelectorMsg{selectedFile: selected})
			m = updated.(Model)
			if m.selectedSourceFile != selected {
				t.Errorf("filter = %q, want %q", m.selectedSourceFile, selected)
			}
			if len(m.filteredHosts) != 0 || len(m.table.Rows()) != 0 {
				t.Errorf("expected empty list, got %+v", m.filteredHosts)
			}
			if !strings.Contains(m.View(), "filtering by") || !strings.Contains(m.View(), "config") {
				t.Error("missing active file banner")
			}
			updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
			m = updated.(Model)
			if m.selectedSourceFile != "" || len(m.filteredHosts) != 1 || m.filteredHosts[0].Name != "other" {
				t.Fatalf("clearing empty filter: got %q, %+v", m.selectedSourceFile, m.filteredHosts)
			}
		})
	}
}

func TestSourceFileFilterSurvivesHiddenToggle(t *testing.T) {
	m := createTestModel()
	m.allHosts = []config.SSHHost{
		{Name: "other", SourceFile: "/x/main"},
		{Name: "secret", SourceFile: "/x/hidden.conf", Tags: []string{"hidden"}},
	}
	toggle := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")}
	updated, _ := m.Update(toggle)
	m = updated.(Model)
	m.fileSelectorPurpose = purposeFilterHosts
	updated, _ = m.Update(fileSelectorMsg{selectedFile: "/x/hidden.conf"})
	m = updated.(Model)
	if len(m.filteredHosts) != 1 || m.filteredHosts[0].Name != "secret" {
		t.Fatalf("initial filter: %+v", m.filteredHosts)
	}
	for _, wantCount := range []int{0, 1} {
		updated, _ = m.Update(toggle)
		m = updated.(Model)
		if m.selectedSourceFile != "/x/hidden.conf" {
			t.Errorf("filter lost after H: %q", m.selectedSourceFile)
		}
		if len(m.filteredHosts) != wantCount || len(m.table.Rows()) != wantCount {
			t.Fatalf("after H: want %d hosts, got %+v", wantCount, m.filteredHosts)
		}
		if wantCount == 1 && m.filteredHosts[0].Name != "secret" {
			t.Fatalf("unexpected host: %+v", m.filteredHosts)
		}
	}
}

func TestSourceFileFilterAfterDeleteOrMove(t *testing.T) {
	for _, action := range []string{"delete", "move"} {
		for _, remaining := range []bool{false, true} {
			name := action + "/last-host"
			if remaining {
				name = action + "/hosts-remain"
			}
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				t.Setenv("XDG_CONFIG_HOME", dir)
				t.Setenv("APPDATA", dir)
				main := filepath.Join(dir, "config")
				selected := filepath.Join(dir, "selected.conf")
				content := "Host target\n  HostName target.example.com\n"
				if remaining {
					content += "Host remaining\n  HostName remaining.example.com\n"
				}
				for path, data := range map[string]string{
					main:     "Include selected.conf\nHost other\n  HostName other.example.com\n",
					selected: content,
				} {
					if err := os.WriteFile(path, []byte(data), 0600); err != nil {
						t.Fatal(err)
					}
				}
				hosts, err := config.ParseSSHConfigFile(main)
				if err != nil {
					t.Fatal(err)
				}
				m := createTestModel()
				m.configFile = main
				m.allHosts = hosts
				m.selectedSourceFile = selected
				m.searchInput.SetValue("target")
				m.rebuildFilteredHosts()
				m.updateTableRows()
				if len(m.filteredHosts) != 1 {
					t.Fatalf("expected target before removal: %+v", m.filteredHosts)
				}
				var updated tea.Model
				if action == "delete" {
					updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
					m = updated.(Model)
					updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				} else {
					// The move form writes the files before emitting its success message.
					if err := config.DeleteSSHHostWithLine(m.filteredHosts[0]); err != nil {
						t.Fatal(err)
					}
					f, err := os.OpenFile(main, os.O_APPEND|os.O_WRONLY, 0600)
					if err != nil {
						t.Fatal(err)
					}
					_, err = f.WriteString("Host target\n  HostName target.example.com\n")
					closeErr := f.Close()
					if err != nil {
						t.Fatal(err)
					}
					if closeErr != nil {
						t.Fatal(closeErr)
					}
					updated, _ = m.Update(moveFormSubmitMsg{})
				}
				m = updated.(Model)
				wantFilter := ""
				if remaining {
					wantFilter = selected
				}
				if m.selectedSourceFile != wantFilter {
					t.Fatalf("filter = %q, want %q", m.selectedSourceFile, wantFilter)
				}
				// Search must not determine whether the file filter is cleared.
				if m.searchInput.Value() != "target" {
					t.Fatal("search was lost")
				}
				m.searchInput.SetValue("")
				m.rebuildFilteredHosts()
				m.updateTableRows()
				wantCount := 1
				if action == "move" && !remaining {
					wantCount = 2
				}
				if len(m.filteredHosts) != wantCount {
					t.Fatalf("want %d hosts, got %+v", wantCount, m.filteredHosts)
				}
				if remaining && m.filteredHosts[0].Name != "remaining" {
					t.Fatalf("unexpected remaining hosts: %+v", m.filteredHosts)
				}
			})
		}
	}
}

func TestSourceFileFilterWithSearchNavigation(t *testing.T) {
	m := createTestModel()
	m.allHosts = []config.SSHHost{
		{Name: "alpha", SourceFile: "/x/work.conf"},
		{Name: "beta", SourceFile: "/x/work.conf"},
		{Name: "gamma", SourceFile: "/x/work.conf"},
		{Name: "outside", SourceFile: "/x/other.conf"},
	}
	m.fileSelectorPurpose = purposeFilterHosts
	updated, _ := m.Update(fileSelectorMsg{selectedFile: "/x/work.conf"})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	m.table.SetCursor(2)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(Model)
	if m.table.Cursor() != 0 {
		t.Fatalf("search cursor = %d, want first result", m.table.Cursor())
	}
	if len(m.filteredHosts) != 3 {
		t.Fatalf("search results = %+v", m.filteredHosts)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.searchInput.Value() != "" || !m.searchMode {
		t.Fatal("Esc must clear search and keep focus")
	}
	if m.selectedSourceFile != "/x/work.conf" || len(m.filteredHosts) != 3 {
		t.Fatalf("Esc lost file filter: %q, %+v", m.selectedSourceFile, m.filteredHosts)
	}
	for _, host := range m.filteredHosts {
		if host.SourceFile != "/x/work.conf" {
			t.Fatalf("host outside selected file: %+v", host)
		}
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	view := m.View()
	for _, shortcut := range []string{"Tab: search", "c: filter by file", "C: clear filter"} {
		if !strings.Contains(view, shortcut) {
			t.Errorf("missing shortcut %q", shortcut)
		}
	}
}
