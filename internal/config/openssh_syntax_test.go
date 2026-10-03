package config

import (
	"os"
	"path/filepath"
	"testing"
)

// parseSingleHost writes a config made of a "Host h" line followed by body,
// parses it and returns the host named h.
func parseSingleHost(t *testing.T, body string) SSHHost {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(configFile, []byte("Host h\n"+body+"\n"), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	hosts, err := ParseSSHConfigFile(configFile)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}
	for _, host := range hosts {
		if host.Name == "h" {
			return host
		}
	}
	t.Fatalf("host h not found in %+v", hosts)
	return SSHHost{}
}

// AC-1: values are read the way OpenSSH reads them (checked with ssh -G, OpenSSH 10.3)
func TestParseFollowsOpenSSHSyntax_AC1(t *testing.T) {
	identity := func(h SSHHost) string { return h.Identity }
	tests := []struct {
		name  string
		line  string
		field func(SSHHost) string
		want  string
	}{
		{"double quotes keep inner spaces", `    IdentityFile "/a  b/id"`, identity, `/a  b/id`},
		{"single quotes", `    IdentityFile '/a b/id'`, identity, `/a b/id`},
		{"escaped space", `    IdentityFile /a\ b/id`, identity, `/a b/id`},
		{"windows path unquoted", `    IdentityFile C:\Users\me\id`, identity, `C:\Users\me\id`},
		{"windows path double-quoted", `    IdentityFile "C:\Users\me\id"`, identity, `C:\Users\me\id`},
		{"escaped backslashes", `    IdentityFile C:\\x\\id`, identity, `C:\x\id`},
		{"escaped quote inside double quotes", `    IdentityFile "/a\"b"`, identity, `/a"b`},
		{"backslash kept inside single quotes", `    IdentityFile '/a\ b'`, identity, `/a\ b`},
		{"backslash-space kept inside double quotes", `    IdentityFile "/a\ b"`, identity, `/a\ b`},
		{"quoted part glued to text", `    IdentityFile "/a"b`, identity, `/ab`},
		{"trailing comment", `    IdentityFile /a/id # old key`, identity, `/a/id`},
		{"hash inside a word", `    IdentityFile /a/id#x`, identity, `/a/id#x`},
		{"hash inside quotes", `    IdentityFile "/a #b"`, identity, `/a #b`},
		{"key=value", `    IdentityFile=/a/id`, identity, `/a/id`},
		{"key = value", `    IdentityFile = /a/id`, identity, `/a/id`},
		{"tab separators", "\tIdentityFile\t/a/id", identity, `/a/id`},
		{"quoted user", `    User "john doe"`, func(h SSHHost) string { return h.User }, `john doe`},
		{"quoted hostname", `    HostName "ex.com"`, func(h SSHHost) string { return h.Hostname }, `ex.com`},
		{"proxycommand kept raw", `    ProxyCommand ssh -W "%h:%p" bastion # c`, func(h SSHHost) string { return h.ProxyCommand }, `ssh -W "%h:%p" bastion # c`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := parseSingleHost(t, tt.line)
			if got := tt.field(host); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// AC-2: Include accepts quoted paths with spaces and several patterns
func TestIncludeQuotedPathsAndSeveralPatterns_AC2(t *testing.T) {
	tempDir := t.TempDir()
	spaced := filepath.Join(tempDir, "conf d")
	if err := os.MkdirAll(spaced, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(spaced, "a.conf"):  "Host spaced-host\n    HostName a.example.com\n",
		filepath.Join(tempDir, "b.conf"): "Host second-host\n    HostName b.example.com\n",
		filepath.Join(tempDir, "c.conf"): "Host third-host\n    HostName c.example.com\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	mainConfig := filepath.Join(tempDir, "config")
	content := "Include \"" + spaced + "/*\"\nInclude b.conf c.conf\n"
	if err := os.WriteFile(mainConfig, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	hosts, err := ParseSSHConfigFile(mainConfig)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}
	found := map[string]bool{}
	for _, host := range hosts {
		found[host.Name] = true
	}
	for _, name := range []string{"spaced-host", "second-host", "third-host"} {
		if !found[name] {
			t.Errorf("host %q not loaded through Include, got %+v", name, found)
		}
	}
}

// AC-3: a ProxyCommand written by sshm (as "ProxyCommand=...") is read back as ProxyCommand
func TestProxyCommandWrittenBySSHMIsReadBack_AC3(t *testing.T) {
	for _, command := range []string{"ssh -W %h:%p bastion", "none"} {
		t.Run(command, func(t *testing.T) {
			configFile := filepath.Join(t.TempDir(), "config")
			host := SSHHost{Name: "proxied", Hostname: "example.com", ProxyCommand: command}
			if err := AddSSHHostToFile(host, configFile); err != nil {
				t.Fatalf("AddSSHHostToFile() error = %v", err)
			}
			hosts, err := ParseSSHConfigFile(configFile)
			if err != nil {
				t.Fatalf("ParseSSHConfigFile() error = %v", err)
			}
			if len(hosts) != 1 {
				t.Fatalf("expected 1 host, got %d", len(hosts))
			}
			if hosts[0].ProxyCommand != command {
				t.Errorf("ProxyCommand = %q, want %q", hosts[0].ProxyCommand, command)
			}
			if hosts[0].Options != "" {
				t.Errorf("Options = %q, want empty", hosts[0].Options)
			}
		})
	}
}
