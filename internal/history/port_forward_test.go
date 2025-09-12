package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPortForwardingHistory(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sshm_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create history manager with temp directory
	historyPath := filepath.Join(tempDir, "test_history.json")
	hm := &HistoryManager{
		historyPath: historyPath,
		history:     &ConnectionHistory{Connections: make(map[string]ConnectionInfo)},
	}

	hostName := "test-server"

	// Test recording port forwarding configuration
	err = hm.RecordPortForwarding(hostName, "local", "8080", "localhost", "80", "127.0.0.1")
	if err != nil {
		t.Fatalf("Failed to record port forwarding: %v", err)
	}

	// Test retrieving port forwarding configuration
	config := hm.GetPortForwardingConfig(hostName)
	if config == nil {
		t.Fatalf("Expected port forwarding config to exist")
	}

	// Verify the saved configuration
	if config.Type != "local" {
		t.Errorf("Expected Type 'local', got %s", config.Type)
	}
	if config.LocalPort != "8080" {
		t.Errorf("Expected LocalPort '8080', got %s", config.LocalPort)
	}
	if config.RemoteHost != "localhost" {
		t.Errorf("Expected RemoteHost 'localhost', got %s", config.RemoteHost)
	}
	if config.RemotePort != "80" {
		t.Errorf("Expected RemotePort '80', got %s", config.RemotePort)
	}
	if config.BindAddress != "127.0.0.1" {
		t.Errorf("Expected BindAddress '127.0.0.1', got %s", config.BindAddress)
	}

	// Test updating configuration with different values
	err = hm.RecordPortForwarding(hostName, "remote", "3000", "app-server", "8000", "")
	if err != nil {
		t.Fatalf("Failed to record updated port forwarding: %v", err)
	}

	// Verify the updated configuration
	config = hm.GetPortForwardingConfig(hostName)
	if config == nil {
		t.Fatalf("Expected port forwarding config to exist after update")
	}

	if config.Type != "remote" {
		t.Errorf("Expected updated Type 'remote', got %s", config.Type)
	}
	if config.LocalPort != "3000" {
		t.Errorf("Expected updated LocalPort '3000', got %s", config.LocalPort)
	}
	if config.RemoteHost != "app-server" {
		t.Errorf("Expected updated RemoteHost 'app-server', got %s", config.RemoteHost)
	}
	if config.RemotePort != "8000" {
		t.Errorf("Expected updated RemotePort '8000', got %s", config.RemotePort)
	}
	if config.BindAddress != "" {
		t.Errorf("Expected updated BindAddress to be empty, got %s", config.BindAddress)
	}

	// Test dynamic forwarding
	err = hm.RecordPortForwarding(hostName, "dynamic", "1080", "", "", "0.0.0.0")
	if err != nil {
		t.Fatalf("Failed to record dynamic port forwarding: %v", err)
	}

	config = hm.GetPortForwardingConfig(hostName)
	if config == nil {
		t.Fatalf("Expected port forwarding config to exist for dynamic forwarding")
	}

	if config.Type != "dynamic" {
		t.Errorf("Expected Type 'dynamic', got %s", config.Type)
	}
	if config.LocalPort != "1080" {
		t.Errorf("Expected LocalPort '1080', got %s", config.LocalPort)
	}
	if config.RemoteHost != "" {
		t.Errorf("Expected RemoteHost to be empty for dynamic forwarding, got %s", config.RemoteHost)
	}
	if config.RemotePort != "" {
		t.Errorf("Expected RemotePort to be empty for dynamic forwarding, got %s", config.RemotePort)
	}
	if config.BindAddress != "0.0.0.0" {
		t.Errorf("Expected BindAddress '0.0.0.0', got %s", config.BindAddress)
	}
}

func TestPortForwardingHistoryPersistence(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sshm_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	historyPath := filepath.Join(tempDir, "test_history.json")

	// Create first history manager and record data
	hm1 := &HistoryManager{
		historyPath: historyPath,
		history:     &ConnectionHistory{Connections: make(map[string]ConnectionInfo)},
	}

	hostName := "persistent-server"
	err = hm1.RecordPortForwarding(hostName, "local", "9090", "db-server", "5432", "")
	if err != nil {
		t.Fatalf("Failed to record port forwarding: %v", err)
	}

	// Create second history manager and load data
	hm2 := &HistoryManager{
		historyPath: historyPath,
		history:     &ConnectionHistory{Connections: make(map[string]ConnectionInfo)},
	}

	err = hm2.loadHistory()
	if err != nil {
		t.Fatalf("Failed to load history: %v", err)
	}

	// Verify the loaded configuration
	config := hm2.GetPortForwardingConfig(hostName)
	if config == nil {
		t.Fatalf("Expected port forwarding config to be loaded from file")
	}

	if config.Type != "local" {
		t.Errorf("Expected loaded Type 'local', got %s", config.Type)
	}
	if config.LocalPort != "9090" {
		t.Errorf("Expected loaded LocalPort '9090', got %s", config.LocalPort)
	}
	if config.RemoteHost != "db-server" {
		t.Errorf("Expected loaded RemoteHost 'db-server', got %s", config.RemoteHost)
	}
	if config.RemotePort != "5432" {
		t.Errorf("Expected loaded RemotePort '5432', got %s", config.RemotePort)
	}
}

func TestGetPortForwardingConfigNonExistent(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sshm_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	historyPath := filepath.Join(tempDir, "test_history.json")
	hm := &HistoryManager{
		historyPath: historyPath,
		history:     &ConnectionHistory{Connections: make(map[string]ConnectionInfo)},
	}

	// Test getting configuration for non-existent host
	config := hm.GetPortForwardingConfig("non-existent-host")
	if config != nil {
		t.Errorf("Expected nil config for non-existent host, got %+v", config)
	}
}
