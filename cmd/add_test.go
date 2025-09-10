package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddCommand(t *testing.T) {
	// Test that the add command is properly configured
	if addCmd.Use != "add [hostname]" {
		t.Errorf("Expected Use 'add [hostname]', got '%s'", addCmd.Use)
	}

	if addCmd.Short != "Add a new SSH host configuration" {
		t.Errorf("Expected Short description, got '%s'", addCmd.Short)
	}

	// Test that it accepts maximum 1 argument
	err := addCmd.Args(addCmd, []string{"host1", "host2"})
	if err == nil {
		t.Error("Expected error for too many arguments")
	}

	// Test that it accepts 0 or 1 argument
	err = addCmd.Args(addCmd, []string{})
	if err != nil {
		t.Errorf("Expected no error for 0 arguments, got %v", err)
	}

	err = addCmd.Args(addCmd, []string{"hostname"})
	if err != nil {
		t.Errorf("Expected no error for 1 argument, got %v", err)
	}
}

func TestAddCommandRegistration(t *testing.T) {
	// Check that add command is registered with root command
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "add" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Add command not found in root command")
	}
}

func TestAddCommandHelp(t *testing.T) {
	// Test help output
	cmd := &cobra.Command{}
	cmd.AddCommand(addCmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"add", "--help"})

	// This should not return an error for help
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Expected no error for help command, got %v", err)
	}

	output := buf.String()
	if !contains(output, "Add a new SSH host configuration") {
		t.Error("Help output should contain command description")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
