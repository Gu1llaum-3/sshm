package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// SSHHost represents an SSH host configuration
type SSHHost struct {
	Name          string
	Hostname      string
	User          string
	Port          string
	Identity      string
	ProxyJump     string
	ProxyCommand  string
	Options       string
	RemoteCommand string // Command to execute after SSH connection
	RequestTTY    string // Request TTY (yes, no, force, auto)
	Tags          []string
	SourceFile    string // Path to the config file where this host is defined

	// Temporary field to handle multiple aliases during parsing
	aliasNames []string `json:"-"` // Do not serialize this field
}

// GetDefaultSSHConfigPath returns the default SSH config path for the current platform
func GetDefaultSSHConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "windows":
		return filepath.Join(homeDir, ".ssh", "config"), nil
	default:
		// Linux, macOS, etc.
		return filepath.Join(homeDir, ".ssh", "config"), nil
	}
}

// GetSSHMConfigDir returns the SSHM config directory
func GetSSHMConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var configDir string
	switch runtime.GOOS {
	case "windows":
		// Use %APPDATA%/sshm on Windows
		appData := os.Getenv("APPDATA")
		if appData != "" {
			configDir = filepath.Join(appData, "sshm")
		} else {
			configDir = filepath.Join(homeDir, ".config", "sshm")
		}
	default:
		// Use XDG Base Directory specification
		xdgConfigDir := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigDir != "" {
			configDir = filepath.Join(xdgConfigDir, "sshm")
		} else {
			configDir = filepath.Join(homeDir, ".config", "sshm")
		}
	}

	return configDir, nil
}

// GetSSHMBackupDir returns the SSHM backup directory
func GetSSHMBackupDir() (string, error) {
	configDir, err := GetSSHMConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "backups"), nil
}

// GetSSHDirectory returns the .ssh directory path
func GetSSHDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".ssh"), nil
}

// ensureSSHDirectory creates the .ssh directory with appropriate permissions
func ensureSSHDirectory() error {
	sshDir, err := GetSSHDirectory()
	if err != nil {
		return err
	}

	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		// 0700 provides owner-only access across platforms
		return os.MkdirAll(sshDir, 0700)
	}
	return nil
}

// configMutex protects SSH config file operations from race conditions
var configMutex sync.Mutex

// backupConfig creates a backup of the SSH config file in ~/.config/sshm/backups/
func backupConfig(configPath string) error {
	// Get backup directory and ensure it exists
	backupDir, err := GetSSHMBackupDir()
	if err != nil {
		return fmt.Errorf("failed to get backup directory: %w", err)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create simple backup filename (overwrites previous backup)
	filename := filepath.Base(configPath)
	backupPath := filepath.Join(backupDir, filename+".backup")

	// Copy file
	src, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// Set appropriate permissions
	return os.Chmod(backupPath, 0600)
}

// ParseSSHConfig parses the SSH config file and returns the list of hosts
func ParseSSHConfig() ([]SSHHost, error) {
	configPath, err := GetDefaultSSHConfigPath()
	if err != nil {
		return nil, err
	}
	return ParseSSHConfigFile(configPath)
}

// ParseSSHConfigFile parses a specific SSH config file and returns the list of hosts
func ParseSSHConfigFile(configPath string) ([]SSHHost, error) {
	return parseSSHConfigFileWithProcessedFiles(configPath, make(map[string]bool))
}

// parseSSHConfigFileWithProcessedFiles parses SSH config with include support
func parseSSHConfigFileWithProcessedFiles(configPath string, processedFiles map[string]bool) ([]SSHHost, error) {
	// Resolve absolute path to prevent infinite recursion
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", configPath, err)
	}

	// Check for circular includes
	if processedFiles[absPath] {
		return []SSHHost{}, nil // Skip already processed files silently
	}
	processedFiles[absPath] = true

	// Check if the file exists, otherwise create it (and the parent directory if needed)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Only create the main config file, not included files
		if absPath == getMainConfigPath() {
			// Ensure .ssh directory exists with proper permissions
			if err := ensureSSHDirectory(); err != nil {
				return nil, fmt.Errorf("failed to create .ssh directory: %w", err)
			}

			file, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return nil, fmt.Errorf("failed to create SSH config file: %w", err)
			}
			file.Close()

			// Set secure permissions on the config file
			if err := SetSecureFilePermissions(configPath); err != nil {
				return nil, fmt.Errorf("failed to set secure permissions: %w", err)
			}
		}

		// File doesn't exist, return empty host list
		return []SSHHost{}, nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var hosts []SSHHost
	var currentHost *SSHHost
	var pendingTags []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore empty lines
		if line == "" {
			continue
		}

		// Check for tags comment
		if strings.HasPrefix(line, "# Tags:") {
			tagsStr := strings.TrimPrefix(line, "# Tags:")
			tagsStr = strings.TrimSpace(tagsStr)
			if tagsStr != "" {
				// Split tags by comma and trim whitespace
				for _, tag := range strings.Split(tagsStr, ",") {
					tag = strings.TrimSpace(tag)
					if tag != "" {
						pendingTags = append(pendingTags, tag)
					}
				}
			}
			continue
		}

		// Ignore other comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Split line into words
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")

		switch key {
		case "include":
			// Handle Include directive
			includeHosts, err := processIncludeDirective(value, configPath, processedFiles)
			if err != nil {
				// Don't fail the entire parse if include fails, just skip it
				continue
			}
			hosts = append(hosts, includeHosts...)
		case "host":
			// New host, save previous one if it exists
			if currentHost != nil {
				hosts = append(hosts, *currentHost)

				// Handle aliases: create duplicate hosts for each alias
				if len(currentHost.aliasNames) > 0 {
					for _, aliasName := range currentHost.aliasNames {
						aliasHost := *currentHost // Copy the host
						aliasHost.Name = aliasName
						aliasHost.aliasNames = nil // Clear temporary field
						hosts = append(hosts, aliasHost)
					}
				}
			}

			// Parse multiple host names from the Host line
			hostNames := strings.Fields(value)

			// Skip hosts with wildcards (*, ?) as they are typically patterns, not actual hosts
			var validHostNames []string
			for _, hostName := range hostNames {
				if !strings.ContainsAny(hostName, "*?") {
					validHostNames = append(validHostNames, hostName)
				}
			}

			if len(validHostNames) == 0 {
				currentHost = nil
				pendingTags = nil
				continue
			}

			// For multiple hosts, we create the first one normally
			// and will duplicate it for others after parsing the block
			currentHost = &SSHHost{
				Name:       validHostNames[0], // First name as reference
				Port:       "22",              // Default port
				Tags:       pendingTags,       // Assign pending tags to this host
				SourceFile: absPath,           // Track which file this host comes from
			}

			// Store additional host names for later processing
			if len(validHostNames) > 1 {
				currentHost.aliasNames = validHostNames[1:]
			}

			// Clear pending tags for next host
			pendingTags = nil
		case "hostname":
			if currentHost != nil {
				currentHost.Hostname = value
			}
		case "user":
			if currentHost != nil {
				currentHost.User = value
			}
		case "port":
			if currentHost != nil {
				currentHost.Port = value
			}
		case "identityfile":
			if currentHost != nil {
				currentHost.Identity = value
			}
		case "proxyjump":
			if currentHost != nil {
				currentHost.ProxyJump = value
			}
		case "proxycommand":
			if currentHost != nil {
				currentHost.ProxyCommand = value
			}
		case "remotecommand":
			if currentHost != nil {
				currentHost.RemoteCommand = value
			}
		case "requesttty":
			if currentHost != nil {
				currentHost.RequestTTY = value
			}
		default:
			// Handle other SSH options
			if currentHost != nil && strings.TrimSpace(line) != "" {
				// Store options in config format (key value), not command format
				if currentHost.Options == "" {
					currentHost.Options = parts[0] + " " + value
				} else {
					currentHost.Options += "\n" + parts[0] + " " + value
				}
			}
		}
	}

	// Add the last host if it exists
	if currentHost != nil {
		hosts = append(hosts, *currentHost)

		// Handle aliases: create duplicate hosts for each alias
		if len(currentHost.aliasNames) > 0 {
			for _, aliasName := range currentHost.aliasNames {
				aliasHost := *currentHost // Copy the host
				aliasHost.Name = aliasName
				aliasHost.aliasNames = nil // Clear temporary field
				hosts = append(hosts, aliasHost)
			}
		}
		// Clear the temporary field from the original
		currentHost.aliasNames = nil
	}

	return hosts, scanner.Err()
}

