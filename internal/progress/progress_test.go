package progress

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestCallbackReporter_SetTotal tests setting total files and bytes
func TestCallbackReporter_SetTotal(t *testing.T) {
	var updates []Update
	var mu sync.Mutex

	reporter := NewCallbackReporter(func(u Update) {
		mu.Lock()
		updates = append(updates, u)
		mu.Unlock()
	})

	reporter.SetTotal(10, 1024*1024)

	// Trigger an update to verify SetTotal worked
	reporter.Start("test.txt", 100)

	mu.Lock()
	defer mu.Unlock()

	if len(updates) == 0 {
		t.Fatal("expected updates")
	}

	update := updates[0]
	if update.FilesTotal != 10 {
		t.Errorf("expected FilesTotal 10, got %d", update.FilesTotal)
	}
	if update.BytesTotal != 1024*1024 {
		t.Errorf("expected BytesTotal 1048576, got %d", update.BytesTotal)
	}
}

// TestCallbackReporter_Start tests starting a file transfer
func TestCallbackReporter_Start(t *testing.T) {
	var update Update
	reporter := NewCallbackReporter(func(u Update) {
		update = u
	})

	reporter.Start("test-file.txt", 500)

	if update.Type != UpdateStart {
		t.Errorf("expected UpdateStart, got %v", update.Type)
	}
	if update.CurrentFile != "test-file.txt" {
		t.Errorf("expected file name 'test-file.txt', got '%s'", update.CurrentFile)
	}
	if update.CurrentTotal != 500 {
		t.Errorf("expected total 500, got %d", update.CurrentTotal)
	}
}

// TestCallbackReporter_Update tests progress updates
func TestCallbackReporter_Update(t *testing.T) {
	var update Update
	reporter := NewCallbackReporter(func(u Update) {
		update = u
	})

	reporter.Start("test.txt", 1000)
	time.Sleep(5 * time.Millisecond) // Small delay for speed calculation
	reporter.Update(250)

	if update.Type != UpdateProgress {
		t.Errorf("expected UpdateProgress, got %v", update.Type)
	}
	if update.CurrentBytes != 250 {
		t.Errorf("expected 250 bytes, got %d", update.CurrentBytes)
	}
	if update.BytesPerSecond == 0 {
		t.Error("expected non-zero bytes per second")
	}
}

// TestCallbackReporter_Complete tests completion callback
func TestCallbackReporter_Complete(t *testing.T) {
	var updates []Update
	var mu sync.Mutex

	reporter := NewCallbackReporter(func(u Update) {
		mu.Lock()
		updates = append(updates, u)
		mu.Unlock()
	})

	reporter.SetTotal(3, 3000)
	reporter.Start("file1.txt", 1000)
	reporter.Complete()

	mu.Lock()
	defer mu.Unlock()

	// Find the complete update
	var completeUpdate *Update
	for i := range updates {
		if updates[i].Type == UpdateComplete {
			completeUpdate = &updates[i]
			break
		}
	}

	if completeUpdate == nil {
		t.Fatal("expected UpdateComplete")
	}

	if completeUpdate.FilesCompleted != 1 {
		t.Errorf("expected 1 file completed, got %d", completeUpdate.FilesCompleted)
	}
	if completeUpdate.BytesCompleted != 1000 {
		t.Errorf("expected 1000 bytes completed, got %d", completeUpdate.BytesCompleted)
	}
}

// TestCallbackReporter_Error tests error reporting
func TestCallbackReporter_Error(t *testing.T) {
	var update Update
	reporter := NewCallbackReporter(func(u Update) {
		update = u
	})

	reporter.Start("failing.txt", 100)
	testErr := io.ErrUnexpectedEOF
	reporter.Error(testErr)

	if update.Type != UpdateError {
		t.Errorf("expected UpdateError, got %v", update.Type)
	}
	if update.Error != testErr {
		t.Errorf("expected error %v, got %v", testErr, update.Error)
	}
}

// TestCallbackReporter_SpeedCalculation tests speed calculation
func TestCallbackReporter_SpeedCalculation(t *testing.T) {
	var lastUpdate Update
	reporter := NewCallbackReporter(func(u Update) {
		lastUpdate = u
	})

	reporter.Start("test.txt", 100000)
	time.Sleep(100 * time.Millisecond)
	reporter.Update(50000)

	// Speed should be roughly 500KB/s (50000 bytes / 0.1s)
	// Allow for some variation due to timing
	if lastUpdate.BytesPerSecond < 400000 || lastUpdate.BytesPerSecond > 600000 {
		t.Errorf("expected speed around 500000 B/s, got %.0f", lastUpdate.BytesPerSecond)
	}
}

