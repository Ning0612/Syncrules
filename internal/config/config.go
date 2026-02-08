package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Ning0612/Syncrules/internal/domain"
)

// Config represents the complete configuration for syncrules
type Config struct {
	// Transports define storage backend configurations
	Transports []domain.Transport `mapstructure:"transports"`

	// Endpoints define specific locations within transports
	Endpoints []domain.Endpoint `mapstructure:"endpoints"`

	// Rules define synchronization relationships
	Rules []domain.SyncRule `mapstructure:"rules"`

	// Settings define global configuration options
	Settings Settings `mapstructure:"settings"`

	// Scheduler defines global scheduler configuration
	Scheduler SchedulerConfig `mapstructure:"scheduler"`

	// Logging defines logging configuration
	Logging LoggingConfig `mapstructure:"logging"`
}

// Settings contains global configuration options
type Settings struct {
	// LockPath specifies the directory for lock files (default: ~/.config/syncrules/locks)
	LockPath string `mapstructure:"lock_path"`

	// DefaultConflict is the default conflict resolution strategy
	DefaultConflict string `mapstructure:"default_conflict"`

	// ChecksumAlgorithm specifies which algorithm to use (md5, sha256)
	ChecksumAlgorithm string `mapstructure:"checksum_algorithm"`

	// Verbose enables verbose logging
	Verbose bool `mapstructure:"verbose"`

	// DryRun enables dry-run mode by default
	DryRun bool `mapstructure:"dry_run"`
}

