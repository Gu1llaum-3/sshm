package validation

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ValidateHostname checks if a hostname is valid
// Accepts regular hostnames, IP addresses, and SSH tokens like %h, %p, %r, %u, %n, %C, %d, %i, %k, %L, %l, %T
func ValidateHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}
	if strings.HasPrefix(hostname, ".") || strings.HasSuffix(hostname, ".") {
		return false
	}
	if strings.Contains(hostname, " ") {
		return false
	}

	// Check if hostname contains SSH tokens (e.g., %h, %p, %r, %u, %n, etc.)
	// SSH tokens are documented in ssh_config(5) man page
	sshTokenRegex := regexp.MustCompile(`%[hprunCdiklLT]`)
	if sshTokenRegex.MatchString(hostname) {
		// If it contains SSH tokens, it's a valid SSH config construct
		return true
	}

	// RFC 1123: each label must start with alphanumeric, end with alphanumeric,
	// and contain only alphanumeric and hyphens. Labels are 1-63 chars.
	// A hostname is one or more labels separated by dots.
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]|[a-zA-Z0-9]{0,62})?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]|[a-zA-Z0-9]{0,62})?)*$`)
	return hostnameRegex.MatchString(hostname)
}

// ValidateIP checks if an IP address is valid
func ValidateIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ValidatePort checks if a port is valid
func ValidatePort(port string) bool {
	if port == "" {
		return true // Empty port defaults to 22
	}
	portNum, err := strconv.Atoi(port)
	return err == nil && portNum >= 1 && portNum <= 65535
}

// ValidateHostName checks if a host name is valid for SSH config
func ValidateHostName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
		return false
	}
	// Host name cannot contain whitespace or special SSH config characters
	return !strings.ContainsAny(name, " \t\n\r#")
}

// ValidateIdentityFile checks if an identity file path is valid
func ValidateIdentityFile(path string) bool {
	if path == "" {
		return true // Optional field
	}
	// SSH tokens (e.g. %d, %h, %r, %u) are resolved by SSH at connection time
	sshTokenRegex := regexp.MustCompile(`%[hprunCdiklLT]`)
	if sshTokenRegex.MatchString(path) {
		return true
	}
	// Expand environment variables ($VAR and ${VAR}); track undefined ones
	hasUndefined := false
	path = os.Expand(path, func(key string) string {
		val, ok := os.LookupEnv(key)
		if !ok {
			hasUndefined = true
			return "$" + key
		}
		return val
	})
	// If any variable was undefined, accept the path (SSH will report the error)
	if hasUndefined {
		return true
	}
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		path = filepath.Join(homeDir, path[2:])
	}
	_, err := os.Stat(path)
	return err == nil
}

// ValidateHost validates all host fields
func ValidateHost(name, hostname, port, identity string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("host name is required")
	}

	if !ValidateHostName(name) {
		return fmt.Errorf("invalid host name: cannot contain spaces or special characters")
	}

	if strings.TrimSpace(hostname) == "" {
		return fmt.Errorf("hostname/IP is required")
	}

	if !ValidateHostname(hostname) && !ValidateIP(hostname) {
		return fmt.Errorf("invalid hostname or IP address format")
	}

	if !ValidatePort(port) {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if identity != "" && !ValidateIdentityFile(identity) {
		return fmt.Errorf("identity file does not exist: %s", identity)
	}

	return nil
}
