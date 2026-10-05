package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// AC-4 (#48): saving a host with SetEnv from the edit form keeps the line
// intact, and ssh still accepts the config
func TestEditFormKeepsSetEnv_AC4(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config")
	content := "Host h\n    HostName h.example.com\n    SetEnv TERM=xterm-256color\n    SendEnv LANG LC_*\n"
	if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	form, err := NewEditForm("h", NewStyles(80), 80, 60, configFile)
	if err != nil {
		t.Fatalf("NewEditForm() error = %v", err)
	}
	if msg, ok := form.submitEditForm()().(editFormSubmitMsg); !ok || msg.err != nil {
		t.Fatalf("submitEditForm() error = %v", msg.err)
	}

	saved, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"SetEnv TERM=xterm-256color", "SendEnv LANG LC_*"} {
		if !strings.Contains(string(saved), "    "+line+"\n") {
			t.Errorf("saved config lost %q:\n%s", line, saved)
		}
	}

	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("ssh client not installed")
	}
	out, err := exec.Command(sshPath, "-F", configFile, "-G", "h").CombinedOutput()
	if err != nil {
		t.Fatalf("ssh rejects the saved config: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "\nsetenv TERM=xterm-256color\n") {
		t.Errorf("ssh -G does not read setenv TERM=xterm-256color:\n%s", out)
	}
}
