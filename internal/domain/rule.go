package domain

// SyncRule defines a synchronization relationship between two endpoints
type SyncRule struct {
	// Name is the unique identifier for this rule
	Name string `mapstructure:"name"`

	// Mode defines the sync direction
	Mode SyncMode `mapstructure:"mode"`

	// SourceEndpoint name reference
	SourceEndpoint string `mapstructure:"source"`

	// TargetEndpoint name reference
	TargetEndpoint string `mapstructure:"target"`

	// IgnorePatterns glob patterns to exclude
	IgnorePatterns []string `mapstructure:"ignore"`

	// ConflictStrategy how to handle conflicts
	ConflictStrategy ConflictStrategy `mapstructure:"conflict"`

	// Enabled allows disabling rules without removing them
	Enabled bool `mapstructure:"enabled"`
}

// Validate checks if the rule is properly configured
func (r SyncRule) Validate() error {
	if r.Name == "" {
		return ErrInvalidRule
	}
	if r.SourceEndpoint == "" || r.TargetEndpoint == "" {
		return ErrInvalidRule
	}
	if r.SourceEndpoint == r.TargetEndpoint {
		return ErrInvalidRule // source and target cannot be the same
	}
	if !r.Mode.IsValid() {
		return ErrInvalidRule
	}
	if r.ConflictStrategy != "" && !r.ConflictStrategy.IsValid() {
		return ErrInvalidRule
	}
	return nil
}

// TransportType identifies the storage backend type
type TransportType string

const (
	TransportLocal  TransportType = "local"
	TransportGDrive TransportType = "gdrive"
)

// IsValid checks if the transport type is a known value
func (t TransportType) IsValid() bool {
	switch t {
	case TransportLocal, TransportGDrive:
		return true
	}
	return false
}

// Transport defines a storage backend configuration
type Transport struct {
	// Name is the unique identifier
	Name string `mapstructure:"name"`

	// Type identifies the backend
	Type TransportType `mapstructure:"type"`

	// Credentials path for auth (gdrive)
	Credentials string `mapstructure:"credentials"`
}

// Endpoint defines a specific location within a transport
type Endpoint struct {
	// Name is the unique identifier
	Name string `mapstructure:"name"`

	// Transport name reference
	Transport string `mapstructure:"transport"`

	// Root path within the transport
	Root string `mapstructure:"root"`
}
