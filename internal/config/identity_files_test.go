package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// keyLines are the lines of a host with several keys, in file order. Saving the
// host without changes must keep all of them, in this order.
var keyLines = []string{
	`IdentityFile ~/.ssh/a`,
	`IdentityFile "~/.ssh/b c"`,
	`ForwardAgent yes`,
}

// directiveLinesOf returns the IdentityFile and ForwardAgent lines of a config
// file, trimmed, in file order.
func directiveLinesOf(t *testing.T, path string) []string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "IdentityFile") || strings.HasPrefix(line, "ForwardAgent") {
			got = append(got, line)
		}
	}
	return got
}

func writeConfig(t *testing.T, path, hostLine string) {
	t.Helper()
	content := hostLine + "\n    HostName keys.example.com\n    " + strings.Join(keyLines, "\n    ") + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func findHost(t *testing.T, path, name string) SSHHost {
	t.Helper()
	hosts, err := ParseSSHConfigFile(path)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}
	for _, host := range hosts {
		if host.Name == name {
			return host
		}
	}
	t.Fatalf("host %q not found", name)
	return SSHHost{}
}

// AC-5: saving or moving a host with several IdentityFile lines keeps them all
func TestSavingHostKeepsAllIdentityFiles_AC5(t *testing.T) {
	t.Run("single host block", func(t *testing.T) {
		configFile := filepath.Join(t.TempDir(), "config")
		writeConfig(t, configFile, "Host keys")
		host := findHost(t, configFile, "keys")
		if err := UpdateSSHHostInFile("keys", host, configFile); err != nil {
			t.Fatalf("UpdateSSHHostInFile() error = %v", err)
		}
		if got := directiveLinesOf(t, configFile); !reflect.DeepEqual(got, keyLines) {
			t.Errorf("lines after save = %q, want %q", got, keyLines)
		}
	})

	t.Run("multi-host block", func(t *testing.T) {
		configFile := filepath.Join(t.TempDir(), "config")
		writeConfig(t, configFile, "Host keys keys2")
		host := findHost(t, configFile, "keys")
		names := []string{"keys", "keys2"}
		if err := UpdateMultiHostBlock(names, names, host, configFile); err != nil {
			t.Fatalf("UpdateMultiHostBlock() error = %v", err)
		}
		if got := directiveLinesOf(t, configFile); !reflect.DeepEqual(got, keyLines) {
			t.Errorf("lines after save = %q, want %q", got, keyLines)
		}
	})

	t.Run("move to another file", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		sshDir := filepath.Join(home, ".ssh")
		if err := os.MkdirAll(sshDir, 0700); err != nil {
			t.Fatal(err)
		}
		mainConfig := filepath.Join(sshDir, "config")
		target := filepath.Join(sshDir, "other.conf")
		writeConfig(t, mainConfig, "Include other.conf\n\nHost keys")
		if err := os.WriteFile(target, []byte("Host other\n    HostName other.example.com\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := MoveHostToFile("keys", target); err != nil {
			t.Fatalf("MoveHostToFile() error = %v", err)
		}
		if got := directiveLinesOf(t, target); !reflect.DeepEqual(got, keyLines) {
			t.Errorf("lines after move = %q, want %q", got, keyLines)
		}
	})
}

// AC-5: the first IdentityFile is the main key (the one OpenSSH tries first),
// the others are kept in file order
func TestParseKeepsEveryIdentityFileInOrder_AC5(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config")
	writeConfig(t, configFile, "Host keys")
	host := findHost(t, configFile, "keys")
	if host.Identity != "~/.ssh/a" {
		t.Errorf("Identity = %q, want %q", host.Identity, "~/.ssh/a")
	}
	if want := []string{"~/.ssh/b c"}; !reflect.DeepEqual(host.ExtraIdentities, want) {
		t.Errorf("ExtraIdentities = %q, want %q", host.ExtraIdentities, want)
	}
}
