package lock

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Ning0612/Syncrules/internal/testutil"
)

func TestNewFileLock(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// Verify lock path
	expectedPath := filepath.Join(dir, LockFileName)
	if lock.lockPath != expectedPath {
		t.Errorf("expected lock path %s, got %s", expectedPath, lock.lockPath)
	}

	// Verify default timeout
	if lock.staleTimeout != DefaultStaleTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultStaleTimeout, lock.staleTimeout)
	}
}

func TestAcquireRelease(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// Acquire lock
	if err := lock.Acquire("test-rule"); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	// Verify lock file exists
	if _, err := os.Stat(lock.lockPath); os.IsNotExist(err) {
		t.Error("lock file does not exist after acquire")
	}

	// Verify lock is held
	if !lock.IsLocked() {
		t.Error("lock should be held")
	}

	// Release lock
	if err := lock.Release(); err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Verify lock file is removed
	if _, err := os.Stat(lock.lockPath); !os.IsNotExist(err) {
		t.Error("lock file still exists after release")
	}

	// Verify lock is not held
	if lock.IsLocked() {
		t.Error("lock should not be held after release")
	}
}

func TestAcquireTwice_SameProcess(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// First acquire
	if err := lock.Acquire("test-rule-1"); err != nil {
		t.Fatalf("First Acquire failed: %v", err)
	}
	defer lock.Release()

	// Second acquire by same process should succeed (update rule name)
	if err := lock.Acquire("test-rule-2"); err != nil {
		t.Fatalf("Second Acquire by same process should succeed: %v", err)
	}

	// Verify rule name was updated
	holder, err := lock.GetHolder()
	if err != nil {
		t.Fatalf("GetHolder failed: %v", err)
	}
	if holder.RuleName != "test-rule-2" {
		t.Errorf("expected rule name 'test-rule-2', got '%s'", holder.RuleName)
	}
}

func TestConcurrentAcquire(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	const goroutines = 10
	var wg sync.WaitGroup
	acquired := make([]bool, goroutines)
	errors := make([]error, goroutines)

	// Launch multiple goroutines trying to acquire the lock
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			lock, err := NewFileLock(dir)
			if err != nil {
				errors[idx] = err
				return
			}

			err = lock.Acquire("concurrent-test")
			if err == nil {
				acquired[idx] = true
				// Hold lock briefly then release
				time.Sleep(10 * time.Millisecond)
				lock.Release()
			} else {
				errors[idx] = err
			}
		}(i)
	}

	wg.Wait()

	// Verify only one goroutine acquired the lock
	acquireCount := 0
	lockErrorCount := 0
	for i := 0; i < goroutines; i++ {
		if acquired[i] {
			acquireCount++
		}
		if errors[i] != nil && IsLockError(errors[i]) {
			lockErrorCount++
		}
	}

	if acquireCount != 1 {
		t.Errorf("expected exactly 1 acquire, got %d", acquireCount)
	}

	if lockErrorCount != goroutines-1 {
		t.Errorf("expected %d lock errors, got %d", goroutines-1, lockErrorCount)
	}
}

func TestAtomicLockCreation(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	const attempts = 100
	var wg sync.WaitGroup
	successes := make([]bool, attempts)

	// Rapidly try to acquire lock from multiple goroutines
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			lock, err := NewFileLock(dir)
			if err != nil {
				return
			}

			if err := lock.Acquire("atomic-test"); err == nil {
				successes[idx] = true
				time.Sleep(1 * time.Millisecond)
				lock.Release()
			}
		}(i)

		// Small delay to create race condition
		if i%10 == 0 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	wg.Wait()

	// Count successful acquisitions (should be reasonable, not all)
	successCount := 0
	for _, s := range successes {
		if s {
			successCount++
		}
	}

	// At least some should succeed (not deterministic due to timing)
	if successCount == 0 {
		t.Error("expected at least some successful acquisitions")
	}

	// Not all should succeed (would indicate no locking)
	if successCount == attempts {
		t.Error("all attempts succeeded, lock not working correctly")
	}

	t.Logf("Successful acquisitions: %d/%d", successCount, attempts)
}

func TestIsLocked(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// Should not be locked initially
	if lock.IsLocked() {
		t.Error("lock should not be held initially")
	}

	// Acquire and check
	lock.Acquire("test-rule")
	if !lock.IsLocked() {
		t.Error("lock should be held after acquire")
	}

	// Release and check
	lock.Release()
	if lock.IsLocked() {
		t.Error("lock should not be held after release")
	}
}

func TestGetHolder(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// No holder initially
	_, err = lock.GetHolder()
	if err == nil {
		t.Error("expected error when no lock is held")
	}

	// Acquire lock
	const ruleName = "test-rule-123"
	if err := lock.Acquire(ruleName); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	defer lock.Release()

	// Get holder info
	holder, err := lock.GetHolder()
	if err != nil {
		t.Fatalf("GetHolder failed: %v", err)
	}

	// Verify holder info
	if holder.PID != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), holder.PID)
	}

	hostname, _ := os.Hostname()
	if holder.Hostname != hostname {
		t.Errorf("expected hostname %s, got %s", hostname, holder.Hostname)
	}

	if holder.RuleName != ruleName {
		t.Errorf("expected rule name %s, got %s", ruleName, holder.RuleName)
	}

	if time.Since(holder.StartTime) > 1*time.Second {
		t.Error("start time should be recent")
	}
}

