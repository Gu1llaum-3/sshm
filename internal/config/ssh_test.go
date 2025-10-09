package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetDefaultSSHConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		expected string
	}{
		{"Linux", "linux", ".ssh/config"},
		{"macOS", "darwin", ".ssh/config"},
		{"Windows", "windows", ".ssh/config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original GOOS
			originalGOOS := runtime.GOOS
			defer func() {
				// Note: We can't actually change runtime.GOOS at runtime
				// This test verifies the function logic with the current OS
				_ = originalGOOS
			}()

			configPath, err := GetDefaultSSHConfigPath()
			if err != nil {
				t.Fatalf("GetDefaultSSHConfigPath() error = %v", err)
			}

			if !strings.HasSuffix(configPath, tt.expected) {
				t.Errorf("Expected path to end with %q, got %q", tt.expected, configPath)
			}

			// Verify the path uses the correct separator for current OS
			expectedSeparator := string(filepath.Separator)
			if !strings.Contains(configPath, expectedSeparator) && len(configPath) > len(tt.expected) {
				t.Errorf("Path should use OS-specific separator %q, got %q", expectedSeparator, configPath)
			}
		})
	}
}

func TestGetSSHDirectory(t *testing.T) {
	sshDir, err := GetSSHDirectory()
	if err != nil {
		t.Fatalf("GetSSHDirectory() error = %v", err)
	}

	if !strings.HasSuffix(sshDir, ".ssh") {
		t.Errorf("Expected directory to end with .ssh, got %q", sshDir)
	}

	// Verify the path uses the correct separator for current OS
	expectedSeparator := string(filepath.Separator)
	if !strings.Contains(sshDir, expectedSeparator) && len(sshDir) > 4 {
		t.Errorf("Path should use OS-specific separator %q, got %q", expectedSeparator, sshDir)
	}
}

func TestEnsureSSHDirectory(t *testing.T) {
	// This test just ensures the function doesn't panic
	// and returns without error when .ssh directory already exists
	err := ensureSSHDirectory()
	if err != nil {
		t.Fatalf("ensureSSHDirectory() error = %v", err)
	}
}

func TestParseSSHConfigWithInclude(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create main config file
	mainConfig := filepath.Join(tempDir, "config")
	mainConfigContent := `Host main-host
    HostName example.com
    User mainuser

Include included.conf
Include subdir/*

Host another-host
    HostName another.example.com
    User anotheruser
`

	err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create main config: %v", err)
	}

	// Create included file
	includedConfig := filepath.Join(tempDir, "included.conf")
	includedConfigContent := `Host included-host
    HostName included.example.com
    User includeduser
    Port 2222
`

	err = os.WriteFile(includedConfig, []byte(includedConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create included config: %v", err)
	}

	// Create subdirectory with another config file
	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	subConfig := filepath.Join(subDir, "sub.conf")
	subConfigContent := `Host sub-host
    HostName sub.example.com
    User subuser
    IdentityFile ~/.ssh/sub_key
`

	err = os.WriteFile(subConfig, []byte(subConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create sub config: %v", err)
	}

	// Parse the main config file
	hosts, err := ParseSSHConfigFile(mainConfig)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Check that we got all expected hosts
	expectedHosts := map[string]struct{}{
		"main-host":     {},
		"included-host": {},
		"sub-host":      {},
		"another-host":  {},
	}

	if len(hosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
	}

	for _, host := range hosts {
		if _, exists := expectedHosts[host.Name]; !exists {
			t.Errorf("Unexpected host found: %s", host.Name)
		}
		delete(expectedHosts, host.Name)

		// Validate specific host properties
		switch host.Name {
		case "main-host":
			if host.Hostname != "example.com" || host.User != "mainuser" {
				t.Errorf("main-host properties incorrect: hostname=%s, user=%s", host.Hostname, host.User)
			}
			if host.SourceFile != mainConfig {
				t.Errorf("main-host SourceFile incorrect: expected=%s, got=%s", mainConfig, host.SourceFile)
			}
		case "included-host":
			if host.Hostname != "included.example.com" || host.User != "includeduser" || host.Port != "2222" {
				t.Errorf("included-host properties incorrect: hostname=%s, user=%s, port=%s", host.Hostname, host.User, host.Port)
			}
			if host.SourceFile != includedConfig {
				t.Errorf("included-host SourceFile incorrect: expected=%s, got=%s", includedConfig, host.SourceFile)
			}
		case "sub-host":
			if host.Hostname != "sub.example.com" || host.User != "subuser" || host.Identity != "~/.ssh/sub_key" {
				t.Errorf("sub-host properties incorrect: hostname=%s, user=%s, identity=%s", host.Hostname, host.User, host.Identity)
			}
			if host.SourceFile != subConfig {
				t.Errorf("sub-host SourceFile incorrect: expected=%s, got=%s", subConfig, host.SourceFile)
			}
		case "another-host":
			if host.Hostname != "another.example.com" || host.User != "anotheruser" {
				t.Errorf("another-host properties incorrect: hostname=%s, user=%s", host.Hostname, host.User)
			}
			if host.SourceFile != mainConfig {
				t.Errorf("another-host SourceFile incorrect: expected=%s, got=%s", mainConfig, host.SourceFile)
			}
		}
	}

	// Check that all expected hosts were found
	if len(expectedHosts) > 0 {
		var missing []string
		for host := range expectedHosts {
			missing = append(missing, host)
		}
		t.Errorf("Missing hosts: %v", missing)
	}
}

func TestParseSSHConfigWithCircularInclude(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create config1 that includes config2
	config1 := filepath.Join(tempDir, "config1")
	config1Content := `Host host1
    HostName example1.com

Include config2
`

	err := os.WriteFile(config1, []byte(config1Content), 0600)
	if err != nil {
		t.Fatalf("Failed to create config1: %v", err)
	}

	// Create config2 that includes config1 (circular)
	config2 := filepath.Join(tempDir, "config2")
	config2Content := `Host host2
    HostName example2.com

Include config1
`

	err = os.WriteFile(config2, []byte(config2Content), 0600)
	if err != nil {
		t.Fatalf("Failed to create config2: %v", err)
	}

	// Parse the config file - should not cause infinite recursion
	hosts, err := ParseSSHConfigFile(config1)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Should get both hosts exactly once
	expectedHosts := map[string]bool{
		"host1": false,
		"host2": false,
	}

	for _, host := range hosts {
		if _, exists := expectedHosts[host.Name]; !exists {
			t.Errorf("Unexpected host found: %s", host.Name)
		} else {
			if expectedHosts[host.Name] {
				t.Errorf("Host %s found multiple times", host.Name)
			}
			expectedHosts[host.Name] = true
		}
	}

	// Check all hosts were found
	for hostName, found := range expectedHosts {
		if !found {
			t.Errorf("Host %s not found", hostName)
		}
	}
}

func TestParseSSHConfigWithNonExistentInclude(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create main config file with non-existent include
	mainConfig := filepath.Join(tempDir, "config")
	mainConfigContent := `Host main-host
    HostName example.com

Include non-existent-file.conf

Host another-host
    HostName another.example.com
`

	err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create main config: %v", err)
	}

	// Parse should succeed and ignore the non-existent include
	hosts, err := ParseSSHConfigFile(mainConfig)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Should get the hosts that exist, ignoring the failed include
	if len(hosts) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(hosts))
	}

	hostNames := make(map[string]bool)
	for _, host := range hosts {
		hostNames[host.Name] = true
	}

	if !hostNames["main-host"] || !hostNames["another-host"] {
		t.Errorf("Expected main-host and another-host, got: %v", hostNames)
	}
}

