# SSH Manager (sshm)

SSH Manager (sshm) is a bash script that simplifies and automates the management of SSH hosts through the SSH configuration file (`~/.ssh/config`). It provides functionalities to list, connect, view, add, edit, and delete SSH host configurations, check the availability of hosts using pings, and manage different SSH configuration contexts.

## Features

- List all SSH hosts in the configuration file (with optional ping check).
- Connect to an SSH host by name or number from the list.
- View the configuration details of a specific SSH host.
- Add a new SSH host configuration.
- Edit an existing SSH host configuration.
- Delete an SSH host configuration.
- Check the availability of an SSH host using ping.
- Manage multiple SSH configuration contexts.

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

## Example

### Listing and Connecting to SSH Hosts

```bash
# Quick list without ping check (fast)
sshm list

# List with availability check (slower if hosts are down)
sshm list --ping
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

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
