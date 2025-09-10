package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/ui"
	"github.com/Gu1llaum-3/sshm/internal/version"

	"github.com/spf13/cobra"
)

// AppVersion will be set at build time via -ldflags
var AppVersion = "dev"

// configFile holds the path to the SSH config file
var configFile string

// RootCmd is the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "sshm",
	Short: "SSH Manager - A modern SSH connection manager",
	Long: `SSHM is a modern SSH manager for your terminal.

Main usage:
  Running 'sshm' (without arguments) opens the interactive TUI window to browse, search, and connect to your SSH hosts graphically.

You can also use sshm in CLI mode for direct operations.

Hosts are read from your ~/.ssh/config file by default.`,
	Version: AppVersion,
	Run: func(cmd *cobra.Command, args []string) {
		// If no arguments provided, run interactive mode
		if len(args) == 0 {
			runInteractiveMode()
			return
		}

		// If a host name is provided, connect directly
		hostName := args[0]
		connectToHost(hostName)
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
	if err := ui.RunInteractiveMode(hosts, configFile, AppVersion); err != nil {
		log.Fatalf("Error running interactive mode: %v", err)
	}
}

func connectToHost(hostName string) {
	// Parse SSH configurations to verify host exists
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

	// Check if host exists
	var hostFound bool
	for _, host := range hosts {
		if host.Name == hostName {
			hostFound = true
			break
		}
	}

	if !hostFound {
		fmt.Printf("Error: Host '%s' not found in SSH configuration.\n", hostName)
		fmt.Println("Use 'sshm' to see available hosts.")
		os.Exit(1)
	}

	// Connect to the host
	fmt.Printf("Connecting to %s...\n", hostName)

	// Build the SSH command with the appropriate config file
	var sshCmd []string
	if configFile != "" {
		sshCmd = []string{"ssh", "-F", configFile, hostName}
	} else {
		sshCmd = []string{"ssh", hostName}
	}

	// Note: In a real implementation, you'd use exec.Command here
	// For now, just print the command that would be executed
	fmt.Printf("%s\n", strings.Join(sshCmd, " "))
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
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Add the config file flag
	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "SSH config file to use (default: ~/.ssh/config)")

	// Set custom version template with update check
	RootCmd.SetVersionTemplate(getVersionWithUpdateCheck())
}
