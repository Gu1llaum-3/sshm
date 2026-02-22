package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type infoResponseForTest struct {
	Schema   string             `json:"schema"`
	OK       bool               `json:"ok"`
	Hostname string             `json:"hostname"`
	Result   *infoResultForTest `json:"result"`
	Error    *infoErrorForTest  `json:"error"`
}

type infoResultForTest struct {
	CanonicalName string             `json:"canonical_name"`
	Target        infoTargetForTest  `json:"target"`
	IdentityFile  *string            `json:"identity_file"`
	ProxyJump     *string            `json:"proxy_jump"`
	ProxyCommand  *string            `json:"proxy_command"`
	Options       *string            `json:"options"`
	Tags          []string           `json:"tags"`
	RemoteCommand *string            `json:"remote_command"`
	RequestTTY    *string            `json:"request_tty"`
	Source        *infoSourceForTest `json:"source"`
}

type infoTargetForTest struct {
	Host     string  `json:"host"`
	Hostname *string `json:"hostname"`
	User     *string `json:"user"`
	Port     *int    `json:"port"`
}

type infoSourceForTest struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

type infoErrorForTest struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details"`
}

func TestInfoCommandConfig(t *testing.T) {
	if infoCmd.Use != "info <hostname>" {
		t.Fatalf("infoCmd.Use=%q", infoCmd.Use)
	}

	err := infoCmd.Args(infoCmd, []string{})
	if err == nil {
		t.Fatalf("expected args error for no args")
	}

	err = infoCmd.Args(infoCmd, []string{"one", "two"})
	if err == nil {
		t.Fatalf("expected args error for too many args")
	}

	err = infoCmd.Args(infoCmd, []string{"host"})
	if err != nil {
		t.Fatalf("expected no args error, got %v", err)
	}
}

func TestInfoCommandRegistration(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "info" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("info command not registered")
	}
}

func TestRunInfoSuccessJSON(t *testing.T) {
	tempDir := t.TempDir()
	cfg := filepath.Join(tempDir, "config")

	cfgContent := `# Tags: prod, web
Host prod-web
    HostName 10.0.0.10
    User deploy
    Port 2222
    IdentityFile ~/.ssh/id_prod
    ProxyJump bastion
    ServerAliveInterval 60
`

	if err := os.WriteFile(cfg, []byte(cfgContent), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	buf := new(bytes.Buffer)
	exitCode := runInfo(buf, "prod-web", cfg, false)
	if exitCode != 0 {
		t.Fatalf("exitCode=%d", exitCode)
	}

	out := buf.String()
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected output")
	}

	var resp infoResponseForTest
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("output not JSON: %v\noutput=%q", err, out)
	}

	if resp.Schema != "sshm.info.v1" {
		t.Fatalf("schema=%q", resp.Schema)
	}
	if !resp.OK {
		t.Fatalf("ok=false")
	}
	if resp.Result == nil {
		t.Fatalf("result is nil")
	}
	if resp.Error != nil {
		t.Fatalf("error is non-nil")
	}

	if resp.Result.CanonicalName != "prod-web" {
		t.Fatalf("canonical_name=%q", resp.Result.CanonicalName)
	}
	if resp.Result.Target.Host != "prod-web" {
		t.Fatalf("target.host=%q", resp.Result.Target.Host)
	}
	if resp.Result.Target.Hostname == nil || *resp.Result.Target.Hostname != "10.0.0.10" {
		t.Fatalf("target.hostname=%v", resp.Result.Target.Hostname)
	}
	if resp.Result.Target.User == nil || *resp.Result.Target.User != "deploy" {
		t.Fatalf("target.user=%v", resp.Result.Target.User)
	}
	if resp.Result.Target.Port == nil || *resp.Result.Target.Port != 2222 {
		t.Fatalf("target.port=%v", resp.Result.Target.Port)
	}
	if resp.Result.Source == nil || resp.Result.Source.File == "" || resp.Result.Source.Line == 0 {
		t.Fatalf("source missing: %#v", resp.Result.Source)
	}
	if resp.Result.IdentityFile == nil || *resp.Result.IdentityFile != "~/.ssh/id_prod" {
		t.Fatalf("identity_file=%v", resp.Result.IdentityFile)
	}
	if resp.Result.ProxyJump == nil || *resp.Result.ProxyJump != "bastion" {
		t.Fatalf("proxy_jump=%v", resp.Result.ProxyJump)
	}
}