// SchedulerConfig contains scheduler configuration
type SchedulerConfig struct {
	// Enabled determines if scheduler is enabled globally
	Enabled bool `mapstructure:"enabled"`

	// DefaultInterval is the default sync interval (e.g., "5m", "1h")
	DefaultInterval string `mapstructure:"default_interval"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	// Level specifies the minimum log level ("debug", "info", "warn", "error")
	Level string `mapstructure:"level"`

	// Format specifies the log format ("text" or "json")
	Format string `mapstructure:"format"`

	// File contains file logging configuration
	File LogFileConfig `mapstructure:"file"`

	// Daemon contains daemon-specific logging configuration
	Daemon DaemonLoggingConfig `mapstructure:"daemon"`
}

// LogFileConfig contains file logging configuration
type LogFileConfig struct {
	// Enabled enables file logging
	Enabled bool `mapstructure:"enabled"`

	// Path specifies the log file path
	Path string `mapstructure:"path"`

	// MaxSizeMB specifies the maximum size in MB before rotation
	MaxSizeMB int `mapstructure:"max_size"`

	// MaxAgeDays specifies the maximum age in days to retain old log files
	MaxAgeDays int `mapstructure:"max_age"`

	// MaxBackups specifies the maximum number of old log files to retain
	MaxBackups int `mapstructure:"max_backups"`

	// Compress enables compression of rotated log files
	Compress bool `mapstructure:"compress"`
}

// DaemonLoggingConfig contains daemon-specific logging configuration
type DaemonLoggingConfig struct {
	// Enabled enables separate daemon logging
	Enabled bool `mapstructure:"enabled"`

	// Level specifies the log level for daemon (can differ from main level)
	Level string `mapstructure:"level"`

	// FilePath specifies the daemon log file path
	FilePath string `mapstructure:"file_path"`
}

// Validate checks if the configuration is complete and consistent
func (c *Config) Validate() error {
	// Check transport name uniqueness
	transportNames := make(map[string]bool)
	for _, t := range c.Transports {
		if t.Name == "" {
			return fmt.Errorf("%w: transport name cannot be empty", domain.ErrConfigInvalid)
		}
		if transportNames[t.Name] {
			return fmt.Errorf("%w: duplicate transport name: %s", domain.ErrConfigInvalid, t.Name)
		}
		if !t.Type.IsValid() {
			return fmt.Errorf("%w: invalid transport type: %s", domain.ErrConfigInvalid, t.Type)
		}
		transportNames[t.Name] = true
	}

	// Check endpoint name uniqueness and transport references
	endpointNames := make(map[string]bool)
	for _, e := range c.Endpoints {
		if e.Name == "" {
			return fmt.Errorf("%w: endpoint name cannot be empty", domain.ErrConfigInvalid)
		}
		if endpointNames[e.Name] {
			return fmt.Errorf("%w: duplicate endpoint name: %s", domain.ErrConfigInvalid, e.Name)
		}
		if e.Transport == "" {
			return fmt.Errorf("%w: endpoint %s has no transport", domain.ErrConfigInvalid, e.Name)
		}
		if !transportNames[e.Transport] {
			return fmt.Errorf("%w: endpoint %s references unknown transport: %s",
				domain.ErrTransportNotFound, e.Name, e.Transport)
		}
		if e.Root == "" {
			return fmt.Errorf("%w: endpoint %s has no root path", domain.ErrConfigInvalid, e.Name)
		}
		endpointNames[e.Name] = true
	}

	// Check rule name uniqueness and endpoint references
	ruleNames := make(map[string]bool)
	for _, r := range c.Rules {
		if r.Name == "" {
			return fmt.Errorf("%w: rule name cannot be empty", domain.ErrConfigInvalid)
		}
		if ruleNames[r.Name] {
			return fmt.Errorf("%w: duplicate rule name: %s", domain.ErrConfigInvalid, r.Name)
		}
		if !endpointNames[r.SourceEndpoint] {
			return fmt.Errorf("%w: rule %s references unknown source endpoint: %s",
				domain.ErrEndpointNotFound, r.Name, r.SourceEndpoint)
		}
		if !endpointNames[r.TargetEndpoint] {
			return fmt.Errorf("%w: rule %s references unknown target endpoint: %s",
				domain.ErrEndpointNotFound, r.Name, r.TargetEndpoint)
		}
		if err := r.Validate(); err != nil {
			return fmt.Errorf("rule %s: %w", r.Name, err)
		}
		ruleNames[r.Name] = true
	}

	// Validate scheduler configuration
	if c.Scheduler.DefaultInterval != "" {
		_, err := time.ParseDuration(c.Scheduler.DefaultInterval)
		if err != nil {
			return fmt.Errorf("%w: invalid scheduler.default_interval '%s': %v",
				domain.ErrConfigInvalid, c.Scheduler.DefaultInterval, err)
		}
	}

	// Validate rule-level schedule intervals
	for _, rule := range c.Rules {
		if rule.Schedule != nil && rule.Schedule.Interval != "" {
			_, err := time.ParseDuration(rule.Schedule.Interval)
			if err != nil {
				return fmt.Errorf("%w: invalid schedule.interval '%s' for rule '%s': %v",
					domain.ErrConfigInvalid, rule.Schedule.Interval, rule.Name, err)
			}
		}
	}

	return nil
}

// ApplyDefaults applies default values to the configuration
func (c *Config) ApplyDefaults() {
	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}

	// File logging defaults
	if c.Logging.File.Enabled {
		if c.Logging.File.MaxSizeMB == 0 {
			c.Logging.File.MaxSizeMB = 100
		}
		if c.Logging.File.MaxAgeDays == 0 {
			c.Logging.File.MaxAgeDays = 30
		}
		if c.Logging.File.MaxBackups == 0 {
			c.Logging.File.MaxBackups = 5
		}
		// Compress is false by default,保持原值
	}

	// Daemon logging defaults
	if c.Logging.Daemon.Enabled {
		if c.Logging.Daemon.Level == "" {
			c.Logging.Daemon.Level = c.Logging.Level // 與主配置相同
		}
	}
}

// GetTransport returns a transport by name
func (c *Config) GetTransport(name string) (*domain.Transport, error) {
	for i := range c.Transports {
		if c.Transports[i].Name == name {
			return &c.Transports[i], nil
		}
	}
	return nil, domain.ErrTransportNotFound
}

// GetEndpoint returns an endpoint by name
func (c *Config) GetEndpoint(name string) (*domain.Endpoint, error) {
	for i := range c.Endpoints {
		if c.Endpoints[i].Name == name {
			return &c.Endpoints[i], nil
		}
	}
	return nil, domain.ErrEndpointNotFound
}

// GetRule returns a rule by name
func (c *Config) GetRule(name string) (*domain.SyncRule, error) {
	for i := range c.Rules {
		if c.Rules[i].Name == name {
			return &c.Rules[i], nil
		}
	}
	return nil, domain.ErrInvalidRule
}

// GetEnabledRules returns all enabled rules
func (c *Config) GetEnabledRules() []domain.SyncRule {
	var rules []domain.SyncRule
	for _, r := range c.Rules {
		if r.Enabled {
			rules = append(rules, r)
		}
	}
	return rules
}

// GetScheduledRules returns rules that should be included in scheduled syncs
func (c *Config) GetScheduledRules() []domain.SyncRule {
	var rules []domain.SyncRule
	for _, r := range c.Rules {
		// Rule must be enabled
		if !r.Enabled {
			continue
		}

		// If rule has schedule config, check if scheduling is enabled
		if r.Schedule != nil {
			if r.Schedule.Enabled {
				rules = append(rules, r)
			}
		} else {
			// No schedule config means include in scheduled syncs by default
			rules = append(rules, r)
		}
	}
	return rules
}

// ExpandPath expands ~ and environment variables in a path
func ExpandPath(path string) string {
	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			if len(path) > 1 && (path[1] == '/' || path[1] == filepath.Separator) {
				path = filepath.Join(home, path[2:])
			} else if len(path) == 1 {
				path = home
			}
		}
	}
	// Expand environment variables
	path = os.ExpandEnv(path)
	return filepath.Clean(path)
}

// GetLockPath returns the lock directory path, using default if not configured
func (c *Config) GetLockPath() string {
	if c.Settings.LockPath != "" {
		return ExpandPath(c.Settings.LockPath)
	}

	// Default to ~/.config/syncrules/locks
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to current directory if UserConfigDir fails
		return filepath.Join(".", ".syncrules", "locks")
	}
	return filepath.Join(configDir, "syncrules", "locks")
}

// GetLogPath returns the log file path, using default if not configured
func (c *Config) GetLogPath() string {
	if c.Logging.File.Path != "" {
		return ExpandPath(c.Logging.File.Path)
	}

	// Default to ~/.config/syncrules/logs/syncrules.log
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to current directory if UserConfigDir fails
		return filepath.Join(".", ".syncrules", "logs", "syncrules.log")
	}
	return filepath.Join(configDir, "syncrules", "logs", "syncrules.log")
}

// GetDaemonLogPath returns the daemon log file path, using default if not configured
func (c *Config) GetDaemonLogPath() string {
	if c.Logging.Daemon.FilePath != "" {
		return ExpandPath(c.Logging.Daemon.FilePath)
	}

	// Default to ~/.config/syncrules/logs/daemon.log
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to current directory if UserConfigDir fails
		return filepath.Join(".", ".syncrules", "logs", "daemon.log")
	}
	return filepath.Join(configDir, "syncrules", "logs", "daemon.log")
}
