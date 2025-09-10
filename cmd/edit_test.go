package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestEditCommand(t *testing.T) {
	// Test that the edit command is properly configured
	if editCmd.Use != "edit <hostname>" {
		t.Errorf("Expected Use 'edit <hostname>', got '%s'", editCmd.Use)
	}

	if editCmd.Short != "Edit an existing SSH host configuration" {
		t.Errorf("Expected Short description, got '%s'", editCmd.Short)
	}

	// Test that it requires exactly 1 argument
	err := editCmd.Args(editCmd, []string{})
	if err == nil {
		t.Error("Expected error for no arguments")
	}

	err = editCmd.Args(editCmd, []string{"host1", "host2"})
	if err == nil {
		t.Error("Expected error for too many arguments")
	}

	err = editCmd.Args(editCmd, []string{"hostname"})
	if err != nil {
		t.Errorf("Expected no error for 1 argument, got %v", err)
	}
}

func TestEditCommandRegistration(t *testing.T) {
	// Check that edit command is registered with root command
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "edit" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Edit command not found in root command")
	}
}

func TestEditCommandHelp(t *testing.T) {
	// Test help output
	cmd := &cobra.Command{}
	cmd.AddCommand(editCmd)

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"edit", "--help"})

	// This should not return an error for help
	err := cmd.Execute()
	if err != nil {
		t.Errorf("Expected no error for help command, got %v", err)
	}

	output := buf.String()
	if !contains(output, "Edit an existing SSH host configuration") {
		t.Error("Help output should contain command description")
	}
}
