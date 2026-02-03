package domain

// SyncRule defines a synchronization relationship between two endpoints
type SyncRule struct {
	// Name is the unique identifier for this rule
	Name string

	// Mode defines the sync direction
	Mode SyncMode

	// SourceEndpoint name reference
	SourceEndpoint string

	// TargetEndpoint name reference
	TargetEndpoint string

	// IgnorePatterns glob patterns to exclude
	IgnorePatterns []string

	// ConflictStrategy how to handle conflicts
	ConflictStrategy ConflictStrategy

	// Enabled allows disabling rules without removing them
	Enabled bool
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
		return ErrCircularDependency
	}
	return nil
}

// TransportType identifies the storage backend type
type TransportType string

const (
	TransportLocal  TransportType = "local"
	TransportGDrive TransportType = "gdrive"
)

// Transport defines a storage backend configuration
type Transport struct {
	// Name is the unique identifier
	Name string

	// Type identifies the backend
	Type TransportType

	// Credentials path for auth (gdrive)
	Credentials string
}

// Endpoint defines a specific location within a transport
type Endpoint struct {
	// Name is the unique identifier
	Name string

	// Transport name reference
	Transport string

	// Root path within the transport
	Root string
}
