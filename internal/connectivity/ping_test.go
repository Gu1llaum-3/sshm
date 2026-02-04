package connectivity

import (
	"context"
	"testing"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/config"
)

func TestNewPingManager(t *testing.T) {
	pm := NewPingManager(5*time.Second, "")
	if pm == nil {
		t.Error("NewPingManager() returned nil")
	}
	if pm.results == nil {
		t.Error("PingManager.results map not initialized")
	}
}

func TestPingManager_PingHost(t *testing.T) {
	pm := NewPingManager(1*time.Second, "")
	ctx := context.Background()

	// Test ping method exists and doesn't panic
	host := config.SSHHost{Name: "test", Hostname: "127.0.0.1", Port: "22"}
	result := pm.PingHost(ctx, host)
	if result == nil {
		t.Error("Expected ping result to be returned")
	}

	// Test with invalid host
	invalidHost := config.SSHHost{Name: "invalid", Hostname: "invalid.host.12345", Port: "22"}
	result = pm.PingHost(ctx, invalidHost)
	if result == nil {
		t.Error("Expected ping result to be returned even for invalid host")
	}
}

func TestPingManager_GetStatus(t *testing.T) {
	pm := NewPingManager(1*time.Second, "")

	// Test unknown host
	status := pm.GetStatus("unknown.host")
	if status != StatusUnknown {
		t.Errorf("Expected StatusUnknown for unknown host, got %v", status)
	}

	// Test after ping
	ctx := context.Background()
	host := config.SSHHost{Name: "test", Hostname: "127.0.0.1", Port: "22"}
	pm.PingHost(ctx, host)
	status = pm.GetStatus("test")
	if status == StatusUnknown {
		t.Error("Expected status to be set after ping")
	}
}

func TestPingManager_PingMultipleHosts(t *testing.T) {
	pm := NewPingManager(1*time.Second, "")
	hosts := []config.SSHHost{
		{Name: "localhost", Hostname: "127.0.0.1", Port: "22"},
		{Name: "invalid", Hostname: "invalid.host.12345", Port: "22"},
	}

	ctx := context.Background()

	// Ping each host individually
	for _, host := range hosts {
		result := pm.PingHost(ctx, host)
		if result == nil {
			t.Errorf("Expected ping result for host %s", host.Name)
		}

		// Check that status was set
		status := pm.GetStatus(host.Name)
		if status == StatusUnknown {
			t.Errorf("Expected status to be set for host %s after ping", host.Name)
		}
	}
}

func TestPingManager_GetResult(t *testing.T) {
	pm := NewPingManager(1*time.Second, "")
	ctx := context.Background()

	// Test getting result for unknown host
	result, exists := pm.GetResult("unknown")
	if exists || result != nil {
		t.Error("Expected no result for unknown host")
	}

	// Test after ping
	host := config.SSHHost{Name: "test", Hostname: "127.0.0.1", Port: "22"}
	pm.PingHost(ctx, host)

	result, exists = pm.GetResult("test")
	if !exists || result == nil {
		t.Error("Expected result to exist after ping")
	}
	if result.HostName != "test" {
		t.Errorf("Expected hostname 'test', got '%s'", result.HostName)
	}
}

func TestPingStatus_String(t *testing.T) {
	tests := []struct {
		status   PingStatus
		expected string
	}{
		{StatusUnknown, "unknown"},
		{StatusConnecting, "connecting"},
		{StatusOnline, "online"},
		{StatusOffline, "offline"},
		{PingStatus(999), "unknown"}, // Invalid status
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("PingStatus.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPingHost_Basic(t *testing.T) {
	// Test that the ping functionality exists
	pm := NewPingManager(1*time.Second, "")
	ctx := context.Background()
	host := config.SSHHost{Name: "test", Hostname: "127.0.0.1", Port: "22"}

	// Just ensure the function doesn't panic
	result := pm.PingHost(ctx, host)
	if result == nil {
		t.Error("Expected ping result to be returned")
	}

	// Test that status is set
	status := pm.GetStatus("test")
	if status == StatusUnknown {
		t.Error("Expected status to be set after ping attempt")
	}
}
