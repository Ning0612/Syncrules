package adapter

import (
	"context"
	"io"

	"github.com/Ning0612/Syncrules/internal/domain"
)

// Adapter defines the interface for storage backends
// All implementations must handle path normalization internally
// and return domain-level errors for consistent error handling
type Adapter interface {
	// List returns all files and directories under the given path
	// Path should be relative to the adapter's root
	// Returns domain.ErrNotFound if path doesn't exist
	// Returns domain.ErrNotDirectory if path is a file
	List(ctx context.Context, path string) ([]domain.FileInfo, error)

	// Read opens a file for reading
	// Caller is responsible for closing the reader
	// Returns domain.ErrNotFound if file doesn't exist
	// Returns domain.ErrNotFile if path is a directory
	Read(ctx context.Context, path string) (io.ReadCloser, error)

	// Write creates or overwrites a file
	// Parent directories should be created automatically
	// Returns domain.ErrPermissionDenied if write not allowed
	Write(ctx context.Context, path string, r io.Reader) error

	// Delete removes a file or empty directory
	// Returns domain.ErrNotFound if path doesn't exist
	// Use DeleteRecursive for non-empty directories
	Delete(ctx context.Context, path string) error

	// Stat returns metadata for a single path
	// Returns domain.ErrNotFound if path doesn't exist
	Stat(ctx context.Context, path string) (domain.FileInfo, error)

	// Mkdir creates a directory and any necessary parents
	// No error if directory already exists
	Mkdir(ctx context.Context, path string) error

	// Exists checks if a path exists
	Exists(ctx context.Context, path string) (bool, error)

	// Close releases any resources held by the adapter
	Close() error
}

// AdapterFactory creates adapters for a given transport configuration
type AdapterFactory interface {
	// Create returns an adapter for the given transport and root path
	Create(transport domain.Transport, root string) (Adapter, error)

	// Supports returns true if this factory can handle the transport type
	Supports(transportType domain.TransportType) bool
}
