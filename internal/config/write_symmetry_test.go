package config

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// trickyValues are values that the tokenizer would split or transform if
// they were written as they are.
var trickyValues = []struct {
	name  string
	value string
}{
	{"space", `/a b/id`},
	{"two spaces", `/a  b/id`},
	{"tab", "/a\tb/id"},
	{"double quote", `/a"b/id`},
	{"single quote", `/a'b/id`},
	{"backslash", `/a\b/id`},
	{"escaped-looking backslashes", `/a\\b\"c\'d\ e`},
	{"trailing backslash with space", `C:\my dir\`},
	{"leading hash", `#a/id`},
	{"leading equals", `=a/id`},
	{"windows path", `C:\Users\me\id`},
	{"windows path with space", `C:\My Drive\key`},
}

// writtenFields are the host fields whose value goes through formatSSHConfigValue,
// with the ssh -G keyword that prints them. ProxyJump is not one of them: OpenSSH
// takes it raw (see TestProxyJumpIsKeptRaw_AC8).
var writtenFields = []struct {
	name   string
	sshKey string
	set    func(*SSHHost, string)
	get    func(SSHHost) string
}{
	{"IdentityFile", "identityfile",
		func(h *SSHHost, v string) { h.Identity = v },
		func(h SSHHost) string { return h.Identity }},
	{"extra IdentityFile", "identityfile",
		func(h *SSHHost, v string) { h.Identity = "/main"; h.ExtraIdentities = []string{v} },
		func(h SSHHost) string {
			if len(h.ExtraIdentities) != 1 {
				return "<missing>"
			}
			return h.ExtraIdentities[0]
		}},
	{"User", "user",
		func(h *SSHHost, v string) { h.User = v },
		func(h SSHHost) string { return h.User }},
	{"HostName", "hostname",
		func(h *SSHHost, v string) { h.Hostname = v },
		func(h SSHHost) string { return h.Hostname }},
}

func writeTrickyHost(t *testing.T, set func(*SSHHost, string), value string) string {
	t.Helper()
	configFile := filepath.Join(t.TempDir(), "config")
	host := SSHHost{Name: "h", Hostname: "h.example.com"}
	set(&host, value)
	if err := AddSSHHostToFile(host, configFile); err != nil {
		t.Fatalf("AddSSHHostToFile() error = %v", err)
	}
	return configFile
}

// AC-8: a value written by sshm is read back unchanged by sshm
func TestWrittenValuesAreReadBackUnchanged_AC8(t *testing.T) {
	for _, field := range writtenFields {
		for _, tt := range trickyValues {
			t.Run(field.name+"/"+tt.name, func(t *testing.T) {
				configFile := writeTrickyHost(t, field.set, tt.value)
				host := findHost(t, configFile, "h")
				if got := field.get(host); got != tt.value {
					t.Errorf("read back %q, want %q", got, tt.value)
				}
			})
		}
	}
}

// AC-8: a value written by sshm is read back unchanged by the ssh client
func TestWrittenValuesAreReadBackBySSHClient_AC8(t *testing.T) {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("ssh client not installed")
	}
	for _, field := range writtenFields {
		for _, tt := range trickyValues {
			t.Run(field.name+"/"+tt.name, func(t *testing.T) {
				configFile := writeTrickyHost(t, field.set, tt.value)
				out, err := exec.Command(sshPath, "-F", configFile, "-G", "h").CombinedOutput()
				if err != nil {
					t.Fatalf("ssh -G failed: %v\n%s", err, out)
				}
				var values []string
				for _, line := range strings.Split(string(out), "\n") {
					if strings.HasPrefix(line, field.sshKey+" ") {
						values = append(values, strings.TrimPrefix(line, field.sshKey+" "))
					}
				}
				for _, v := range values {
					if v == tt.value {
						return
					}
				}
				t.Errorf("ssh -G %s = %q, want it to contain %q", field.sshKey, values, tt.value)
			})
		}
	}
}

// AC-8: OpenSSH takes ProxyJump raw, quotes included, without its trailing
// comment, so sshm reads and writes it the same way
func TestProxyJumpIsKeptRaw_AC8(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		for line, want := range map[string]string{
			`    ProxyJump "user@bastion" # c`: `"user@bastion"`,
			`    ProxyJump a, b`:               `a, b`,
			`    ProxyJump=a,b`:                `a,b`,
		} {
			if got := parseSingleHost(t, line).ProxyJump; got != want {
				t.Errorf("%s: ProxyJump = %q, want %q", line, got, want)
			}
		}
	})

	sshPath, _ := exec.LookPath("ssh")
	for _, value := range []string{`user@bastion:2222,admin@other`, `a, b`, `"user@bastion"`} {
		t.Run("round trip/"+value, func(t *testing.T) {
			configFile := writeTrickyHost(t, func(h *SSHHost, v string) { h.ProxyJump = v }, value)
			if got := findHost(t, configFile, "h").ProxyJump; got != value {
				t.Errorf("sshm reads back %q, want %q", got, value)
			}
			if sshPath == "" {
				return
			}
			// Some OpenSSH versions reject the value even written by hand
			// (9.6 rejects "a, b"): sshm writes it raw, so there is nothing to compare
			handWritten := filepath.Join(t.TempDir(), "config")
			writeFile(t, handWritten, "Host h\n    ProxyJump "+value+"\n")
			if out, err := exec.Command(sshPath, "-F", handWritten, "-G", "h").CombinedOutput(); err != nil {
				t.Skipf("this ssh rejects ProxyJump %s written by hand: %s", value, out)
			}
			out, err := exec.Command(sshPath, "-F", configFile, "-G", "h").CombinedOutput()
			if err != nil {
				t.Fatalf("ssh -G failed: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), "\nproxyjump "+value+"\n") {
				t.Errorf("ssh -G does not read proxyjump %q:\n%s", value, out)
			}
		})
	}
}

// AC-8: a host name that needs quotes can be added, found, edited and deleted
func TestQuotedHostNameRoundTrip_AC8(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config")
	if err := AddSSHHostToFile(SSHHost{Name: "my host", Hostname: "a.example.com"}, configFile); err != nil {
		t.Fatalf("AddSSHHostToFile() error = %v", err)
	}
	host := findHost(t, configFile, "my host")
	if found, err := HostExistsInSpecificFile("my host", configFile); err != nil || !found {
		t.Fatalf("HostExistsInSpecificFile() = %v, %v, want true", found, err)
	}
	host.Hostname = "b.example.com"
	if err := UpdateSSHHostInFile("my host", host, configFile); err != nil {
		t.Fatalf("UpdateSSHHostInFile() error = %v", err)
	}
	if got := findHost(t, configFile, "my host").Hostname; got != "b.example.com" {
		t.Errorf("HostName after update = %q, want %q", got, "b.example.com")
	}
	if err := DeleteSSHHostFromFileWithLine("my host", configFile, host.LineNumber); err != nil {
		t.Fatalf("DeleteSSHHostFromFileWithLine() error = %v", err)
	}
	if found, _ := HostExistsInSpecificFile("my host", configFile); found {
		t.Error("host still present after delete")
	}
}
