# Syncrules Daemon Usage Guide

## Overview

The Syncrules daemon provides automated, scheduled synchronization of your configured rules. It runs in the background and executes syncs at regular intervals.

---

## Prerequisites

- Syncrules installed and configured
- A valid `sync.yaml` configuration file
- Scheduler enabled in configuration

---

## Configuration

### Enable the Scheduler

Edit your `sync.yaml` file:

```yaml
scheduler:
  enabled: true
  default_interval: "5m"  # Sync every 5 minutes
```

### Configure Rules for Scheduling

#### Include All Rules (Default)
```yaml
rules:
  - name: documents
    enabled: true
    source: local-home
    target: gdrive-backup
    # No schedule config = included in scheduled syncs
```

#### Exclude Specific Rules
```yaml
rules:
  - name: temp-sync
    enabled: true
    schedule:
      enabled: false  # Exclude from daemon
```

#### Custom Interval Per Rule
```yaml
rules:
  - name: critical-files
    enabled: true
    schedule:
      enabled: true
      interval: "1m"  # Sync every minute
      
  - name: archives
    enabled: true
    schedule:
      enabled: true
      interval: "1h"  # Sync every hour
```

---

## Starting the Daemon

### Foreground Mode (Testing)

For development and testing:

```bash
syncrules daemon start --interval 5m
```

Press `Ctrl+C` to stop.

### Background Mode (Production)

For production use:

```bash
syncrules daemon start --detach
```

The daemon will run in the background and survive terminal closure.

### Custom Configuration

```bash
syncrules daemon start --config /path/to/config.yaml --detach
```

### Custom Interval

Override the config default:

```bash
syncrules daemon start --interval 10m --detach
```

**Interval Formats**:
- `30s` - 30 seconds
- `5m` - 5 minutes
- `1h` - 1 hour
- `2h30m` - 2 hours 30 minutes

---

## Checking Status

```bash
syncrules daemon status
```

**Example Output**:

When running:
```
Status: running
  PID: 12345
  PID file: C:\Users\user\.config\syncrules\daemon.pid
```

When stopped:
```
Status: stopped
  (PID file not found or invalid: PID file does not exist...)
```

---

## Stopping the Daemon

```bash
syncrules daemon stop
```

The daemon will shut down gracefully, completing any in-progress syncs.

**Timeout**: 10 seconds (daemon is force-killed if it doesn't stop)

---

## PID File Location

### Default Locations

**Linux/Mac**:
```
~/.config/syncrules/daemon.pid
```

**Windows**:
```
C:\Users\<username>\.config\syncrules\daemon.pid
```

### Custom PID File

```bash
syncrules daemon start --pid-file /custom/path/daemon.pid --detach
```

---

## Common Usage Patterns

### 1. Development/Testing

```bash
# Start in foreground with short interval for quick testing
syncrules daemon start --interval 30s

# Watch the logs
# Press Ctrl+C when done
```

### 2. Production Deployment

```bash
# Start daemon in background
syncrules daemon start --detach

# Verify it's running
syncrules daemon status

# Later, when needed
syncrules daemon stop
```

### 3. System Service Integration

#### systemd (Linux)

Create `/etc/systemd/system/syncrules.service`:

```ini
[Unit]
Description=Syncrules Sync Daemon
After=network.target

[Service]
Type=forking
User=youruser
ExecStart=/usr/local/bin/syncrules daemon start --detach
ExecStop=/usr/local/bin/syncrules daemon stop
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable syncrules
sudo systemctl start syncrules
sudo systemctl status syncrules
```

#### launchd (macOS)

Create `~/Library/LaunchAgents/com.syncrules.daemon.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" 
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.syncrules.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/syncrules</string>
        <string>daemon</string>
        <string>start</string>
        <string>--detach</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load and start:
```bash
launchctl load ~/Library/LaunchAgents/com.syncrules.daemon.plist
launchctl start com.syncrules.daemon
```

---

## Troubleshooting

### Daemon Won't Start

**Error**: `scheduler is not enabled in configuration`

**Solution**: Add to `sync.yaml`:
```yaml
scheduler:
  enabled: true
  default_interval: "5m"
```

---

### Daemon Already Running

**Error**: `daemon is already running (PID file exists...)`

**Solutions**:
1. Check if actually running: `syncrules daemon status`
2. If not running, remove stale PID file manually
3. Or just start again (stale files are auto-cleaned)

---

### Invalid Interval

**Error**: `invalid interval format`

**Solution**: Use valid Go duration format:
- ✅ `5m`, `1h`, `30s`, `2h30m`
- ❌ `5`, `1hour`, `30sec`

---

### Permission Denied (Linux/Mac)

If PID file directory isn't writable:

```bash
# Create directory with correct permissions
mkdir -p ~/.config/syncrules
chmod 755 ~/.config/syncrules
```

Or specify custom location:
```bash
syncrules daemon start --pid-file /tmp/syncrules.pid --detach
```

---

## Best Practices

1. **Start Small**: Test with short intervals in foreground mode first
2. **Monitor Logs**: Check sync results before going to production
3. **Rule Selection**: Use `schedule.enabled: false` to exclude heavy rules
4. **Custom Intervals**: Critical files more frequently, archives less
5. **Graceful Stop**: Always use `daemon stop` rather than killing process

---

## See Also

- [Configuration Guide](config-examples.md) - More configuration examples
- [README](../README.md) - General Syncrules documentation
- [Walkthrough](walkthrough.md) - Implementation details