// processIncludeDirective processes an Include directive and returns hosts from included files
func processIncludeDirective(pattern string, baseConfigPath string, processedFiles map[string]bool) ([]SSHHost, error) {
	// Expand tilde to home directory
	if strings.HasPrefix(pattern, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		pattern = filepath.Join(homeDir, pattern[1:])
	}

	// If pattern is not absolute, make it relative to the base config directory
	if !filepath.IsAbs(pattern) {
		baseDir := filepath.Dir(baseConfigPath)
		pattern = filepath.Join(baseDir, pattern)
	}

	// Use glob to find matching files
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
	}

	var allHosts []SSHHost
	for _, match := range matches {
		// Skip directories
		if info, err := os.Stat(match); err == nil && info.IsDir() {
			continue
		}

		// Skip backup files created by sshm (*.backup)
		if strings.HasSuffix(match, ".backup") {
			continue
		}

		// Skip markdown files (*.md)
		if strings.HasSuffix(match, ".md") {
			continue
		}

		// Skip common non-SSH config file types
		if isNonSSHConfigFile(match) {
			continue
		}

		// Recursively parse the included file
		hosts, err := parseSSHConfigFileWithProcessedFiles(match, processedFiles)
		if err != nil {
			// Skip files that can't be parsed rather than failing completely
			continue
		}
		allHosts = append(allHosts, hosts...)
	}

	return allHosts, nil
}

// isNonSSHConfigFile checks if a file should be excluded from SSH config parsing
func isNonSSHConfigFile(filePath string) bool {
	fileName := strings.ToLower(filepath.Base(filePath))

	// Skip common documentation files
	if fileName == "readme" || fileName == "readme.txt" {
		return true
	}

	// Skip files with common non-config extensions
	excludedExtensions := []string{
		".txt", ".md", ".rst", ".doc", ".docx", ".pdf",
		".log", ".tmp", ".bak", ".old", ".orig",
		".json", ".xml", ".yaml", ".yml", ".toml",
		".sh", ".bash", ".zsh", ".fish", ".ps1", ".bat", ".cmd",
		".py", ".pl", ".rb", ".js", ".php", ".go", ".c", ".cpp",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".svg",
		".zip", ".tar", ".gz", ".bz2", ".xz",
	}

	for _, ext := range excludedExtensions {
		if strings.HasSuffix(fileName, ext) {
			return true
		}
	}

	// Skip hidden files (starting with .)
	if strings.HasPrefix(fileName, ".") {
		return true
	}

	// Additional check: if file contains common non-SSH content indicators
	// This is a more expensive check, so we do it last
	if hasNonSSHContent(filePath) {
		return true
	}

	return false
}

// hasNonSSHContent performs a quick content check to identify non-SSH files
func hasNonSSHContent(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false // If we can't read it, don't exclude it
	}
	defer file.Close()

	// Read only the first few KB to check content
	buffer := make([]byte, 2048)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	content := strings.ToLower(string(buffer[:n]))

	// Check for common non-SSH file indicators
	nonSSHIndicators := []string{
		"<!doctype", "<html>", "<xml>", "<?xml",
		"#!/bin/", "#!/usr/bin/",
		"# readme", "# documentation", "# license",
		"package main", "function ", "class ", "def ",
		"import ", "require ", "#include",
		"SELECT ", "INSERT ", "UPDATE ", "DELETE ",
	}

	for _, indicator := range nonSSHIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}

	return false
}

