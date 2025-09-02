package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"sshm/internal/config"
)

// ConnectionHistory represents the history of SSH connections
type ConnectionHistory struct {
	Connections map[string]ConnectionInfo `json:"connections"`
}

// ConnectionInfo stores information about a specific connection
type ConnectionInfo struct {
	HostName     string    `json:"host_name"`
	LastConnect  time.Time `json:"last_connect"`
	ConnectCount int       `json:"connect_count"`
}

// HistoryManager manages the connection history
type HistoryManager struct {
	historyPath string
	history     *ConnectionHistory
}

// NewHistoryManager creates a new history manager
func NewHistoryManager() (*HistoryManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	historyPath := filepath.Join(homeDir, ".ssh", "sshm_history.json")
	
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
