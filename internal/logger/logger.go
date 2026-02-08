package logger

import (
	"fmt"
	"os"
	"sync"
)

var (
	defaultLogger Logger
	mu            sync.RWMutex
	initialized   bool
)

// Init 初始化全域 logger
func Init(config Config) error {
	mu.Lock()
	defer mu.Unlock()

	// Prevent duplicate initialization
	if initialized {
		return fmt.Errorf("logger already initialized; call Shutdown() before re-initializing")
	}

	// 檢查是否使用舊版 logger（回退機制）
	if os.Getenv("SYNCRULES_USE_LEGACY_LOGGER") == "true" {
		defaultLogger = NewLegacyLogger()
		initialized = true
		return nil
	}

	logger, err := NewSlogLogger(config)
	if err != nil {
		return fmt.Errorf("failed to create slog logger: %w", err)
	}

	defaultLogger = logger
	initialized = true
	return nil
}

// Get 取得全域 logger
func Get() Logger {
	mu.RLock()
	defer mu.RUnlock()

	if !initialized {
		// 未初始化時回傳 null logger（避免 panic）
		return &NullLogger{}
	}

	return defaultLogger
}

// With 建立帶 context 的子 logger
func With(args ...any) Logger {
	return Get().With(args...)
}

// Sync 強制 flush
func Sync() error {
	return Get().Sync()
}

// Shutdown 優雅關閉
func Shutdown() error {
	mu.Lock()
	if !initialized {
		mu.Unlock()
		return nil
	}

	logger := defaultLogger
	initialized = false
	mu.Unlock() // Release lock before calling logger.Shutdown() to avoid deadlock

	return logger.Shutdown()
}

// SetLevel 動態調整日誌級別（僅支援 legacy logger）
func SetLevel(level Level) {
	mu.RLock()
	defer mu.RUnlock()

	// slog 不支援動態調整，僅用於未來擴充
	if legacy, ok := defaultLogger.(*LegacyLogger); ok {
		legacy.SetLevel(level)
	}
}

// NullLogger 空 logger（不做任何事）
type NullLogger struct{}

func (n *NullLogger) Debug(msg string, args ...any) {}
func (n *NullLogger) Info(msg string, args ...any)  {}
func (n *NullLogger) Warn(msg string, args ...any)  {}
func (n *NullLogger) Error(msg string, args ...any) {}
func (n *NullLogger) With(args ...any) Logger       { return n }
func (n *NullLogger) Sync() error                   { return nil }
func (n *NullLogger) Shutdown() error               { return nil }
