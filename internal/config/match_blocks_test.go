package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// matchBlock is the Match block of #30, as it must stay in the file
const matchBlock = "Match Host raspi exec \"true\"\n    HostName 192.168.1.100\n    User pi\n"

// matchAfterHostConfig returns the config of #30 with the Match block after
// the block of hostLine, separated from it by separator ("\n" or "")
func matchAfterHostConfig(hostLine, separator string) string {
	return hostLine + "\n    HostName 192.168.1.200\n" + separator + matchBlock + "\nHost other\n    HostName other.example.com\n"
}

// assertMatchBlockKept checks that the file holds the Match block once, unchanged
func assertMatchBlockKept(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(content), matchBlock) != 1 || strings.Count(string(content), "Match ") != 1 {
		t.Errorf("%s does not hold the Match block once, unchanged:\n%s", filepath.Base(path), content)
	}
}

// AC-1 (#30): a Match block ends the Host block before it, whatever the case,
// indentation or "=" form of the keyword
func TestMatchEndsTheHostBlock_AC1(t *testing.T) {
	for _, matchLine := range []string{`Match Host raspi exec "true"`, `  match host raspi`, `MATCH=host raspi`} {
		t.Run(matchLine, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config")
			writeFile(t, configFile, "Host raspi\n    HostName 192.168.1.200\n"+matchLine+"\n    HostName 192.168.1.100\n    User pi\n    Compression yes\n")

			hosts := hostsByName(t, configFile)["raspi"]
			if len(hosts) != 1 {
				t.Fatalf("raspi hosts = %+v, want one", hosts)
			}
			raspi := hosts[0]
			if raspi.Hostname != "192.168.1.200" || raspi.User != "" || raspi.Options != "" {
				t.Errorf("raspi = HostName %q, User %q, Options %q; want 192.168.1.200 and nothing from the Match block",
					raspi.Hostname, raspi.User, raspi.Options)
			}
		})
	}
}

// AC-5 (#30): the config of the issue, with the Match block first, still reads
// the Host block's HostName
func TestMatchBeforeHostIsIgnored_AC5(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config")
	writeFile(t, configFile, "Match Host raspi exec \"true\"\n    HostName 192.168.1.100\n\nHost raspi\n    HostName 192.168.1.200\n")

	hosts := hostsByName(t, configFile)["raspi"]
	if len(hosts) != 1 || hosts[0].Hostname != "192.168.1.200" {
		t.Errorf("raspi = %+v, want one host with HostName 192.168.1.200", hosts)
	}
}

// resolvedBySSH returns the hostname and user that ssh resolves for a host
func resolvedBySSH(t *testing.T, sshPath, configFile, host string) string {
	t.Helper()
	out, err := exec.Command(sshPath, "-F", configFile, "-G", host).CombinedOutput()
	if err != nil {
		t.Fatalf("ssh -G failed: %v\n%s", err, out)
	}
	var resolved []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "hostname ") || strings.HasPrefix(line, "user ") {
			resolved = append(resolved, line)
		}
	}
	return strings.Join(resolved, ", ")
}

// assertSavingKeepsTheMatchBlock saves the config with save, then checks that
// the Match block is still in the file and ssh resolves host the same way
func assertSavingKeepsTheMatchBlock(t *testing.T, configFile, host string, save func() error) {
	t.Helper()
	sshPath, sshErr := exec.LookPath("ssh")
	var before string
	if sshErr == nil {
		before = resolvedBySSH(t, sshPath, configFile, host)
	}

	if err := save(); err != nil {
		t.Fatalf("save error = %v", err)
	}

	assertMatchBlockKept(t, configFile)
	if sshErr != nil {
		t.Skip("ssh client not installed, ssh -G check skipped")
	}
	if after := resolvedBySSH(t, sshPath, configFile, host); after != before {
		t.Errorf("ssh -G %s after save = %q, want %q as before", host, after, before)
	}
}

// AC-2 (#30): saving a host followed by a Match block, unchanged, keeps the
// Match block and what ssh resolves for the host
func TestSavingHostKeepsTheMatchBlock_AC2(t *testing.T) {
	for name, separator := range map[string]string{"blank line": "\n", "no blank line": ""} {
		t.Run(name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config")
			writeFile(t, configFile, matchAfterHostConfig("Host raspi", separator))
			raspi := hostsByName(t, configFile)["raspi"][0]
			assertSavingKeepsTheMatchBlock(t, configFile, "raspi", func() error {
				return UpdateSSHHostInFile("raspi", raspi, configFile)
			})
		})

		t.Run("multi-host block/"+name, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config")
			writeFile(t, configFile, matchAfterHostConfig("Host raspi pi4", separator))
			raspi := hostsByName(t, configFile)["raspi"][0]
			names := []string{"raspi", "pi4"}
			assertSavingKeepsTheMatchBlock(t, configFile, "raspi", func() error {
				return UpdateMultiHostBlock(names, names, raspi, configFile)
			})
		})
	}
}

// AC-5 (#30): saving the host of the issue, with the Match block first, keeps
// the Match block
func TestSavingHostAfterMatchKeepsTheMatchBlock_AC5(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config")
	writeFile(t, configFile, matchBlock+"\nHost raspi\n    HostName 192.168.1.200\n")
	raspi := hostsByName(t, configFile)["raspi"][0]
	assertSavingKeepsTheMatchBlock(t, configFile, "raspi", func() error {
		return UpdateSSHHostInFile("raspi", raspi, configFile)
	})
}

// AC-3 (#30): deleting a host followed by a Match block, with no blank line
// between them, keeps the Match block
func TestDeletingHostKeepsTheMatchBlock_AC3(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config")
	writeFile(t, configFile, matchAfterHostConfig("Host raspi", ""))

	if err := DeleteSSHHostFromFile("raspi", configFile); err != nil {
		t.Fatalf("DeleteSSHHostFromFile() error = %v", err)
	}

	assertMatchBlockKept(t, configFile)
	byName := hostsByName(t, configFile)
	if len(byName["raspi"]) != 0 || len(byName["other"]) != 1 {
		t.Errorf("hosts after delete = %v, want only other", byName)
	}
}

// AC-4 (#30): moving a host followed by a Match block leaves the Match block
// in the source file and copies nothing of it to the target file
func TestMovingHostLeavesTheMatchBlock_AC4(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sshDir, "config")
	target := filepath.Join(sshDir, "other")
	writeFile(t, source, matchAfterHostConfig("Host raspi", ""))
	writeFile(t, target, "")

	if err := MoveHostToFile("raspi", target); err != nil {
		t.Fatalf("MoveHostToFile() error = %v", err)
	}

	assertMatchBlockKept(t, source)
	targetContent, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(targetContent)), "match") || strings.Contains(string(targetContent), "192.168.1.100") {
		t.Errorf("target after move holds part of the Match block:\n%s", targetContent)
	}
	if moved := hostsByName(t, target)["raspi"]; len(moved) != 1 || moved[0].Hostname != "192.168.1.200" {
		t.Errorf("raspi in target = %+v, want HostName 192.168.1.200", moved)
	}
}
