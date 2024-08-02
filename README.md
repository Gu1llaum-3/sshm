Voici le contenu complet du fichier `README.md` sous format code :

```markdown
# SSH Manager (sshm)

SSH Manager (sshm) is a bash script that simplifies and automates the management of SSH hosts through the SSH configuration file (`~/.ssh/config`). It provides functionalities to list, connect, view, add, edit, and delete SSH host configurations, as well as to check the availability of hosts using pings.

## Features

- List all SSH hosts in the configuration file.
- Connect to an SSH host by number or name.
- View the configuration details of a specific SSH host.
- Add a new SSH host configuration.
- Edit an existing SSH host configuration.
- Delete an SSH host configuration.
- Check the availability of an SSH host using ping.

## Requirements

- Bash
- SSH
- awk
- sed
- ping

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/sshm.git
   cd sshm
   ```

2. Make the script executable:

   ```bash
   chmod +x sshm
   ```

3. Move the script to a directory in your PATH, for example:

   ```bash
   sudo mv sshm /usr/local/bin/
   ```

## Usage

### List SSH Hosts

```bash
sshm list
```

### Connect to an SSH Host

```bash
sshm connect <name>
```

### View SSH Host Configuration

```bash
sshm view <name>
```

### Add a New SSH Host Configuration

```bash
sshm add
```

The script will prompt you to enter the host details.

### Edit an Existing SSH Host Configuration

```bash
sshm edit <name>
```

The script will prompt you to enter the new details for the host.

### Delete an SSH Host Configuration

```bash
sshm delete <name>
```

### Check SSH Host Availability

```bash
sshm ping <name>
```

## Example

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

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.