package progress

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Reporter handles progress reporting for sync operations
type Reporter interface {
	// Start begins tracking a new file transfer
	Start(path string, totalBytes int64)
	// Update reports progress on current transfer
	Update(bytesTransferred int64)
	// Complete marks the current transfer as complete
	Complete()
	// Error reports an error on current transfer
	Error(err error)
	// SetTotal sets the total number of files to process
	SetTotal(totalFiles int, totalBytes int64)
	// OverallProgress reports overall sync progress
	OverallProgress(filesCompleted int, bytesCompleted int64)
}

// Callback is a function that receives progress updates
type Callback func(update Update)

// Update represents a progress update
type Update struct {
	Type            UpdateType
	CurrentFile     string
	CurrentBytes    int64
	CurrentTotal    int64
	FilesCompleted  int
	FilesTotal      int
	BytesCompleted  int64
	BytesTotal      int64
	BytesPerSecond  float64
	Error           error
}

// UpdateType indicates the type of progress update
type UpdateType int

const (
	UpdateStart UpdateType = iota
	UpdateProgress
	UpdateComplete
	UpdateError
	UpdateOverall
)

// CallbackReporter implements Reporter with a callback function
type CallbackReporter struct {
	callback       Callback
	mu             sync.Mutex
	currentFile    string
	currentTotal   int64
	currentBytes   int64
	filesTotal     int
	bytesTotal     int64
	filesCompleted int
	bytesCompleted int64
	startTime      time.Time
}

// NewCallbackReporter creates a new CallbackReporter
func NewCallbackReporter(callback Callback) *CallbackReporter {
	return &CallbackReporter{
		callback: callback,
	}
}

// SetTotal sets the total number of files and bytes to sync
func (r *CallbackReporter) SetTotal(totalFiles int, totalBytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.filesTotal = totalFiles
	r.bytesTotal = totalBytes
}

// Start begins tracking a new file transfer
func (r *CallbackReporter) Start(path string, totalBytes int64) {
	r.mu.Lock()
	r.currentFile = path
	r.currentTotal = totalBytes
	r.currentBytes = 0
	r.startTime = time.Now()

	// Capture values for callback outside lock
	update := Update{
		Type:           UpdateStart,
		CurrentFile:    path,
		CurrentTotal:   totalBytes,
		FilesCompleted: r.filesCompleted,
		FilesTotal:     r.filesTotal,
		BytesCompleted: r.bytesCompleted,
		BytesTotal:     r.bytesTotal,
	}
	callback := r.callback
	r.mu.Unlock()

	// Call callback outside lock to prevent deadlock
	if callback != nil {
		callback(update)
	}
}

// Update reports progress on current transfer
func (r *CallbackReporter) Update(bytesTransferred int64) {
	r.mu.Lock()
	r.currentBytes = bytesTransferred

	var bytesPerSecond float64
	elapsed := time.Since(r.startTime).Seconds()
	if elapsed > 0 {
		bytesPerSecond = float64(bytesTransferred) / elapsed
	}

	update := Update{
		Type:           UpdateProgress,
		CurrentFile:    r.currentFile,
		CurrentBytes:   bytesTransferred,
		CurrentTotal:   r.currentTotal,
		FilesCompleted: r.filesCompleted,
		FilesTotal:     r.filesTotal,
		BytesCompleted: r.bytesCompleted + bytesTransferred,
		BytesTotal:     r.bytesTotal,
		BytesPerSecond: bytesPerSecond,
	}
	callback := r.callback
	r.mu.Unlock()

	if callback != nil {
		callback(update)
	}
}

// Complete marks the current transfer as complete
func (r *CallbackReporter) Complete() {
	r.mu.Lock()
	r.filesCompleted++
	r.bytesCompleted += r.currentTotal

	update := Update{
		Type:           UpdateComplete,
		CurrentFile:    r.currentFile,
		CurrentBytes:   r.currentTotal,
		CurrentTotal:   r.currentTotal,
		FilesCompleted: r.filesCompleted,
		FilesTotal:     r.filesTotal,
		BytesCompleted: r.bytesCompleted,
		BytesTotal:     r.bytesTotal,
	}
	callback := r.callback
	r.mu.Unlock()

	if callback != nil {
		callback(update)
	}
}

// Error reports an error on current transfer
func (r *CallbackReporter) Error(err error) {
	r.mu.Lock()
	update := Update{
		Type:           UpdateError,
		CurrentFile:    r.currentFile,
		FilesCompleted: r.filesCompleted,
		FilesTotal:     r.filesTotal,
		BytesCompleted: r.bytesCompleted,
		BytesTotal:     r.bytesTotal,
		Error:          err,
	}
	callback := r.callback
	r.mu.Unlock()

	if callback != nil {
		callback(update)
	}
}

// OverallProgress reports overall sync progress
func (r *CallbackReporter) OverallProgress(filesCompleted int, bytesCompleted int64) {
	r.mu.Lock()
	update := Update{
		Type:           UpdateOverall,
		FilesCompleted: filesCompleted,
		FilesTotal:     r.filesTotal,
		BytesCompleted: bytesCompleted,
		BytesTotal:     r.bytesTotal,
	}
	callback := r.callback
	r.mu.Unlock()

	if callback != nil {
		callback(update)
	}
}

// ProgressReader wraps an io.Reader to track read progress
type ProgressReader struct {
	reader      io.Reader
	reporter    Reporter
	transferred int64
}

// NewProgressReader creates a new progress-tracking reader
func NewProgressReader(r io.Reader, reporter Reporter) *ProgressReader {
	return &ProgressReader{
		reader:   r,
		reporter: reporter,
	}
}

// Read implements io.Reader
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	if n > 0 {
		pr.transferred += int64(n)
		if pr.reporter != nil {
			pr.reporter.Update(pr.transferred)
		}
	}
	return n, err
}

// ProgressWriter wraps an io.Writer to track write progress
type ProgressWriter struct {
	writer      io.Writer
	reporter    Reporter
	transferred int64
}

// NewProgressWriter creates a new progress-tracking writer
func NewProgressWriter(w io.Writer, reporter Reporter) *ProgressWriter {
	return &ProgressWriter{
		writer:   w,
		reporter: reporter,
	}
}

// Write implements io.Writer
func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.writer.Write(p)
	if n > 0 {
		pw.transferred += int64(n)
		if pw.reporter != nil {
			pw.reporter.Update(pw.transferred)
		}
	}
	return n, err
}

// NullReporter is a no-op reporter
type NullReporter struct{}

func (NullReporter) Start(path string, totalBytes int64)                  {}
func (NullReporter) Update(bytesTransferred int64)                        {}
func (NullReporter) Complete()                                            {}
func (NullReporter) Error(err error)                                      {}
func (NullReporter) SetTotal(totalFiles int, totalBytes int64)            {}
func (NullReporter) OverallProgress(filesCompleted int, bytesCompleted int64) {}

// FormatBytes formats bytes into human-readable string
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSpeed formats bytes per second into human-readable string
func FormatSpeed(bytesPerSecond float64) string {
	return FormatBytes(int64(bytesPerSecond)) + "/s"
}

// FormatProgress returns a progress bar string
func FormatProgress(current, total int64, width int) string {
	if total == 0 {
		return ""
	}

	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))
	if filled > width {
		filled = width
	}

	bar := make([]byte, width)
	for i := 0; i < width; i++ {
		if i < filled {
			bar[i] = '='
		} else if i == filled {
			bar[i] = '>'
		} else {
			bar[i] = ' '
		}
	}

	return fmt.Sprintf("[%s] %5.1f%%", string(bar), percent*100)
}
