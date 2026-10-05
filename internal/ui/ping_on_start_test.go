package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Gu1llaum-3/sshm/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

// pingedHosts runs the command and every command it batches, and returns the
// names of the hosts pinged, sorted
func pingedHosts(cmd tea.Cmd) []string {
	var names []string
	if cmd == nil {
		return names
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			names = append(names, pingedHosts(c)...)
		}
	case pingResultMsg:
		names = append(names, msg.HostName)
	}
	slices.Sort(names)
	return names
}

// AC-6 (#73): at startup the TUI pings every visible host, unless
// ping_on_start is false; p pings them again in both cases
func TestPingOnStart_AC6(t *testing.T) {
	// Port 1 on localhost refuses the connection right away
	hosts := []config.SSHHost{
		{Name: "web", Hostname: "127.0.0.1", Port: "1"},
		{Name: "db", Hostname: "127.0.0.1", Port: "1"},
		{Name: "old", Hostname: "127.0.0.1", Port: "1", Tags: []string{"hidden"}},
	}
	visible := []string{"db", "web"}

	for _, tt := range []struct {
		name       string
		configJSON string
		want       []string
	}{
		{"default", "", visible},
		{"option true", `{"ping_on_start": true}`, visible},
		{"option false", `{"ping_on_start": false}`, nil},
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

			m := NewModel(hosts, "", false, "", true)
			if got := pingedHosts(m.Init()); !slices.Equal(got, tt.want) {
				t.Errorf("hosts pinged at startup = %v, want %v", got, tt.want)
			}

			m, _ = press(t, m, tea.KeyMsg{Type: tea.KeyTab})
			_, cmd := press(t, m, typed("p")...)
			if got := pingedHosts(cmd); !slices.Equal(got, visible) {
				t.Errorf("hosts pinged by p = %v, want %v", got, visible)
			}
		})
	}
}
