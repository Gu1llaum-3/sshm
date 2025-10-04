# SSHM Configuration

SSHM supports configurable key bindings through a configuration file located at:
- Linux/macOS: `~/.config/sshm/config.json`
- Windows: `%APPDATA%\sshm\config.json`

## Configuration Options

### Key Bindings

The key bindings section allows you to customize how you exit the application.

#### Example Configuration

```json
{
  "key_bindings": {
    "quit_keys": ["q", "ctrl+c"],
    "disable_esc_quit": true
  }
}
```

#### Options

- **quit_keys**: Array of keys that will quit the application. Default: `["q", "ctrl+c"]`
- **disable_esc_quit**: Boolean flag to disable ESC key from quitting the application. Default: `false`

## For Vim Users

If you're a vim user and frequently press ESC accidentally causing the application to quit, set `disable_esc_quit` to `true`:

```json
{
  "key_bindings": {
    "quit_keys": ["q", "ctrl+c"],
    "disable_esc_quit": true
  }
}
```

With this configuration:
- ESC will no longer quit the application
- You can still quit using 'q' or Ctrl+C
- All other functionality remains the same

## Default Configuration

If no configuration file exists, SSHM will create one with these defaults:

```json
{
  "key_bindings": {
    "quit_keys": ["q", "ctrl+c"],
    "disable_esc_quit": false
  }
}
```

This ensures backward compatibility - ESC will continue to work as a quit key by default.

## Configuration Location

The configuration file will be automatically created when you first run SSHM. You can manually edit it to customize the key bindings to your preference.

If you encounter any issues with the configuration file, you can delete it and SSHM will recreate it with default settings on the next run.