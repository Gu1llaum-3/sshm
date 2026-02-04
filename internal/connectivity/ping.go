package connectivity

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/config"

	"golang.org/x/crypto/ssh"
)

// PingStatus represents the connectivity status of an SSH host
type PingStatus int

const (
	StatusUnknown PingStatus = iota
	StatusConnecting
	StatusOnline
	StatusOffline
)

func (s PingStatus) String() string {
	switch s {
	case StatusUnknown:
		return "unknown"
	case StatusConnecting:
		return "connecting"
	case StatusOnline:
		return "online"
	case StatusOffline:
		return "offline"
	}
	return "unknown"
}

// HostPingResult represents the result of pinging a host
type HostPingResult struct {
	HostName string
	Status   PingStatus
	Error    error
	Duration time.Duration
}

// PingManager manages SSH connectivity checks for multiple hosts
type PingManager struct {
	results    map[string]*HostPingResult
	mutex      sync.RWMutex
	timeout    time.Duration
	configFile string
}

// NewPingManager creates a new ping manager with the specified timeout
func NewPingManager(timeout time.Duration, configFile string) *PingManager {
	return &PingManager{
		results:    make(map[string]*HostPingResult),
		timeout:    timeout,
		configFile: configFile,
	}
}

// GetStatus returns the current status for a host
func (pm *PingManager) GetStatus(hostName string) PingStatus {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	if result, exists := pm.results[hostName]; exists {
		return result.Status
	}
	return StatusUnknown
}

// GetResult returns the complete result for a host
func (pm *PingManager) GetResult(hostName string) (*HostPingResult, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result, exists := pm.results[hostName]
	return result, exists
}

// updateStatus updates the status for a host
func (pm *PingManager) updateStatus(hostName string, status PingStatus, err error, duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.results[hostName] = &HostPingResult{
		HostName: hostName,
		Status:   status,
		Error:    err,
		Duration: duration,
	}
}

// PingHost performs an SSH connectivity check for a single host
func (pm *PingManager) PingHost(ctx context.Context, host config.SSHHost) *HostPingResult {
	start := time.Now()

	// Mark as connecting
	pm.updateStatus(host.Name, StatusConnecting, nil, 0)

	// If the host uses a ProxyJump or ProxyCommand, we need to use the external SSH command
	// because implementing jump host support with pure Go ssh library requires
	// handling authentication for the jump host, which is complex and requires
	// access to the user's SSH agent or keys.
	if host.ProxyJump != "" || host.ProxyCommand != "" {
		return pm.pingWithExternalCommand(ctx, host, start)
	}

	// Determine the actual hostname and port
	hostname := host.Hostname
	if hostname == "" {
		hostname = host.Name
	}

	port := host.Port
	if port == "" {
		port = "22"
	}

	// Create context with timeout
	pingCtx, cancel := context.WithTimeout(ctx, pm.timeout)
	defer cancel()

	// Try to establish a TCP connection first (faster than SSH handshake)
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(pingCtx, "tcp", net.JoinHostPort(hostname, port))
	if err != nil {
		duration := time.Since(start)
		pm.updateStatus(host.Name, StatusOffline, err, duration)
		return &HostPingResult{
			HostName: host.Name,
			Status:   StatusOffline,
			Error:    err,
			Duration: duration,
		}
	}
	defer conn.Close()

	// If TCP connection succeeds, try SSH handshake
	sshConfig := &ssh.ClientConfig{
		User:            host.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For ping purposes only
		Timeout:         time.Second * 2,             // Short timeout for handshake
	}

	// We don't need to authenticate, just check if SSH is responding
	sshConn, _, _, err := ssh.NewClientConn(conn, net.JoinHostPort(hostname, port), sshConfig)
	if sshConn != nil {
		sshConn.Close()
	}

	duration := time.Since(start)

	// Even if SSH handshake fails, if we got a TCP connection, consider it online
	// This handles cases where authentication fails but the host is reachable
	status := StatusOnline
	if err != nil && isConnectionError(err) {
		status = StatusOffline
	}

	pm.updateStatus(host.Name, status, err, duration)
	return &HostPingResult{
		HostName: host.Name,
		Status:   status,
		Error:    err,
		Duration: duration,
	}
}

// pingWithExternalCommand pings a host using the external SSH command
func (pm *PingManager) pingWithExternalCommand(ctx context.Context, host config.SSHHost, start time.Time) *HostPingResult {
	// Construct the SSH command
	// ssh -q -o BatchMode=yes -o StrictHostKeyChecking=no -o ConnectTimeout=5 host exit
	args := []string{"-q", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no"}

	// Set timeout matching the manager's timeout
	// Convert duration to seconds (rounding up to ensure we don't timeout too early in the command)
	timeoutSec := int(pm.timeout.Seconds())
	if timeoutSec < 1 {
		timeoutSec = 1
	}
	args = append(args, "-o", fmt.Sprintf("ConnectTimeout=%d", timeoutSec))

	// If we have a specific config file, use it
	if pm.configFile != "" {
		args = append(args, "-F", pm.configFile)
	}

	// Add the host name and the command to run (exit)
	args = append(args, host.Name, "exit")

	// Create command with context for timeout cancellation
	// Note: We used pm.timeout for the ssh command option, but we also respect the context deadline
	cmd := exec.CommandContext(ctx, "ssh", args...)

	// Run the command
	err := cmd.Run()
	duration := time.Since(start)

	var status PingStatus
	if err != nil {
		// SSH returns non-zero exit code on connection failure
		status = StatusOffline
	} else {
		status = StatusOnline
	}

	pm.updateStatus(host.Name, status, err, duration)
	return &HostPingResult{
		HostName: host.Name,
		Status:   status,
		Error:    err,
		Duration: duration,
	}
}

// PingAllHosts pings all hosts concurrently and returns a channel of results
func (pm *PingManager) PingAllHosts(ctx context.Context, hosts []config.SSHHost) <-chan *HostPingResult {
	resultChan := make(chan *HostPingResult, len(hosts))

	var wg sync.WaitGroup

	for _, host := range hosts {
		wg.Add(1)
		go func(h config.SSHHost) {
			defer wg.Done()
			result := pm.PingHost(ctx, h)
			select {
			case resultChan <- result:
			case <-ctx.Done():
				return
			}
		}(host)
	}

	// Close the channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	return resultChan
}

// isConnectionError determines if an error is a connection-related error
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"no route to host",
		"network is unreachable",
		"timeout",
		"connection timed out",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(strings.ToLower(errStr), connErr) {
			return true
		}
	}

	return false
}
