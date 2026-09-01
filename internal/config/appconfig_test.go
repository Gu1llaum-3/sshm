package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultKeyBindings(t *testing.T) {
	kb := GetDefaultKeyBindings()

	// Test default configuration
	if kb.DisableEscQuit {
		t.Error("Default configuration should allow ESC to quit (backward compatibility)")
	}

	// Test default quit keys
	expectedQuitKeys := []string{"q", "ctrl+c"}
	if len(kb.QuitKeys) != len(expectedQuitKeys) {
		t.Errorf("Expected %d quit keys, got %d", len(expectedQuitKeys), len(kb.QuitKeys))
	}

	for i, expected := range expectedQuitKeys {
		if i >= len(kb.QuitKeys) || kb.QuitKeys[i] != expected {
			t.Errorf("Expected quit key %s, got %s", expected, kb.QuitKeys[i])
		}
	}
}

func TestShouldQuitOnKey(t *testing.T) {
	tests := []struct {
		name           string
		keyBindings    KeyBindings
		key            string
		expectedResult bool
	}{
		{
			name: "Default config - ESC should quit",
			keyBindings: KeyBindings{
				QuitKeys:       []string{"q", "ctrl+c"},
				DisableEscQuit: false,
			},
			key:            "esc",
			expectedResult: true,
		},
		{
			name: "Disabled ESC quit - ESC should not quit",
			keyBindings: KeyBindings{
				QuitKeys:       []string{"q", "ctrl+c"},
				DisableEscQuit: true,
			},
			key:            "esc",
			expectedResult: false,
		},
		{
			name: "Q key should quit",
			keyBindings: KeyBindings{
				QuitKeys:       []string{"q", "ctrl+c"},
				DisableEscQuit: true,
			},
			key:            "q",
			expectedResult: true,
		},
		{
			name: "Ctrl+C should quit",
			keyBindings: KeyBindings{
				QuitKeys:       []string{"q", "ctrl+c"},
				DisableEscQuit: true,
			},
			key:            "ctrl+c",
			expectedResult: true,
		},
		{
			name: "Other keys should not quit",
			keyBindings: KeyBindings{
				QuitKeys:       []string{"q", "ctrl+c"},
				DisableEscQuit: true,
			},
			key:            "enter",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.keyBindings.ShouldQuitOnKey(tt.key)
			if result != tt.expectedResult {
				t.Errorf("ShouldQuitOnKey(%q) = %v, expected %v", tt.key, result, tt.expectedResult)
			}
		})
	}
}