// getMainConfigPath returns the main SSH config path for comparison
func getMainConfigPath() string {
	configPath, _ := GetDefaultSSHConfigPath()
	absPath, _ := filepath.Abs(configPath)
	return absPath
}

// formatSSHConfigValue formats a value for SSH config file, adding quotes if necessary
func formatSSHConfigValue(value string) string {
	if value == "" {
		return value
	}

	// If the value contains spaces, wrap it in quotes
	if strings.Contains(value, " ") {
		return `"` + value + `"`
	}

	return value
}

// AddSSHHost adds a new SSH host to the config file
func AddSSHHost(host SSHHost) error {
	configPath, err := GetDefaultSSHConfigPath()
	if err != nil {
		return err
	}
	return AddSSHHostToFile(host, configPath)
}

// AddSSHHostToFile adds a new SSH host to a specific config file
func AddSSHHostToFile(host SSHHost, configPath string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// Create backup before modification if file exists
	if _, err := os.Stat(configPath); err == nil {
		if err := backupConfig(configPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Check if host already exists in the specified config file
	exists, err := HostExistsInFile(host.Name, configPath)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("host '%s' already exists", host.Name)
	}

	// Open file in append mode
	file, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the configuration
	_, err = file.WriteString("\n")
	if err != nil {
		return err
	}

	// Write tags if present
	if len(host.Tags) > 0 {
		_, err = file.WriteString("# Tags: " + strings.Join(host.Tags, ", ") + "\n")
		if err != nil {
			return err
		}
	}

	// Write host configuration
	_, err = file.WriteString(fmt.Sprintf("Host %s\n", host.Name))
	if err != nil {
		return err
	}

	_, err = file.WriteString(fmt.Sprintf("    HostName %s\n", host.Hostname))
	if err != nil {
		return err
	}

	if host.User != "" {
		_, err = file.WriteString(fmt.Sprintf("    User %s\n", host.User))
		if err != nil {
			return err
		}
	}

	if host.Port != "" && host.Port != "22" {
		_, err = file.WriteString(fmt.Sprintf("    Port %s\n", host.Port))
		if err != nil {
			return err
		}
	}

	if host.Identity != "" {
		_, err = file.WriteString(fmt.Sprintf("    IdentityFile %s\n", formatSSHConfigValue(host.Identity)))
		if err != nil {
			return err
		}
	}

	if host.ProxyJump != "" {
		_, err = file.WriteString(fmt.Sprintf("    ProxyJump %s\n", host.ProxyJump))
		if err != nil {
			return err
		}
	}

	if host.ProxyCommand != "" {
		_, err = file.WriteString(fmt.Sprintf("    ProxyCommand=%s\n", host.ProxyCommand))
		if err != nil {
			return err
		}
	}

	if host.RemoteCommand != "" {
		_, err = file.WriteString(fmt.Sprintf("    RemoteCommand %s\n", host.RemoteCommand))
		if err != nil {
			return err
		}
	}

	if host.RequestTTY != "" {
		_, err = file.WriteString(fmt.Sprintf("    RequestTTY %s\n", host.RequestTTY))
		if err != nil {
			return err
		}
	}

	// Write SSH options
	if host.Options != "" {
		// Split options by newlines and write each one
		options := strings.Split(host.Options, "\n")
		for _, option := range options {
			option = strings.TrimSpace(option)
			if option != "" {
				_, err = file.WriteString(fmt.Sprintf("    %s\n", option))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ParseSSHOptionsFromCommand converts SSH command line options to config format
// Input: "-o Compression=yes -o ServerAliveInterval=60"
// Output: "Compression yes\nServerAliveInterval 60"
func ParseSSHOptionsFromCommand(options string) string {
	if options == "" {
		return ""
	}

	var result []string
	parts := strings.Split(options, "-o")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Replace = with space for SSH config format
		option := strings.ReplaceAll(part, "=", " ")
		result = append(result, option)
	}

	return strings.Join(result, "\n")
}

// FormatSSHOptionsForCommand converts SSH config options to command line format
// Input: "Compression yes\nServerAliveInterval 60"
// Output: "-o Compression=yes -o ServerAliveInterval=60"
func FormatSSHOptionsForCommand(options string) string {
	if options == "" {
		return ""
	}

	var result []string
	lines := strings.Split(options, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Replace space with = for command line format
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			result = append(result, fmt.Sprintf("-o %s=%s", parts[0], parts[1]))
		} else {
			result = append(result, fmt.Sprintf("-o %s", line))
		}
	}

	return strings.Join(result, " ")
}

// HostExists checks if a host already exists in the config
func HostExists(hostName string) (bool, error) {
	hosts, err := ParseSSHConfig()
	if err != nil {
		return false, err
	}

	for _, host := range hosts {
		if host.Name == hostName {
			return true, nil
		}
	}
	return false, nil
}

// HostExistsInFile checks if a host exists in a specific config file
func HostExistsInFile(hostName string, configPath string) (bool, error) {
	// Parse only the specific file, not its includes
	return HostExistsInSpecificFile(hostName, configPath)
}

// HostExistsInSpecificFile checks if a host exists in a specific file only (no includes)
func HostExistsInSpecificFile(hostName string, configPath string) (bool, error) {
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for Host declaration
		if strings.HasPrefix(strings.ToLower(line), "host ") {
			// Extract host names (can be multiple hosts on one line)
			hostPart := strings.TrimSpace(line[5:]) // Remove "host "
			hostNames := strings.Fields(hostPart)

			for _, name := range hostNames {
				if name == hostName {
					return true, nil
				}
			}
		}
	}

	return false, scanner.Err()
}

// GetSSHHost retrieves a specific host configuration by name
func GetSSHHost(hostName string) (*SSHHost, error) {
	hosts, err := ParseSSHConfig()
	if err != nil {
		return nil, err
	}

	for _, host := range hosts {
		if host.Name == hostName {
			return &host, nil
		}
	}
	return nil, fmt.Errorf("host '%s' not found", hostName)
}

// GetSSHHostFromFile retrieves a specific host configuration by name from a specific config file
func GetSSHHostFromFile(hostName string, configPath string) (*SSHHost, error) {
	hosts, err := ParseSSHConfigFile(configPath)
	if err != nil {
		return nil, err
	}

	for _, host := range hosts {
		if host.Name == hostName {
			return &host, nil
		}
	}
	return nil, fmt.Errorf("host '%s' not found", hostName)
}

// QuickHostExists performs a fast check if a host exists without full parsing
// This is optimized for connection scenarios where we just need to verify existence
func QuickHostExists(hostName string) (bool, error) {
	configPath, err := GetDefaultSSHConfigPath()
	if err != nil {
		return false, err
	}
	return QuickHostExistsInFile(hostName, configPath)
}

// QuickHostExistsInFile performs a fast check if a host exists in config files
// This stops parsing as soon as the host is found, making it much faster for connection scenarios
func QuickHostExistsInFile(hostName string, configPath string) (bool, error) {
	return quickHostSearchInFile(hostName, configPath, make(map[string]bool))
}

// quickHostSearchInFile performs optimized host search with early termination
func quickHostSearchInFile(hostName string, configPath string, processedFiles map[string]bool) (bool, error) {
	// Resolve absolute path to prevent infinite recursion
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve absolute path for %s: %w", configPath, err)
	}

	// Check for circular includes
	if processedFiles[absPath] {
		return false, nil // Skip already processed files silently
	}
	processedFiles[absPath] = true

	// Check if the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, nil // File doesn't exist, host not found
	}

	file, err := os.Open(configPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore empty lines and comments (except includes)
		if line == "" || (strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "# Tags:")) {
			continue
		}

		// Split line into words
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")

		switch key {
		case "include":
			// Handle Include directive - search in included files
			if found, err := quickSearchInclude(hostName, value, configPath, processedFiles); err == nil && found {
				return true, nil // Found in included file
			}
		case "host":
			// Parse multiple host names from the Host line
			hostNames := strings.Fields(value)

			// Check if our target host is in this Host declaration
			for _, candidateHostName := range hostNames {
				// Skip hosts with wildcards (*, ?) as they are typically patterns
				if !strings.ContainsAny(candidateHostName, "*?") && candidateHostName == hostName {
					return true, nil // Found the host!
				}
			}
		}
	}

	return false, scanner.Err()
}

