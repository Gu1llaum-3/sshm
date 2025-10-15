package history

import (
	"testing"
)

func TestParseSSHArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantConn *ManualConnection
		wantOk   bool
	}{
		{
			name: "user@host",
			args: []string{"user@example.com"},
			wantConn: &ManualConnection{
				User:     "user",
				Hostname: "example.com",
				Port:     "22",
			},
			wantOk: true,
		},
		{
			name: "user@host with -p port",
			args: []string{"-p", "2222", "user@example.com"},
			wantConn: &ManualConnection{
				User:     "user",
				Hostname: "example.com",
				Port:     "2222",
			},
			wantOk: true,
		},
		{
			name: "user@host with -p2222 (no space)",
			args: []string{"-p2222", "user@example.com"},
			wantConn: &ManualConnection{
				User:     "user",
				Hostname: "example.com",
				Port:     "2222",
			},
			wantOk: true,
		},
		{
			name: "user@host with -i identity",
			args: []string{"-i", "~/.ssh/id_rsa", "user@example.com"},
			wantConn: &ManualConnection{
				User:     "user",
				Hostname: "example.com",
				Port:     "22",
				Identity: "~/.ssh/id_rsa",
			},
			wantOk: true,
		},
		{
			name: "complete connection",
			args: []string{"-p", "2222", "-i", "~/.ssh/id_rsa", "guillaume@127.0.0.1"},
			wantConn: &ManualConnection{
				User:     "guillaume",
				Hostname: "127.0.0.1",
				Port:     "2222",
				Identity: "~/.ssh/id_rsa",
			},
			wantOk: true,
		},
		{
			name: "just hostname (no user)",
			args: []string{"example.com"},
			wantConn: &ManualConnection{
				Hostname: "example.com",
				Port:     "22",
				// User will be current system user, so we don't check it
			},
			wantOk: true,
		},
		{
			name:     "config file args should return false",
			args:     []string{"-F", "~/.ssh/config", "host"},
			wantConn: nil,
			wantOk:   false,
		},
		{
			name:     "empty args",
			args:     []string{},
			wantConn: nil,
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConn, gotOk := ParseSSHArgs(tt.args)

			if gotOk != tt.wantOk {
				t.Errorf("ParseSSHArgs() gotOk = %v, want %v", gotOk, tt.wantOk)
				return
			}

			if !tt.wantOk {
				if gotConn != nil {
					t.Errorf("ParseSSHArgs() gotConn = %v, want nil", gotConn)
				}
				return
			}

			if gotConn == nil {
				t.Errorf("ParseSSHArgs() gotConn = nil, want non-nil")
				return
			}

			if gotConn.User != tt.wantConn.User {
				// Skip user check if wantConn.User is empty (current user)
				if tt.wantConn.User != "" {
					t.Errorf("ParseSSHArgs() User = %v, want %v", gotConn.User, tt.wantConn.User)
				}
			}
			if gotConn.Hostname != tt.wantConn.Hostname {
				t.Errorf("ParseSSHArgs() Hostname = %v, want %v", gotConn.Hostname, tt.wantConn.Hostname)
			}
			if gotConn.Port != tt.wantConn.Port {
				t.Errorf("ParseSSHArgs() Port = %v, want %v", gotConn.Port, tt.wantConn.Port)
			}
			if gotConn.Identity != tt.wantConn.Identity {
				t.Errorf("ParseSSHArgs() Identity = %v, want %v", gotConn.Identity, tt.wantConn.Identity)
			}
		})
	}
}

func TestIsManualSSHCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "user@host is manual",
			args: []string{"user@example.com"},
			want: true,
		},
		{
			name: "with -p flag is manual",
			args: []string{"-p", "2222", "host"},
			want: true,
		},
		{
			name: "with -p2222 is manual",
			args: []string{"-p2222", "host"},
			want: true,
		},
		{
			name: "just hostname is not manual",
			args: []string{"myhost"},
			want: false,
		},
		{
			name: "empty is not manual",
			args: []string{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsManualSSHCommand(tt.args); got != tt.want {
				t.Errorf("IsManualSSHCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManualConnectionID(t *testing.T) {
	tests := []struct {
		name         string
		conn         ManualConnection
		wantHostID   string
		wantUser     string
		wantHostname string
		wantPort     string
	}{
		{
			name: "complete connection",
			conn: ManualConnection{
				User:     "guillaume",
				Hostname: "127.0.0.1",
				Port:     "2222",
				Identity: "~/.ssh/id_rsa",
			},
			wantHostID:   "manual:guillaume@127.0.0.1:2222",
			wantUser:     "guillaume",
			wantHostname: "127.0.0.1",
			wantPort:     "2222",
		},
		{
			name: "default port",
			conn: ManualConnection{
				User:     "user",
				Hostname: "example.com",
				Port:     "",
			},
			wantHostID:   "manual:user@example.com:22",
			wantUser:     "user",
			wantHostname: "example.com",
			wantPort:     "22",
		},
		{
			name: "no user specified",
			conn: ManualConnection{
				Hostname: "example.com",
				Port:     "2222",
			},
			wantHostID:   "manual:default@example.com:2222",
			wantUser:     "default",
			wantHostname: "example.com",
			wantPort:     "2222",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test generation
			gotHostID := generateManualHostID(tt.conn)
			if gotHostID != tt.wantHostID {
				t.Errorf("generateManualHostID() = %v, want %v", gotHostID, tt.wantHostID)
			}

			// Test IsManualConnection
			if !IsManualConnection(gotHostID) {
				t.Errorf("IsManualConnection(%v) = false, want true", gotHostID)
			}

			// Test parsing
			user, hostname, port, ok := ParseManualConnectionID(gotHostID)
			if !ok {
				t.Errorf("ParseManualConnectionID() ok = false, want true")
			}
			if user != tt.wantUser {
				t.Errorf("ParseManualConnectionID() user = %v, want %v", user, tt.wantUser)
			}
			if hostname != tt.wantHostname {
				t.Errorf("ParseManualConnectionID() hostname = %v, want %v", hostname, tt.wantHostname)
			}
			if port != tt.wantPort {
				t.Errorf("ParseManualConnectionID() port = %v, want %v", port, tt.wantPort)
			}
		})
	}
}

func TestParseManualConnectionID_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		hostID string
	}{
		{
			name:   "not a manual connection",
			hostID: "myhost",
		},
		{
			name:   "missing components",
			hostID: "manual:invalid",
		},
		{
			name:   "no @ sign",
			hostID: "manual:hostname:22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, ok := ParseManualConnectionID(tt.hostID)
			if ok {
				t.Errorf("ParseManualConnectionID() ok = true, want false for invalid input")
			}
		})
	}
}
