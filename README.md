

<p align="center">
    <img src="images/logo.png" alt="SSHM Logo" width="120" />
</p>

# 🚀 SSHM - SSH Manager

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![Release](https://img.shields.io/github/v/release/Gu1llaum-3/sshm?style=for-the-badge)](https://github.com/Gu1llaum-3/sshm/releases)
[![License](https://img.shields.io/github/license/Gu1llaum-3/sshm?style=for-the-badge)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey?style=for-the-badge)](https://github.com/Gu1llaum-3/sshm/releases)

> **A modern, interactive SSH Manager for your terminal** 🔥

SSHM is a beautiful command-line tool that transforms how you manage and connect to your SSH hosts. Built with Go and featuring an intuitive TUI interface, it makes SSH connection management effortless and enjoyable.

<p align="center">
    <a href="images/sshm.gif" target="_blank">
        <img src="images/sshm.gif" alt="Demo SSHM Terminal" width="800" />
    </a>
    <br>
    <em>🖱️ Click on the image to view in full size</em>
</p>

## ✨ Features

### 🚀 **Core Capabilities**
- **🎨 Beautiful TUI Interface** - Navigate your SSH hosts with an elegant, interactive terminal UI
- **⚡ Quick Connect** - Connect to any host instantly through the TUI or the CLI with `sshm <host>`
- **🔄 Port Forwarding** - Easy setup for Local, Remote, and Dynamic (SOCKS) forwarding with history persistence
- **📝 Easy Management** - Add, edit, move, and manage SSH configurations seamlessly
- **🏷️ Tag Support** - Organize your hosts with custom tags for better categorization
- **🔍 Smart Search** - Find hosts quickly with built-in filtering and search
- **📝 Real-time Status** - Live SSH connectivity indicators with asynchronous ping checks and color-coded status
- **🔔 Smart Updates** - Automatic version checking with update notifications
- **📈 Connection History** - Track your SSH connections with last login timestamps

### 🛠️ **Technical Features**
- **🔒 Secure** - Works directly with your existing `~/.ssh/config` file
- **📁 Custom Config Support** - Use any SSH configuration file with the `-c` flag
- **📂 SSH Include Support** - Full support for SSH Include directives to organize configurations across multiple files
- **⚙️ SSH Options Support** - Add any SSH configuration option through intuitive forms
- **🔄 Automatic Conversion** - Seamlessly converts between command-line and config formats
- **🔄 Automatic Backups** - Backup configurations automatically before changes
- **✅ Validation** - Prevent configuration errors with built-in validation
- **🔗 ProxyJump/ProxyCommand Support** - Secure connection tunneling through bastion hosts
- **⌨️ Keyboard Shortcuts** - Power user navigation with vim-like shortcuts
- **🌐 Cross-platform** - Supports Linux, macOS (Intel & Apple Silicon), and Windows
- **⚡ Lightweight** - Single binary with no dependencies, zero configuration required

## 🚀 Quick Start

### Installation

**Homebrew (Recommended for macOS):**
```bash
brew install Gu1llaum-3/sshm/sshm
```

**Unix/Linux/macOS (One-line install):**
```bash
curl -sSL https://raw.githubusercontent.com/Gu1llaum-3/sshm/main/install/unix.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/Gu1llaum-3/sshm/main/install/windows.ps1 | iex
```

**Alternative methods:**

*Linux/macOS:*
```bash
# Download specific release
wget https://github.com/Gu1llaum-3/sshm/releases/latest/download/sshm-linux-amd64.tar.gz

# Extract and install
tar -xzf sshm-linux-amd64.tar.gz
sudo mv sshm-linux-amd64 /usr/local/bin/sshm
```

*Windows:*
```powershell
# Download and extract
Invoke-WebRequest -Uri "https://github.com/Gu1llaum-3/sshm/releases/latest/download/sshm-windows-amd64.zip" -OutFile "sshm-windows-amd64.zip"
Expand-Archive sshm-windows-amd64.zip -DestinationPath C:\tools\
# Add C:\tools to your PATH environment variable
```

## 📖 Usage

### Interactive Mode

Launch SSHM without arguments to enter the beautiful TUI interface:

```bash
sshm
```

**Navigation:**
- `↑/↓` or `j/k` - Navigate hosts
- `Enter` - Connect to selected host
- `a` - Add new host
- `e` - Edit selected host
- `d` - Delete selected host
- `m` - Move host to another config file (requires SSH Include directives)
- `f` - Port forwarding setup
- `q` - Quit
- `/` - Search/filter hosts

**Real-time Status Indicators:**
- 🟢 **Online** - Host is reachable via SSH
- 🟡 **Connecting** - Currently checking host connectivity
- 🔴 **Offline** - Host is unreachable or SSH connection failed
- ⚫ **Unknown** - Connectivity status not yet determined

**Sorting & Filtering:**
- `s` - Switch between sorting modes (name ↔ last login)
- `n` - Sort by **name** (alphabetical)
- `r` - Sort by **recent** (last login time)
- `Tab` - Cycle between filtering modes
- Filter by **name** (default) - Search through host names
- Filter by **last login** - Sort and filter by most recently used connections

The interactive forms will guide you through configuration:
- **Hostname/IP** - Server address
- **Username** - SSH user
- **Port** - SSH port (default: 22)
- **Identity File** - Private key path
- **ProxyJump** - Jump server for connection tunneling
- **ProxyCommand** - Jump command for connection tunneling
- **SSH Options** - Additional SSH options in `-o` format (e.g., `-o Compression=yes -o ServerAliveInterval=60`)
- **Tags** - Comma-separated tags for organization

### Port Forwarding

SSHM provides an intuitive interface for setting up SSH port forwarding. Press `f` while selecting a host to open the port forwarding setup:

**Forward Types:**
- **Local (-L)** - Forward a local port to a remote host/port through the SSH connection
  - Example: Access a remote database on `localhost:5432` via local port `15432`
  - Use case: `ssh -L 15432:localhost:5432 server` → Database accessible on `localhost:15432`

- **Remote (-R)** - Forward a remote port back to a local host/port
  - Example: Expose local web server on remote host's port `8080`
  - Use case: `ssh -R 8080:localhost:3000 server` → Local app accessible from remote host's port 8080
  - ⚠️ **Requirements for external access:**
    - **SSH Server Config**: Add `GatewayPorts yes` to `/etc/ssh/sshd_config` and restart SSH service
    - **Firewall**: Open the remote port in the server's firewall (`ufw allow 8080` or equivalent)
    - **Port Availability**: Ensure the remote port is not already in use
    - **Bind Address**: Use `0.0.0.0` for external access, `127.0.0.1` for local-only

- **Dynamic (-D)** - Create a SOCKS proxy for secure browsing
  - Example: Route web traffic through the SSH connection
  - Use case: `ssh -D 1080 server` → Configure browser to use `localhost:1080` as SOCKS proxy
  - ⚠️ **Configuration requirements:**
    - **Browser Setup**: Configure SOCKS v5 proxy in browser settings
    - **DNS**: Enable "Proxy DNS when using SOCKS v5" for full privacy
    - **Applications**: Only SOCKS-aware applications will use the proxy
    - **Bind Address**: Use `127.0.0.1` for security (local access only)

**Port Forwarding Interface:**
- Choose forward type with ←/→ arrow keys
- Configure ports and addresses with guided forms
- Optional bind address configuration (defaults to 127.0.0.1)
- Real-time validation of port numbers and addresses
- **Port forwarding history** - Save frequently used configurations for quick reuse
- Connect automatically with configured forwarding options

**Troubleshooting Port Forwarding:**

*Remote Forwarding Issues:*
```bash
# Error: "remote port forwarding failed for listen port X"
# Solutions:
1. Check if port is already in use: ssh server "netstat -tln | grep :X"
2. Use a different port that's available
3. Enable GatewayPorts in SSH config for external access
```

*SSH Server Configuration for Remote Forwarding:*
```bash
# Edit SSH daemon config on the server:
sudo nano /etc/ssh/sshd_config

# Add or uncomment:
GatewayPorts yes

# Restart SSH service:
sudo systemctl restart sshd  # Ubuntu/Debian/CentOS 7+
# OR
sudo service ssh restart     # Older systems
```

*Firewall Configuration:*
```bash
# Ubuntu/Debian (UFW):
sudo ufw allow [port_number]

# CentOS/RHEL/Rocky (firewalld):
sudo firewall-cmd --add-port=[port_number]/tcp --permanent
sudo firewall-cmd --reload

# Check if port is accessible:
telnet [server_ip] [port_number]
```

*Dynamic Forwarding (SOCKS) Browser Setup:*
```
Firefox: about:preferences → Network Settings
- Manual proxy configuration
- SOCKS Host: localhost, Port: [your_port]
- SOCKS v5: ✓
- Proxy DNS when using SOCKS v5: ✓

Chrome: Launch with proxy
chrome --proxy-server="socks5://localhost:[your_port]"
```

### CLI Usage

SSHM provides both command-line operations and an interactive TUI interface:

```bash
# Launch interactive TUI mode for browsing and connecting to hosts
sshm

# Connect directly to a specific host (with history tracking)
sshm my-server

# Execute a command on a remote host
sshm my-server uptime

# Execute command with arguments
sshm my-server ls -la /var/log

# Force TTY allocation for interactive commands
sshm -t my-server sudo systemctl restart nginx

# Launch TUI with custom SSH config file
sshm -c /path/to/custom/ssh_config

# Connect directly with custom SSH config file
sshm my-server -c /path/to/custom/ssh_config

# Add a new host using interactive form
sshm add

# Add a new host with pre-filled hostname
sshm add hostname

# Add a new host with custom SSH config file
sshm add hostname -c /path/to/custom/ssh_config

# Edit an existing host configuration
sshm edit my-server

# Edit host with custom SSH config file
sshm edit my-server -c /path/to/custom/ssh_config

# Move a host to another SSH config file (requires Include directives)
sshm move my-server

# Move host with custom SSH config file (requires Include directives)
sshm move my-server -c /path/to/custom/ssh_config

# Search for hosts (interactive filter)
sshm search

# Print machine-readable info (JSON) for scripting
sshm info prod-server
sshm info prod-server --pretty

# With a custom SSH config file
sshm -c /path/to/custom/ssh_config info prod-server

# Pipe to jq
sshm info prod-server | jq -r '.result.target.hostname'
sshm info prod-server | jq -r '.result.target.user'

# Show version information (includes update check)
sshm --version

# Show help and available commands
sshm --help
```

### Host Info (JSON)

`sshm info <hostname>` prints a single JSON object to stdout so you can script against it with `jq`.

```bash
# Extract fields
sshm info prod-server | jq -r '.result.target.hostname'
sshm info prod-server | jq -r '.result.target.port'

# Check not-found (exit code 2)
sshm info does-not-exist | jq -r '.error.code'
```

### Shell Completion

SSHM supports shell completion for host names, making it easy to connect to hosts without typing full names:

```bash
sshm <TAB>           # Lists all available hosts
sshm pro<TAB>        # Completes to hosts starting with "pro" (e.g., prod-server)
```

**Setup Instructions:**

**Bash:**
```bash
# Enable for current session
source <(sshm completion bash)

# Enable permanently (add to ~/.bashrc)
echo 'source <(sshm completion bash)' >> ~/.bashrc
```

**Zsh:**
```bash
# Enable for current session
source <(sshm completion zsh)

# Enable permanently (add to ~/.zshrc)
echo 'source <(sshm completion zsh)' >> ~/.zshrc
```

**Fish:**
```bash
# Enable for current session
sshm completion fish | source

# Enable permanently
sshm completion fish > ~/.config/fish/completions/sshm.fish
```

**PowerShell:**
```powershell
# Enable for current session
sshm completion powershell | Out-String | Invoke-Expression

# Enable permanently (add to your PowerShell profile)
Add-Content $PROFILE 'sshm completion powershell | Out-String | Invoke-Expression'
```

### Direct Host Connection

SSHM supports direct connection to hosts via the command line, making it easy to integrate into your existing workflow:

```bash
# Connect directly to any configured host
sshm production-server
sshm db-staging
sshm web-01

# All direct connections are tracked in your history
# Use the TUI to see your most recently connected hosts
```

**Features of Direct Connection:**
- **Instant connection** - No TUI navigation required
- **History tracking** - All connections are recorded with timestamps
- **Error handling** - Clear messages if host doesn't exist or configuration issues
- **Config file support** - Works with custom config files using `-c` flag

### Remote Command Execution

Execute commands on remote hosts without opening an interactive shell:

```bash
# Execute a single command
sshm prod-server uptime

# Execute command with arguments
sshm prod-server ls -la /var/log

# Check disk usage
sshm prod-server df -h

# View logs (pipe to local commands)
sshm prod-server 'cat /var/log/nginx/access.log' | grep 404

# Force TTY allocation for interactive commands (sudo, vim, etc.)
sshm -t prod-server sudo systemctl restart nginx
```

**Features:**
- **Exit code propagation** - Remote command exit codes are passed through
- **TTY support** - Use `-t` flag for commands requiring terminal interaction
- **Pipe-friendly** - Output can be piped to local commands for processing
- **History tracking** - Command executions are recorded in connection history

### Backup Configuration

SSHM automatically creates backups of your SSH configuration files before making any changes to ensure your configurations are safe.

**Backup Location:**
- **Unix/Linux/macOS**: `~/.config/sshm/backups/` (or `$XDG_CONFIG_HOME/sshm/backups/` if set)
- **Windows**: `%APPDATA%\sshm\backups\` (fallback: `%USERPROFILE%\.config\sshm\backups\`)

**Key Features:**
- Automatic backup before any modification
- One backup per file (overwrites previous backup)
- Stored separately to avoid SSH Include conflicts
- Easy manual recovery if needed

**Additional Storage:**
- **Connection History**: Stored in the same config directory for persistent tracking
- **Port Forwarding History**: Saved configurations for quick reuse of common forwarding setups

**Quick Recovery:**
```bash
# Unix/Linux/macOS
cp ~/.config/sshm/backups/config.backup ~/.ssh/config

# Windows
copy "%APPDATA%\sshm\backups\config.backup" "%USERPROFILE%\.ssh\config"
```

### Configuration File Options

By default, SSHM uses the standard SSH configuration file at `~/.ssh/config`. You can specify a different configuration file using the `-c` flag:

```bash
# Use custom config file in TUI mode
sshm -c /path/to/custom/ssh_config

# Use custom config file with commands
sshm add hostname -c /path/to/custom/ssh_config
sshm edit hostname -c /path/to/custom/ssh_config
sshm move hostname -c /path/to/custom/ssh_config
```

### Advanced Features

#### Host Movement Between Config Files

SSHM provides a powerful `move` command to relocate SSH hosts between different configuration files. **This feature requires SSH Include directives to be present in your SSH configuration.**

```bash
# Move a host to another config file (requires Include directives)
sshm move my-server

# Move with custom config file (requires Include directives)
sshm move my-server -c /path/to/custom/ssh_config
```

**⚠️ Important Requirements:**
- **SSH Include directives must be present** in your SSH config file (either `~/.ssh/config` or the file specified with `-c`)
- The config file must contain `Include` statements referencing other SSH configuration files
- Without Include directives, the move command will display an error message

**Features:**
- **Interactive file selector** - Choose destination config file from Include directives
- **Include support** - Works seamlessly with SSH Include directives structure
- **Atomic operations** - Safe host movement with automatic backups
- **Validation** - Prevents conflicts and ensures configuration integrity
- **Error handling** - Clear messages when Include files are needed but not found

**Use Cases:**
- Reorganize hosts from main config to specialized include files
- Move development hosts to separate environment-specific configs
- Consolidate configurations for better organization

**Example Setup Required:**
Your main SSH config file must contain Include directives like:
```ssh
# ~/.ssh/config
Include ~/.ssh/config.d/*
Include work-servers.conf
Include projects/*.conf

Host personal-server
    HostName personal.example.com
    User myuser
```

#### Real-time Connectivity Status

SSHM features asynchronous SSH connectivity checking that provides visual indicators of host availability:

**Status Indicators:**
- 🟢 **Online** - SSH connection successful (shows response time)
- 🟡 **Connecting** - Currently testing connectivity
- 🔴 **Offline** - SSH connection failed or host unreachable
- ⚫ **Unknown** - Status not yet determined

**Features:**
- **Non-blocking checks** - Status updates happen in the background
- **Response time tracking** - See connection latency for online hosts
- **Automatic refresh** - Status indicators update continuously
- **Error details** - Detailed error information for failed connections

#### Automatic Update Checking

SSHM includes built-in version checking that notifies you of available updates:

**Features:**
- **Background checking** - Version check happens asynchronously
- **Release notifications** - Clear indicators when updates are available
- **Pre-release detection** - Identifies beta and development versions
- **GitHub integration** - Direct links to release pages
- **Non-intrusive** - Updates don't interrupt your workflow

**Update notifications appear:**
- In the main TUI interface as a subtle notification
- In the `sshm --version` command output
- Only when a newer stable version is available

#### Port Forwarding History

SSHM remembers your port forwarding configurations for easy reuse:

**Features:**
- **Automatic saving** - Successful forwarding setups are saved automatically
- **Quick reuse** - Previously used configurations appear as suggestions
- **Per-host history** - Forwarding history is tracked per SSH host
- **All forward types** - Supports Local (-L), Remote (-R), and Dynamic (-D) forwarding history
- **Persistent storage** - History survives application restarts

### Platform-Specific Notes

**Windows:**
- SSHM works with the built-in OpenSSH client (Windows 10/11)
- Configuration file location: `%USERPROFILE%\.ssh\config`
- Compatible with WSL SSH configurations
- Supports the same SSH options as Unix systems

**Unix/Linux/macOS:**
- Standard SSH configuration file: `~/.ssh/config`
- Full compatibility with OpenSSH features
- Preserves file permissions automatically

## 🏗️ Configuration

SSHM works directly with your standard SSH configuration file (`~/.ssh/config`). It adds special comment tags for enhanced functionality while maintaining full compatibility with standard SSH tools.

### SSH Include Support

SSHM fully supports SSH Include directives, allowing you to organize your SSH configurations across multiple files. This is particularly useful for managing large numbers of hosts or organizing configurations by environment, project, or team.

**Include Examples:**
```ssh
# Main ~/.ssh/config file
Host personal-server
    HostName personal.example.com
    User myuser

# Include work-related configurations
Include work-servers.conf

# Include all configurations from a directory
Include projects/*

# Include with relative paths
Include ~/.ssh/configs/production.conf
```

**Organization Examples:**

*work-servers.conf:*
```ssh
# Tags: work, production
Host prod-web-01
    HostName 10.0.1.10
    User deploy
    ProxyJump bastion.company.com

# Tags: work, staging  
Host staging-api
    HostName staging-api.company.com
    User developer
```

*projects/client-alpha.conf:*
```ssh
# Tags: client, development
Host client-alpha-dev
    HostName dev.client-alpha.com
    User admin
    Port 2222
```

**Example configuration:**
Include ~/.ssh/conf.d/*

```ssh
# Tags: production, web, frontend
Host web-prod-01
    HostName 192.168.1.10
    User deploy
    Port 22
    IdentityFile ~/.ssh/production_key
    Compression yes
    ServerAliveInterval 60

# Tags: development, database
Host db-dev
    HostName dev-db.company.com
    User admin
    Port 2222
    IdentityFile ~/.ssh/dev_key
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null

# Tags: production, backend
Host backend-prod
    HostName 10.0.1.50
    User app
    Port 22
    ProxyJump bastion.company.com
    ProxyCommand ssh -W %h:%p Jumphost
    IdentityFile ~/.ssh/production_key
    Compression yes
    ServerAliveInterval 300
    BatchMode yes
```

### Supported SSH Options

SSHM supports all standard SSH configuration options:

**Built-in Fields:**
- `HostName` - Server hostname or IP address
- `User` - Username for SSH connection
- `Port` - SSH port number
- `IdentityFile` - Path to private key file
- `ProxyJump` - Jump server for connection tunneling (e.g., `user@jumphost:port`)
- `ProxyCommand` - Jump command for connection tunneling (e.g, `ssh -W %h:%p Jumphost`)
- `Tags` - Custom tags (SSHM extension)

**Additional SSH Options:**
You can add any valid SSH option using the "SSH Options" field in the interactive forms. Enter them in command-line format (e.g., `-o Compression=yes -o ServerAliveInterval=60`) and SSHM will automatically convert them to the proper SSH config format.

**Common SSH Options:**
- `Compression` - Enable/disable compression (`yes`/`no`)
- `ServerAliveInterval` - Interval in seconds for keepalive messages
- `ServerAliveCountMax` - Maximum number of keepalive messages
- `StrictHostKeyChecking` - Host key verification (`yes`/`no`/`ask`)
- `UserKnownHostsFile` - Path to known hosts file
- `BatchMode` - Disable interactive prompts (`yes`/`no`)
- `ConnectTimeout` - Connection timeout in seconds
- `ControlMaster` - Connection multiplexing (`yes`/`no`/`auto`)
- `ControlPath` - Path for control socket
- `ControlPersist` - Keep connection alive duration
- `ForwardAgent` - Forward SSH agent (`yes`/`no`)
- `LocalForward` - Local port forwarding (e.g., `8080:localhost:80`)
- `RemoteForward` - Remote port forwarding
- `DynamicForward` - SOCKS proxy port forwarding

**Example usage in forms:**
```
SSH Options: -o Compression=yes -o ServerAliveInterval=60 -o StrictHostKeyChecking=no
```

This will be automatically converted to:
```ssh
    Compression yes
    ServerAliveInterval 60
    StrictHostKeyChecking no
```

### Custom Key Bindings

SSHM supports customizable key bindings through a configuration file. This is particularly useful for users who want to modify the default quit behavior.

**Configuration File Location:**
- **Linux/macOS**: `~/.config/sshm/config.json`
- **Windows**: `%APPDATA%\sshm\config.json`

**Example Configuration:**
```json
{
  "key_bindings": {
    "quit_keys": ["q", "ctrl+c"],
    "disable_esc_quit": true
  }
}
```

**Available Options:**
- **quit_keys**: Array of keys that will quit the application. Default: `["q", "ctrl+c"]`
- **disable_esc_quit**: Boolean flag to disable ESC key from quitting the application. Default: `false`

**For Vim Users:**
If you frequently press ESC accidentally causing the application to quit, set `disable_esc_quit` to `true`. This will disable ESC as a quit key while preserving all other functionality.

**Default Configuration:**
If no configuration file exists, SSHM will automatically create one with default settings that maintain backward compatibility.

## 🛠️ Development

### Prerequisites

- Go 1.23+ 
- Git

### Build from Source

```bash
# Clone the repository
git clone https://github.com/Gu1llaum-3/sshm.git
cd sshm

# Build the binary
go build -o sshm .

# Run
./sshm
```

### Project Structure

```
sshm/
├── main.go             # Application entry point
├── cmd/                # CLI commands (Cobra)
│   ├── root.go         # Root command and interactive mode
│   ├── add.go          # Add host command
│   ├── edit.go         # Edit host command
│   ├── move.go         # Move host command
│   └── search.go       # Search command
├── internal/
│   ├── config/         # SSH configuration management
│   │   └── ssh.go      # Config parsing and manipulation
│   ├── connectivity/   # SSH connectivity checking
│   │   └── ping.go     # Asynchronous SSH ping functionality
│   ├── history/        # Connection history tracking
│   │   ├── history.go  # History management and last login tracking
│   │   └── port_forward_test.go # Port forwarding history tests
│   ├── version/        # Version checking and updates
│   │   ├── version.go  # GitHub release checking and version comparison
│   │   └── version_test.go # Version parsing and comparison tests
│   ├── ui/             # Terminal UI components (Bubble Tea)
│   │   ├── tui.go      # Main TUI interface and program setup
│   │   ├── model.go    # Core TUI model and state
│   │   ├── update.go   # Message handling and state updates
│   │   ├── view.go     # UI rendering and layout
│   │   ├── table.go    # Host list table component with status indicators
│   │   ├── add_form.go # Add host form interface
│   │   ├── edit_form.go# Edit host form interface
│   │   ├── move_form.go# Move host form interface
│   │   ├── port_forward_form.go # Port forwarding setup with history
│   │   ├── styles.go   # Lip Gloss styling definitions
│   │   ├── sort.go     # Sorting and filtering logic
│   │   └── utils.go    # UI utility functions
│   └── validation/     # Input validation
│       └── ssh.go      # SSH config validation
├── images/             # Documentation assets
│   ├── logo.png        # Project logo
│   └── sshm.gif        # Demo animation
├── install/            # Installation scripts
│   ├── unix.sh         # Unix/Linux/macOS installer
│   └── README.md       # Installation guide
├── .github/            # GitHub configuration
│   ├── copilot-instructions.md # Development guidelines
│   └── workflows/      # CI/CD pipelines
│       └── build.yml   # Multi-platform builds
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── LICENSE             # MIT license
└── README.md           # Project documentation
```

### Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Go Crypto SSH](https://golang.org/x/crypto/ssh) - SSH connectivity checking

## 📦 Releases

Automated releases are built for multiple platforms:

| Platform | Architecture | Download |
|----------|-------------|----------|
| Linux | AMD64 | [sshm-linux-amd64.tar.gz](https://github.com/Gu1llaum-3/sshm/releases/latest/download/sshm-linux-amd64.tar.gz) |
| Linux | ARM64 | [sshm-linux-arm64.tar.gz](https://github.com/Gu1llaum-3/sshm/releases/latest/download/sshm-linux-arm64.tar.gz) |
| macOS | Intel | [sshm-darwin-amd64.tar.gz](https://github.com/Gu1llaum-3/sshm/releases/latest/download/sshm-darwin-amd64.tar.gz) |
| macOS | Apple Silicon | [sshm-darwin-arm64.tar.gz](https://github.com/Gu1llaum-3/sshm/releases/latest/download/sshm-darwin-arm64.tar.gz) |
| Windows | AMD64 | [sshm-windows-amd64.zip](https://github.com/Gu1llaum-3/sshm/releases/latest/download/sshm-windows-amd64.zip) |
| Windows | ARM64 | [sshm-windows-arm64.zip](https://github.com/Gu1llaum-3/sshm/releases/latest/download/sshm-windows-arm64.zip) |

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Workflow

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Charm](https://charm.sh/) for the amazing TUI libraries
- [Cobra](https://cobra.dev/) for the excellent CLI framework
- [@yimeng](https://github.com/yimeng) for contributing SSH Include directive support
- [@ldreux](https://github.com/ldreux) for contributing multi-word search functionality
- [@qingfengzxr](https://github.com/qingfengzxr) for contributing custom key bindings support
- The Go community for building such fantastic tools

---

<div align="center">

**Made with ❤️ by [Guillaume](https://github.com/Gu1llaum-3)**

⭐ **Star this repo if you found it useful!** ⭐

</div>
