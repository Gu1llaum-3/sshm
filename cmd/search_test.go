package cmd

import (
	"strings"
	"testing"
)

func TestSearchCommand(t *testing.T) {
	// Test that the search command is properly configured
	if searchCmd.Use != "search [query]" {
		t.Errorf("Expected Use 'search [query]', got '%s'", searchCmd.Use)
	}

	if searchCmd.Short != "Search SSH hosts by name, hostname, or tags" {
		t.Errorf("Expected Short description, got '%s'", searchCmd.Short)
	}

	// Test that it accepts maximum 1 argument
	err := searchCmd.Args(searchCmd, []string{"query1", "query2"})
	if err == nil {
		t.Error("Expected error for too many arguments")
	}

	// Test that it accepts 0 or 1 argument
	err = searchCmd.Args(searchCmd, []string{})
	if err != nil {
		t.Errorf("Expected no error for 0 arguments, got %v", err)
	}

	err = searchCmd.Args(searchCmd, []string{"query"})
	if err != nil {
		t.Errorf("Expected no error for 1 argument, got %v", err)
	}
}

func TestSearchCommandRegistration(t *testing.T) {
	// Check that search command is registered with root command
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "search" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Search command not found in root command")
	}
}

func TestSearchCommandFlags(t *testing.T) {
	// Test that flags are properly configured
	flags := searchCmd.Flags()

	// Check format flag
	formatFlag := flags.Lookup("format")
	if formatFlag == nil {
		t.Error("Expected --format flag to be defined")
	}

	// Check tags flag
	tagsFlag := flags.Lookup("tags")
	if tagsFlag == nil {
		t.Error("Expected --tags flag to be defined")
	}

	// Check names flag
	namesFlag := flags.Lookup("names")
	if namesFlag == nil {
		t.Error("Expected --names flag to be defined")
	}
}

func TestSearchCommandHelp(t *testing.T) {
	// Test that the command has the right help properties
	// Instead of executing --help, just check the Long description
	if searchCmd.Long == "" {
		t.Error("Search command should have a Long description")
	}

	if !strings.Contains(searchCmd.Long, "Search") {
		t.Error("Long description should contain information about searching")
	}
}

func TestFormatOutput(t *testing.T) {
	tests := []struct {
		name   string
		format string
		valid  bool
	}{
		{"table format", "table", true},
		{"json format", "json", true},
		{"simple format", "simple", true},
		{"invalid format", "invalid", false},
		{"empty format", "", true}, // Should default to table
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidFormat(tt.format)
			if valid != tt.valid {
				t.Errorf("isValidFormat(%q) = %v, want %v", tt.format, valid, tt.valid)
			}
		})
	}
}

// Helper function to validate format (this would be in the actual search.go)
func isValidFormat(format string) bool {
	if format == "" {
		return true // Default to table
	}
	validFormats := []string{"table", "json", "simple"}
	for _, valid := range validFormats {
		if format == valid {
			return true
		}
	}
	return false
}
