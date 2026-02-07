package lock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ning0612/Syncrules/internal/testutil"
)

// TestAcquireTwice_ThenRelease is a regression test for the bug where
// re-acquiring with different RuleName updates file but not l.info,
// causing Release to fail with "lock stolen" error
func TestAcquireTwice_ThenRelease(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// First acquire
	if err := lock.Acquire("rule-a"); err != nil {
		t.Fatalf("First acquire failed: %v", err)
	}

	// Second acquire with different rule name (should succeed)
	if err := lock.Acquire("rule-b"); err != nil {
		t.Fatalf("Second acquire failed: %v", err)
	}

	// Release should succeed (NOT fail with "lock stolen")
	if err := lock.Release(); err != nil {
		t.Fatalf("Release after re-acquire failed: %v (this was the bug!)", err)
	}

	// Verify lock file is gone
	lockPath := filepath.Join(dir, LockFileName)
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("Lock file still exists after release")
	}
}

// TestAcquireTwice_RuleNamePersisted verifies RuleName is properly updated
func TestAcquireTwice_RuleNamePersisted(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// Acquire with rule-a
	if err := lock.Acquire("rule-a"); err != nil {
		t.Fatalf("First acquire failed: %v", err)
	}

	// Re-acquire with rule-b
	if err := lock.Acquire("rule-b"); err != nil {
		t.Fatalf("Second acquire failed: %v", err)
	}

	// Read lock info and verify RuleName was updated
	info, err := lock.readLockInfo()
	if err != nil {
		t.Fatalf("Failed to read lock info: %v", err)
	}

	if info.RuleName != "rule-b" {
		t.Errorf("Expected RuleName 'rule-b', got %q", info.RuleName)
	}

	// Also verify internal state matches
	if lock.info.RuleName != "rule-b" {
		t.Errorf("Internal l.info.RuleName should be 'rule-b', got %q", lock.info.RuleName)
	}

	lock.Release()
}
