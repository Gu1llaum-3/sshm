package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/history"
	"github.com/Gu1llaum-3/sshm/internal/ui"
	"github.com/Gu1llaum-3/sshm/internal/version"

	"github.com/spf13/cobra"
)

// AppVersion will be set at build time via -ldflags
var AppVersion = "dev"

// configFile holds the path to the SSH config file
var configFile string

// searchMode enables the focus on search mode at startup
var searchMode bool

// RootCmd is the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "sshm [host]",
	Short: "SSH Manager - A modern SSH connection manager",
	Long: `SSHM is a modern SSH manager for your terminal.

Main usage:
  Running 'sshm' (without arguments) opens the interactive TUI window to browse, search, and connect to your SSH hosts graphically.
  Running 'sshm <host>' connects directly to the specified host and records the connection in your history.

You can also use sshm in CLI mode for other operations like adding, editing, or searching hosts.

Hosts are read from your ~/.ssh/config file by default.`,
	Version:       AppVersion,
	Args:          cobra.ArbitraryArgs,
	SilenceUsage:  true,
	SilenceErrors: true, // We'll handle errors ourselves
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no arguments provided, run interactive mode
		if len(args) == 0 {
			runInteractiveMode()
			return nil
		}

		// If a host name is provided, connect directly
		hostName := args[0]
		connectToHost(hostName)
		return nil
	},
}

func runInteractiveMode() {
	// Parse SSH configurations
	var hosts []config.SSHHost
	var err error

	if configFile != "" {
		hosts, err = config.ParseSSHConfigFile(configFile)
	} else {
		hosts, err = config.ParseSSHConfig()
	}

	if err != nil {
		log.Fatalf("Error reading SSH config file: %v", err)
	}

	if len(hosts) == 0 {
		fmt.Println("No SSH hosts found in your ~/.ssh/config file.")
		fmt.Print("Would you like to add a new host now? [y/N]: ")
		var response string
		_, err := fmt.Scanln(&response)
		if err == nil && (response == "y" || response == "Y") {
			err := ui.RunAddForm("", configFile)
			if err != nil {
				fmt.Printf("Error adding host: %v\n", err)
			}
			// After adding, try to reload hosts and continue if any exist
			if configFile != "" {
				hosts, err = config.ParseSSHConfigFile(configFile)
			} else {
				hosts, err = config.ParseSSHConfig()
			}
			if err != nil || len(hosts) == 0 {
				fmt.Println("No hosts available, exiting.")
				os.Exit(1)
			}
		} else {
			fmt.Println("No hosts available, exiting.")
			os.Exit(1)
		}
	}

	// Run the interactive TUI
	if err := ui.RunInteractiveMode(hosts, configFile, searchMode, AppVersion); err != nil {
		log.Fatalf("Error running interactive mode: %v", err)
	}
}

func connectToHost(hostName string) {
	// Quick check if host exists without full parsing (optimized for connection)
	var hostFound bool
	var err error

	if configFile != "" {
		hostFound, err = config.QuickHostExistsInFile(hostName, configFile)
	} else {
		hostFound, err = config.QuickHostExists(hostName)
	}

	if err != nil {
		log.Fatalf("Error checking SSH config: %v", err)
	}

	if !hostFound {
		fmt.Printf("Error: Host '%s' not found in SSH configuration.\n", hostName)
		fmt.Println("Use 'sshm' to see available hosts.")
		os.Exit(1)
	}

	// Record the connection in history
	historyManager, err := history.NewHistoryManager()
	if err != nil {
		// Log the error but don't prevent the connection
		fmt.Printf("Warning: Could not initialize connection history: %v\n", err)
	} else {
		err = historyManager.RecordConnection(hostName)
		if err != nil {
			// Log the error but don't prevent the connection
			fmt.Printf("Warning: Could not record connection history: %v\n", err)
		}
	}

	// Build and execute the SSH command
	fmt.Printf("Connecting to %s...\n", hostName)

	var sshCmd *exec.Cmd
	var args []string

	if configFile != "" {
		args = append(args, "-F", configFile)
	}
	args = append(args, hostName)

	// Note: We don't add RemoteCommand here because if it's configured in SSH config,
	// SSH will handle it automatically. Adding it as a command line argument would conflict.

	sshCmd = exec.Command("ssh", args...)

	// Set up the command to use the same stdin, stdout, and stderr as the parent process
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	// Execute the SSH command
	err = sshCmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// SSH command failed, exit with the same code
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		fmt.Printf("Error executing SSH command: %v\n", err)
		os.Exit(1)
	}
}

// getVersionWithUpdateCheck returns a custom version string with update check
func getVersionWithUpdateCheck() string {
	versionText := fmt.Sprintf("sshm version %s", AppVersion)

	// Check for updates
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	updateInfo, err := version.CheckForUpdates(ctx, AppVersion)
	if err != nil {
		// Return just version if check fails
		return versionText + "\n"
	}

	if updateInfo != nil && updateInfo.Available {
		versionText += fmt.Sprintf("\n🚀 Update available: %s → %s (%s)",
			updateInfo.CurrentVer,
			updateInfo.LatestVer,
			updateInfo.ReleaseURL)
	}

	return versionText + "\n"
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	// Custom error handling for unknown commands that might be host names
	if err := RootCmd.Execute(); err != nil {
		// Check if this is an "unknown command" error and the argument might be a host name
		errStr := err.Error()
		if strings.Contains(errStr, "unknown command") {
			// Extract the command name from the error
			parts := strings.Split(errStr, "\"")
			if len(parts) >= 2 {
				potentialHost := parts[1]
				// Try to connect to this as a host
				connectToHost(potentialHost)
				return
			}
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Add the config file flag
	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "SSH config file to use (default: ~/.ssh/config)")
	RootCmd.PersistentFlags().BoolVarP(&searchMode, "search", "s", false, "Focus on search input at startup")

	// Set custom version template with update check
	RootCmd.SetVersionTemplate(getVersionWithUpdateCheck())
}
