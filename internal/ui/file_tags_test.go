package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Gu1llaum-3/sshm/internal/config"
	tea "github.com/charmbracelet/bubbletea"
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