// quickSearchInclude handles Include directives during quick host search
func quickSearchInclude(hostName, pattern, baseConfigPath string, processedFiles map[string]bool) (bool, error) {
	// Expand tilde to home directory
	if strings.HasPrefix(pattern, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return false, fmt.Errorf("failed to get home directory: %w", err)
		}
		pattern = filepath.Join(homeDir, pattern[1:])
	}

	// If pattern is not absolute, make it relative to the base config directory
	if !filepath.IsAbs(pattern) {
		baseDir := filepath.Dir(baseConfigPath)
		pattern = filepath.Join(baseDir, pattern)
	}

	// Use glob to find matching files
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false, fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
	}

	for _, match := range matches {
		// Skip directories
		if info, err := os.Stat(match); err == nil && info.IsDir() {
			continue
		}

		// Skip non-SSH config files (this avoids parsing README, etc.)
		if isNonSSHConfigFile(match) {
			continue
		}

		// Search in the included file
		if found, err := quickHostSearchInFile(hostName, match, processedFiles); err == nil && found {
			return true, nil // Found in this included file
		}
	}

	return false, nil
}

// UpdateSSHHost updates an existing SSH host configuration
func UpdateSSHHost(oldName string, newHost SSHHost) error {
	return UpdateSSHHostV2(oldName, newHost)
}

// IsPartOfMultiHostDeclaration checks if a host is part of a multi-host declaration
func IsPartOfMultiHostDeclaration(hostName string, configPath string) (bool, []string, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return false, nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(strings.ToLower(line), "host ") {
			// Extract host names (can be multiple hosts on one line)
			hostPart := strings.TrimSpace(line[5:]) // Remove "host "
			hostNames := strings.Fields(hostPart)

			// Check if our target host is in this Host declaration
			for _, name := range hostNames {
				if name == hostName {
					if len(hostNames) > 1 {
						return true, hostNames, nil
					}
					return false, hostNames, nil
				}
			}
		}
	}

	return false, nil, scanner.Err()
}

