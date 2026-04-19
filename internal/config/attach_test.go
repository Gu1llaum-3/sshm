package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeAttachedIdentityExpandsWindowsStyleHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	privateKey := filepath.Join(home, ".ssh", "id_ed25519_demo")
	if err := os.MkdirAll(filepath.Dir(privateKey), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := NormalizeAttachedIdentityForKey(`~\.ssh\id_ed25519_demo`)
	if err != nil {
		t.Fatalf("NormalizeAttachedIdentityForKey() error = %v", err)
	}
	if got != privateKey {
		t.Fatalf("path = %q, want %q", got, privateKey)
	}
}

func TestNormalizeAttachedIdentityExpandsUserProfilePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)

	privateKey := filepath.Join(home, ".ssh", "id_ed25519_demo")
	if err := os.MkdirAll(filepath.Dir(privateKey), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := NormalizeAttachedIdentityForKey(`%USERPROFILE%\.ssh\id_ed25519_demo`)
	if err != nil {
		t.Fatalf("NormalizeAttachedIdentityForKey() error = %v", err)
	}
	if got != privateKey {
		t.Fatalf("path = %q, want %q", got, privateKey)
	}
}

func TestNormalizeAttachedIdentityRejectsBlankPath(t *testing.T) {
	_, err := NormalizeAttachedIdentityForKey("   ")
	if err == nil {
		t.Fatal("NormalizeAttachedIdentityForKey() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "identity path is required") {
		t.Fatalf("error = %q, want identity path required", err)
	}
}

func TestSetSSHHostIdentityStopsBeforeMatchBlock(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	content := `Host demo
    HostName 203.0.113.10

Match user root
    IdentityFile ~/.ssh/id_match
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	host := SSHHost{
		Name:       "demo",
		SourceFile: configPath,
		LineNumber: 1,
	}
	if err := SetSSHHostIdentity(host, privateKey); err != nil {
		t.Fatalf("SetSSHHostIdentity() error = %v", err)
	}

	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(updated)
	if !strings.Contains(got, "Host demo\n    HostName 203.0.113.10\n    IdentityFile "+privateKey+"\n\nMatch user root") {
		t.Fatalf("identity not inserted before Match block:\n%s", got)
	}
	if !strings.Contains(got, "Match user root\n    IdentityFile ~/.ssh/id_match") {
		t.Fatalf("match block changed unexpectedly:\n%s", got)
	}
}
