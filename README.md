# SSH Manager (sshm)

SSH Manager (sshm) is a bash script that simplifies and automates the management of SSH hosts through the SSH configuration file (`~/.ssh/config`). It provides functionalities to list, connect, view, add, edit, and delete SSH host configurations, check the availability of hosts using pings, and manage different SSH configuration contexts.

## Features

- List all SSH hosts in the configuration file (with optional ping check).
- Filter SSH hosts by tags for better organization.
- Connect to an SSH host by name or number from the list.
- View the configuration details of a specific SSH host.
- Add a new SSH host configuration.
- Edit an existing SSH host configuration.
- Delete an SSH host configuration.
- Check the availability of an SSH host using ping.
- Manage multiple SSH configuration contexts.
- Upgrade sshm to the latest version automatically.

## Requirements

- Bash
- SSH
- awk
- sed
- ping

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/Gu1llaum-3/sshm.git
   cd sshm
   ```

2. Make the script executable:

   ```bash
   chmod +x sshm
   ```

3. Move the script to a directory in your PATH, for example:

   ```bash
   sudo mv sshm.bash /usr/local/bin/sshm
   ```

## Usage

### List SSH Hosts

```bash
sshm list
```

To check host availability with ping (may be slower if hosts are unreachable):

```bash
sshm list --ping
```

To filter hosts by a specific tag:

```bash
sshm list --tag production
```

You can combine options:

```bash
sshm list --ping --tag staging
```

### Connect to an SSH Host

```bash
sshm <host>
```

You can also connect by selecting a number from the `sshm list` output:

```bash
sshm list
# Select a number when prompted, e.g., type "1" to connect to the first host
```

### View SSH Host Configuration

```bash
sshm view <host>
```

### Add a New SSH Host Configuration

```bash
sshm add
```

The script will prompt you to enter the host details.

### Edit an Existing SSH Host Configuration

```bash
sshm edit <host>
```

The script will prompt you to enter the new details for the host.

### Delete an SSH Host Configuration

```bash
sshm delete <host>
```

### Check SSH Host Availability

```bash
sshm ping <host>
```

### Manage SSH Contexts

#### List Available Contexts

```bash
sshm context list
```

This will list all available SSH configuration contexts and highlight the currently active one.

#### Switch to a Different Context

```bash
sshm context use <context_name>
```

Switches the active SSH configuration to the specified context.

#### Create a New Context

```bash
sshm context create <context_name>
```

Creates a new SSH configuration context.

#### Delete a Context

```bash
sshm context delete <context_name>
```

Deletes the specified SSH configuration context.

### Upgrade sshm

```bash
sshm upgrade
```

Automatically checks for the latest version on GitHub and upgrades sshm if a newer version is available. The script will:
- Check for updates from the GitHub repository
- Display the current and available versions
- Show the installation path (`/usr/local/bin/sshm`)
- Ask for confirmation before proceeding
- Download and install the latest version
- Verify the installation

The upgrade will attempt to install to `/usr/local/bin/sshm` (may require sudo), and fall back to `~/.local/bin/sshm` if needed.

## Example

### Listing and Connecting to SSH Hosts

```bash
# Quick list without ping check (fast)
sshm list

# List with availability check (slower if hosts are down)
sshm list --ping

# Filter by specific tag
sshm list --tag production

# Combine filtering and ping check
sshm list --ping --tag staging
```

### Adding a New SSH Host

```bash
sshm add
```

You will be prompted to enter the following details:
- Host name
- HostName (IP address or domain)
- User (default: current user)
- Port (default: 22)
- IdentityFile (default: `~/.ssh/id_rsa`)
- ProxyJump (optional)
- Tags (optional, comma-separated)

### Editing an Existing SSH Host

```bash
sshm edit myhost
```

You will be prompted to update the details for the host `myhost`.

### Viewing a Host Configuration

```bash
sshm view myhost
```

### Checking Host Availability

```bash
sshm ping myhost
```

### Switching to a Different SSH Context

```bash
sshm context use myconfig
```

Switches to the `myconfig` SSH configuration context.

### Upgrading sshm

```bash
sshm upgrade
```

Checks for and installs the latest version of sshm. The command will show you the current version, the available version, and ask for confirmation before upgrading.

## Tags

SSH Manager supports tagging hosts for better organization and filtering. Tags are comma-separated labels that help you categorize your SSH hosts.

### Using Tags

When adding or editing a host, you can specify tags:

```bash
sshm add
# When prompted, enter tags like: production, webserver, ubuntu
```

### Filtering by Tags

Use the `--tag` option to filter hosts:

```bash
# Show only production hosts
sshm list --tag production

# Show only development hosts with ping check
sshm list --ping --tag development
```

### Tag Display

Tags are displayed in the list view with a `#` prefix and are automatically sorted alphabetically:
- Tags: `#database #production #ubuntu`

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
