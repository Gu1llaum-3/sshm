package key

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
)

func TestBuildGenerateArgs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	result, args, err := BuildGenerateArgs(GenerateOptions{
		Name:      "id_ed25519_demo",
		Algorithm: "ed25519",
		Comment:   "demo@test",
		Directory: "~/.ssh",
		KDFRounds: 100,
	})
	if err != nil {
		t.Fatalf("BuildGenerateArgs() error = %v", err)
	}

	privatePath := filepath.Join(home, ".ssh", "id_ed25519_demo")
	if result.PrivateKeyPath != privatePath {
		t.Fatalf("private path = %q", result.PrivateKeyPath)
	}
	want := []string{"-t", "ed25519", "-f", privatePath, "-C", "demo@test", "-a", "100"}
	if !slices.Equal(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestBuildGenerateArgsNormalizesAlgorithmCase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, args, err := BuildGenerateArgs(GenerateOptions{
		Name:      "id_rsa_demo",
		Algorithm: "RSA",
		Directory: "~/.ssh",
	})
	if err != nil {
		t.Fatalf("BuildGenerateArgs() error = %v", err)
	}
	if got, want := args[1], "rsa"; got != want {
		t.Fatalf("algorithm arg = %q, want %q", got, want)
	}
}

func TestGenerateDryRunDoesNotExecute(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	runner := &fakeRunner{
		outputs: make(map[string][]byte),
		errors:  make(map[string]error),
	}

	_, err := Generate(context.Background(), runner, GenerateOptions{
		Name:   "id_ed25519_demo",
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(runner.runCalls) != 0 {
		t.Fatalf("runCalls = %#v, want none", runner.runCalls)
	}
}

func TestBuildGenerateArgsRejectsUnsupportedAlgorithm(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, _, err := BuildGenerateArgs(GenerateOptions{
		Name:      "id_demo",
		Algorithm: "dsa",
		Directory: "~/.ssh",
	})
	if err == nil {
		t.Fatal("BuildGenerateArgs() error = nil, want validation error")
	}
}

func TestBuildGenerateArgsRejectsPathLikeName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, _, err := BuildGenerateArgs(GenerateOptions{
		Name:      "nested/id_demo",
		Directory: "~/.ssh",
	})
	if err == nil {
		t.Fatal("BuildGenerateArgs() error = nil, want path-like name validation error")
	}

	_, _, err = BuildGenerateArgs(GenerateOptions{
		Name:      `nested\id_demo`,
		Directory: "~/.ssh",
	})
	if err == nil {
		t.Fatal("BuildGenerateArgs() error = nil, want Windows path-like name validation error")
	}
}
