# Syncrules Configuration Examples

## Scheduler Configuration

### Basic Setup

Minimal configuration to enable scheduled syncs:

```yaml
scheduler:
  enabled: true
  default_interval: "5m"
```

### Common Intervals

```yaml
scheduler:
  enabled: true
  default_interval: "15m"  # Every 15 minutes
```

**Popular Intervals**:
- `1m` - Every minute (high-frequency)
- `5m` - Every 5 minutes (dev/testing)
- `15m` - Every 15 minutes (balanced)
- `1h` - Every hour (production)
- `6h` - Every 6 hours (low-priority)
- `24h` or `1d` - Daily

---

## Rule-Level Scheduling

### Example 1: Mixed Frequencies

```yaml
scheduler:
  enabled: true
  default_interval: "1h"

rules:
  # Critical files - sync frequently
  - name: work-documents
    enabled: true
    source: laptop-docs
    target: gdrive-backup
    schedule:
      enabled: true
      interval: "5m"  # Every 5 minutes

  # Config files - sync hourly (use default)
  - name: dotfiles
    enabled: true
    source: home-config
    target: github-backup
    # No schedule = use default interval (1h)

  # Archives - sync daily
  - name: photo-archives
    enabled: true
    source: photo-library
    target: nas-backup
    schedule:
      enabled: true
      interval: "24h"  # Daily

  # Manual only - exclude from daemon
  - name: large-media
    enabled: true
    source: video-projects
    target: external-drive
    schedule:
      enabled: false  # Run manually only
```

### Example 2: Development Setup

Quick iterations for testing:

```yaml
scheduler:
  enabled: true
  default_interval: "30s"  # Fast for development

rules:
  - name: test-sync
    enabled: true
    source: test-source
    target: test-target
    mode: bidirectional
    schedule:
      enabled: true  # Use 30s default
```

### Example 3: Production Setup

Conservative intervals for production:

```yaml
scheduler:
  enabled: true
  default_interval: "1h"  # Safe default

rules:
  - name: production-data
    enabled: true
    source: prod-server
    target: backup-server
    mode: mirror
    schedule:
      enabled: true
      interval: "15m"  # More frequent for critical data
      
  - name: logs
    enabled: true
    source: app-logs
    target: log-archive
    schedule:
      enabled: true
      interval: "6h"  # Less critical
```

---

## Complete Configuration Template

```yaml
# Global scheduler settings
scheduler:
  enabled: true
  default_interval: "1h"

# Storage backends
transports:
  - name: local
    type: filesystem

  - name: gdrive
    type: googledrive
    credentials_path: ~/.config/syncrules/gdrive-creds.json

# Sync locations
endpoints:
  - name: documents
    transport: local
    path: ~/Documents

  - name: gdrive-docs
    transport: gdrive
    path: /Backup/Documents

  - name: config
    transport: local
    path: ~/.config

  - name: gdrive-config
    transport: gdrive
    path: /Backup/Config

# Sync rules with scheduling
rules:
  # High-priority sync
  - name: important-docs
    enabled: true
    source: documents
    target: gdrive-docs
    mode: bidirectional
    schedule:
      enabled: true
      interval: "10m"
    conflict: newer

  # Normal priority sync
  - name: config-backup
    enabled: true
    source: config
    target: gdrive-config
    mode: mirror
    schedule:
      enabled: true
      # Uses default interval (1h)
    ignore:
      - "*.log"
      - "cache/*"

# Global settings
settings:
  lock_path: ~/.config/syncrules/locks
```

---

## Advanced Patterns

### Multi-Tier Backup

Different intervals for different backup tiers:

```yaml
scheduler:
  enabled: true
  default_interval: "1h"

rules:
  # Tier 1: Hot backup (frequent)
  - name: current-work
    source: active-projects
    target: local-backup
    schedule:
      enabled: true
      interval: "5m"

  # Tier 2: Warm backup (hourly)
  - name: recent-files
    source: active-projects
    target: nas-backup
    schedule:
      enabled: true
      interval: "1h"

  # Tier 3: Cold backup (daily)
  - name: archive-sync
    source: active-projects
    target: cloud-archive
    schedule:
      enabled: true
      interval: "24h"
```

### Selective Scheduling

Enable daemon only for specific rules:

```yaml
scheduler:
  enabled: true
  default_interval: "30m"

rules:
  # Auto-sync via daemon
  - name: auto-backup
    enabled: true
    schedule:
      enabled: true

  # Manual sync only
  - name: manual-transfer
    enabled: true
    schedule:
      enabled: false

  # Disabled completely
  - name: archived-rule
    enabled: false
```

---

## Platform-Specific Examples

### Windows

```yaml
scheduler:
  enabled: true
  default_interval: "15m"

endpoints:
  - name: documents
    transport: local
    path: C:\Users\username\Documents

  - name: backup
    transport: local
    path: D:\Backups\Documents

rules:
  - name: doc-backup
    source: documents
    target: backup
    schedule:
      enabled: true
```

### Linux/macOS

```yaml
scheduler:
  enabled: true
  default_interval: "1h"

endpoints:
  - name: home-docs
    transport: local
    path: ~/Documents

  - name: backup-mount
    transport: local
    path: /mnt/backup/documents

rules:
  - name: doc-backup
    source: home-docs
    target: backup-mount
    schedule:
      enabled: true
```

---

## Validation Tips

### Valid Intervals

✅ **Valid**:
- `30s`, `5m`, `1h`, `2h30m`
- Combinations: `1h30m`, `2h15m30s`

❌ **Invalid**:
- `5` (missing unit)
- `5minutes` (use `5m`)
- `1hour` (use `1h`)
- `-5m` (negative)
- `0s` (zero)

### Testing Your Configuration

```bash
# Load config and check for errors
syncrules sync --dry-run

# Verify scheduler is enabled
syncrules daemon start --interval 1m
# Ctrl+C after verifying it starts

# Check what rules will be scheduled
# (observe the GetScheduledRules behavior via logs)
```

---

## See Also

- [Daemon Usage Guide](daemon-usage.md) - How to use daemon commands
- [Main README](../README.md) - General Syncrules documentation
- [Walkthrough](walkthrough.md) - Implementation details
