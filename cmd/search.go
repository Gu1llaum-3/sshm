package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"

	"github.com/spf13/cobra"
)

var (
	// outputFormat defines the output format (table, json, simple)
	outputFormat string
	// tagsOnly limits search to tags only
	tagsOnly bool
	// namesOnly limits search to host names only
	namesOnly bool
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search SSH hosts by name, hostname, or tags",
	Long: `Search through your SSH hosts configuration by name, hostname, or tags.
The search is case-insensitive and will match partial strings.

Examples:
  sshm search web          # Search for hosts containing "web"
  sshm search --tags dev   # Search only in tags for "dev"
  sshm search --names prod # Search only in host names for "prod"
  sshm search --format json server # Output results in JSON format`,
	Args: cobra.MaximumNArgs(1),
	Run:  runSearch,
}

func runSearch(cmd *cobra.Command, args []string) {
	// Parse SSH configurations
	var hosts []config.SSHHost
	var err error

	if configFile != "" {
		hosts, err = config.ParseSSHConfigFile(configFile)
	} else {
		hosts, err = config.ParseSSHConfig()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading SSH config file: %v\n", err)
		os.Exit(1)
	}

	if len(hosts) == 0 {
		fmt.Println("No SSH hosts found in your configuration file.")
		os.Exit(1)
	}

	// Get search query
	var query string
	if len(args) > 0 {
		query = args[0]
	}

	// Filter hosts based on search criteria
	filteredHosts := filterHosts(hosts, query, tagsOnly, namesOnly)

	// Display results
	if len(filteredHosts) == 0 {
		if query == "" {
			fmt.Println("No hosts found.")
		} else {
			fmt.Printf("No hosts found matching '%s'.\n", query)
		}
		return
	}

	// Output results in specified format
	switch outputFormat {
	case "json":
		outputJSON(filteredHosts)
	case "simple":
		outputSimple(filteredHosts)
	default:
		outputTable(filteredHosts)
	}
}

// filterHosts filters hosts according to the search query and options
func filterHosts(hosts []config.SSHHost, query string, tagsOnly, namesOnly bool) []config.SSHHost {
	var filtered []config.SSHHost

	if query == "" {
		return hosts
	}

	query = strings.ToLower(query)

	for _, host := range hosts {
		matched := false

		// Search in names if not tags-only
		if !tagsOnly {
			// Check the host name
			if strings.Contains(strings.ToLower(host.Name), query) {
				matched = true
			}

			// Check the hostname if not names-only
			if !namesOnly && !matched && strings.Contains(strings.ToLower(host.Hostname), query) {
				matched = true
			}
		}

		// Search in tags if not names-only
		if !namesOnly && !matched {
			for _, tag := range host.Tags {
				if strings.Contains(strings.ToLower(tag), query) {
					matched = true
					break
				}
			}
		}

		if matched {
			filtered = append(filtered, host)
		}
	}

	return filtered
}

// outputTable displays results in a formatted table
func outputTable(hosts []config.SSHHost) {
	if len(hosts) == 0 {
		return
	}

	// Calculate column widths
	nameWidth := 4 // "Name"
	hostWidth := 8 // "Hostname"
	userWidth := 4 // "User"
	tagsWidth := 4 // "Tags"

	for _, host := range hosts {
		if len(host.Name) > nameWidth {
			nameWidth = len(host.Name)
		}
		if len(host.Hostname) > hostWidth {
			hostWidth = len(host.Hostname)
		}
		if len(host.User) > userWidth {
			userWidth = len(host.User)
		}
		tagsStr := strings.Join(host.Tags, ", ")
		if len(tagsStr) > tagsWidth {
			tagsWidth = len(tagsStr)
		}
	}

	// Add padding
	nameWidth += 2
	hostWidth += 2
	userWidth += 2
	tagsWidth += 2

	// Print header
	fmt.Printf("%-*s %-*s %-*s %-*s\n", nameWidth, "Name", hostWidth, "Hostname", userWidth, "User", tagsWidth, "Tags")
	fmt.Printf("%s %s %s %s\n",
		strings.Repeat("-", nameWidth),
		strings.Repeat("-", hostWidth),
		strings.Repeat("-", userWidth),
		strings.Repeat("-", tagsWidth))

	// Print hosts
	for _, host := range hosts {
		user := host.User
		if user == "" {
			user = "-"
		}
		tags := strings.Join(host.Tags, ", ")
		if tags == "" {
			tags = "-"
		}
		fmt.Printf("%-*s %-*s %-*s %-*s\n", nameWidth, host.Name, hostWidth, host.Hostname, userWidth, user, tagsWidth, tags)
	}

	fmt.Printf("\nFound %d host(s)\n", len(hosts))
}

// outputSimple displays results in simple format (one per line)
func outputSimple(hosts []config.SSHHost) {
	for _, host := range hosts {
		fmt.Println(host.Name)
	}
}

// outputJSON displays results in JSON format
func outputJSON(hosts []config.SSHHost) {
	fmt.Println("[")
	for i, host := range hosts {
		fmt.Printf("  {\n")
		fmt.Printf("    \"name\": \"%s\",\n", escapeJSON(host.Name))
		fmt.Printf("    \"hostname\": \"%s\",\n", escapeJSON(host.Hostname))
		fmt.Printf("    \"user\": \"%s\",\n", escapeJSON(host.User))
		fmt.Printf("    \"port\": \"%s\",\n", escapeJSON(host.Port))
		fmt.Printf("    \"identity\": \"%s\",\n", escapeJSON(host.Identity))
		fmt.Printf("    \"proxy_jump\": \"%s\",\n", escapeJSON(host.ProxyJump))
		fmt.Printf("    \"proxy_command\": \"%s\",\n", escapeJSON(host.ProxyCommand))
		fmt.Printf("    \"options\": \"%s\",\n", escapeJSON(host.Options))
		fmt.Printf("    \"tags\": [")
		for j, tag := range host.Tags {
			fmt.Printf("\"%s\"", escapeJSON(tag))
			if j < len(host.Tags)-1 {
				fmt.Printf(", ")
			}
		}
		fmt.Printf("]\n")
		if i < len(hosts)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}

// escapeJSON escapes special characters for JSON output
func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func init() {
	// Add search command to root
	RootCmd.AddCommand(searchCmd)

	// Add flags
	searchCmd.Flags().StringVarP(&outputFormat, "format", "f", "table", "Output format (table, json, simple)")
	searchCmd.Flags().BoolVar(&tagsOnly, "tags", false, "Search only in tags")
	searchCmd.Flags().BoolVar(&namesOnly, "names", false, "Search only in host names")
}