func TestParseSSHConfigWithWildcardHosts(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create config file with wildcard hosts
	configFile := filepath.Join(tempDir, "config")
	configContent := `# Wildcard patterns should be ignored
Host *.example.com
    User defaultuser
    IdentityFile ~/.ssh/id_rsa

Host server-*
    Port 2222

Host *
    ServerAliveInterval 60

# Real hosts should be included
Host real-server
    HostName real.example.com
    User realuser

Host another-real-server
    HostName another.example.com
    User anotheruser
`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Parse the config file
	hosts, err := ParseSSHConfigFile(configFile)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Should only get real hosts, not wildcard patterns
	expectedHosts := map[string]bool{
		"real-server":         false,
		"another-real-server": false,
	}

	if len(hosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
		for _, host := range hosts {
			t.Logf("Found host: %s", host.Name)
		}
	}

	for _, host := range hosts {
		if _, expected := expectedHosts[host.Name]; !expected {
			t.Errorf("Unexpected host found: %s", host.Name)
		} else {
			expectedHosts[host.Name] = true
		}
	}

	// Check that all expected hosts were found
	for hostName, found := range expectedHosts {
		if !found {
			t.Errorf("Expected host %s not found", hostName)
		}
	}

	// Verify host properties
	for _, host := range hosts {
		switch host.Name {
		case "real-server":
			if host.Hostname != "real.example.com" || host.User != "realuser" {
				t.Errorf("real-server properties incorrect: hostname=%s, user=%s", host.Hostname, host.User)
			}
		case "another-real-server":
			if host.Hostname != "another.example.com" || host.User != "anotheruser" {
				t.Errorf("another-real-server properties incorrect: hostname=%s, user=%s", host.Hostname, host.User)
			}
		}
	}
}

