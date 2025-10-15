package history

import (
	"os/user"
	"strings"
)

// ParseSSHArgs parses SSH command line arguments and extracts connection details
// It handles formats like: user@host, -p port, -i identity, etc.
func ParseSSHArgs(args []string) (*ManualConnection, bool) {
	if len(args) == 0 {
		return nil, false
	}

	conn := &ManualConnection{
		Port: "22", // Default SSH port
	}

	// Get current user as default
	currentUser, err := user.Current()
	if err == nil {
		conn.User = currentUser.Username
	}

	// Parse arguments
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Handle -p <port> or -p<port>
		if arg == "-p" {
			if i+1 < len(args) {
				conn.Port = args[i+1]
				i++
			}
		} else if strings.HasPrefix(arg, "-p") {
			conn.Port = arg[2:]
		} else if arg == "-i" {
			// Handle -i <identity>
			if i+1 < len(args) {
				conn.Identity = args[i+1]
				i++
			}
		} else if arg == "-F" || arg == "-c" || arg == "--config" {
			// Skip config file arguments - these are handled separately
			if i+1 < len(args) {
				i++
			}
			return nil, false
		} else if strings.HasPrefix(arg, "-") {
			// Skip other SSH options like -v, -A, -X, etc.
			// If they have a value, skip it too
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
			continue
		} else if strings.Contains(arg, "@") {
			// Parse user@hostname
			parts := strings.SplitN(arg, "@", 2)
			if len(parts) == 2 {
				conn.User = parts[0]
				conn.Hostname = parts[1]
			}
		} else if conn.Hostname == "" {
			// If no @, treat as just hostname
			conn.Hostname = arg
		}
	}

	// If we got a hostname, this is a valid manual connection
	if conn.Hostname != "" {
		return conn, true
	}

	return nil, false
}

// IsManualSSHCommand checks if the arguments represent a manual SSH connection
// (not a configured host name)
func IsManualSSHCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}

	// Check for SSH flags that indicate manual connection
	for _, arg := range args {
		if arg == "-p" || strings.HasPrefix(arg, "-p") {
			return true
		}
		if strings.Contains(arg, "@") {
			return true
		}
	}

	return false
}
