package key

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	outputs  map[string][]byte
	errors   map[string]error
	runCalls []string
}

func (f *fakeRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	call := name + " " + shellCall(args)
	if out, ok := f.outputs[call]; ok {
		return out, f.errors[call]
	}
	return nil, fmt.Errorf("unexpected output call: %s", call)
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) error {
	call := name + " " + shellCall(args)
	f.runCalls = append(f.runCalls, call)
	return f.errors[call]
}

func shellCall(args []string) string {
	return strings.Join(args, " ")
}

func TestInventoryReturnsDeclaredReferencesOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(sshDir, "id_ed25519_test")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privateKey+".pub", []byte("public"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte("ignored"), 0600); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(home, "ssh_config")
	configContent := `Host demo
    HostName example.com
    IdentityFile ~/.ssh/id_ed25519_test

Host effective-only
    HostName example.net
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		outputs: map[string][]byte{
			"ssh-keygen -lf " + privateKey: []byte("256 SHA256:testfingerprint " + privateKey + " (ED25519)\n"),
		},
		errors: make(map[string]error),
	}

	items, err := Inventory(context.Background(), runner, configPath)
	if err != nil {
		t.Fatalf("Inventory() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Path != privateKey {
		t.Fatalf("path = %q", items[0].Path)
	}
	if items[0].Fingerprint != "SHA256:testfingerprint" {
		t.Fatalf("fingerprint = %q", items[0].Fingerprint)
	}
	if items[0].Algorithm != "ED25519" {
		t.Fatalf("algorithm = %q", items[0].Algorithm)
	}
	if items[0].PublicKeyPath != privateKey+".pub" {
		t.Fatalf("public key path = %q", items[0].PublicKeyPath)
	}
	if len(items[0].References) != 1 || items[0].References[0].Host != "demo" {
		t.Fatalf("references = %#v", items[0].References)
	}
	if items[0].CanDelete {
		t.Fatalf("CanDelete = true, want false when declared references exist")
	}
}

func TestInventorySkipsFilesWithoutFingerprint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(sshDir, "id_ed25519_test")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		outputs: map[string][]byte{},
		errors: map[string]error{
			"ssh-keygen -lf " + privateKey: fmt.Errorf("fingerprint failed"),
		},
	}

	items, err := Inventory(context.Background(), runner, "")
	if err != nil {
		t.Fatalf("Inventory() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func TestInventoryMarksKeyAsDeletableWhenNoDeclaredReferencesExist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(sshDir, "id_ed25519_free")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		outputs: map[string][]byte{
			"ssh-keygen -lf " + privateKey: []byte("256 SHA256:freefingerprint " + privateKey + " (ED25519)\n"),
		},
		errors: make(map[string]error),
	}

	items, err := Inventory(context.Background(), runner, "")
	if err != nil {
		t.Fatalf("Inventory() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if !items[0].CanDelete {
		t.Fatalf("CanDelete = false, want true when no declared references exist")
	}
	if items[0].PublicKeyPath != "" {
		t.Fatalf("public key path = %q, want empty when .pub file is missing", items[0].PublicKeyPath)
	}
}
