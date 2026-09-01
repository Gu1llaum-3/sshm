package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KeyBindings represents configurable key bindings for the application
type KeyBindings struct {
	// Quit keys - keys that will quit the application
	QuitKeys []string `json:"quit_keys"`

	// DisableEscQuit - if true, ESC key won't quit the application (useful for vim users)
	DisableEscQuit bool `json:"disable_esc_quit"`
}

// AppConfig represents the main application configuration
type AppConfig struct {
	CheckForUpdates *bool       `json:"check_for_updates,omitempty"`
	KeyBindings     KeyBindings `json:"key_bindings"`

	// SSHCommand is the argv prefix used to launch SSH sessions (e.g. ["ssh"] or
	// ["mosh"]). Defaults to ["ssh"] when absent or invalid.
	SSHCommand []string `json:"ssh_command,omitempty"`
}

// IsUpdateCheckEnabled returns true if the update check is enabled (default: true)
func (c *AppConfig) IsUpdateCheckEnabled() bool {
	if c == nil || c.CheckForUpdates == nil {
		return true
	}
	return *c.CheckForUpdates
}

// GetDefaultKeyBindings returns the default key bindings configuration
func GetDefaultKeyBindings() KeyBindings {
	return KeyBindings{
		QuitKeys:       []string{"q", "ctrl+c"}, // Default keeps current behavior minus ESC
		DisableEscQuit: false,                   // Default to false for backward compatibility
	}
}

// GetDefaultSSHCommand returns the default argv prefix used to launch SSH sessions
func GetDefaultSSHCommand() []string {
	return []string{"ssh"}
}

// GetDefaultAppConfig returns the default application configuration
func GetDefaultAppConfig() AppConfig {
	return AppConfig{
		KeyBindings: GetDefaultKeyBindings(),
		SSHCommand:  GetDefaultSSHCommand(),
	}
}

// GetSSHCommand returns the configured SSH launcher command, defaulting to ["ssh"]
// if unset. This is safe to call even if the config wasn't loaded via LoadAppConfig.
func (c *AppConfig) GetSSHCommand() []string {
	if c == nil || len(c.SSHCommand) == 0 {
		return GetDefaultSSHCommand()
	}
	return c.SSHCommand
}

// GetAppConfigPath returns the path to the application config file
func GetAppConfigPath() (string, error) {
	configDir, err := GetSSHMConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

// LoadAppConfig loads the application configuration from file
// If the file doesn't exist, it returns the default configuration
func LoadAppConfig() (*AppConfig, error) {
	configPath, err := GetAppConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return default config and create the file
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := GetDefaultAppConfig()

		// Create config directory if it doesn't exist
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, err
		}

		// Save default config to file
		if err := SaveAppConfig(&defaultConfig); err != nil {
			// If we can't save, just return the default config without erroring
			// This allows the app to work even if config file can't be created
			return &defaultConfig, nil
		}

		return &defaultConfig, nil
	}

	// Read existing config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Validate and fill in missing fields with defaults
	config = mergeWithDefaults(config)

	return &config, nil
}

// SaveAppConfig saves the application configuration to file
func SaveAppConfig(config *AppConfig) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	configPath, err := GetAppConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// mergeWithDefaults ensures all required fields are set with defaults if missing
func mergeWithDefaults(config AppConfig) AppConfig {
	defaults := GetDefaultAppConfig()

	// If QuitKeys is empty, use defaults
	if len(config.KeyBindings.QuitKeys) == 0 {
		config.KeyBindings.QuitKeys = defaults.KeyBindings.QuitKeys
	}

	config.SSHCommand = validateSSHCommand(config.SSHCommand)

	return config
}

// validateSSHCommand ensures ssh_command is a non-empty list of non-empty strings.
// A missing field (nil) silently defaults to ["ssh"]. A field that was explicitly
// set but is invalid (empty list, or containing blank entries) also falls back to
// ["ssh"], but prints a warning since that likely indicates a config mistake.
func validateSSHCommand(cmd []string) []string {
	if cmd == nil {
		return GetDefaultSSHCommand()
	}

	if len(cmd) == 0 {
		fmt.Fprintln(os.Stderr, "Warning: ssh_command is empty in config, falling back to default [\"ssh\"]")
		return GetDefaultSSHCommand()
	}

	for _, part := range cmd {
		if strings.TrimSpace(part) == "" {
			fmt.Fprintln(os.Stderr, "Warning: ssh_command contains an empty entry in config, falling back to default [\"ssh\"]")
			return GetDefaultSSHCommand()
		}
	}

	return cmd
}

// BuildSSHArgv builds the full exec argv (launcher command followed by SSH flags,
// the host name, and any remote command) used to launch an SSH session.
func BuildSSHArgv(sshCommand []string, configFile string, forceTTY bool, hostName string, remoteCommand []string) []string {
	launcher := sshCommand
	if len(launcher) == 0 {
		launcher = GetDefaultSSHCommand()
	}

	argv := make([]string, 0, len(launcher)+len(remoteCommand)+4)
	argv = append(argv, launcher...)

	if configFile != "" {
		argv = append(argv, "-F", configFile)
	}

	if forceTTY {
		argv = append(argv, "-t")
	}

	argv = append(argv, hostName)
	argv = append(argv, remoteCommand...)

	return argv
}

// ShouldQuitOnKey checks if the given key should trigger quit based on configuration
func (kb *KeyBindings) ShouldQuitOnKey(key string) bool {
	// Special handling for ESC key
	if key == "esc" {
		return !kb.DisableEscQuit
	}

	// Check if key is in the quit keys list
	for _, quitKey := range kb.QuitKeys {
		if quitKey == key {
			return true
		}
	}

	return false
}
