package config

import (
	"os"
	"path/filepath"
	"testing"
)

// hostLineVariants are Host lines that OpenSSH and the sshm parser read as "Host foo"
var hostLineVariants = []string{"host foo", "Host\tfoo", "Host=foo", `Host "foo"`, "HOST  foo # main"}

func hostsByName(t *testing.T, path string) map[string][]SSHHost {
	t.Helper()
	hosts, err := ParseSSHConfigFile(path)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}
	byName := map[string][]SSHHost{}
	for _, host := range hosts {
		byName[host.Name] = append(byName[host.Name], host)
	}
	return byName
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// AC-7: a block the parser reads can be found again to be edited or deleted
func TestHostBlocksWrittenAnyWayCanBeEditedAndDeleted_AC7(t *testing.T) {
	for _, hostLine := range hostLineVariants {
		for _, prefix := range []string{"", "# Tags: t\n"} {
			config := prefix + hostLine + "\n    HostName foo.example.com\nHost bar\n    HostName bar.example.com\n"

			t.Run("exists/"+prefix+hostLine, func(t *testing.T) {
				configFile := filepath.Join(t.TempDir(), "config")
				writeFile(t, configFile, config)
				if found, err := HostExistsInSpecificFile("foo", configFile); err != nil || !found {
					t.Errorf("HostExistsInSpecificFile() = %v, %v, want true", found, err)
				}
			})

			t.Run("update/"+prefix+hostLine, func(t *testing.T) {
				configFile := filepath.Join(t.TempDir(), "config")
				writeFile(t, configFile, config)
				host := hostsByName(t, configFile)["foo"][0]
				host.Hostname = "new.example.com"
				if err := UpdateSSHHostInFile("foo", host, configFile); err != nil {
					t.Fatalf("UpdateSSHHostInFile() error = %v", err)
				}
				byName := hostsByName(t, configFile)
				if len(byName["foo"]) != 1 || byName["foo"][0].Hostname != "new.example.com" {
					t.Errorf("foo after update = %+v, want one host with the new HostName", byName["foo"])
				}
				if len(byName["bar"]) != 1 || byName["bar"][0].Hostname != "bar.example.com" {
					t.Errorf("bar after update = %+v, want it untouched", byName["bar"])
				}
			})

			t.Run("delete/"+prefix+hostLine, func(t *testing.T) {
				configFile := filepath.Join(t.TempDir(), "config")
				writeFile(t, configFile, config)
				host := hostsByName(t, configFile)["foo"][0]
				if err := DeleteSSHHostFromFileWithLine("foo", configFile, host.LineNumber); err != nil {
					t.Fatalf("DeleteSSHHostFromFileWithLine() error = %v", err)
				}
				byName := hostsByName(t, configFile)
				if len(byName["foo"]) != 0 {
					t.Errorf("foo still present after delete: %+v", byName["foo"])
				}
				if len(byName["bar"]) != 1 || byName["bar"][0].Hostname != "bar.example.com" {
					t.Errorf("bar after delete = %+v, want it untouched", byName["bar"])
				}
			})
		}
	}
}

// AC-7: a multi-host block written another way is found as a multi-host block
func TestMultiHostBlocksWrittenAnyWayCanBeEdited_AC7(t *testing.T) {
	for _, hostLine := range []string{"host foo other", "Host\tfoo\tother", "Host=foo other", `Host "foo" 'other'`} {
		t.Run(hostLine, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config")
			writeFile(t, configFile, hostLine+"\n    HostName shared.example.com\n")

			isMulti, names, err := IsPartOfMultiHostDeclaration("foo", configFile)
			if err != nil || !isMulti || len(names) != 2 || names[1] != "other" {
				t.Fatalf("IsPartOfMultiHostDeclaration() = %v, %q, %v, want true, [foo other]", isMulti, names, err)
			}

			host := hostsByName(t, configFile)["foo"][0]
			host.Hostname = "new.example.com"
			names = []string{"foo", "other"}
			if err := UpdateMultiHostBlock(names, names, host, configFile); err != nil {
				t.Fatalf("UpdateMultiHostBlock() error = %v", err)
			}
			byName := hostsByName(t, configFile)
			for _, name := range names {
				if len(byName[name]) != 1 || byName[name][0].Hostname != "new.example.com" {
					t.Errorf("%s after update = %+v, want one host with the new HostName", name, byName[name])
				}
			}
		})
	}
}

// AC-7: with the same host name in two files, or twice in one file, edits and
// deletes only touch the targeted block
func TestDuplicateHostNamesTargetTheRightBlock_AC7(t *testing.T) {
	t.Run("two files", func(t *testing.T) {
		dir := t.TempDir()
		fileA, fileB := filepath.Join(dir, "a.conf"), filepath.Join(dir, "b.conf")
		writeFile(t, fileA, "host dup\n    HostName a.example.com\n")
		writeFile(t, fileB, "Host=dup\n    HostName b.example.com\n")

		host := hostsByName(t, fileB)["dup"][0]
		host.Hostname = "b2.example.com"
		if err := UpdateSSHHostInFile("dup", host, fileB); err != nil {
			t.Fatalf("UpdateSSHHostInFile() error = %v", err)
		}
		hostA := hostsByName(t, fileA)["dup"][0]
		if err := DeleteSSHHostFromFileWithLine("dup", fileA, hostA.LineNumber); err != nil {
			t.Fatalf("DeleteSSHHostFromFileWithLine() error = %v", err)
		}
		if got := hostsByName(t, fileA)["dup"]; len(got) != 0 {
			t.Errorf("a.conf still has dup: %+v", got)
		}
		if got := hostsByName(t, fileB)["dup"]; len(got) != 1 || got[0].Hostname != "b2.example.com" {
			t.Errorf("b.conf dup = %+v, want the updated host", got)
		}
	})

	t.Run("one file", func(t *testing.T) {
		configFile := filepath.Join(t.TempDir(), "config")
		writeFile(t, configFile, "host dup\n    HostName first.example.com\n\nHost\tdup\n    HostName second.example.com\n")
		second := hostsByName(t, configFile)["dup"][1]
		if err := DeleteSSHHostFromFileWithLine("dup", configFile, second.LineNumber); err != nil {
			t.Fatalf("DeleteSSHHostFromFileWithLine() error = %v", err)
		}
		if got := hostsByName(t, configFile)["dup"]; len(got) != 1 || got[0].Hostname != "first.example.com" {
			t.Errorf("dup after delete = %+v, want only the first block", got)
		}
	})
}
