package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateHostname(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     bool
	}{
		{"valid hostname", "example.com", true},
		{"valid IP", "192.168.1.1", true}, // IPs are valid hostnames too
		{"valid subdomain", "sub.example.com", true},
		{"valid single word", "localhost", true},
		{"empty hostname", "", false},
		{"hostname too long", strings.Repeat("a", 254), false},
		{"hostname with space", "example .com", false},
		{"hostname starting with dot", ".example.com", false},
		{"hostname ending with dot", "example.com.", false},
		{"hostname with hyphen", "my-server.com", true},
		{"hostname starting with number", "1example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateHostname(tt.hostname); got != tt.want {
				t.Errorf("ValidateHostname(%q) = %v, want %v", tt.hostname, got, tt.want)
			}
		})
	}
}

func TestValidateIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv6", "2001:db8::1", true},
		{"invalid IP", "256.256.256.256", false},
		{"empty IP", "", false},
		{"hostname not IP", "example.com", false},
		{"localhost", "127.0.0.1", true},
		{"zero IP", "0.0.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateIP(tt.ip); got != tt.want {
				t.Errorf("ValidateIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name string
		port string
		want bool
	}{
		{"valid port 22", "22", true},
		{"valid port 80", "80", true},
		{"valid port 65535", "65535", true},
		{"valid port 1", "1", true},
		{"empty port", "", true}, // Empty defaults to 22
		{"invalid port 0", "0", false},
		{"invalid port 65536", "65536", false},
		{"invalid port negative", "-1", false},
		{"invalid port string", "abc", false},
		{"invalid port with space", "22 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidatePort(tt.port); got != tt.want {
				t.Errorf("ValidatePort(%q) = %v, want %v", tt.port, got, tt.want)
			}
		})
	}
}

func TestValidateHostName(t *testing.T) {
	tests := []struct {
		name     string
		hostName string
		want     bool
	}{
		{"valid host name", "myserver", true},
		{"valid host name with hyphen", "my-server", true},
		{"valid host name with number", "server1", true},
		{"empty host name", "", false},
		{"host name too long", strings.Repeat("a", 51), false},
		{"host name with space", "my server", false},
		{"host name with tab", "my\tserver", false},
		{"host name with newline", "my\nserver", false},
		{"host name with hash", "my#server", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateHostName(tt.hostName); got != tt.want {
				t.Errorf("ValidateHostName(%q) = %v, want %v", tt.hostName, got, tt.want)
			}
		})
	}
}

func TestValidateIdentityFile(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(validFile, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"empty path", "", true}, // Optional field
		{"valid file", validFile, true},
		{"non-existent file", "/path/to/nonexistent", false},
		{"tilde path", "~/.ssh/id_rsa", true}, // Will pass if file exists
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateIdentityFile(tt.path); got != tt.want {
				t.Errorf("ValidateIdentityFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestValidateHost(t *testing.T) {
	// Create a temporary file for identity testing
	tmpDir := t.TempDir()
	validIdentity := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(validIdentity, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		hostName string
		hostname string
		port     string
		identity string
		wantErr  bool
	}{
		{"valid host", "myserver", "example.com", "22", "", false},
		{"valid host with identity", "myserver", "192.168.1.1", "2222", validIdentity, false},
		{"empty host name", "", "example.com", "22", "", true},
		{"invalid host name", "my server", "example.com", "22", "", true},
		{"empty hostname", "myserver", "", "22", "", true},
		{"invalid hostname", "myserver", "invalid..hostname", "22", "", true},
		{"invalid port", "myserver", "example.com", "99999", "", true},
		{"invalid identity", "myserver", "example.com", "22", "/nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHost(tt.hostName, tt.hostname, tt.port, tt.identity)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHost() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}