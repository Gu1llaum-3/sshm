package key

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDeployPlanUsesSSHCopyIDForDirectTarget(t *testing.T) {
	tempDir := t.TempDir()
	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	original := hasSSHCopyID
	hasSSHCopyID = func() bool { return true }
	t.Cleanup(func() {
		hasSSHCopyID = original
	})

	plan, err := BuildDeployPlan(DeployOptions{
		Target:   "203.0.113.10",
		User:     "root",
		Port:     "2222",
		Identity: privateKey,
	})
	if err != nil {
		t.Fatalf("BuildDeployPlan() error = %v", err)
	}
	if plan.Command != "ssh-copy-id" {
		t.Fatalf("command = %q, want ssh-copy-id", plan.Command)
	}
	want := []string{"-i", publicKey, "-p", "2222", "root@203.0.113.10"}
	if got := shellCall(plan.Args); got != shellCall(want) {
		t.Fatalf("args = %#v, want %#v", plan.Args, want)
	}
}

func TestBuildDeployPlanUsesConfigAliasWhenUnique(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	configContent := `Host demo
    HostName 203.0.113.10
    User deploy
    Port 2222
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	original := hasSSHCopyID
	hasSSHCopyID = func() bool { return true }
	t.Cleanup(func() {
		hasSSHCopyID = original
	})

	plan, err := BuildDeployPlan(DeployOptions{
		Target:     "demo",
		Identity:   privateKey,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("BuildDeployPlan() error = %v", err)
	}
	want := []string{"-i", publicKey, "-F", configPath, "demo"}
	if got := shellCall(plan.Args); got != shellCall(want) {
		t.Fatalf("args = %#v, want %#v", plan.Args, want)
	}
}

func TestBuildDeployPlanKeepsConfigWhenDirectOverridesAreSet(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	configContent := `Host demo
    HostName 203.0.113.10
    User deploy
    Port 2222
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	original := hasSSHCopyID
	hasSSHCopyID = func() bool { return true }
	t.Cleanup(func() {
		hasSSHCopyID = original
	})

	plan, err := BuildDeployPlan(DeployOptions{
		Target:     "demo",
		User:       "root",
		Port:       "2200",
		Identity:   privateKey,
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("BuildDeployPlan() error = %v", err)
	}
	want := []string{"-i", publicKey, "-F", configPath, "-p", "2200", "root@demo"}
	if got := shellCall(plan.Args); got != shellCall(want) {
		t.Fatalf("args = %#v, want %#v", plan.Args, want)
	}
}

func TestBuildDeployPlanRejectsAmbiguousConfigAlias(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	configContent := `Host demo
    HostName 203.0.113.10

Include included.conf
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "included.conf"), []byte("Host demo\n    HostName 203.0.113.11\n"), 0600); err != nil {
		t.Fatal(err)
	}

	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := BuildDeployPlan(DeployOptions{
		Target:     "demo",
		Identity:   privateKey,
		ConfigPath: configPath,
	})
	if err == nil {
		t.Fatal("BuildDeployPlan() error = nil, want ambiguity error")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("error = %q, want ambiguity wording", err)
	}
}

func TestBuildDeployPlanFallsBackToManualSSH(t *testing.T) {
	tempDir := t.TempDir()
	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	original := hasSSHCopyID
	hasSSHCopyID = func() bool { return false }
	t.Cleanup(func() {
		hasSSHCopyID = original
	})

	plan, err := BuildDeployPlan(DeployOptions{
		Target:   "203.0.113.10",
		User:     "root",
		Port:     "2200",
		Identity: privateKey,
	})
	if err != nil {
		t.Fatalf("BuildDeployPlan() error = %v", err)
	}
	if plan.Command != "ssh" {
		t.Fatalf("command = %q, want ssh", plan.Command)
	}
	if got, want := plan.Args[0], "-p"; got != want {
		t.Fatalf("args[0] = %q, want %q", got, want)
	}
	if !strings.Contains(shellCall(plan.Args), "grep -qxF 'ssh-ed25519 AAAATEST demo' ~/.ssh/authorized_keys") {
		t.Fatalf("manual ssh command missing idempotent grep:\n%#v", plan.Args)
	}
}

func TestBuildDeployPlanIncludesProxyOptionsForDirectTarget(t *testing.T) {
	tempDir := t.TempDir()
	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	original := hasSSHCopyID
	hasSSHCopyID = func() bool { return true }
	t.Cleanup(func() {
		hasSSHCopyID = original
	})

	plan, err := BuildDeployPlan(DeployOptions{
		Target:       "203.0.113.10",
		User:         "root",
		Port:         "2222",
		ProxyJump:    "bastion",
		ProxyCommand: "ssh -W %h:%p jump-host",
		Identity:     privateKey,
	})
	if err != nil {
		t.Fatalf("BuildDeployPlan() error = %v", err)
	}

	got := shellCall(plan.Args)
	if !strings.Contains(got, "-o ProxyJump=bastion") {
		t.Fatalf("args missing ProxyJump option: %#v", plan.Args)
	}
	if !strings.Contains(got, "-o ProxyCommand=ssh -W %h:%p jump-host") {
		t.Fatalf("args missing ProxyCommand option: %#v", plan.Args)
	}
}

func TestDeployDryRunDoesNotExecute(t *testing.T) {
	tempDir := t.TempDir()
	privateKey := filepath.Join(tempDir, "id_ed25519_demo")
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	original := hasSSHCopyID
	hasSSHCopyID = func() bool { return true }
	t.Cleanup(func() {
		hasSSHCopyID = original
	})

	runner := &fakeRunner{
		outputs: make(map[string][]byte),
		errors:  make(map[string]error),
	}

	plan, err := Deploy(context.Background(), runner, DeployOptions{
		Target:   "203.0.113.10",
		User:     "root",
		Identity: privateKey,
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if plan.Command != "ssh-copy-id" {
		t.Fatalf("command = %q, want ssh-copy-id", plan.Command)
	}
	if len(runner.runCalls) != 0 {
		t.Fatalf("runCalls = %#v, want none", runner.runCalls)
	}
}