func TestAppConfigBasics(t *testing.T) {
	// Test default config creation
	defaultConfig := GetDefaultAppConfig()

	if defaultConfig.KeyBindings.DisableEscQuit {
		t.Error("Default configuration should allow ESC to quit")
	}

	expectedQuitKeys := []string{"q", "ctrl+c"}
	if len(defaultConfig.KeyBindings.QuitKeys) != len(expectedQuitKeys) {
		t.Errorf("Expected %d quit keys, got %d", len(expectedQuitKeys), len(defaultConfig.KeyBindings.QuitKeys))
	}

	// CheckForUpdates should be nil by default
	if defaultConfig.CheckForUpdates != nil {
		t.Error("Default configuration should have CheckForUpdates as nil")
	}

	// IsUpdateCheckEnabled should return true by default
	if !defaultConfig.IsUpdateCheckEnabled() {
		t.Error("IsUpdateCheckEnabled should return true when CheckForUpdates is nil")
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func TestIsUpdateCheckEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *AppConfig
		expected bool
	}{
		{
			name:     "nil AppConfig returns true",
			config:   nil,
			expected: true,
		},
		{
			name:     "CheckForUpdates nil returns true",
			config:   &AppConfig{},
			expected: true,
		},
		{
			name:     "CheckForUpdates true returns true",
			config:   &AppConfig{CheckForUpdates: boolPtr(true)},
			expected: true,
		},
		{
			name:     "CheckForUpdates false returns false",
			config:   &AppConfig{CheckForUpdates: boolPtr(false)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsUpdateCheckEnabled()
			if result != tt.expected {
				t.Errorf("IsUpdateCheckEnabled() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestMergeWithDefaults(t *testing.T) {
	// Test config with missing QuitKeys
	incompleteConfig := AppConfig{
		KeyBindings: KeyBindings{
			DisableEscQuit: true,
			// QuitKeys is missing
		},
	}

	mergedConfig := mergeWithDefaults(incompleteConfig)

	// Should preserve DisableEscQuit
	if !mergedConfig.KeyBindings.DisableEscQuit {
		t.Error("Should preserve DisableEscQuit as true")
	}

	// Should fill in default QuitKeys
	expectedQuitKeys := []string{"q", "ctrl+c"}
	if len(mergedConfig.KeyBindings.QuitKeys) != len(expectedQuitKeys) {
		t.Errorf("Expected %d quit keys, got %d", len(expectedQuitKeys), len(mergedConfig.KeyBindings.QuitKeys))
	}
}

func TestSaveAndLoadAppConfigIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sshm_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a custom config file directly in temp directory
	configPath := filepath.Join(tempDir, "config.json")

	customConfig := AppConfig{
		CheckForUpdates: boolPtr(false),
		KeyBindings: KeyBindings{
			QuitKeys:       []string{"q"},
			DisableEscQuit: true,
		},
	}

	// Save config directly to file
	data, err := json.MarshalIndent(customConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Read and unmarshal config
	readData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var loadedConfig AppConfig
	err = json.Unmarshal(readData, &loadedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify the loaded config matches what we saved
	if !loadedConfig.KeyBindings.DisableEscQuit {
		t.Error("DisableEscQuit should be true")
	}

	if len(loadedConfig.KeyBindings.QuitKeys) != 1 || loadedConfig.KeyBindings.QuitKeys[0] != "q" {
		t.Errorf("Expected quit keys to be ['q'], got %v", loadedConfig.KeyBindings.QuitKeys)
	}

	// Verify CheckForUpdates is correctly persisted and reloaded
	if loadedConfig.CheckForUpdates == nil {
		t.Fatal("CheckForUpdates should not be nil after round-trip")
	}
	if *loadedConfig.CheckForUpdates != false {
		t.Errorf("CheckForUpdates should be false after round-trip, got %v", *loadedConfig.CheckForUpdates)
	}
	if loadedConfig.IsUpdateCheckEnabled() {
		t.Error("IsUpdateCheckEnabled should return false when CheckForUpdates is false")
	}
}

// captureStderr redirects os.Stderr for the duration of fn and returns what was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	original := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = original }()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	return buf.String()
}

func TestSSHCommandDefaultsWhenAbsent(t *testing.T) {
	var stderr string
	var loaded AppConfig

	stderr = captureStderr(t, func() {
		raw := []byte(`{"key_bindings": {"quit_keys": ["q"]}}`)
		if err := json.Unmarshal(raw, &loaded); err != nil {
			t.Fatalf("Failed to unmarshal config: %v", err)
		}
		loaded = mergeWithDefaults(loaded)
	})

	if len(loaded.SSHCommand) != 1 || loaded.SSHCommand[0] != "ssh" {
		t.Errorf("Expected ssh_command to default to [\"ssh\"], got %v", loaded.SSHCommand)
	}
	if stderr != "" {
		t.Errorf("Expected no warning when ssh_command is absent, got: %q", stderr)
	}
}

func TestSSHCommandCustomValue(t *testing.T) {
	raw := []byte(`{"ssh_command": ["kitten", "ssh"]}`)

	var loaded AppConfig
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}
	loaded = mergeWithDefaults(loaded)

	expected := []string{"kitten", "ssh"}
	if len(loaded.SSHCommand) != len(expected) {
		t.Fatalf("Expected ssh_command %v, got %v", expected, loaded.SSHCommand)
	}
	for i, v := range expected {
		if loaded.SSHCommand[i] != v {
			t.Errorf("Expected ssh_command[%d] = %q, got %q", i, v, loaded.SSHCommand[i])
		}
	}
}

func TestSSHCommandInvalidFallsBackWithWarning(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty list", raw: `{"ssh_command": []}`},
		{name: "list with empty string", raw: `{"ssh_command": ["ssh", ""]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var loaded AppConfig

			stderr := captureStderr(t, func() {
				if err := json.Unmarshal([]byte(tt.raw), &loaded); err != nil {
					t.Fatalf("Failed to unmarshal config: %v", err)
				}
				loaded = mergeWithDefaults(loaded)
			})

			if len(loaded.SSHCommand) != 1 || loaded.SSHCommand[0] != "ssh" {
				t.Errorf("Expected fallback to [\"ssh\"], got %v", loaded.SSHCommand)
			}
			if stderr == "" {
				t.Error("Expected a warning to be printed to stderr for invalid ssh_command")
			}
		})
	}
}

func TestBuildSSHArgv(t *testing.T) {
	tests := []struct {
		name          string
		sshCommand    []string
		configFile    string
		forceTTY      bool
		hostName      string
		remoteCommand []string
		expected      []string
	}{
		{
			name:       "default launcher with plain host",
			sshCommand: []string{"ssh"},
			hostName:   "myhost",
			expected:   []string{"ssh", "myhost"},
		},
		{
			name:       "custom launcher with plain host",
			sshCommand: []string{"kitten", "ssh"},
			hostName:   "myhost",
			expected:   []string{"kitten", "ssh", "myhost"},
		},
		{
			name:          "custom launcher with remote command",
			sshCommand:    []string{"kitten", "ssh"},
			hostName:      "myhost",
			remoteCommand: []string{"uptime"},
			expected:      []string{"kitten", "ssh", "myhost", "uptime"},
		},
		{
			name:       "config file and force tty are preserved",
			sshCommand: []string{"ssh"},
			configFile: "/tmp/custom_config",
			forceTTY:   true,
			hostName:   "myhost",
			expected:   []string{"ssh", "-F", "/tmp/custom_config", "-t", "myhost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argv := BuildSSHArgv(tt.sshCommand, tt.configFile, tt.forceTTY, tt.hostName, tt.remoteCommand)

			if len(argv) != len(tt.expected) {
				t.Fatalf("Expected argv %v, got %v", tt.expected, argv)
			}
			for i, v := range tt.expected {
				if argv[i] != v {
					t.Errorf("Expected argv[%d] = %q, got %q", i, v, argv[i])
				}
			}
		})
	}
}
