# local-start-page

A lightweight personal start page that runs locally. Organize links into draggable, resizable groups with a custom background and per-group colors.

## Features

- **Link groups** — draggable, horizontally resizable, with custom background color per group
- **Links** — favicon auto-fetch, emoji or custom icon URL support
- **Background** — solid color, CSS value, or uploaded image
- **Edit mode** — toggle with the Edit button; drag groups, resize them, add/rename/delete links
- **Service mode** — install as a system service (launchd / systemd / Task Scheduler)

## Quick start

```bash
go build -o start-page .
./start-page
# open http://localhost:1221
```

## Command-line flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `./config.toml` | Path to config file |
| `--port` | `1221` | Port (overrides config; value is saved to config) |
| `--daemon` | — | Run server in the background (PID → `/tmp/local-start-page.pid`) |
| `--stop` | — | Stop background daemon |
| `--status` | — | Show daemon status |
| `--install` | — | Install as system service |
| `--uninstall` | — | Remove system service |

## Install as a service

The `--install` command copies the binary to a user-level location and registers it with the native service manager. No root or administrator privileges required.

```bash
./start-page --install              # uses port from config.toml (default 1221)
./start-page --install --port 8080  # install with a specific port
./start-page --uninstall            # remove the service
```

### macOS — launchd LaunchAgent

| Path | Value |
|------|-------|
| Binary | `~/.local/bin/local-start-page` |
| Config | `~/Library/Application Support/local-start-page/config.toml` |
| Logs | `~/Library/Logs/local-start-page/server.log` |
| Service file | `~/Library/LaunchAgents/com.local-start-page.plist` |

Starts on login, restarts automatically if it crashes (`KeepAlive`).

```bash
# View logs
tail -f ~/Library/Logs/local-start-page/server.log
```

### Linux — systemd user service

| Path | Value |
|------|-------|
| Binary | `~/.local/bin/local-start-page` |
| Config | `~/.config/local-start-page/config.toml` |
| Logs | `~/.config/local-start-page/server.log` |
| Service file | `~/.config/systemd/user/local-start-page.service` |

```bash
# View logs
journalctl --user -u local-start-page.service -f

# Enable autostart on headless servers (no graphical login)
loginctl enable-linger $USER
```

### Windows — Task Scheduler

| Path | Value |
|------|-------|
| Binary | `%LOCALAPPDATA%\local-start-page\local-start-page.exe` |
| Config | `%APPDATA%\local-start-page\config.toml` |
| Logs | `%APPDATA%\local-start-page\server.log` |
| Task name | `local-start-page` |

Starts on login. No administrator rights required.

```cmd
REM View logs
type %APPDATA%\local-start-page\server.log
```

## Background daemon (without install)

If you prefer not to install as a service, you can run the server detached from the terminal:

```bash
./start-page --daemon           # start in background
./start-page --status           # check if running
./start-page --stop             # stop
tail -f /tmp/local-start-page.log
```

## Configuration

`config.toml` is created automatically on first run.

```toml
background = "#0f0f1a"
port = 1221

[[group]]
name = "Dev"
color = ""          # optional CSS color, e.g. "#1a2a1a"

  [[group.link]]
  name = "GitHub"
  url  = "https://github.com"
  icon = ""         # emoji, https://… URL, or empty (auto favicon)
```

### Group background color

Each group can have an independent background color. Click **Edit → rename** on any group to open the group editor and pick a color. Leave the field empty to use the default surface color.

## Building

```bash
go build -o start-page .     # current platform
GOOS=linux  go build -o start-page-linux .
GOOS=darwin go build -o start-page-darwin .
GOOS=windows go build -o start-page.exe .
```
