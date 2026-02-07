package config

import (
	"fmt"
	"os"
	"path/filepath"

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

	return nil
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