// UpdateSSHHostInFile updates an existing SSH host configuration in a specific file
func UpdateSSHHostInFile(oldName string, newHost SSHHost, configPath string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// Create backup before modification
	if err := backupConfig(configPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Check if this host is part of a multi-host declaration
	isMultiHost, hostNames, err := IsPartOfMultiHostDeclaration(oldName, configPath)
	if err != nil {
		return fmt.Errorf("failed to check multi-host declaration: %w", err)
	}

	// Read the current config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	i := 0
	hostFound := false

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Check for tags comment followed by Host
		if strings.HasPrefix(line, "# Tags:") && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])

			// Check if this is a Host line that contains our target host
			if strings.HasPrefix(nextLine, "Host ") {
				hostPart := strings.TrimSpace(nextLine[5:]) // Remove "Host "
				foundHostNames := strings.Fields(hostPart)

				// Check if our target host is in this Host declaration
				targetHostIndex := -1
				for idx, hostName := range foundHostNames {
					if hostName == oldName {
						targetHostIndex = idx
						break
					}
				}

				if targetHostIndex != -1 {
					hostFound = true

					if isMultiHost && len(hostNames) > 1 {
						// Strategy: Remove old host from the line, add new host as separate entry
						// Remove the old host name from the Host line
						var remainingHosts []string
						for idx, hostName := range foundHostNames {
							if idx != targetHostIndex {
								remainingHosts = append(remainingHosts, hostName)
							}
						}

						// Keep the tags comment
						newLines = append(newLines, lines[i])

						// Update the Host line with remaining hosts
						if len(remainingHosts) > 0 {
							newLines = append(newLines, "Host "+strings.Join(remainingHosts, " "))

							// Copy the existing configuration for remaining hosts
							i += 2 // Skip tags and original Host line
							for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
								newLines = append(newLines, lines[i])
								i++
							}
						} else {
							// No remaining hosts, skip the entire block
							i += 2 // Skip tags and Host line
							for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
								i++
							}
						}

						// Add the new host as a separate entry
						newLines = append(newLines, "")
						if len(newHost.Tags) > 0 {
							newLines = append(newLines, "# Tags: "+strings.Join(newHost.Tags, ", "))
						}
						newLines = append(newLines, "Host "+newHost.Name)
						newLines = append(newLines, "    HostName "+newHost.Hostname)
						if newHost.User != "" {
							newLines = append(newLines, "    User "+newHost.User)
						}
						if newHost.Port != "" && newHost.Port != "22" {
							newLines = append(newLines, "    Port "+newHost.Port)
						}
						if newHost.Identity != "" {
							newLines = append(newLines, "    IdentityFile "+formatSSHConfigValue(newHost.Identity))
						}
						if newHost.ProxyJump != "" {
							newLines = append(newLines, "    ProxyJump "+newHost.ProxyJump)
						}
						if newHost.ProxyCommand != "" {
							newLines = append(newLines, "    ProxyCommand="+newHost.ProxyCommand)
						}
						if newHost.RemoteCommand != "" {
							newLines = append(newLines, "    RemoteCommand "+newHost.RemoteCommand)
						}
						if newHost.RequestTTY != "" {
							newLines = append(newLines, "    RequestTTY "+newHost.RequestTTY)
						}
						// Write SSH options
						if newHost.Options != "" {
							options := strings.Split(newHost.Options, "\n")
							for _, option := range options {
								option = strings.TrimSpace(option)
								if option != "" {
									newLines = append(newLines, "    "+option)
								}
							}
						}
						newLines = append(newLines, "")

						continue
					} else {
						// Simple case: only one host, replace entire block
						// Skip until we find the end of this host block (empty line or next Host)
						i += 2 // Skip tags and Host line
						for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
							i++
						}

						// Skip any trailing empty lines after the host block
						for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
							i++
						}

						// Insert new configuration at this position
						// Add empty line only if the previous line is not empty
						if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
							newLines = append(newLines, "")
						}
						if len(newHost.Tags) > 0 {
							newLines = append(newLines, "# Tags: "+strings.Join(newHost.Tags, ", "))
						}
						newLines = append(newLines, "Host "+newHost.Name)
						newLines = append(newLines, "    HostName "+newHost.Hostname)
						if newHost.User != "" {
							newLines = append(newLines, "    User "+newHost.User)
						}
						if newHost.Port != "" && newHost.Port != "22" {
							newLines = append(newLines, "    Port "+newHost.Port)
						}
						if newHost.Identity != "" {
							newLines = append(newLines, "    IdentityFile "+formatSSHConfigValue(newHost.Identity))
						}
						if newHost.ProxyJump != "" {
							newLines = append(newLines, "    ProxyJump "+newHost.ProxyJump)
						}
						if newHost.ProxyCommand != "" {
							newLines = append(newLines, "    ProxyCommand="+newHost.ProxyCommand)
						}
						if newHost.RemoteCommand != "" {
							newLines = append(newLines, "    RemoteCommand "+newHost.RemoteCommand)
						}
						if newHost.RequestTTY != "" {
							newLines = append(newLines, "    RequestTTY "+newHost.RequestTTY)
						}
						// Write SSH options
						if newHost.Options != "" {
							options := strings.Split(newHost.Options, "\n")
							for _, option := range options {
								option = strings.TrimSpace(option)
								if option != "" {
									newLines = append(newLines, "    "+option)
								}
							}
						}

						// Add empty line after the host configuration for separation
						newLines = append(newLines, "")

						continue
					}
				}
			}
		}

		// Check for Host line without tags
		if strings.HasPrefix(line, "Host ") {
			hostPart := strings.TrimSpace(line[5:]) // Remove "Host "
			foundHostNames := strings.Fields(hostPart)

			// Check if our target host is in this Host declaration
			targetHostIndex := -1
			for idx, hostName := range foundHostNames {
				if hostName == oldName {
					targetHostIndex = idx
					break
				}
			}

			if targetHostIndex != -1 {
				hostFound = true

				if isMultiHost && len(hostNames) > 1 {
					// Strategy: Remove old host from the line, add new host as separate entry
					// Remove the old host name from the Host line
					var remainingHosts []string
					for idx, hostName := range foundHostNames {
						if idx != targetHostIndex {
							remainingHosts = append(remainingHosts, hostName)
						}
					}

					// Update the Host line with remaining hosts
					if len(remainingHosts) > 0 {
						newLines = append(newLines, "Host "+strings.Join(remainingHosts, " "))

						// Copy the existing configuration for remaining hosts
						i++ // Skip original Host line
						for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
							newLines = append(newLines, lines[i])
							i++
						}
					} else {
						// No remaining hosts, skip the entire block
						i++ // Skip Host line
						for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
							i++
						}
					}

					// Add the new host as a separate entry
					newLines = append(newLines, "")
					if len(newHost.Tags) > 0 {
						newLines = append(newLines, "# Tags: "+strings.Join(newHost.Tags, ", "))
					}
					newLines = append(newLines, "Host "+newHost.Name)
					newLines = append(newLines, "    HostName "+newHost.Hostname)
					if newHost.User != "" {
						newLines = append(newLines, "    User "+newHost.User)
					}
					if newHost.Port != "" && newHost.Port != "22" {
						newLines = append(newLines, "    Port "+newHost.Port)
					}
					if newHost.Identity != "" {
						newLines = append(newLines, "    IdentityFile "+formatSSHConfigValue(newHost.Identity))
					}
					if newHost.ProxyJump != "" {
						newLines = append(newLines, "    ProxyJump "+newHost.ProxyJump)
					}
					if newHost.ProxyCommand != "" {
						newLines = append(newLines, "    ProxyCommand="+newHost.ProxyCommand)
					}
					if newHost.RemoteCommand != "" {
						newLines = append(newLines, "    RemoteCommand "+newHost.RemoteCommand)
					}
					if newHost.RequestTTY != "" {
						newLines = append(newLines, "    RequestTTY "+newHost.RequestTTY)
					}
					// Write SSH options
					if newHost.Options != "" {
						options := strings.Split(newHost.Options, "\n")
						for _, option := range options {
							option = strings.TrimSpace(option)
							if option != "" {
								newLines = append(newLines, "    "+option)
							}
						}
					}
					newLines = append(newLines, "")

					continue
				} else {
					// Simple case: only one host, replace entire block
					// Skip until we find the end of this host block
					i++ // Skip Host line
					for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
						i++
					}

					// Skip any trailing empty lines after the host block
					for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
						i++
					}

					// Insert new configuration
					// Add empty line only if the previous line is not empty
					if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
						newLines = append(newLines, "")
					}
					if len(newHost.Tags) > 0 {
						newLines = append(newLines, "# Tags: "+strings.Join(newHost.Tags, ", "))
					}
					newLines = append(newLines, "Host "+newHost.Name)
					newLines = append(newLines, "    HostName "+newHost.Hostname)
					if newHost.User != "" {
						newLines = append(newLines, "    User "+newHost.User)
					}
					if newHost.Port != "" && newHost.Port != "22" {
						newLines = append(newLines, "    Port "+newHost.Port)
					}
					if newHost.Identity != "" {
						newLines = append(newLines, "    IdentityFile "+formatSSHConfigValue(newHost.Identity))
					}
					if newHost.ProxyJump != "" {
						newLines = append(newLines, "    ProxyJump "+newHost.ProxyJump)
					}
					if newHost.ProxyCommand != "" {
						newLines = append(newLines, "    ProxyCommand="+newHost.ProxyCommand)
					}
					if newHost.RemoteCommand != "" {
						newLines = append(newLines, "    RemoteCommand "+newHost.RemoteCommand)
					}
					if newHost.RequestTTY != "" {
						newLines = append(newLines, "    RequestTTY "+newHost.RequestTTY)
					}
					// Write SSH options
					if newHost.Options != "" {
						options := strings.Split(newHost.Options, "\n")
						for _, option := range options {
							option = strings.TrimSpace(option)
							if option != "" {
								newLines = append(newLines, "    "+option)
							}
						}
					}

					// Add empty line after the host configuration for separation
					newLines = append(newLines, "")

					continue
				}
			}
		}

		// Keep other lines as-is
		newLines = append(newLines, lines[i])
		i++
	}

	if !hostFound {
		return fmt.Errorf("host '%s' not found", oldName)
	}

	// Write back to file
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(configPath, []byte(newContent), 0600)
}

