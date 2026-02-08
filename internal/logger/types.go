package logger

import (
	"io"
	"strings"
)

// Logger 統一日誌介面
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	Sync() error     // 強制 flush
	Shutdown() error // 優雅關閉
}

// Level 日誌級別
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

// ParseLevel parses a string into a Level (case-insensitive)
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo // default to info
	}
}

// Format 日誌格式
type Format int

const (
	FormatText Format = iota
	FormatJSON
)

// String returns the string representation of the format
func (f Format) String() string {
	switch f {
	case FormatText:
		return "text"
	case FormatJSON:
		return "json"
	default:
		return "text"
	}
}

// ParseFormat parses a string into a Format (case-insensitive)
func ParseFormat(s string) Format {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON
	case "text":
		return FormatText
	default:
		return FormatText
	}
}

// Output 日誌輸出目標
type Output int

const (
	OutputStdout Output = iota
	OutputStderr
	OutputFile
)

// Config 日誌配置
type Config struct {
	Level   Level
	Format  Format
	Outputs []OutputConfig
	File    FileConfig
	Daemon  DaemonConfig
}

// OutputConfig 輸出配置
type OutputConfig struct {
	Type   Output
	Writer io.Writer // 可選，用於測試
}

// FileConfig 檔案日誌配置
type FileConfig struct {
	Enabled    bool
	Path       string
	MaxSizeMB  int  // 單位：MB
	MaxAgeDays int  // 保留天數
	MaxBackups int  // 保留備份數
	Compress   bool // 是否壓縮
}

// DaemonConfig daemon 專用配置
type DaemonConfig struct {
	Enabled  bool
	Level    Level
	FilePath string
}
