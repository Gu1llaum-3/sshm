package cmd

import "testing"

func TestKeyCommandRegistration(t *testing.T) {
	found := false
	for _, command := range RootCmd.Commands() {
		if command.Name() == "key" {
			found = true
			if !command.HasSubCommands() {
				t.Fatal("key command should register subcommands")
			}
		}
	}
	if !found {
		t.Fatal("key command not registered")
	}
}
