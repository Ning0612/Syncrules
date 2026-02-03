package domain

import "errors"

// Adapter errors - 儲存適配器層錯誤
var (
	// ErrNotFound indicates the requested resource does not exist
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists indicates the resource already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrPermissionDenied indicates insufficient permissions
	ErrPermissionDenied = errors.New("permission denied")

	// ErrNotDirectory indicates expected a directory but got a file
	ErrNotDirectory = errors.New("not a directory")

	// ErrNotFile indicates expected a file but got a directory
	ErrNotFile = errors.New("not a file")

	// ErrVersionConflict indicates a version/etag mismatch during update
	ErrVersionConflict = errors.New("version conflict")

	// ErrQuotaExceeded indicates storage quota has been exceeded
	ErrQuotaExceeded = errors.New("quota exceeded")

	// ErrNetworkError indicates a network-related failure
	ErrNetworkError = errors.New("network error")

	// ErrTimeout indicates operation timed out
	ErrTimeout = errors.New("operation timed out")
)

// Sync errors - 同步邏輯層錯誤
var (
	// ErrSyncConflict indicates an unresolved sync conflict
	ErrSyncConflict = errors.New("sync conflict")

	// ErrInvalidRule indicates a malformed sync rule
	ErrInvalidRule = errors.New("invalid sync rule")

	// ErrCircularDependency indicates circular sync rule dependencies
	ErrCircularDependency = errors.New("circular dependency detected")

	// ErrSyncInProgress indicates another sync is already running
	ErrSyncInProgress = errors.New("sync already in progress")
)

// Config errors - 設定檔錯誤
var (
	// ErrConfigNotFound indicates config file not found
	ErrConfigNotFound = errors.New("config file not found")

	// ErrConfigInvalid indicates config file is malformed
	ErrConfigInvalid = errors.New("invalid config")

	// ErrEndpointNotFound indicates referenced endpoint doesn't exist
	ErrEndpointNotFound = errors.New("endpoint not found")

	// ErrTransportNotFound indicates referenced transport doesn't exist
	ErrTransportNotFound = errors.New("transport not found")
)
