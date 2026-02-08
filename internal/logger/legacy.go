package logger

import (
	"fmt"
	"os"
	"sync"
)

// LegacyLogger 舊版 logger（使用 fmt.Print*，用於回退）
type LegacyLogger struct {
	level Level
	mu    sync.RWMutex
}

// NewLegacyLogger 建立 legacy logger
func NewLegacyLogger() *LegacyLogger {
	return &LegacyLogger{
		level: LevelInfo,
	}
}

// SetLevel 設定日誌級別
func (l *LegacyLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// shouldLog 判斷是否應該記錄
func (l *LegacyLogger) shouldLog(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

// Debug 記錄 debug 級別日誌
func (l *LegacyLogger) Debug(msg string, args ...any) {
	if !l.shouldLog(LevelDebug) {
		return
	}
	fmt.Fprintf(os.Stdout, "[DEBUG] %s %v\n", msg, args)
}

// Info 記錄 info 級別日誌
func (l *LegacyLogger) Info(msg string, args ...any) {
	if !l.shouldLog(LevelInfo) {
		return
	}
	fmt.Fprintf(os.Stdout, "[INFO] %s %v\n", msg, args)
}

// Warn 記錄 warn 級別日誌
func (l *LegacyLogger) Warn(msg string, args ...any) {
	if !l.shouldLog(LevelWarn) {
		return
	}
	fmt.Fprintf(os.Stderr, "[WARN] %s %v\n", msg, args)
}

// Error 記錄 error 級別日誌
func (l *LegacyLogger) Error(msg string, args ...any) {
	if !l.shouldLog(LevelError) {
		return
	}
	fmt.Fprintf(os.Stderr, "[ERROR] %s %v\n", msg, args)
}

// With 建立帶 context 的子 logger（legacy 不支援，回傳自己）
func (l *LegacyLogger) With(args ...any) Logger {
	return l
}

// Sync 強制 flush
func (l *LegacyLogger) Sync() error {
	return nil
}

// Shutdown 優雅關閉
func (l *LegacyLogger) Shutdown() error {
	return nil
}