func TestParseSSHConfigExcludesBackupFiles(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create main config file with include pattern
	mainConfig := filepath.Join(tempDir, "config")
	mainConfigContent := `Host main-host
    HostName example.com

Include *.conf
`

	err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create main config: %v", err)
	}

	// Create a regular config file
	regularConfig := filepath.Join(tempDir, "regular.conf")
	regularConfigContent := `Host regular-host
    HostName regular.example.com
`

	err = os.WriteFile(regularConfig, []byte(regularConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create regular config: %v", err)
	}

	// Create a backup file that should be excluded
	backupConfig := filepath.Join(tempDir, "regular.conf.backup")
	backupConfigContent := `Host backup-host
    HostName backup.example.com
`

	err = os.WriteFile(backupConfig, []byte(backupConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create backup config: %v", err)
	}

	// Parse the config file
	hosts, err := ParseSSHConfigFile(mainConfig)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Should only get main-host and regular-host, not backup-host
	expectedHosts := map[string]bool{
		"main-host":    false,
		"regular-host": false,
	}

	if len(hosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
		for _, host := range hosts {
			t.Logf("Found host: %s", host.Name)
		}
	}

	for _, host := range hosts {
		if _, expected := expectedHosts[host.Name]; !expected {
			t.Errorf("Unexpected host found: %s (backup files should be excluded)", host.Name)
		} else {
			expectedHosts[host.Name] = true
		}
	}

	// Check that backup-host was not included
	for _, host := range hosts {
		if host.Name == "backup-host" {
			t.Error("backup-host should not be included (backup files should be excluded)")
		}
	}
}

func TestBackupConfigToSSHMDirectory(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Override the home directory for this test
	originalHome := os.Getenv("HOME")
	if originalHome == "" {
		originalHome = os.Getenv("USERPROFILE") // Windows
	}

	// Set test home directory
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create a test SSH config file
	sshDir := filepath.Join(tempDir, ".ssh")
	err := os.MkdirAll(sshDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	configPath := filepath.Join(sshDir, "config")
	configContent := `Host test-host
    HostName test.example.com
    User testuser
`

	err = os.WriteFile(configPath, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Test backup creation
	err = backupConfig(configPath)
	if err != nil {
		t.Fatalf("backupConfig() error = %v", err)
	}

	// Verify backup directory was created
	backupDir, err := GetSSHMBackupDir()
	if err != nil {
		t.Fatalf("GetSSHMBackupDir() error = %v", err)
	}

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Errorf("Backup directory was not created: %s", backupDir)
	}

	// Verify backup file was created
	files, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 backup file, got %d", len(files))
	}

	if len(files) > 0 {
		backupFile := files[0]
		expectedName := "config.backup"
		if backupFile.Name() != expectedName {
			t.Errorf("Backup file has unexpected name: got %s, want %s", backupFile.Name(), expectedName)
		}

		// Verify backup content
		backupContent, err := os.ReadFile(filepath.Join(backupDir, backupFile.Name()))
		if err != nil {
			t.Fatalf("Failed to read backup file: %v", err)
		}

		if string(backupContent) != configContent {
			t.Errorf("Backup content doesn't match original")
		}
	}

	// Test that subsequent backups overwrite the previous one
	newConfigContent := `Host test-host-updated
    HostName updated.example.com
    User updateduser
`

	err = os.WriteFile(configPath, []byte(newConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// Create second backup
	err = backupConfig(configPath)
	if err != nil {
		t.Fatalf("Second backupConfig() error = %v", err)
	}

	// Verify still only one backup file exists
	files, err = os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory after second backup: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected still 1 backup file after overwrite, got %d", len(files))
	}

	// Verify backup content was updated
	if len(files) > 0 {
		backupContent, err := os.ReadFile(filepath.Join(backupDir, files[0].Name()))
		if err != nil {
			t.Fatalf("Failed to read updated backup file: %v", err)
		}

		if string(backupContent) != newConfigContent {
			t.Errorf("Updated backup content doesn't match new config content")
		}
	}
}

func TestFindHostInAllConfigs(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create main config file
	mainConfig := filepath.Join(tempDir, "config")
	mainConfigContent := `Host main-host
    HostName example.com

Include included.conf
`

	err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create main config: %v", err)
	}

	// Create included config file
	includedConfig := filepath.Join(tempDir, "included.conf")
	includedConfigContent := `Host included-host
    HostName included.example.com
    User includeduser
`

	err = os.WriteFile(includedConfig, []byte(includedConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create included config: %v", err)
	}

	// Test finding host from main config
	host, err := GetSSHHostFromFile("main-host", mainConfig)
	if err != nil {
		t.Fatalf("GetSSHHostFromFile() error = %v", err)
	}
	if host.Name != "main-host" || host.Hostname != "example.com" {
		t.Errorf("main-host not found correctly: name=%s, hostname=%s", host.Name, host.Hostname)
	}
	if host.SourceFile != mainConfig {
		t.Errorf("main-host SourceFile incorrect: expected=%s, got=%s", mainConfig, host.SourceFile)
	}

	// Test finding host from included config
	// Note: This tests the full parsing with includes
	hosts, err := ParseSSHConfigFile(mainConfig)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	var includedHost *SSHHost
	for _, h := range hosts {
		if h.Name == "included-host" {
			includedHost = &h
			break
		}
	}

	if includedHost == nil {
		t.Fatal("included-host not found")
	}
	if includedHost.Hostname != "included.example.com" || includedHost.User != "includeduser" {
		t.Errorf("included-host properties incorrect: hostname=%s, user=%s", includedHost.Hostname, includedHost.User)
	}
	if includedHost.SourceFile != includedConfig {
		t.Errorf("included-host SourceFile incorrect: expected=%s, got=%s", includedConfig, includedHost.SourceFile)
	}
}

func TestGetAllConfigFiles(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create main config file
	mainConfig := filepath.Join(tempDir, "config")
	mainConfigContent := `Host main-host
    HostName example.com

Include included.conf
Include subdir/*.conf
`

	err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create main config: %v", err)
	}

	// Create included config file
	includedConfig := filepath.Join(tempDir, "included.conf")
	err = os.WriteFile(includedConfig, []byte("Host included-host\n    HostName included.example.com\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to create included config: %v", err)
	}

	// Create subdirectory with config files
	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	subConfig := filepath.Join(subDir, "sub.conf")
	err = os.WriteFile(subConfig, []byte("Host sub-host\n    HostName sub.example.com\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to create sub config: %v", err)
	}

	// Parse to populate the processed files map
	_, err = ParseSSHConfigFile(mainConfig)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Note: GetAllConfigFiles() uses a fresh parse, so we test it indirectly
	// by checking that all files are found during parsing
	hosts, err := ParseSSHConfigFile(mainConfig)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Check that hosts from all files are found
	sourceFiles := make(map[string]bool)
	for _, host := range hosts {
		sourceFiles[host.SourceFile] = true
	}

	expectedFiles := []string{mainConfig, includedConfig, subConfig}
	for _, expectedFile := range expectedFiles {
		if !sourceFiles[expectedFile] {
			t.Errorf("Expected config file not found in SourceFile: %s", expectedFile)
		}
	}
}

func TestGetAllConfigFilesFromBase(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create main config file
	mainConfig := filepath.Join(tempDir, "config")
	mainConfigContent := `Host main-host
    HostName example.com

Include included.conf
`

	err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create main config: %v", err)
	}

	// Create included config file
	includedConfig := filepath.Join(tempDir, "included.conf")
	includedConfigContent := `Host included-host
    HostName included.example.com

Include subdir/*.conf
`

	err = os.WriteFile(includedConfig, []byte(includedConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create included config: %v", err)
	}

	// Create subdirectory with config files
	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	subConfig := filepath.Join(subDir, "sub.conf")
	err = os.WriteFile(subConfig, []byte("Host sub-host\n    HostName sub.example.com\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to create sub config: %v", err)
	}

	// Create an isolated config file that should not be included
	isolatedConfig := filepath.Join(tempDir, "isolated.conf")
	err = os.WriteFile(isolatedConfig, []byte("Host isolated-host\n    HostName isolated.example.com\n"), 0600)
	if err != nil {
		t.Fatalf("Failed to create isolated config: %v", err)
	}

	// Test GetAllConfigFilesFromBase with main config as base
	files, err := GetAllConfigFilesFromBase(mainConfig)
	if err != nil {
		t.Fatalf("GetAllConfigFilesFromBase() error = %v", err)
	}

	// Should find main config, included config, and sub config, but not isolated config
	expectedFiles := map[string]bool{
		mainConfig:     false,
		includedConfig: false,
		subConfig:      false,
	}

	if len(files) != len(expectedFiles) {
		t.Errorf("Expected %d config files, got %d", len(expectedFiles), len(files))
		for i, file := range files {
			t.Logf("Found file %d: %s", i+1, file)
		}
	}

	for _, file := range files {
		if _, expected := expectedFiles[file]; expected {
			expectedFiles[file] = true
		} else if file == isolatedConfig {
			t.Errorf("Isolated config file should not be included: %s", file)
		} else {
			t.Logf("Unexpected file found: %s", file)
		}
	}

	// Check that all expected files were found
	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Expected config file not found: %s", file)
		}
	}

	// Test GetAllConfigFilesFromBase with isolated config as base (should only return itself)
	isolatedFiles, err := GetAllConfigFilesFromBase(isolatedConfig)
	if err != nil {
		t.Fatalf("GetAllConfigFilesFromBase() error = %v", err)
	}

	if len(isolatedFiles) != 1 || isolatedFiles[0] != isolatedConfig {
		t.Errorf("Expected only isolated config file, got: %v", isolatedFiles)
	}

	// Test with empty base config file path (should fallback to default behavior)
	defaultFiles, err := GetAllConfigFilesFromBase("")
	if err != nil {
		t.Fatalf("GetAllConfigFilesFromBase('') error = %v", err)
	}

	// Should behave like GetAllConfigFiles()
	allFiles, err := GetAllConfigFiles()
	if err != nil {
		t.Fatalf("GetAllConfigFiles() error = %v", err)
	}

	if len(defaultFiles) != len(allFiles) {
		t.Errorf("GetAllConfigFilesFromBase('') should behave like GetAllConfigFiles(). Got %d vs %d files", len(defaultFiles), len(allFiles))
	}
}