func TestForceRelease(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// Acquire lock
	lock.Acquire("test-rule")

	// Force release
	if err := lock.ForceRelease(); err != nil {
		t.Fatalf("ForceRelease failed: %v", err)
	}

	// Verify lock file is removed
	if _, err := os.Stat(lock.lockPath); !os.IsNotExist(err) {
		t.Error("lock file should be removed after force release")
	}

	// Verify not locked
	if lock.IsLocked() {
		t.Error("lock should not be held after force release")
	}
}

func TestStaleDetection_ProcessDead(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// Create lock info with non-existent PID
	hostname, _ := os.Hostname()
	staleInfo := &LockInfo{
		PID:       999999, // Unlikely to exist
		Hostname:  hostname,
		StartTime: time.Now().Add(-1 * time.Hour),
		RuleName:  "stale-test",
	}

	// Write stale lock info
	if err := lock.writeLockInfo(staleInfo); err != nil {
		t.Fatalf("failed to write stale lock info: %v", err)
	}

	// Try to acquire - should succeed by removing stale lock
	if err := lock.Acquire("new-rule"); err != nil {
		t.Fatalf("should acquire stale lock: %v", err)
	}
	defer lock.Release()

	// Verify new lock holder
	holder, err := lock.GetHolder()
	if err != nil {
		t.Fatalf("GetHolder failed: %v", err)
	}
	if holder.PID != os.Getpid() {
		t.Error("expected current process to be holder")
	}
}

func TestStaleDetection_LongRunning(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// Set very short stale timeout
	lock.SetStaleTimeout(100 * time.Millisecond)

	// Acquire lock
	if err := lock.Acquire("long-running"); err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	defer lock.Release()

	// Wait longer than stale timeout
	time.Sleep(200 * time.Millisecond)

	// Lock should STILL be held (not stale, process is alive)
	if !lock.IsLocked() {
		t.Error("long-running lock should not be considered stale")
	}

	// Try to acquire with another lock instance - should fail
	lock2, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	err = lock2.Acquire("competing")
	if err == nil {
		t.Error("should not acquire lock held by living process")
		lock2.Release()
	}
	if !IsLockError(err) {
		t.Errorf("expected LockError, got: %v", err)
	}
}

func TestStaleDetection_DifferentHost(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// Set short timeout for testing
	lock.SetStaleTimeout(100 * time.Millisecond)

	// Create lock info from different host (old enough to be stale)
	foreignInfo := &LockInfo{
		PID:       12345,
		Hostname:  "foreign-host-" + testutil.RandomString(8),
		StartTime: time.Now().Add(-1 * time.Hour),
		RuleName:  "foreign-rule",
	}

	if err := lock.writeLockInfo(foreignInfo); err != nil {
		t.Fatalf("failed to write foreign lock info: %v", err)
	}

	// Should be considered stale due to timeout (can't check process on different host)
	if err := lock.Acquire("local-rule"); err != nil {
		t.Fatalf("should acquire stale foreign lock: %v", err)
	}
	defer lock.Release()
}

func TestLockError(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock1, _ := NewFileLock(dir)
	lock2, _ := NewFileLock(dir)

	// Acquire with lock1
	lock1.Acquire("first")
	defer lock1.Release()

	// Try to acquire with lock2
	err := lock2.Acquire("second")
	if err == nil {
		t.Fatal("expected error when lock is held")
	}

	// Verify it's a LockError
	if !IsLockError(err) {
		t.Errorf("expected LockError, got: %T", err)
	}

	// Verify error message contains holder info
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("error message should not be empty")
	}
	t.Logf("Lock error message: %s", errMsg)
}

// Security Tests

func TestSecurity_StaleLock_OnlyDeadProcess(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	// Acquire lock
	lock.Acquire("security-test")

	// Read lock info
	info, err := lock.readLockInfo()
	if err != nil {
		t.Fatalf("failed to read lock info: %v", err)
	}

	// Verify lock is not considered stale (process is alive)
	if lock.isStale(info) {
		t.Error("lock with alive process should not be stale")
	}

	lock.Release()

	// Create lock with dead process
	deadInfo := &LockInfo{
		PID:       999999,
		Hostname:  info.Hostname,
		StartTime: time.Now().Add(-1 * time.Hour),
		RuleName:  "dead-process",
	}

	lock.writeLockInfo(deadInfo)

	// Verify dead process lock is stale
	if !lock.isStale(deadInfo) {
		t.Error("lock with dead process should be stale")
	}
}

func TestSetStaleTimeout(t *testing.T) {
	dir, cleanup := testutil.TempDir(t)
	defer cleanup()

	lock, err := NewFileLock(dir)
	if err != nil {
		t.Fatalf("NewFileLock failed: %v", err)
	}

	customTimeout := 5 * time.Minute
	lock.SetStaleTimeout(customTimeout)

	if lock.staleTimeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, lock.staleTimeout)
	}
}
