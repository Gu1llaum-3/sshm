package key

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttachSetsIdentityFileForUniqueHost(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	if err := os.WriteFile(configPath, []byte("Host demo\n    HostName 203.0.113.10\n"), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := Attach(context.Background(), AttachOptions{
		Host:       "demo",
		Identity:   privateKey,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if result.HostName != "demo" {
		t.Fatalf("HostName = %q, want demo", result.HostName)
	}
	if result.Identity != privateKey {
		t.Fatalf("Identity = %q, want %q", result.Identity, privateKey)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "IdentityFile "+privateKey) {
		t.Fatalf("config missing attached identity:\n%s", string(content))
	}
}

func TestAttachAcceptsPubPathAndNormalizesToPrivateKey(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	if err := os.WriteFile(configPath, []byte("Host demo\n    HostName 203.0.113.10\n"), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privateKey+".pub", []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Attach(context.Background(), AttachOptions{
		Host:       "demo",
		Identity:   privateKey + ".pub",
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if result.Identity != privateKey {
		t.Fatalf("Identity = %q, want %q", result.Identity, privateKey)
	}
}

func TestAttachDryRunDoesNotWriteConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	original := "Host demo\n    HostName 203.0.113.10\n"
	if err := os.WriteFile(configPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := Attach(context.Background(), AttachOptions{
		Host:       "demo",
		Identity:   privateKey,
		ConfigPath: configPath,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if result.AlreadyAttached {
		t.Fatalf("AlreadyAttached = true, want false")
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != original {
		t.Fatalf("config changed during dry-run:\n%s", string(content))
	}
}

func TestAttachRejectsAmbiguousHost(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	if err := os.WriteFile(configPath, []byte("Host demo\n    HostName 203.0.113.10\nInclude extra.conf\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "extra.conf"), []byte("Host demo\n    HostName 203.0.113.11\n"), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Attach(context.Background(), AttachOptions{
		Host:       "demo",
		Identity:   privateKey,
		ConfigPath: configPath,
	})
	if err == nil {
		t.Fatal("Attach() error = nil, want ambiguity error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("error = %q, want ambiguity wording", err)
	}
}

func TestAttachRejectsMultiHostDeclaration(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	if err := os.WriteFile(configPath, []byte("Host demo demo-alt\n    HostName 203.0.113.10\n"), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Attach(context.Background(), AttachOptions{
		Host:       "demo",
		Identity:   privateKey,
		ConfigPath: configPath,
	})
	if err == nil {
		t.Fatal("Attach() error = nil, want multi-host rejection")
	}
	if !strings.Contains(err.Error(), "multi-host declaration") {
		t.Fatalf("error = %q, want multi-host wording", err)
	}
}

func TestAttachRejectsMultipleIdentityFiles(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	content := `Host demo
    HostName 203.0.113.10
    IdentityFile ~/.ssh/id_one
    IdentityFile ~/.ssh/id_two
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Attach(context.Background(), AttachOptions{
		Host:       "demo",
		Identity:   privateKey,
		ConfigPath: configPath,
	})
	if err == nil {
		t.Fatal("Attach() error = nil, want multiple IdentityFile rejection")
	}
	if !strings.Contains(err.Error(), "multiple explicit IdentityFile") {
		t.Fatalf("error = %q, want multiple IdentityFile wording", err)
	}
}
