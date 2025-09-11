package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createTestHistoryManager creates a history manager with a temporary file for testing
func createTestHistoryManager(t *testing.T) *HistoryManager {
	// Create temporary directory
	tempDir := t.TempDir()
	historyPath := filepath.Join(tempDir, "test_sshm_history.json")

	hm := &HistoryManager{
		historyPath: historyPath,
		history:     &ConnectionHistory{Connections: make(map[string]ConnectionInfo)},
	}

	return hm
}

func TestNewHistoryManager(t *testing.T) {
	hm, err := NewHistoryManager()
	if err != nil {
		t.Fatalf("NewHistoryManager() error = %v", err)
	}
	if hm == nil {
		t.Fatal("NewHistoryManager() returned nil")
	}
	if hm.historyPath == "" {
		t.Error("Expected historyPath to be set")
	}
}

func TestHistoryManager_RecordConnection(t *testing.T) {
	hm := createTestHistoryManager(t)

	// Add a connection
	err := hm.RecordConnection("testhost")
	if err != nil {
		t.Errorf("RecordConnection() error = %v", err)
	}

	// Check that the connection was added
	lastUsed, exists := hm.GetLastConnectionTime("testhost")
	if !exists || lastUsed.IsZero() {
		t.Error("Expected connection to be recorded")
	}
}

func TestHistoryManager_GetLastConnectionTime(t *testing.T) {
	hm := createTestHistoryManager(t)

	// Test with no connections
	lastUsed, exists := hm.GetLastConnectionTime("nonexistent-testhost")
	if exists || !lastUsed.IsZero() {
		t.Error("Expected no connection for non-existent host")
	}

	// Add a connection
	err := hm.RecordConnection("testhost")
	if err != nil {
		t.Errorf("RecordConnection() error = %v", err)
	}

	// Test with existing connection
	lastUsed, exists = hm.GetLastConnectionTime("testhost")
	if !exists || lastUsed.IsZero() {
		t.Error("Expected non-zero time for existing host")
	}

	// Check that the time is recent (within last minute)
	if time.Since(lastUsed) > time.Minute {
		t.Error("Last used time seems too old")
	}
}

func TestHistoryManager_GetConnectionCount(t *testing.T) {
	hm := createTestHistoryManager(t)

	// Add same host multiple times
	for i := 0; i < 3; i++ {
		err := hm.RecordConnection("testhost-count")
		if err != nil {
			t.Errorf("RecordConnection() error = %v", err)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// Should have correct count
	count := hm.GetConnectionCount("testhost-count")
	if count != 3 {
		t.Errorf("Expected connection count 3, got %d", count)
	}
}

func TestMigrateOldHistoryFile(t *testing.T) {
	// This test verifies that migration doesn't fail when called
	// The actual migration logic will be tested in integration tests

	tempDir := t.TempDir()
	newHistoryPath := filepath.Join(tempDir, "sshm_history.json")

	// Test that migration works when no old file exists (common case)
	if err := migrateOldHistoryFile(newHistoryPath); err != nil {
		t.Errorf("migrateOldHistoryFile() with no old file error = %v", err)
	}

	// Test that migration skips when new file already exists
	if err := os.WriteFile(newHistoryPath, []byte(`{"connections":{}}`), 0644); err != nil {
		t.Fatalf("Failed to write new history file: %v", err)
	}

	if err := migrateOldHistoryFile(newHistoryPath); err != nil {
		t.Errorf("migrateOldHistoryFile() with existing new file error = %v", err)
	}

	// File should be unchanged
	data, err := os.ReadFile(newHistoryPath)
	if err != nil {
		t.Errorf("Failed to read new file: %v", err)
	}
	if string(data) != `{"connections":{}}` {
		t.Error("New file was modified when it shouldn't have been")
	}
}

func TestMigrateOldHistoryFile_NoOldFile(t *testing.T) {
	// Test migration when no old file exists
	tempDir := t.TempDir()
	newHistoryPath := filepath.Join(tempDir, "sshm_history.json")

	// Should not return error when old file doesn't exist
	if err := migrateOldHistoryFile(newHistoryPath); err != nil {
		t.Errorf("migrateOldHistoryFile() with no old file error = %v", err)
	}
}

func TestMigrateOldHistoryFile_NewFileExists(t *testing.T) {
	// Test migration when new file already exists (should skip migration)
	tempDir := t.TempDir()
	newHistoryPath := filepath.Join(tempDir, "sshm_history.json")

	// Create new file first
	if err := os.WriteFile(newHistoryPath, []byte(`{"connections":{}}`), 0644); err != nil {
		t.Fatalf("Failed to write new history file: %v", err)
	}

	// Migration should skip when new file exists
	if err := migrateOldHistoryFile(newHistoryPath); err != nil {
		t.Errorf("migrateOldHistoryFile() with existing new file error = %v", err)
	}

	// New file should be unchanged
	data, err := os.ReadFile(newHistoryPath)
	if err != nil {
		t.Errorf("Failed to read new file: %v", err)
	}
	if string(data) != `{"connections":{}}` {
		t.Error("New file was modified when it shouldn't have been")
	}
}
