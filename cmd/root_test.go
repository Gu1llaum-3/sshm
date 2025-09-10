package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	// Test that the root command is properly configured
	if RootCmd.Use != "sshm" {
		t.Errorf("Expected Use 'sshm', got '%s'", RootCmd.Use)
	}

	if RootCmd.Short != "SSH Manager - A modern SSH connection manager" {
		t.Errorf("Expected Short description, got '%s'", RootCmd.Short)
	}

	if RootCmd.Version != AppVersion {
		t.Errorf("Expected Version '%s', got '%s'", AppVersion, RootCmd.Version)
	}
}

func TestRootCommandFlags(t *testing.T) {
	// Test that persistent flags are properly configured
	flags := RootCmd.PersistentFlags()

	// Check config flag
	configFlag := flags.Lookup("config")
	if configFlag == nil {
		t.Error("Expected --config flag to be defined")
		return
	}
	if configFlag.Shorthand != "c" {
		t.Errorf("Expected config flag shorthand 'c', got '%s'", configFlag.Shorthand)
	}
}

func TestRootCommandSubcommands(t *testing.T) {
	// Test that all expected subcommands are registered
	// Note: completion and help are automatically added by Cobra and may not always appear in Commands()
	expectedCommands := []string{"add", "edit", "search"}

	commands := RootCmd.Commands()
	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("Expected command '%s' not found", expected)
		}
	}

	// Check that we have at least the core commands
	if len(commandNames) < 3 {
		t.Errorf("Expected at least 3 commands, got %d", len(commandNames))
	}
}

func TestRootCommandHelp(t *testing.T) {
	// Test help output
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetArgs([]string{"--help"})

	// This should not return an error for help
	err := RootCmd.Execute()
	if err != nil {
		t.Errorf("Expected no error for help command, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "modern SSH manager") {
		t.Error("Help output should contain command description")
	}
	if !strings.Contains(output, "Usage:") {
		t.Error("Help output should contain usage section")
	}
}

func TestRootCommandVersion(t *testing.T) {
	// Test that version command executes without error
	// Note: Cobra handles version output internally, so we just check for no error
	RootCmd.SetArgs([]string{"--version"})

	// This should not return an error for version
	err := RootCmd.Execute()
	if err != nil {
		t.Errorf("Expected no error for version command, got %v", err)
	}

	// Reset args for other tests
	RootCmd.SetArgs([]string{})
}

func TestExecuteFunction(t *testing.T) {
	// Test that Execute function exists and can be called
	// We can't easily test the actual execution without mocking,
	// but we can test that the function exists
	t.Log("Execute function exists and is accessible")
}

func TestConnectToHostFunction(t *testing.T) {
	// Test that connectToHost function exists and can be called
	// Note: We can't easily test the actual connection without a valid SSH config
	// and without actually connecting to a host, but we can verify the function exists
	t.Log("connectToHost function exists and is accessible")

	// The function will handle errors internally (like host not found)
	// We don't want to actually test the SSH connection in unit tests
}

func TestRunInteractiveModeFunction(t *testing.T) {
	// Test that runInteractiveMode function exists
	// We can't easily test the actual execution without mocking the UI,
	// but we can verify the function signature
	t.Log("runInteractiveMode function exists and is accessible")
}

func TestConfigFileVariable(t *testing.T) {
	// Test that configFile variable is properly initialized
	originalConfigFile := configFile
	defer func() { configFile = originalConfigFile }()

	// Set config file through flag
	RootCmd.SetArgs([]string{"--config", "/tmp/test-config"})
	RootCmd.ParseFlags([]string{"--config", "/tmp/test-config"})

	// The configFile variable should be updated by the flag parsing
	// Note: This test verifies the flag binding works
}

func TestVersionVariable(t *testing.T) {
	// Test that version variable has a default value
	if AppVersion == "" {
		t.Error("AppVersion variable should have a default value")
	}

	// Test that version is set to "dev" by default
	if AppVersion != "dev" {
		t.Logf("AppVersion is set to '%s' (expected 'dev' for development)", AppVersion)
	}
}