func TestRunInfoNotFoundJSON(t *testing.T) {
	tempDir := t.TempDir()
	cfg := filepath.Join(tempDir, "config")
	cfgContent := `Host known
    HostName example.com
`
	if err := os.WriteFile(cfg, []byte(cfgContent), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	buf := new(bytes.Buffer)
	exitCode := runInfo(buf, "missing", cfg, false)
	if exitCode != 2 {
		t.Fatalf("exitCode=%d", exitCode)
	}

	var resp infoResponseForTest
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v", err)
	}
	if resp.OK {
		t.Fatalf("ok=true")
	}
	if resp.Error == nil {
		t.Fatalf("error is nil")
	}
	if resp.Error.Code != "NOT_FOUND" {
		t.Fatalf("error.code=%q", resp.Error.Code)
	}
}

func TestRunInfoPrettyJSON(t *testing.T) {
	tempDir := t.TempDir()
	cfg := filepath.Join(tempDir, "config")
	cfgContent := `Host known
    HostName 127.0.0.1
`
	if err := os.WriteFile(cfg, []byte(cfgContent), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	buf := new(bytes.Buffer)
	exitCode := runInfo(buf, "known", cfg, true)
	if exitCode != 0 {
		t.Fatalf("exitCode=%d", exitCode)
	}

	out := buf.String()
	if !strings.Contains(out, "\n") {
		t.Fatalf("expected pretty output")
	}

	var resp infoResponseForTest
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v", err)
	}
	if !resp.OK {
		t.Fatalf("ok=false")
	}
}

func TestInfoValidArgsFunction(t *testing.T) {
	if infoCmd.ValidArgsFunction == nil {
		t.Fatalf("expected ValidArgsFunction to be set on infoCmd")
	}
}

func TestInfoValidArgsFunctionWithSSHConfig(t *testing.T) {
	tmpDir := t.TempDir()
	testConfigFile := filepath.Join(tmpDir, "config")

	sshConfig := `Host prod-server
    HostName 192.168.1.1
    User admin

Host dev-server
    HostName 192.168.1.2
    User developer

Host staging-db
    HostName 192.168.1.3
    User dbadmin
`
	if err := os.WriteFile(testConfigFile, []byte(sshConfig), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()
	configFile = testConfigFile

	tests := []struct {
		name       string
		toComplete string
		args       []string
		wantCount  int
		wantHosts  []string
	}{
		{
			name:       "empty prefix returns all hosts",
			toComplete: "",
			args:       []string{},
			wantCount:  3,
			wantHosts:  []string{"prod-server", "dev-server", "staging-db"},
		},
		{
			name:       "prefix filters hosts",
			toComplete: "prod",
			args:       []string{},
			wantCount:  1,
			wantHosts:  []string{"prod-server"},
		},
		{
			name:       "prefix case insensitive",
			toComplete: "DEV",
			args:       []string{},
			wantCount:  1,
			wantHosts:  []string{"dev-server"},
		},
		{
			name:       "no match returns empty",
			toComplete: "nonexistent",
			args:       []string{},
			wantCount:  0,
			wantHosts:  []string{},
		},
		{
			name:       "already has host arg returns nothing",
			toComplete: "",
			args:       []string{"existing-host"},
			wantCount:  0,
			wantHosts:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, directive := infoCmd.ValidArgsFunction(infoCmd, tt.args, tt.toComplete)

			if len(completions) != tt.wantCount {
				t.Fatalf("Expected %d completions, got %d: %v", tt.wantCount, len(completions), completions)
			}

			if directive != cobra.ShellCompDirectiveNoFileComp {
				t.Fatalf("Expected ShellCompDirectiveNoFileComp, got %v", directive)
			}

			for _, wantHost := range tt.wantHosts {
				found := false
				for _, comp := range completions {
					if comp == wantHost {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("Expected completion %q not found in %v", wantHost, completions)
				}
			}
		})
	}
}