// DeleteSSHHost removes an SSH host configuration from the config file
func DeleteSSHHost(hostName string) error {
	return DeleteSSHHostV2(hostName)
}

// DeleteSSHHostFromFile deletes an SSH host from a specific config file
func DeleteSSHHostFromFile(hostName, configPath string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// Create backup before modification
	if err := backupConfig(configPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Check if this host is part of a multi-host declaration
	isMultiHost, hostNames, err := IsPartOfMultiHostDeclaration(hostName, configPath)
	if err != nil {
		return fmt.Errorf("failed to check multi-host declaration: %w", err)
	}

	// Read the current config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	i := 0
	hostFound := false

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Check for tags comment followed by Host
		if strings.HasPrefix(line, "# Tags:") && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])

			// Check if this is a Host line that contains our target host
			if strings.HasPrefix(nextLine, "Host ") {
				hostPart := strings.TrimSpace(nextLine[5:]) // Remove "Host "
				foundHostNames := strings.Fields(hostPart)

				// Check if our target host is in this Host declaration
				targetHostIndex := -1
				for idx, host := range foundHostNames {
					if host == hostName {
						targetHostIndex = idx
						break
					}
				}

				if targetHostIndex != -1 {
					hostFound = true

					if isMultiHost && len(hostNames) > 1 {
						// Remove the target host from the multi-host line
						var remainingHosts []string
						for idx, host := range foundHostNames {
							if idx != targetHostIndex {
								remainingHosts = append(remainingHosts, host)
							}
						}

						// Keep the tags comment
						newLines = append(newLines, lines[i])

						if len(remainingHosts) > 0 {
							// Update the Host line with remaining hosts
							newLines = append(newLines, "Host "+strings.Join(remainingHosts, " "))

							// Copy the existing configuration for remaining hosts
							i += 2 // Skip tags and original Host line
							for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
								newLines = append(newLines, lines[i])
								i++
							}
						} else {
							// No remaining hosts, skip the entire block
							i += 2 // Skip tags and Host line
							for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
								i++
							}
						}

						// Skip any trailing empty lines after the host block
						for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
							i++
						}

						continue
					} else {
						// Single host or last host in multi-host block, delete entire block
						// Skip tags comment and Host line
						i += 2

						// Skip until we find the end of this host block (empty line or next Host)
						for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
							i++
						}

						// Skip any trailing empty lines after the host block
						for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
							i++
						}

						continue
					}
				}
			}
		}

		// Check for Host line without tags
		if strings.HasPrefix(line, "Host ") {
			hostPart := strings.TrimSpace(line[5:]) // Remove "Host "
			foundHostNames := strings.Fields(hostPart)

			// Check if our target host is in this Host declaration
			targetHostIndex := -1
			for idx, host := range foundHostNames {
				if host == hostName {
					targetHostIndex = idx
					break
				}
			}

			if targetHostIndex != -1 {
				hostFound = true

				if isMultiHost && len(hostNames) > 1 {
					// Remove the target host from the multi-host line
					var remainingHosts []string
					for idx, host := range foundHostNames {
						if idx != targetHostIndex {
							remainingHosts = append(remainingHosts, host)
						}
					}

					if len(remainingHosts) > 0 {
						// Update the Host line with remaining hosts
						newLines = append(newLines, "Host "+strings.Join(remainingHosts, " "))

						// Copy the existing configuration for remaining hosts
						i++ // Skip original Host line
						for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
							newLines = append(newLines, lines[i])
							i++
						}
					} else {
						// No remaining hosts, skip the entire block
						i++ // Skip Host line
						for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
							i++
						}
					}

					// Skip any trailing empty lines after the host block
					for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
						i++
					}

					continue
				} else {
					// Single host, delete entire block
					// Skip Host line
					i++

					// Skip until we find the end of this host block
					for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
						i++
					}

					// Skip any trailing empty lines after the host block
					for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
						i++
					}

					continue
				}
			}
		}

		// Keep other lines as-is
		newLines = append(newLines, lines[i])
		i++
	}

	if !hostFound {
		return fmt.Errorf("host '%s' not found", hostName)
	}

	// Write back to file
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(configPath, []byte(newContent), 0600)
}