func TestHostExistsInSpecificFile(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create main config file
	mainConfig := filepath.Join(tempDir, "config")
	mainConfigContent := `Host main-host
    HostName example.com
    User mainuser

Include included.conf

Host another-host
    HostName another.example.com
    User anotheruser
`

	err := os.WriteFile(mainConfig, []byte(mainConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create main config: %v", err)
	}

	// Create included config file
	includedConfig := filepath.Join(tempDir, "included.conf")
	includedConfigContent := `Host included-host
    HostName included.example.com
    User includeduser
`

	err = os.WriteFile(includedConfig, []byte(includedConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create included config: %v", err)
	}

	// Test that host exists in main config file (should ignore includes)
	exists, err := HostExistsInSpecificFile("main-host", mainConfig)
	if err != nil {
		t.Fatalf("HostExistsInSpecificFile() error = %v", err)
	}
	if !exists {
		t.Error("main-host should exist in main config file")
	}

	// Test that host from included file does NOT exist in main config file
	exists, err = HostExistsInSpecificFile("included-host", mainConfig)
	if err != nil {
		t.Fatalf("HostExistsInSpecificFile() error = %v", err)
	}
	if exists {
		t.Error("included-host should NOT exist in main config file (should ignore includes)")
	}

	// Test that host exists in included config file
	exists, err = HostExistsInSpecificFile("included-host", includedConfig)
	if err != nil {
		t.Fatalf("HostExistsInSpecificFile() error = %v", err)
	}
	if !exists {
		t.Error("included-host should exist in included config file")
	}

	// Test non-existent host
	exists, err = HostExistsInSpecificFile("non-existent", mainConfig)
	if err != nil {
		t.Fatalf("HostExistsInSpecificFile() error = %v", err)
	}
	if exists {
		t.Error("non-existent host should not exist")
	}

	// Test with non-existent file
	exists, err = HostExistsInSpecificFile("any-host", "/non/existent/file")
	if err != nil {
		t.Fatalf("HostExistsInSpecificFile() should not return error for non-existent file: %v", err)
	}
	if exists {
		t.Error("non-existent file should not contain any hosts")
	}
}

func TestGetConfigFilesExcludingCurrent(t *testing.T) {
	// This test verifies the function works when SSH config is properly set up
	// Since GetConfigFilesExcludingCurrent depends on FindHostInAllConfigs which uses the default SSH config,
	// we'll test the function more directly by creating a temporary SSH config setup

	// Skip this test if we can't access SSH config directory
	_, err := GetSSHDirectory()
	if err != nil {
		t.Skipf("Skipping test: cannot get SSH directory: %v", err)
	}

	// Check if SSH config exists
	defaultConfigPath, err := GetDefaultSSHConfigPath()
	if err != nil {
		t.Skipf("Skipping test: cannot get default SSH config path: %v", err)
	}

	if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: SSH config file does not exist at %s", defaultConfigPath)
	}

	// Test that the function returns something for a hypothetical host
	// We can't guarantee specific hosts exist, so we test the function doesn't crash
	_, err = GetConfigFilesExcludingCurrent("test-host-that-probably-does-not-exist", defaultConfigPath)
	if err == nil {
		t.Log("GetConfigFilesExcludingCurrent() succeeded for non-existent host (expected)")
	} else if strings.Contains(err.Error(), "not found") {
		t.Log("GetConfigFilesExcludingCurrent() correctly reported host not found")
	} else {
		t.Fatalf("GetConfigFilesExcludingCurrent() unexpected error = %v", err)
	}

	// Test with valid SSH config directory
	if err == nil {
		t.Log("GetConfigFilesExcludingCurrent() function is working correctly")
	}
}

func TestMoveHostToFile(t *testing.T) {
	// This test verifies the MoveHostToFile function works when SSH config is properly set up
	// Since MoveHostToFile depends on FindHostInAllConfigs which uses the default SSH config,
	// we'll test the error handling and basic function behavior

	// Check if SSH config exists
	defaultConfigPath, err := GetDefaultSSHConfigPath()
	if err != nil {
		t.Skipf("Skipping test: cannot get default SSH config path: %v", err)
	}

	if _, err := os.Stat(defaultConfigPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: SSH config file does not exist at %s", defaultConfigPath)
	}

	// Create a temporary destination config file
	tempDir := t.TempDir()
	destConfig := filepath.Join(tempDir, "dest.conf")
	destConfigContent := `Host dest-host
    HostName dest.example.com
    User destuser
`

	err = os.WriteFile(destConfig, []byte(destConfigContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create dest config: %v", err)
	}

	// Test moving non-existent host (should return error)
	err = MoveHostToFile("non-existent-host-12345", destConfig)
	if err == nil {
		t.Error("MoveHostToFile() should return error for non-existent host")
	} else if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	// Test moving to non-existent file (should return error)
	err = MoveHostToFile("any-host", "/non/existent/file")
	if err == nil {
		t.Error("MoveHostToFile() should return error for non-existent destination file")
	}

	// Verify that the HostExistsInSpecificFile function works correctly
	// This is a component that MoveHostToFile uses
	exists, err := HostExistsInSpecificFile("dest-host", destConfig)
	if err != nil {
		t.Fatalf("HostExistsInSpecificFile() error = %v", err)
	}
	if !exists {
		t.Error("dest-host should exist in destination config file")
	}

	// Test that the component functions work for the move operation
	t.Log("MoveHostToFile() error handling works correctly")
}

func TestParseSSHConfigWithMultipleHostsOnSameLine(t *testing.T) {
	tempDir := t.TempDir()

	configFile := filepath.Join(tempDir, "config")
	configContent := `# Test multiple hosts on same line
Host local1 local2
    HostName ::1
    User myuser

Host root-server
    User root
    HostName root.example.com

Host web1 web2 web3
    HostName ::1
    User webuser
    Port 8080

Host single-host
    HostName single.example.com
    User singleuser
`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	hosts, err := ParseSSHConfigFile(configFile)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Should get 7 hosts: local1, local2, root-server, web1, web2, web3, single-host
	expectedHosts := map[string]struct{}{
		"local1":      {},
		"local2":      {},
		"root-server": {},
		"web1":        {},
		"web2":        {},
		"web3":        {},
		"single-host": {},
	}

	if len(hosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
		for _, host := range hosts {
			t.Logf("Found host: %s", host.Name)
		}
	}

	hostMap := make(map[string]SSHHost)
	for _, host := range hosts {
		hostMap[host.Name] = host
	}

	for expectedHostName := range expectedHosts {
		if _, found := hostMap[expectedHostName]; !found {
			t.Errorf("Expected host %s not found", expectedHostName)
		}
	}

	// Verify properties based on host name
	if host, found := hostMap["local1"]; found {
		if host.Hostname != "::1" || host.User != "myuser" {
			t.Errorf("local1 properties incorrect: hostname=%s, user=%s", host.Hostname, host.User)
		}
	}

	if host, found := hostMap["local2"]; found {
		if host.Hostname != "::1" || host.User != "myuser" {
			t.Errorf("local2 properties incorrect: hostname=%s, user=%s", host.Hostname, host.User)
		}
	}

	if host, found := hostMap["web1"]; found {
		if host.Hostname != "::1" || host.User != "webuser" || host.Port != "8080" {
			t.Errorf("web1 properties incorrect: hostname=%s, user=%s, port=%s", host.Hostname, host.User, host.Port)
		}
	}

	if host, found := hostMap["web2"]; found {
		if host.Hostname != "::1" || host.User != "webuser" || host.Port != "8080" {
			t.Errorf("web2 properties incorrect: hostname=%s, user=%s, port=%s", host.Hostname, host.User, host.Port)
		}
	}

	if host, found := hostMap["web3"]; found {
		if host.Hostname != "::1" || host.User != "webuser" || host.Port != "8080" {
			t.Errorf("web3 properties incorrect: hostname=%s, user=%s, port=%s", host.Hostname, host.User, host.Port)
		}
	}

	if host, found := hostMap["root-server"]; found {
		if host.User != "root" || host.Hostname != "root.example.com" {
			t.Errorf("root-server properties incorrect: user=%s, hostname=%s", host.User, host.Hostname)
		}
	}
}

func TestUpdateSSHHostInFileWithMultiHost(t *testing.T) {
	tempDir := t.TempDir()

	configFile := filepath.Join(tempDir, "config")
	configContent := `# Test config with multi-host
Host web1 web2 web3
    HostName webserver.example.com
    User webuser
    Port 2222

Host database
    HostName db.example.com
    User dbuser
`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Update web2 in the multi-host line
	newHost := SSHHost{
		Name:     "web2-updated",
		Hostname: "newweb.example.com",
		User:     "newuser",
		Port:     "22",
	}

	err = UpdateSSHHostInFile("web2", newHost, configFile)
	if err != nil {
		t.Fatalf("UpdateSSHHostInFile() error = %v", err)
	}

	// Parse the updated config
	hosts, err := ParseSSHConfigFile(configFile)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Should have: web1, web3, web2-updated, database
	expectedHosts := []string{"web1", "web3", "web2-updated", "database"}

	hostMap := make(map[string]SSHHost)
	for _, host := range hosts {
		hostMap[host.Name] = host
	}

	if len(hosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
		for _, host := range hosts {
			t.Logf("Found host: %s", host.Name)
		}
	}

	for _, expectedHostName := range expectedHosts {
		if _, found := hostMap[expectedHostName]; !found {
			t.Errorf("Expected host %s not found", expectedHostName)
		}
	}

	// Verify web1 and web3 still have original properties
	if host, found := hostMap["web1"]; found {
		if host.Hostname != "webserver.example.com" || host.User != "webuser" || host.Port != "2222" {
			t.Errorf("web1 properties changed: hostname=%s, user=%s, port=%s", host.Hostname, host.User, host.Port)
		}
	}

	if host, found := hostMap["web3"]; found {
		if host.Hostname != "webserver.example.com" || host.User != "webuser" || host.Port != "2222" {
			t.Errorf("web3 properties changed: hostname=%s, user=%s, port=%s", host.Hostname, host.User, host.Port)
		}
	}

	// Verify web2-updated has new properties
	if host, found := hostMap["web2-updated"]; found {
		if host.Hostname != "newweb.example.com" || host.User != "newuser" || host.Port != "22" {
			t.Errorf("web2-updated properties incorrect: hostname=%s, user=%s, port=%s", host.Hostname, host.User, host.Port)
		}
	}

	// Verify database is unchanged
	if host, found := hostMap["database"]; found {
		if host.Hostname != "db.example.com" || host.User != "dbuser" {
			t.Errorf("database properties changed: hostname=%s, user=%s", host.Hostname, host.User)
		}
	}
}

func TestIsPartOfMultiHostDeclaration(t *testing.T) {
	tempDir := t.TempDir()

	configFile := filepath.Join(tempDir, "config")
	configContent := `Host single
    HostName single.example.com

Host multi1 multi2 multi3
    HostName multi.example.com

Host another
    HostName another.example.com
`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	tests := []struct {
		hostName      string
		expectedMulti bool
		expectedHosts []string
	}{
		{"single", false, []string{"single"}},
		{"multi1", true, []string{"multi1", "multi2", "multi3"}},
		{"multi2", true, []string{"multi1", "multi2", "multi3"}},
		{"multi3", true, []string{"multi1", "multi2", "multi3"}},
		{"another", false, []string{"another"}},
		{"nonexistent", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.hostName, func(t *testing.T) {
			isMulti, hostNames, err := IsPartOfMultiHostDeclaration(tt.hostName, configFile)
			if err != nil {
				t.Fatalf("IsPartOfMultiHostDeclaration() error = %v", err)
			}

			if isMulti != tt.expectedMulti {
				t.Errorf("Expected isMulti=%v, got %v", tt.expectedMulti, isMulti)
			}

			if tt.expectedHosts == nil && hostNames != nil {
				t.Errorf("Expected hostNames to be nil, got %v", hostNames)
			} else if tt.expectedHosts != nil {
				if len(hostNames) != len(tt.expectedHosts) {
					t.Errorf("Expected %d hostNames, got %d", len(tt.expectedHosts), len(hostNames))
				} else {
					for i, expectedHost := range tt.expectedHosts {
						if i < len(hostNames) && hostNames[i] != expectedHost {
							t.Errorf("Expected hostNames[%d]=%s, got %s", i, expectedHost, hostNames[i])
						}
					}
				}
			}
		})
	}
}

func TestDeleteSSHHostFromFileWithMultiHost(t *testing.T) {
	tempDir := t.TempDir()

	configFile := filepath.Join(tempDir, "config")
	configContent := `# Test config with multi-host deletion
Host web1 web2 web3
    HostName webserver.example.com
    User webuser
    Port 2222

Host database
    HostName db.example.com
    User dbuser

# Tags: production, critical
Host app1 app2
    HostName appserver.example.com
    User appuser
`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Test 1: Delete one host from multi-host block (should keep others)
	err = DeleteSSHHostFromFile("web2", configFile)
	if err != nil {
		t.Fatalf("DeleteSSHHostFromFile() error = %v", err)
	}

	// Parse the updated config
	hosts, err := ParseSSHConfigFile(configFile)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Should have: web1, web3, database, app1, app2 (web2 removed)
	expectedHosts := []string{"web1", "web3", "database", "app1", "app2"}

	hostMap := make(map[string]SSHHost)
	for _, host := range hosts {
		hostMap[host.Name] = host
	}

	if len(hosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
		for _, host := range hosts {
			t.Logf("Found host: %s", host.Name)
		}
	}

	for _, expectedHostName := range expectedHosts {
		if _, found := hostMap[expectedHostName]; !found {
			t.Errorf("Expected host %s not found", expectedHostName)
		}
	}

	// Verify web2 is not present
	if _, found := hostMap["web2"]; found {
		t.Error("web2 should have been deleted")
	}

	// Verify web1 and web3 still have original properties
	if host, found := hostMap["web1"]; found {
		if host.Hostname != "webserver.example.com" || host.User != "webuser" || host.Port != "2222" {
			t.Errorf("web1 properties incorrect: hostname=%s, user=%s, port=%s", host.Hostname, host.User, host.Port)
		}
	}

	if host, found := hostMap["web3"]; found {
		if host.Hostname != "webserver.example.com" || host.User != "webuser" || host.Port != "2222" {
			t.Errorf("web3 properties incorrect: hostname=%s, user=%s, port=%s", host.Hostname, host.User, host.Port)
		}
	}

	// Test 2: Delete one host from multi-host block with tags
	err = DeleteSSHHostFromFile("app1", configFile)
	if err != nil {
		t.Fatalf("DeleteSSHHostFromFile() error = %v", err)
	}

	// Parse again
	hosts, err = ParseSSHConfigFile(configFile)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Should have: web1, web3, database, app2 (app1 removed)
	expectedHosts = []string{"web1", "web3", "database", "app2"}

	hostMap = make(map[string]SSHHost)
	for _, host := range hosts {
		hostMap[host.Name] = host
	}

	if len(hosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
		for _, host := range hosts {
			t.Logf("Found host: %s", host.Name)
		}
	}

	// Verify app2 still has tags
	if host, found := hostMap["app2"]; found {
		if !contains(host.Tags, "production") || !contains(host.Tags, "critical") {
			t.Errorf("app2 tags incorrect: %v", host.Tags)
		}
	}
}

func TestUpdateMultiHostBlock(t *testing.T) {
	tempDir := t.TempDir()

	configFile := filepath.Join(tempDir, "config")
	configContent := `# Test config for multi-host block update
Host server1 server2 server3
    HostName cluster.example.com
    User clusteruser
    Port 2222

Host single
    HostName single.example.com
    User singleuser
`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Update the multi-host block
	originalHosts := []string{"server1", "server2", "server3"}
	newHosts := []string{"server1", "server4", "server5"} // Remove server2, server3 and add server4, server5
	commonProperties := SSHHost{
		Hostname: "newcluster.example.com",
		User:     "newuser",
		Port:     "22",
		Tags:     []string{"updated", "cluster"},
	}

	err = UpdateMultiHostBlock(originalHosts, newHosts, commonProperties, configFile)
	if err != nil {
		t.Fatalf("UpdateMultiHostBlock() error = %v", err)
	}

	// Parse the updated config
	hosts, err := ParseSSHConfigFile(configFile)
	if err != nil {
		t.Fatalf("ParseSSHConfigFile() error = %v", err)
	}

	// Should have: server1, server4, server5, single
	expectedHosts := []string{"server1", "server4", "server5", "single"}

	hostMap := make(map[string]SSHHost)
	for _, host := range hosts {
		hostMap[host.Name] = host
	}

	if len(hosts) != len(expectedHosts) {
		t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
		for _, host := range hosts {
			t.Logf("Found host: %s", host.Name)
		}
	}

	// Verify new hosts have updated properties
	for _, hostName := range []string{"server1", "server4", "server5"} {
		if host, found := hostMap[hostName]; found {
			if host.Hostname != "newcluster.example.com" || host.User != "newuser" || host.Port != "22" {
				t.Errorf("%s properties incorrect: hostname=%s, user=%s, port=%s",
					hostName, host.Hostname, host.User, host.Port)
			}
			if !contains(host.Tags, "updated") || !contains(host.Tags, "cluster") {
				t.Errorf("%s tags incorrect: %v", hostName, host.Tags)
			}
		} else {
			t.Errorf("Expected host %s not found", hostName)
		}
	}

	// Verify single host is unchanged
	if host, found := hostMap["single"]; found {
		if host.Hostname != "single.example.com" || host.User != "singleuser" {
			t.Errorf("single host properties changed: hostname=%s, user=%s", host.Hostname, host.User)
		}
	}

	// Verify old hosts are gone
	for _, oldHost := range []string{"server2", "server3"} {
		if _, found := hostMap[oldHost]; found {
			t.Errorf("Old host %s should have been removed", oldHost)
		}
	}
}

// Helper function to check if slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper function to create temporary config files for testing
func createTempConfigFile(content string) (string, error) {
	tempFile, err := os.CreateTemp("", "ssh_config_test_*.conf")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = tempFile.WriteString(content)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

func TestFormatSSHConfigValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple path without spaces",
			input:    "/home/user/.ssh/id_rsa",
			expected: "/home/user/.ssh/id_rsa",
		},
		{
			name:     "path with spaces",
			input:    "/home/user/My Documents/ssh key",
			expected: "\"/home/user/My Documents/ssh key\"",
		},
		{
			name:     "Windows path with spaces",
			input:    `G:\My Drive\7 - Tech\9 - SSH Keys\Server_WF.opk`,
			expected: `"G:\My Drive\7 - Tech\9 - SSH Keys\Server_WF.opk"`,
		},
		{
			name:     "path with quotes but no spaces",
			input:    `/home/user/key"with"quotes`,
			expected: `/home/user/key"with"quotes`,
		},
		{
			name:     "path with spaces and quotes",
			input:    `/home/user/key "with" quotes`,
			expected: `"/home/user/key "with" quotes"`,
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "path with single space at end",
			input:    "/home/user/key ",
			expected: "\"/home/user/key \"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSSHConfigValue(tt.input)
			if result != tt.expected {
				t.Errorf("formatSSHConfigValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAddSSHHostWithSpacesInPath(t *testing.T) {
	// Create temporary config file
	configFile, err := createTempConfigFile(`Host existing
    HostName existing.com
`)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(configFile)

	// Test adding host with path containing spaces
	host := SSHHost{
		Name:     "test-spaces",
		Hostname: "test.com",
		User:     "testuser",
		Identity: "/path/with spaces/key file",
	}

	err = AddSSHHostToFile(host, configFile)
	if err != nil {
		t.Fatalf("AddSSHHostToFile failed: %v", err)
	}

	// Read the file and verify quotes are added
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	contentStr := string(content)
	expectedIdentityLine := `    IdentityFile "/path/with spaces/key file"`
	if !strings.Contains(contentStr, expectedIdentityLine) {
		t.Errorf("Expected identity file line with quotes not found.\nContent:\n%s\nExpected line: %s", contentStr, expectedIdentityLine)
	}
}
