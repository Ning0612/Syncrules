package lock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// LockFileName is the name of the lock file
	LockFileName = ".syncrules.lock"
	// DefaultStaleTimeout is the default duration after which a lock is considered stale
	DefaultStaleTimeout = 30 * time.Minute
)

// LockInfo contains metadata about the lock holder
type LockInfo struct {
	PID       int       `json:"pid"`
	Hostname  string    `json:"hostname"`
	StartTime time.Time `json:"start_time"`
	RuleName  string    `json:"rule_name,omitempty"`
}

// FileLock represents a file-based lock for preventing concurrent sync operations
type FileLock struct {
	lockPath     string
	staleTimeout time.Duration
	info         *LockInfo
}

// NewFileLock creates a new file lock instance
func NewFileLock(lockDir string) (*FileLock, error) {
	if lockDir == "" {
		// Default to user config directory
		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get config dir: %w", err)
		}
		lockDir = filepath.Join(configDir, "syncrules")
	}

	// Ensure lock directory exists
	if err := os.MkdirAll(lockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	return &FileLock{
		lockPath:     filepath.Join(lockDir, LockFileName),
		staleTimeout: DefaultStaleTimeout,
	}, nil
}

// SetStaleTimeout sets the duration after which a lock is considered stale
func (l *FileLock) SetStaleTimeout(d time.Duration) {
	l.staleTimeout = d
}

// Acquire attempts to acquire the lock
// Returns error if lock is already held by another process
func (l *FileLock) Acquire(ruleName string) error {
	// Check if this instance already holds the lock
	if l.info != nil {
		// This instance already holds the lock, just update rule name
		existingInfo, err := l.readLockInfo()
		if err == nil && l.isHeldByThisInstance(existingInfo) {
			existingInfo.RuleName = ruleName
			if err := l.writeLockInfo(existingInfo); err != nil {
				return err
			}
			// CRITICAL: Update l.info to match what we wrote to file
			// Otherwise Release() will fail thinking lock was stolen
			l.info.RuleName = ruleName
			return nil
		}
	}

	// Check for existing lock
	existingInfo, err := l.readLockInfo()
	if err == nil {
		// Lock file exists, check if it's stale
		if l.isStale(existingInfo) {
			// Remove stale lock
			if err := os.Remove(l.lockPath); err != nil {
				return fmt.Errorf("failed to remove stale lock: %w", err)
			}
		} else {
			// Lock is held by another process/instance
			return &LockError{
				Holder: existingInfo,
				Reason: "lock is held by another process",
			}
		}
	}

	// Create lock info
	hostname, _ := os.Hostname()
	info := &LockInfo{
		PID:       os.Getpid(),
		Hostname:  hostname,
		StartTime: time.Now(),
		RuleName:  ruleName,
	}

	// Try to create lock file atomically using O_CREATE|O_EXCL
	file, err := os.OpenFile(l.lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			// Another process acquired the lock between our check and create
			existingInfo, readErr := l.readLockInfo()
			if readErr != nil {
				return fmt.Errorf("lock acquisition race condition: %w", err)
			}
			return &LockError{
				Holder: existingInfo,
				Reason: "lock acquired by another process during acquisition",
			}
		}
		return fmt.Errorf("failed to create lock file: %w", err)
	}
	defer file.Close()

	// Write lock info
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(info); err != nil {
		os.Remove(l.lockPath)
		return fmt.Errorf("failed to write lock info: %w", err)
	}

	l.info = info
	return nil
}

// Release releases the lock
func (l *FileLock) Release() error {
	if l.info == nil {
		return nil // Not holding lock
	}

	// Verify we still own the lock before removing
	existingInfo, err := l.readLockInfo()
	if err != nil {
		l.info = nil
		return nil // Lock file doesn't exist, consider it released
	}

	if !l.isHeldByThisInstance(existingInfo) {
		l.info = nil
		return fmt.Errorf("lock was stolen by another process")
	}

	if err := os.Remove(l.lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	l.info = nil
	return nil
}

// IsLocked checks if a lock is currently held
func (l *FileLock) IsLocked() bool {
	info, err := l.readLockInfo()
	if err != nil {
		return false
	}
	return !l.isStale(info)
}

// GetHolder returns information about the current lock holder
func (l *FileLock) GetHolder() (*LockInfo, error) {
	info, err := l.readLockInfo()
	if err != nil {
		return nil, err
	}
	if l.isStale(info) {
		return nil, fmt.Errorf("lock is stale")
	}
	return info, nil
}

// ForceRelease forcibly removes the lock file
// Use with caution - only when certain the lock holder has crashed
func (l *FileLock) ForceRelease() error {
	if err := os.Remove(l.lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to force remove lock: %w", err)
	}
	l.info = nil
	return nil
}

// readLockInfo reads the lock information from file
func (l *FileLock) readLockInfo() (*LockInfo, error) {
	data, err := os.ReadFile(l.lockPath)
	if err != nil {
		return nil, err
	}

	var info LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("invalid lock file format: %w", err)
	}

	return &info, nil
}

// writeLockInfo writes lock information to file
func (l *FileLock) writeLockInfo(info *LockInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.lockPath, data, 0644)
}

// isStale checks if a lock is stale (process dead)
// Note: We only consider a lock stale if the process is dead, not based on timeout alone.
// Timeout is only used as a fallback for cross-host scenarios where we can't check process.
func (l *FileLock) isStale(info *LockInfo) bool {
	hostname, _ := os.Hostname()

	// Same host: check if process is still running
	if info.Hostname == hostname {
		if !processExists(info.PID) {
			return true
		}
		// Process is alive, lock is NOT stale regardless of timeout
		return false
	}

	// Different host: can't check process, use timeout as fallback
	// This is a conservative approach - only consider stale after timeout
	if time.Since(info.StartTime) > l.staleTimeout {
		return true
	}

	return false
}

// isHeldByCurrentProcess checks if the lock is held by the current process
func (l *FileLock) isHeldByCurrentProcess(info *LockInfo) bool {
	hostname, _ := os.Hostname()
	return info.PID == os.Getpid() && info.Hostname == hostname
}

// isHeldByThisInstance checks if the lock is held by this specific FileLock instance
func (l *FileLock) isHeldByThisInstance(info *LockInfo) bool {
	if l.info == nil {
		return false
	}
	return l.isHeldByCurrentProcess(info) &&
		l.info.StartTime.Equal(info.StartTime) &&
		l.info.RuleName == info.RuleName
}

// LockError represents an error when lock cannot be acquired
type LockError struct {
	Holder *LockInfo
	Reason string
}

func (e *LockError) Error() string {
	if e.Holder != nil {
		return fmt.Sprintf("cannot acquire lock: %s (held by PID %d on %s since %s, rule: %s)",
			e.Reason,
			e.Holder.PID,
			e.Holder.Hostname,
			e.Holder.StartTime.Format(time.RFC3339),
			e.Holder.RuleName,
		)
	}
	return fmt.Sprintf("cannot acquire lock: %s", e.Reason)
}

// IsLockError checks if an error is a LockError
func IsLockError(err error) bool {
	_, ok := err.(*LockError)
	return ok
}