// FindHostInAllConfigs finds a host in all configuration files and returns the host with its source file
func FindHostInAllConfigs(hostName string) (*SSHHost, error) {
	hosts, err := ParseSSHConfig()
	if err != nil {
		return nil, err
	}

	for _, host := range hosts {
		if host.Name == hostName {
			return &host, nil
		}
	}

	return nil, fmt.Errorf("host '%s' not found in any configuration file", hostName)
}

// GetAllConfigFiles returns all SSH config files (main + included files)
func GetAllConfigFiles() ([]string, error) {
	configPath, err := GetDefaultSSHConfigPath()
	if err != nil {
		return nil, err
	}

	processedFiles := make(map[string]bool)
	_, _ = parseSSHConfigFileWithProcessedFiles(configPath, processedFiles)

	files := make([]string, 0, len(processedFiles))
	for file := range processedFiles {
		files = append(files, file)
	}

	return files, nil
}

// GetAllConfigFilesFromBase returns all SSH config files starting from a specific base config file
func GetAllConfigFilesFromBase(baseConfigPath string) ([]string, error) {
	if baseConfigPath == "" {
		// Fallback to default behavior
		return GetAllConfigFiles()
	}

	processedFiles := make(map[string]bool)
	_, _ = parseSSHConfigFileWithProcessedFiles(baseConfigPath, processedFiles)

	files := make([]string, 0, len(processedFiles))
	for file := range processedFiles {
		files = append(files, file)
	}

	return files, nil
} // UpdateSSHHostV2 updates an existing SSH host configuration, searching in all config files
func UpdateSSHHostV2(oldName string, newHost SSHHost) error {
	// Find the host to determine which file it's in
	existingHost, err := FindHostInAllConfigs(oldName)
	if err != nil {
		return err
	}

	// Update the host in its source file
	newHost.SourceFile = existingHost.SourceFile
	return UpdateSSHHostInFile(oldName, newHost, existingHost.SourceFile)
}

// DeleteSSHHostV2 removes an SSH host configuration, searching in all config files
func DeleteSSHHostV2(hostName string) error {
	// Find the host to determine which file it's in
	existingHost, err := FindHostInAllConfigs(hostName)
	if err != nil {
		return err
	}

	// Delete the host from its source file
	return DeleteSSHHostFromFile(hostName, existingHost.SourceFile)
}

// AddSSHHostWithFileSelection adds a new SSH host to a user-specified config file
func AddSSHHostWithFileSelection(host SSHHost, targetFile string) error {
	if targetFile == "" {
		// Use default file if none specified
		return AddSSHHost(host)
	}
	return AddSSHHostToFile(host, targetFile)
}

// GetIncludedConfigFiles returns a list of config files that can be used for adding hosts
func GetIncludedConfigFiles() ([]string, error) {
	allFiles, err := GetAllConfigFiles()
	if err != nil {
		return nil, err
	}

	// Filter out files that don't exist or can't be written to
	var writableFiles []string
	mainConfig, err := GetDefaultSSHConfigPath()
	if err == nil {
		writableFiles = append(writableFiles, mainConfig)
	}

	for _, file := range allFiles {
		if file == mainConfig {
			continue // Already added
		}

		// Check if file exists and is writable
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			writableFiles = append(writableFiles, file)
		}
	}

	return writableFiles, nil
}

// MoveHostToFile moves an SSH host from its current config file to a target config file
func MoveHostToFile(hostName string, targetConfigFile string) error {
	// Find the host in all configs to get its current location and data
	host, err := FindHostInAllConfigs(hostName)
	if err != nil {
		return err
	}

	// Check if the target file is different from the current source file
	if host.SourceFile == targetConfigFile {
		return fmt.Errorf("host '%s' is already in the target config file '%s'", hostName, targetConfigFile)
	}

	// First, add the host to the target config file
	err = AddSSHHostToFile(*host, targetConfigFile)
	if err != nil {
		return fmt.Errorf("failed to add host to target file: %v", err)
	}

	// Then, remove the host from its current source file
	err = DeleteSSHHostFromFile(hostName, host.SourceFile)
	if err != nil {
		// If removal fails, we should try to rollback the addition, but for simplicity
		// we'll just return the error. In a production environment, you might want
		// to implement a proper rollback mechanism.
		return fmt.Errorf("failed to remove host from source file: %v", err)
	}

	return nil
}

// GetConfigFilesExcludingCurrent returns all config files except the one containing the specified host
func GetConfigFilesExcludingCurrent(hostName string, baseConfigFile string) ([]string, error) {
	// Get all config files
	var allFiles []string
	var err error

	if baseConfigFile != "" {
		allFiles, err = GetAllConfigFilesFromBase(baseConfigFile)
	} else {
		allFiles, err = GetAllConfigFiles()
	}

	if err != nil {
		return nil, err
	}

	// Find the host to get its current source file
	host, err := FindHostInAllConfigs(hostName)
	if err != nil {
		return nil, err
	}

	// Filter out the current source file
	var filteredFiles []string
	for _, file := range allFiles {
		if file != host.SourceFile {
			filteredFiles = append(filteredFiles, file)
		}
	}

	return filteredFiles, nil
}

