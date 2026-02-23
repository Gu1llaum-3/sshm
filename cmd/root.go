package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/history"
	"github.com/Gu1llaum-3/sshm/internal/ui"

	"github.com/spf13/cobra"
)

// AppVersion will be set at build time via -ldflags
var AppVersion = "dev"

// configFile holds the path to the SSH config file
var configFile string

// forceTTY forces pseudo-TTY allocation for remote commands
var forceTTY bool

// searchMode enables the focus on search mode at startup
var searchMode bool

// noUpdateCheck disables the async update check in the TUI
var noUpdateCheck bool

// RootCmd is the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "sshm [host] [command...]",
	Short: "SSH Manager - A modern SSH connection manager",
	Long: `SSHM is a modern SSH manager for your terminal.

Main usage:
  Running 'sshm' (without arguments) opens the interactive TUI window to browse, search, and connect to your SSH hosts graphically.
  Running 'sshm <host>' connects directly to the specified host and records the connection in your history.
  Running 'sshm <host> <command>' executes the command on the remote host and returns the output.

You can also use sshm in CLI mode for other operations like adding, editing, or searching hosts.

Hosts are read from your ~/.ssh/config file by default.

Examples:
  sshm                           # Open interactive TUI
  sshm prod-server               # Connect to host interactively
  sshm prod-server uptime        # Execute 'uptime' on remote host
  sshm prod-server ls -la /var   # Execute command with arguments
  sshm -t prod-server sudo reboot # Force TTY for interactive commands`,
	Version:       AppVersion,
	Args:          cobra.ArbitraryArgs,
	SilenceUsage:  true,
	SilenceErrors: true, // We'll handle errors ourselves
	// ValidArgsFunction provides shell completion for host names
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Only complete the first positional argument (host name)
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var hosts []config.SSHHost
		var err error

		if configFile != "" {
			hosts, err = config.ParseSSHConfigFile(configFile)
		} else {
			hosts, err = config.ParseSSHConfig()
		}

		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		toCompleteLower := strings.ToLower(toComplete)
		for _, host := range hosts {
			if strings.HasPrefix(strings.ToLower(host.Name), toCompleteLower) {
				completions = append(completions, host.Name)
			}
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			runInteractiveMode()
			return nil
		}

		hostName := args[0]
		var remoteCommand []string
		if len(args) > 1 {
			remoteCommand = args[1:]
		}
		connectToHost(hostName, remoteCommand)
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
	if err := ui.RunInteractiveMode(hosts, configFile, searchMode, AppVersion, noUpdateCheck); err != nil {
		log.Fatalf("Error running interactive mode: %v", err)
	}
}

func connectToHost(hostName string, remoteCommand []string) {
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

	historyManager, err := history.NewHistoryManager()
	if err != nil {
		fmt.Printf("Warning: Could not initialize connection history: %v\n", err)
	} else {
		err = historyManager.RecordConnection(hostName)
		if err != nil {
			fmt.Printf("Warning: Could not record connection history: %v\n", err)
		}
	}

	var args []string

	if configFile != "" {
		args = append(args, "-F", configFile)
	}

	if forceTTY {
		args = append(args, "-t")
	}

	args = append(args, hostName)

	if len(remoteCommand) > 0 {
		args = append(args, remoteCommand...)
	} else {
		fmt.Printf("Connecting to %s...\n", hostName)
	}

	sshCmd := exec.Command("ssh", args...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	err = sshCmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		fmt.Printf("Error executing SSH command: %v\n", err)
		os.Exit(1)
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "unknown command") {
			parts := strings.Split(errStr, "\"")
			if len(parts) >= 2 {
				potentialHost := parts[1]
				connectToHost(potentialHost, nil)
				return
			}
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "SSH config file to use (default: ~/.ssh/config)")
	RootCmd.Flags().BoolVarP(&forceTTY, "tty", "t", false, "Force pseudo-TTY allocation (useful for interactive remote commands)")
	RootCmd.PersistentFlags().BoolVarP(&searchMode, "search", "s", false, "Focus on search input at startup")
	RootCmd.PersistentFlags().BoolVar(&noUpdateCheck, "no-update-check", false, "Disable automatic update check")

	RootCmd.SetVersionTemplate("{{.Name}} version {{.Version}}\n")
}
