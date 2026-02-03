package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/Ning0612/Syncrules/internal/domain"
)

// DefaultConfigPaths returns the default paths to search for config files
func DefaultConfigPaths() []string {
	paths := []string{
		".",
		"./configs",
	}

	// Add user config directory
	if configDir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(configDir, "syncrules"))
	}

	// Add home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config", "syncrules"))
		paths = append(paths, filepath.Join(homeDir, ".syncrules"))
	}

	return paths
}

// Load reads and parses a configuration file
// If path is empty, searches default locations for config.yaml
func Load(path string) (*Config, error) {
	v := viper.New()

	if path != "" {
		// Use specific file
		v.SetConfigFile(path)
	} else {
		// Search default paths
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		for _, p := range DefaultConfigPaths() {
			v.AddConfigPath(p)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrConfigInvalid, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrConfigInvalid, err)
	}

	// Set defaults for rules
	for i := range cfg.Rules {
		// Default to enabled if not specified
		if !v.IsSet(fmt.Sprintf("rules.%d.enabled", i)) {
			cfg.Rules[i].Enabled = true
		}
		// Default conflict strategy to manual
		if cfg.Rules[i].ConflictStrategy == "" {
			cfg.Rules[i].ConflictStrategy = domain.ConflictManual
		}
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadFromString parses configuration from a YAML string
func LoadFromString(yamlContent string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	if err := v.ReadConfig(strings.NewReader(yamlContent)); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrConfigInvalid, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrConfigInvalid, err)
	}

	// Set defaults
	for i := range cfg.Rules {
		if cfg.Rules[i].ConflictStrategy == "" {
			cfg.Rules[i].ConflictStrategy = domain.ConflictManual
		}
		// Default to enabled
		cfg.Rules[i].Enabled = true
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
