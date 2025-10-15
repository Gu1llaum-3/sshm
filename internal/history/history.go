package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/config"
)

// ConnectionHistory represents the history of SSH connections
type ConnectionHistory struct {
	Connections map[string]ConnectionInfo `json:"connections"`
}

// PortForwardConfig stores port forwarding configuration
type PortForwardConfig struct {
	Type        string `json:"type"` // "local", "remote", "dynamic"
	LocalPort   string `json:"local_port"`
	RemoteHost  string `json:"remote_host"`
	RemotePort  string `json:"remote_port"`
	BindAddress string `json:"bind_address"`
}

// ConnectionInfo stores information about a specific connection
type ConnectionInfo struct {
	HostName       string             `json:"host_name"`
	LastConnect    time.Time          `json:"last_connect"`
	ConnectCount   int                `json:"connect_count"`
	PortForwarding *PortForwardConfig `json:"port_forwarding,omitempty"`
}

// HistoryManager manages the connection history
type HistoryManager struct {
	historyPath string
	history     *ConnectionHistory
}

// NewHistoryManager creates a new history manager
func NewHistoryManager() (*HistoryManager, error) {
	configDir, err := config.GetSSHMConfigDir()
	if err != nil {
		return nil, err
	}

	// Ensure config dir exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	historyPath := filepath.Join(configDir, "sshm_history.json")

	// Migration: check if old history file exists and migrate it
	if err := migrateOldHistoryFile(historyPath); err != nil {
		// Don't fail if migration fails, just log it
		// In a production environment, you might want to log this properly
	}

	hm := &HistoryManager{
		historyPath: historyPath,
		history:     &ConnectionHistory{Connections: make(map[string]ConnectionInfo)},
	}

	// Load existing history if it exists
	err = hm.loadHistory()
	if err != nil {
		// If file doesn't exist, that's okay - we'll create it when needed
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return hm, nil
}

// migrateOldHistoryFile migrates the old history file from ~/.ssh to ~/.config/sshm
// TODO: Remove this migration logic in v2.0.0 (introduced in v1.6.0)
func migrateOldHistoryFile(newHistoryPath string) error {
	// Check if new file already exists, skip migration
	if _, err := os.Stat(newHistoryPath); err == nil {
		return nil // New file exists, no migration needed
	}

	// Get old history file path - use same logic as SSH config location
	sshDir, err := config.GetSSHDirectory()
	if err != nil {
		return err
	}
	oldHistoryPath := filepath.Join(sshDir, "sshm_history.json")

	// Check if old file exists
	if _, err := os.Stat(oldHistoryPath); os.IsNotExist(err) {
		return nil // Old file doesn't exist, nothing to migrate
	}

	// Read old file
	data, err := os.ReadFile(oldHistoryPath)
	if err != nil {
		return err
	}

	// Write to new location
	if err := os.WriteFile(newHistoryPath, data, 0644); err != nil {
		return err
	}

	// Remove old file only if write was successful
	if err := os.Remove(oldHistoryPath); err != nil {
		// Don't fail if we can't remove the old file
		// The migration was successful even if cleanup failed
	}

	return nil
}

// loadHistory loads the connection history from the JSON file
func (hm *HistoryManager) loadHistory() error {
	data, err := os.ReadFile(hm.historyPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, hm.history)
}

// saveHistory saves the connection history to the JSON file
func (hm *HistoryManager) saveHistory() error {
	// Ensure the directory exists
	dir := filepath.Dir(hm.historyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(hm.history, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(hm.historyPath, data, 0600)
}

// RecordConnection records a new connection for the specified host
func (hm *HistoryManager) RecordConnection(hostName string) error {
	now := time.Now()

	if conn, exists := hm.history.Connections[hostName]; exists {
		// Update existing connection
		conn.LastConnect = now
		conn.ConnectCount++
		hm.history.Connections[hostName] = conn
	} else {
		// Create new connection record
		hm.history.Connections[hostName] = ConnectionInfo{
			HostName:     hostName,
			LastConnect:  now,
			ConnectCount: 1,
		}
	}

	return hm.saveHistory()
}

// GetLastConnectionTime returns the last connection time for a host
func (hm *HistoryManager) GetLastConnectionTime(hostName string) (time.Time, bool) {
	if conn, exists := hm.history.Connections[hostName]; exists {
		return conn.LastConnect, true
	}
	return time.Time{}, false
}

// GetConnectionCount returns the total number of connections for a host
func (hm *HistoryManager) GetConnectionCount(hostName string) int {
	if conn, exists := hm.history.Connections[hostName]; exists {
		return conn.ConnectCount
	}
	return 0
}

// SortHostsByLastUsed sorts hosts by their last connection time (most recent first)
func (hm *HistoryManager) SortHostsByLastUsed(hosts []config.SSHHost) []config.SSHHost {
	sorted := make([]config.SSHHost, len(hosts))
	copy(sorted, hosts)

	sort.Slice(sorted, func(i, j int) bool {
		timeI, existsI := hm.GetLastConnectionTime(sorted[i].Name)
		timeJ, existsJ := hm.GetLastConnectionTime(sorted[j].Name)

		// If both have history, sort by most recent first
		if existsI && existsJ {
			return timeI.After(timeJ)
		}

		// Hosts with history come before hosts without history
		if existsI && !existsJ {
			return true
		}
		if !existsI && existsJ {
			return false
		}

		// If neither has history, sort alphabetically
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// SortHostsByMostUsed sorts hosts by their connection count (most used first)
func (hm *HistoryManager) SortHostsByMostUsed(hosts []config.SSHHost) []config.SSHHost {
	sorted := make([]config.SSHHost, len(hosts))
	copy(sorted, hosts)

	sort.Slice(sorted, func(i, j int) bool {
		countI := hm.GetConnectionCount(sorted[i].Name)
		countJ := hm.GetConnectionCount(sorted[j].Name)

		// If counts are different, sort by count (highest first)
		if countI != countJ {
			return countI > countJ
		}

		// If counts are equal, sort by most recent
		timeI, existsI := hm.GetLastConnectionTime(sorted[i].Name)
		timeJ, existsJ := hm.GetLastConnectionTime(sorted[j].Name)

		if existsI && existsJ {
			return timeI.After(timeJ)
		}

		// If neither has history, sort alphabetically
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// CleanupOldEntries removes connection history for hosts that no longer exist
func (hm *HistoryManager) CleanupOldEntries(currentHosts []config.SSHHost) error {
	// Create a set of current host names
	currentHostNames := make(map[string]bool)
	for _, host := range currentHosts {
		currentHostNames[host.Name] = true
	}

	// Remove entries for hosts that no longer exist
	for hostName := range hm.history.Connections {
		if !currentHostNames[hostName] {
			delete(hm.history.Connections, hostName)
		}
	}

	return hm.saveHistory()
}

// GetAllConnectionsInfo returns all connection information sorted by last connection time
func (hm *HistoryManager) GetAllConnectionsInfo() []ConnectionInfo {
	var connections []ConnectionInfo
	for _, conn := range hm.history.Connections {
		connections = append(connections, conn)
	}

	sort.Slice(connections, func(i, j int) bool {
		return connections[i].LastConnect.After(connections[j].LastConnect)
	})

	return connections
}

// RecordPortForwarding saves port forwarding configuration for a host
func (hm *HistoryManager) RecordPortForwarding(hostName, forwardType, localPort, remoteHost, remotePort, bindAddress string) error {
	now := time.Now()

	portForwardConfig := &PortForwardConfig{
		Type:        forwardType,
		LocalPort:   localPort,
		RemoteHost:  remoteHost,
		RemotePort:  remotePort,
		BindAddress: bindAddress,
	}

	if conn, exists := hm.history.Connections[hostName]; exists {
		// Update existing connection
		conn.LastConnect = now
		conn.ConnectCount++
		conn.PortForwarding = portForwardConfig
		hm.history.Connections[hostName] = conn
	} else {
		// Create new connection record
		hm.history.Connections[hostName] = ConnectionInfo{
			HostName:       hostName,
			LastConnect:    now,
			ConnectCount:   1,
			PortForwarding: portForwardConfig,
		}
	}

	return hm.saveHistory()
}

// GetPortForwardingConfig retrieves the last used port forwarding configuration for a host
func (hm *HistoryManager) GetPortForwardingConfig(hostName string) *PortForwardConfig {
	if conn, exists := hm.history.Connections[hostName]; exists {
		return conn.PortForwarding
	}
	return nil
}

// ManualConnection represents a manual SSH connection (e.g., ssh user@host -p 2222)
type ManualConnection struct {
	User     string
	Hostname string
	Port     string
	Identity string
}

// RecordManualConnection records a manual SSH connection (like ssh user@host -p 2222 -i key)
// These are stored with a generated host name like "manual:user@host:port"
func (hm *HistoryManager) RecordManualConnection(conn ManualConnection) error {
	// Generate a unique identifier for this manual connection
	hostID := generateManualHostID(conn)

	now := time.Now()

	if existingConn, exists := hm.history.Connections[hostID]; exists {
		// Update existing connection
		existingConn.LastConnect = now
		existingConn.ConnectCount++
		hm.history.Connections[hostID] = existingConn
	} else {
		// Create new connection record
		hm.history.Connections[hostID] = ConnectionInfo{
			HostName:     hostID,
			LastConnect:  now,
			ConnectCount: 1,
		}
	}

	return hm.saveHistory()
}

// generateManualHostID generates a unique ID for manual connections
func generateManualHostID(conn ManualConnection) string {
	// Format: manual:user@hostname:port
	user := conn.User
	if user == "" {
		user = "default"
	}
	port := conn.Port
	if port == "" {
		port = "22"
	}
	return fmt.Sprintf("manual:%s@%s:%s", user, conn.Hostname, port)
}

// IsManualConnection checks if a hostname represents a manual connection
func IsManualConnection(hostName string) bool {
	return len(hostName) > 7 && hostName[:7] == "manual:"
}

// ParseManualConnectionID parses a manual connection ID back into its components
func ParseManualConnectionID(hostID string) (user, hostname, port string, ok bool) {
	if !IsManualConnection(hostID) {
		return "", "", "", false
	}

	// Remove "manual:" prefix
	parts := hostID[7:] // Skip "manual:"

	// Split by last ':'
	lastColon := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == ':' {
			lastColon = i
			break
		}
	}

	if lastColon == -1 {
		return "", "", "", false
	}

	port = parts[lastColon+1:]
	userHost := parts[:lastColon]

	// Split user@host
	atSign := -1
	for i := 0; i < len(userHost); i++ {
		if userHost[i] == '@' {
			atSign = i
			break
		}
	}

	if atSign == -1 {
		return "", "", "", false
	}

	user = userHost[:atSign]
	hostname = userHost[atSign+1:]

	return user, hostname, port, true
}
