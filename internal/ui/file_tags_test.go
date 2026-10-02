package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Gu1llaum-3/sshm/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestInheritedTagsInTableSearchAndVisibility(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	hosts := []config.SSHHost{
		{Name: "database", Hostname: "db.example.com", Tags: []string{"db", "prod"}, InheritedTags: []string{"prod"}},
		{Name: "secret", Hostname: "secret.example.com", InheritedTags: []string{"hidden", "prod"}},
	}
	m := NewModel(hosts, "", false, "", true)
	if len(m.filteredHosts) != 1 {
		t.Fatalf("visible hosts = %+v", m.filteredHosts)
	}
	assertTags := func() {
		t.Helper()
		if rows := m.table.Rows(); len(rows) != 1 || rows[0][2] != "%prod #db" {
			t.Fatalf("table rows = %v", rows)
		}
	}
	assertTags()
	m.updateTableRows()
	assertTags()
	if got := m.filterHosts("prod"); len(got) != 1 || got[0].Name != "database" {
		t.Fatalf("search = %+v", got)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	m = updated.(Model)
	if len(m.filteredHosts) != 2 {
		t.Fatalf("after H = %+v", m.filteredHosts)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	m = updated.(Model)
	if len(m.filteredHosts) != 1 {
		t.Fatalf("after second H = %+v", m.filteredHosts)
	}
}

func TestInheritedTagsStayReadOnlyOnEdit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte("# FileTags: prod\n# Tags: db\nHost database\n HostName db.example.com\n"), 0600); err != nil {
		t.Fatal(err)
	}
	form, err := NewEditForm("database", NewStyles(100), 100, 50, path)
	if err != nil {
		t.Fatal(err)
	}
	if got := form.inputs[7].Value(); got != "db" {
		t.Fatalf("editable tags = %q", got)
	}
	view := form.View()
	if !strings.Contains(view, "%prod") || !strings.Contains(view, "not editable here") {
		t.Fatalf("missing inherited tag hint: %s", view)
	}
	form.inputs[7].SetValue("")
	result := form.submitEditForm()().(editFormSubmitMsg)
	if result.err != nil {
		t.Fatal(result.err)
	}
	host, err := config.GetSSHHostFromFile("database", path)
	if err != nil {
		t.Fatal(err)
	}
	if len(host.Tags) != 0 || !reflect.DeepEqual(host.InheritedTags, []string{"prod"}) {
		t.Fatalf("tags after save: %+v", host)
	}
	info, err := NewInfoForm("database", NewStyles(100), 100, 50, path)
	if err != nil {
		t.Fatal(err)
	}
	if view := info.View(); !strings.Contains(view, "Inherited") || !strings.Contains(view, "prod") {
		t.Fatalf("missing inherited info: %s", view)
	}
}

func TestConfigWarningsSurviveReloadsWithoutStderr(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	path := filepath.Join(dir, "config")
	content := "Host first\n HostName first.example.com\n\n#FileTags: ignored\nHost second\n HostName second.example.com\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	hosts, warnings, err := config.ParseSSHConfigFileWithWarnings(path)
	if err != nil {
		t.Fatal(err)
	}
	capture, err := os.CreateTemp(dir, "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Close()
	original := os.Stderr
	os.Stderr = capture
	defer func() { os.Stderr = original }()
	m := NewModel(hosts, path, false, "", true, warnings...)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m = updated.(Model)
	if !strings.Contains(m.View(), "FileTags") {
		t.Fatal("missing startup warning")
	}
	for _, msg := range []tea.Msg{addFormSubmitMsg{}, editFormSubmitMsg{}, moveFormSubmitMsg{}} {
		updated, _ = m.Update(msg)
		m = updated.(Model)
		if len(m.configWarnings) != 1 {
			t.Fatalf("warnings after %T: %v", msg, m.configWarnings)
		}
		if !strings.Contains(m.View(), "FileTags") {
			t.Fatalf("missing warning after %T", msg)
		}
	}
	// The normal delete path also updates diagnostics. Deleting the first host
	// neutralizes the promoted directive, so the stale warning must disappear.
	m.deleteMode = true
	m.deleteHost = &hosts[0]
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if len(m.configWarnings) != 0 || strings.Contains(m.View(), "FileTags") {
		t.Fatalf("stale warning after delete: %v", m.configWarnings)
	}
	if info, err := capture.Stat(); err != nil || info.Size() != 0 {
		t.Fatalf("unexpected stderr: info=%v, err=%v", info, err)
	}
}

func TestConfigWarningsFitOneStatusLine(t *testing.T) {
	for _, width := range []int{40, 80} {
		m := createTestModel()
		updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 40})
		m = updated.(Model)
		baselineLines := strings.Count(m.View(), "\n")
		warning := "# FileTags: ignored after first Host or Include (/" + strings.Repeat("directory/", 20) + "config:3)"
		m.configWarnings = []string{warning, warning}
		view := m.View()
		if got := strings.Count(view, "\n"); got != baselineLines+1 {
			t.Fatalf("width %d: status added %d lines", width, got-baselineLines)
		}
		found := false
		for _, line := range strings.Split(view, "\n") {
			if strings.Contains(line, "# FileTags:") {
				found = true
				// JoinVertical pads every line to the widest existing component.
				visible := strings.TrimRight(ansi.Strip(line), " ")
				if ansi.StringWidth(visible) > width {
					t.Fatalf("width %d: status width %d", width, ansi.StringWidth(visible))
				}
			}
		}
		if !found {
			t.Fatalf("width %d: diagnostic was truncated away", width)
		}
	}
}
