package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// SlogLogger slog 實作
type SlogLogger struct {
	logger    *slog.Logger
	sanitizer *Sanitizer
	writers   []io.WriteCloser // 需要關閉的 writers
}

// NewSlogLogger 建立新的 slog logger
func NewSlogLogger(config Config) (*SlogLogger, error) {
	sanitizer := NewSanitizer()

	// 建立 multi-writer
	var writers []io.Writer
	var closeableWriters []io.WriteCloser

	// 根據配置新增輸出目標
	for _, output := range config.Outputs {
		switch output.Type {
		case OutputStdout:
			if output.Writer != nil {
				writers = append(writers, output.Writer)
				// Check if custom writer needs closing (exclude standard streams)
				if wc, ok := output.Writer.(io.WriteCloser); ok {
					if wc != os.Stdout && wc != os.Stderr && wc != os.Stdin {
						closeableWriters = append(closeableWriters, wc)
					}
				}
			} else {
				writers = append(writers, os.Stdout)
			}
		case OutputStderr:
			if output.Writer != nil {
				writers = append(writers, output.Writer)
				// Check if custom writer needs closing (exclude standard streams)
				if wc, ok := output.Writer.(io.WriteCloser); ok {
					if wc != os.Stdout && wc != os.Stderr && wc != os.Stdin {
						closeableWriters = append(closeableWriters, wc)
					}
				}
			} else {
				writers = append(writers, os.Stderr)
			}
		case OutputFile:
			if config.File.Enabled {
				fileWriter, err := createFileWriter(config.File)
				if err != nil {
					return nil, fmt.Errorf("failed to create file writer: %w", err)
				}
				writers = append(writers, fileWriter)
				closeableWriters = append(closeableWriters, fileWriter)
			}
		}
	}

	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	multiWriter := io.MultiWriter(writers...)

	// 建立 handler
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: convertLevel(config.Level),
	}

	switch config.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(multiWriter, opts)
	case FormatText:
		handler = slog.NewTextHandler(multiWriter, opts)
	default:
		handler = slog.NewTextHandler(multiWriter, opts)
	}

	return &SlogLogger{
		logger:    slog.New(handler),
		sanitizer: sanitizer,
		writers:   closeableWriters,
	}, nil
}

// createFileWriter 建立檔案 writer（使用 lumberjack 支援 rotation）
func createFileWriter(config FileConfig) (io.WriteCloser, error) {
	// Validate path is not empty
	if config.Path == "" {
		return nil, fmt.Errorf("log file path cannot be empty")
	}

	// 確保目錄存在
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	return &lumberjack.Logger{
		Filename:   config.Path,
		MaxSize:    config.MaxSizeMB,
		MaxAge:     config.MaxAgeDays,
		MaxBackups: config.MaxBackups,
		Compress:   config.Compress,
	}, nil
}

// convertLevel 轉換內部 Level 到 slog.Level
func convertLevel(level Level) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug 記錄 debug 級別日誌
func (l *SlogLogger) Debug(msg string, args ...any) {
	sanitizedMsg := l.sanitizer.Sanitize(msg)
	sanitizedArgs := l.sanitizer.SanitizeArgs(args)
	l.logger.Debug(sanitizedMsg, sanitizedArgs...)
}

// Info 記錄 info 級別日誌
func (l *SlogLogger) Info(msg string, args ...any) {
	sanitizedMsg := l.sanitizer.Sanitize(msg)
	sanitizedArgs := l.sanitizer.SanitizeArgs(args)
	l.logger.Info(sanitizedMsg, sanitizedArgs...)
}

// Warn 記錄 warn 級別日誌
func (l *SlogLogger) Warn(msg string, args ...any) {
	sanitizedMsg := l.sanitizer.Sanitize(msg)
	sanitizedArgs := l.sanitizer.SanitizeArgs(args)
	l.logger.Warn(sanitizedMsg, sanitizedArgs...)
}

// Error 記錄 error 級別日誌
func (l *SlogLogger) Error(msg string, args ...any) {
	sanitizedMsg := l.sanitizer.Sanitize(msg)
	sanitizedArgs := l.sanitizer.SanitizeArgs(args)
	l.logger.Error(sanitizedMsg, sanitizedArgs...)
}

// With 建立帶 context 的子 logger
// 子 logger 不擁有 writers，避免重複關閉
func (l *SlogLogger) With(args ...any) Logger {
	sanitizedArgs := l.sanitizer.SanitizeArgs(args)
	return &childLogger{
		logger:    l.logger.With(sanitizedArgs...),
		sanitizer: l.sanitizer,
	}
}

// Sync 強制 flush 所有緩衝
func (l *SlogLogger) Sync() error {
	// slog 本身沒有 Sync 方法，但 lumberjack 會自動 flush
	// 這裡保留介面以支援未來可能的需求
	return nil
}

// Shutdown 優雅關閉，flush 並關閉所有 writers
func (l *SlogLogger) Shutdown() error {
	var lastErr error
	for _, w := range l.writers {
		if err := w.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// childLogger 子 logger，不擁有 writers，避免重複關閉
type childLogger struct {
	logger    *slog.Logger
	sanitizer *Sanitizer
}

func (c *childLogger) Debug(msg string, args ...any) {
	sanitizedMsg := c.sanitizer.Sanitize(msg)
	sanitizedArgs := c.sanitizer.SanitizeArgs(args)
	c.logger.Debug(sanitizedMsg, sanitizedArgs...)
}

func (c *childLogger) Info(msg string, args ...any) {
	sanitizedMsg := c.sanitizer.Sanitize(msg)
	sanitizedArgs := c.sanitizer.SanitizeArgs(args)
	c.logger.Info(sanitizedMsg, sanitizedArgs...)
}

func (c *childLogger) Warn(msg string, args ...any) {
	sanitizedMsg := c.sanitizer.Sanitize(msg)
	sanitizedArgs := c.sanitizer.SanitizeArgs(args)
	c.logger.Warn(sanitizedMsg, sanitizedArgs...)
}

func (c *childLogger) Error(msg string, args ...any) {
	sanitizedMsg := c.sanitizer.Sanitize(msg)
	sanitizedArgs := c.sanitizer.SanitizeArgs(args)
	c.logger.Error(sanitizedMsg, sanitizedArgs...)
}

func (c *childLogger) With(args ...any) Logger {
	sanitizedArgs := c.sanitizer.SanitizeArgs(args)
	return &childLogger{
		logger:    c.logger.With(sanitizedArgs...),
		sanitizer: c.sanitizer,
	}
}

func (c *childLogger) Sync() error {
	return nil
}

func (c *childLogger) Shutdown() error {
	// Child logger 不擁有 writers，不執行關閉
	return nil
}
