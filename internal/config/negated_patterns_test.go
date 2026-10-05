package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC-1 (#66): a pattern starting with "!" is a negation, not a host
func TestNegatedPatternsAreNotHosts_AC1(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config")
	writeFile(t, configFile, "Host some-hosts* !except-this-host\n    User u\n\nHost web !web-old\n    HostName web.example.com\n")

	byName := hostsByName(t, configFile)
	if len(byName) != 1 || len(byName["web"]) != 1 {
		t.Errorf("hosts = %v, want only web", byName)
	}

	for _, name := range []string{"!except-this-host", "!web-old"} {
		if found, err := QuickHostExistsInFile(name, configFile); err != nil || found {
			t.Errorf("QuickHostExistsInFile(%q) = %v, %v, want false", name, found, err)
		}
	}
}

// AC-2 (#66): a host with negations on its Host line is a single-host block,
// and saving it, changed or not, keeps the negations on that line
func TestSavingHostKeepsItsNegations_AC2(t *testing.T) {
	for _, tt := range []struct {
		name, prefix, hostname string
	}{
		{"changed", "", "new.example.com"},
		{"changed with tags", "# Tags: t\n", "new.example.com"},
		{"unchanged", "", "web.example.com"},
		{"unchanged with tags", "# Tags: t\n", "web.example.com"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config")
			writeFile(t, configFile, tt.prefix+"Host web !web-old\n    HostName web.example.com\n\nHost other\n    HostName other.example.com\n")

			host := hostsByName(t, configFile)["web"][0]
			host.Hostname = tt.hostname
			if err := UpdateSSHHostInFile("web", host, configFile); err != nil {
				t.Fatalf("UpdateSSHHostInFile() error = %v", err)
			}

			lines := hostLinesOf(t, configFile)
			want := []string{"Host web !web-old", "Host other"}
			if len(lines) != len(want) || lines[0] != want[0] || lines[1] != want[1] {
				t.Errorf("Host lines = %q, want %q", lines, want)
			}
			if got := hostsByName(t, configFile)["web"]; len(got) != 1 || got[0].Hostname != tt.hostname {
				t.Errorf("web after save = %+v, want HostName %q", got, tt.hostname)
			}
		})
	}
}

// hostLinesOf returns the Host lines of a config file, trimmed, in file order
func hostLinesOf(t *testing.T, path string) []string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, line := range strings.Split(string(content), "\n") {
		if isHostLine(line) {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return lines
}

// AC-2 (#66): saving a multi-host block keeps its negations too
func TestSavingMultiHostBlockKeepsItsNegations_AC2(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config")
	writeFile(t, configFile, "Host a b !c\n    HostName shared.example.com\n")

	isMulti, names, err := IsPartOfMultiHostDeclaration("a", configFile)
	if err != nil || !isMulti || len(names) != 2 {
		t.Fatalf("IsPartOfMultiHostDeclaration() = %v, %q, %v, want true, [a b]", isMulti, names, err)
	}
	host := hostsByName(t, configFile)["a"][0]
	if err := UpdateMultiHostBlock(names, names, host, configFile); err != nil {
		t.Fatalf("UpdateMultiHostBlock() error = %v", err)
	}
	if lines := hostLinesOf(t, configFile); len(lines) != 1 || lines[0] != "Host a b !c" {
		t.Errorf("Host lines = %q, want [\"Host a b !c\"]", lines)
	}
}
