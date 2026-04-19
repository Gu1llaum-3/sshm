package key

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestBuildAddArgs(t *testing.T) {
	privateKey := filepath.Join(t.TempDir(), "id_ed25519_demo")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	path, args, err := BuildAddArgs(privateKey)
	if err != nil {
		t.Fatalf("BuildAddArgs() error = %v", err)
	}
	if path != privateKey {
		t.Fatalf("path = %q", path)
	}
	if !slices.Equal(args, []string{privateKey}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildAddArgsExpandsWindowsStyleHomePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	privateKey := filepath.Join(home, ".ssh", "id_ed25519_demo")
	if err := os.MkdirAll(filepath.Dir(privateKey), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	path, args, err := BuildAddArgs(`~\.ssh\id_ed25519_demo`)
	if err != nil {
		t.Fatalf("BuildAddArgs() error = %v", err)
	}
	if path != privateKey {
		t.Fatalf("path = %q, want %q", path, privateKey)
	}
	if !slices.Equal(args, []string{privateKey}) {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildAddArgsExpandsUserProfilePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)

	privateKey := filepath.Join(home, ".ssh", "id_ed25519_demo")
	if err := os.MkdirAll(filepath.Dir(privateKey), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	path, _, err := BuildAddArgs(`%USERPROFILE%\.ssh\id_ed25519_demo`)
	if err != nil {
		t.Fatalf("BuildAddArgs() error = %v", err)
	}
	if path != privateKey {
		t.Fatalf("path = %q, want %q", path, privateKey)
	}
}

func TestBuildAddArgsRejectsBlankPath(t *testing.T) {
	_, _, err := BuildAddArgs("   ")
	if err == nil {
		t.Fatal("BuildAddArgs() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("error = %q, want path required", err)
	}
}

func TestAddDryRun(t *testing.T) {
	privateKey := filepath.Join(t.TempDir(), "id_ed25519_demo")
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}

	runner := &fakeRunner{
		outputs: make(map[string][]byte),
		errors:  make(map[string]error),
	}

	path, args, err := Add(context.Background(), runner, AddOptions{
		Path:   privateKey,
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if path != privateKey {
		t.Fatalf("path = %q", path)
	}
	if !slices.Equal(args, []string{privateKey}) {
		t.Fatalf("args = %#v", args)
	}
	if len(runner.runCalls) != 0 {
		t.Fatalf("runCalls = %#v, want none", runner.runCalls)
	}
}

func TestSupportsAppleUseKeychain(t *testing.T) {
	runner := &fakeRunner{
		outputs: map[string][]byte{
			"ssh-add --apple-use-keychain -l": []byte("Could not open a connection to your authentication agent.\n"),
		},
		errors: map[string]error{
			"ssh-add --apple-use-keychain -l": fmt.Errorf("agent unavailable"),
		},
	}

	supported, err := SupportsAppleUseKeychain(context.Background(), runner)
	if runtime.GOOS == "darwin" {
		if err != nil {
			t.Fatalf("SupportsAppleUseKeychain() error = %v", err)
		}
		if !supported {
			t.Fatal("expected support on darwin-compatible probe output")
		}
		return
	}

	if err != nil {
		t.Fatalf("SupportsAppleUseKeychain() error = %v", err)
	}
	if supported {
		t.Fatal("expected no support outside darwin")
	}
}

func TestAppleUseKeychainProbeRejectsUsageOutput(t *testing.T) {
	supported, err := appleUseKeychainProbeSupported([]byte("usage: ssh-add [-cDdKkLlqvXx] [-E fingerprint_hash]\n"), fmt.Errorf("usage"))
	if err != nil {
		t.Fatalf("appleUseKeychainProbeSupported() error = %v", err)
	}
	if supported {
		t.Fatal("supported = true, want false for generic usage output")
	}
}

func TestAppleUseKeychainProbeAllowsAgentUnavailableOutput(t *testing.T) {
	supported, err := appleUseKeychainProbeSupported([]byte("Could not open a connection to your authentication agent.\n"), fmt.Errorf("agent unavailable"))
	if err != nil {
		t.Fatalf("appleUseKeychainProbeSupported() error = %v", err)
	}
	if !supported {
		t.Fatal("supported = false, want true for Apple-compatible probe output")
	}
}