// TestProgressReader tests the ProgressReader wrapper
func TestProgressReader(t *testing.T) {
	data := []byte("Hello, World!")
	reader := bytes.NewReader(data)

	var bytesRead int64
	reporter := NewCallbackReporter(func(u Update) {
		if u.Type == UpdateProgress {
			bytesRead = u.CurrentBytes
		}
	})

	reporter.Start("test.txt", int64(len(data)))
	pr := NewProgressReader(reader, reporter)

	buf := make([]byte, 1024)
	n, err := pr.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected to read %d bytes, got %d", len(data), n)
	}
	if bytesRead != int64(n) {
		t.Errorf("expected progress update of %d bytes, got %d", n, bytesRead)
	}
}

// TestProgressWriter tests the ProgressWriter wrapper
func TestProgressWriter(t *testing.T) {
	var buf bytes.Buffer
	data := []byte("Test data for writer")

	var bytesWritten int64
	reporter := NewCallbackReporter(func(u Update) {
		if u.Type == UpdateProgress {
			bytesWritten = u.CurrentBytes
		}
	})

	reporter.Start("test.txt", int64(len(data)))
	pw := NewProgressWriter(&buf, reporter)

	n, err := pw.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected to write %d bytes, got %d", len(data), n)
	}
	if bytesWritten != int64(n) {
		t.Errorf("expected progress update of %d bytes, got %d", n, bytesWritten)
	}
	if buf.String() != string(data) {
		t.Error("written data does not match")
	}
}

// TestCallbackReporter_Concurrent tests concurrent progress updates
func TestCallbackReporter_Concurrent(t *testing.T) {
	var mu sync.Mutex
	var updates []Update

	reporter := NewCallbackReporter(func(u Update) {
		mu.Lock()
		updates = append(updates, u)
		mu.Unlock()
	})

	reporter.SetTotal(10, 10000)

	// Simulate concurrent file transfers
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			reporter.Start("file.txt", 100)
			for j := 0; j < 10; j++ {
				reporter.Update(int64(j * 10))
				time.Sleep(1 * time.Millisecond)
			}
			reporter.Complete()
		}(i)
	}

	wg.Wait()

	// Verify we received updates (exact count may vary due to concurrency)
	mu.Lock()
	count := len(updates)
	mu.Unlock()

	if count == 0 {
		t.Error("expected some updates")
	}
	t.Logf("Received %d updates from concurrent operations", count)
}

// TestSecurity_CallbackDeadlock tests that callbacks don't cause deadlock
func TestSecurity_CallbackDeadlock(t *testing.T) {
	done := make(chan bool, 1)

	var reporter *CallbackReporter
	reporter = NewCallbackReporter(func(u Update) {
		// REAL re-entrance test: callback calls reporter methods
		// This would deadlock if reporter holds the lock during callback
		switch u.Type {
		case UpdateStart:
			reporter.Update(10) // ← Re-entrance attempt
		case UpdateProgress:
			reporter.OverallProgress(1, 100) // ← Another re-entrance
		case UpdateComplete:
			_ = u.FilesCompleted // Just read
		}
	})

	go func() {
		reporter.SetTotal(1, 100)
		reporter.Start("test.txt", 100)
		reporter.Update(50)
		reporter.Complete()
		done <- true
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock detected - callback was called while holding lock")
	}
}

// TestFormatBytes tests byte formatting
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024, "1.5 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536 * 1024 * 1024, "1.5 GB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.bytes)
		if got != tt.expected {
			t.Errorf("FormatBytes(%d) = %s, want %s", tt.bytes, got, tt.expected)
		}
	}
}

// TestFormatSpeed tests speed formatting
func TestFormatSpeed(t *testing.T) {
	speed := 1024.0 * 1024.0 // 1 MB/s
	result := FormatSpeed(speed)
	if result != "1.0 MB/s" {
		t.Errorf("FormatSpeed(1048576) = %s, want '1.0 MB/s'", result)
	}
}

// TestFormatProgress tests progress bar generation
func TestFormatProgress(t *testing.T) {
	tests := []struct {
		current  int64
		total    int64
		width    int
		contains string // Check if this string is in output
	}{
		{0, 100, 20, "[>"},       // Empty bar
		{50, 100, 20, "50.0%"},   // Half complete
		{100, 100, 20, "100.0%"}, // Full
		{0, 0, 20, ""},           // Zero total (empty result)
	}

	for _, tt := range tests {
		got := FormatProgress(tt.current, tt.total, tt.width)
		if tt.contains != "" && !strings.Contains(got, tt.contains) {
			t.Errorf("FormatProgress(%d, %d, %d) = %s, should contain '%s'",
				tt.current, tt.total, tt.width, got, tt.contains)
		}
	}
}

// TestNullReporter tests that NullReporter doesn't panic
func TestNullReporter(t *testing.T) {
	var nr NullReporter

	// Should not panic
	nr.SetTotal(10, 1000)
	nr.Start("test.txt", 100)
	nr.Update(50)
	nr.Complete()
	nr.Error(io.EOF)
	nr.OverallProgress(5, 500)
}