// UpdateMultiHostBlock updates a multi-host block configuration
func UpdateMultiHostBlock(originalHosts, newHosts []string, commonProperties SSHHost, configPath string) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	// Create backup before modification
	if err := backupConfig(configPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Read the current config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	i := 0
	blockFound := false

	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Check for tags comment followed by Host
		if strings.HasPrefix(line, "# Tags:") && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])

			// Check if this is a Host line that contains any of our original hosts
			if strings.HasPrefix(nextLine, "Host ") {
				hostPart := strings.TrimSpace(nextLine[5:]) // Remove "Host "
				foundHostNames := strings.Fields(hostPart)

				// Check if any of our original hosts are in this Host declaration
				hasOriginalHost := false
				for _, origHost := range originalHosts {
					for _, foundHost := range foundHostNames {
						if foundHost == origHost {
							hasOriginalHost = true
							break
						}
					}
					if hasOriginalHost {
						break
					}
				}

				if hasOriginalHost {
					blockFound = true

					// Skip the old block entirely
					i += 2 // Skip tags and Host line
					for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
						i++
					}

					// Skip any trailing empty lines after the host block
					for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
						i++
					}

					// Insert new multi-host configuration
					if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
						newLines = append(newLines, "")
					}

					// Add tags if present
					if len(commonProperties.Tags) > 0 {
						newLines = append(newLines, "# Tags: "+strings.Join(commonProperties.Tags, ", "))
					}

					// Add Host line with new host names
					newLines = append(newLines, "Host "+strings.Join(newHosts, " "))

					// Add common properties
					newLines = append(newLines, "    HostName "+commonProperties.Hostname)
					if commonProperties.User != "" {
						newLines = append(newLines, "    User "+commonProperties.User)
					}
					if commonProperties.Port != "" && commonProperties.Port != "22" {
						newLines = append(newLines, "    Port "+commonProperties.Port)
					}
					if commonProperties.Identity != "" {
						newLines = append(newLines, "    IdentityFile "+formatSSHConfigValue(commonProperties.Identity))
					}
					if commonProperties.ProxyJump != "" {
						newLines = append(newLines, "    ProxyJump "+commonProperties.ProxyJump)
					}
					if commonProperties.ProxyCommand != "" {
						newLines = append(newLines, "    ProxyCommand="+commonProperties.ProxyCommand)
					}
					if commonProperties.RemoteCommand != "" {
						newLines = append(newLines, "    RemoteCommand "+commonProperties.RemoteCommand)
					}
					if commonProperties.RequestTTY != "" {
						newLines = append(newLines, "    RequestTTY "+commonProperties.RequestTTY)
					}

					// Write SSH options
					if commonProperties.Options != "" {
						options := strings.Split(commonProperties.Options, "\n")
						for _, option := range options {
							option = strings.TrimSpace(option)
							if option != "" {
								newLines = append(newLines, "    "+option)
							}
						}
					}

					// Add empty line after the block
					newLines = append(newLines, "")

					continue
				}
			}
		}

		// Check for Host line without tags (same logic)
		if strings.HasPrefix(line, "Host ") {
			hostPart := strings.TrimSpace(line[5:]) // Remove "Host "
			foundHostNames := strings.Fields(hostPart)

			// Check if any of our original hosts are in this Host declaration
			hasOriginalHost := false
			for _, origHost := range originalHosts {
				for _, foundHost := range foundHostNames {
					if foundHost == origHost {
						hasOriginalHost = true
						break
					}
				}
				if hasOriginalHost {
					break
				}
			}

			if hasOriginalHost {
				blockFound = true

				// Skip the old block entirely
				i++ // Skip Host line
				for i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Host ") {
					i++
				}

				// Skip any trailing empty lines after the host block
				for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
					i++
				}

				// Insert new multi-host configuration
				if len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) != "" {
					newLines = append(newLines, "")
				}

				// Add tags if present
				if len(commonProperties.Tags) > 0 {
					newLines = append(newLines, "# Tags: "+strings.Join(commonProperties.Tags, ", "))
				}

				// Add Host line with new host names
				newLines = append(newLines, "Host "+strings.Join(newHosts, " "))

				// Add common properties
				newLines = append(newLines, "    HostName "+commonProperties.Hostname)
				if commonProperties.User != "" {
					newLines = append(newLines, "    User "+commonProperties.User)
				}
				if commonProperties.Port != "" && commonProperties.Port != "22" {
					newLines = append(newLines, "    Port "+commonProperties.Port)
				}
				if commonProperties.Identity != "" {
					newLines = append(newLines, "    IdentityFile "+formatSSHConfigValue(commonProperties.Identity))
				}
				if commonProperties.ProxyJump != "" {
					newLines = append(newLines, "    ProxyJump "+commonProperties.ProxyJump)
				}
				if commonProperties.ProxyCommand != "" {
					newLines = append(newLines, "    ProxyCommand="+commonProperties.ProxyCommand)
				}
				if commonProperties.RemoteCommand != "" {
					newLines = append(newLines, "    RemoteCommand "+commonProperties.RemoteCommand)
				}
				if commonProperties.RequestTTY != "" {
					newLines = append(newLines, "    RequestTTY "+commonProperties.RequestTTY)
				}

				// Write SSH options
				if commonProperties.Options != "" {
					options := strings.Split(commonProperties.Options, "\n")
					for _, option := range options {
						option = strings.TrimSpace(option)
						if option != "" {
							newLines = append(newLines, "    "+option)
						}
					}
				}

				// Add empty line after the block
				newLines = append(newLines, "")

				continue
			}
		}

		// Keep other lines as-is
		newLines = append(newLines, lines[i])
		i++
	}

	if !blockFound {
		return fmt.Errorf("multi-host block not found")
	}

	// Write back to file
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(configPath, []byte(newContent), 0600)
}
