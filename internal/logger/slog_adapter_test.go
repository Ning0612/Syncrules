package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlogLogger_Basic(t *testing.T) {
	buf := &bytes.Buffer{}
	config := Config{
		Level:  LevelDebug,
		Format: FormatText,
		Outputs: []OutputConfig{
			{Type: OutputStdout, Writer: buf},
		},
	}

	logger, err := NewSlogLogger(config)
	if err != nil {
		t.Fatalf("NewSlogLogger() error = %v", err)
	}
	defer logger.Shutdown()

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("log output missing message: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("log output missing key-value: %s", output)
	}
}

func TestSlogLogger_Levels(t *testing.T) {
	tests := []struct {
		name      string
		level     Level
		logFunc   func(*SlogLogger)
		shouldLog bool
	}{
		{
			name:  "debug at debug level",
			level: LevelDebug,
			logFunc: func(l *SlogLogger) {
				l.Debug("debug msg")
			},
			shouldLog: true,
		},
		{
			name:  "debug at info level",
			level: LevelInfo,
			logFunc: func(l *SlogLogger) {
				l.Debug("debug msg")
			},
			shouldLog: false,
		},
		{
			name:  "error at warn level",
			level: LevelWarn,
			logFunc: func(l *SlogLogger) {
				l.Error("error msg")
			},
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			config := Config{
				Level:  tt.level,
				Format: FormatText,
				Outputs: []OutputConfig{
					{Type: OutputStdout, Writer: buf},
				},
			}

			logger, err := NewSlogLogger(config)
			if err != nil {
				t.Fatalf("NewSlogLogger() error = %v", err)
			}
			defer logger.Shutdown()

			tt.logFunc(logger)

			output := buf.String()
			hasLog := len(output) > 0

			if hasLog != tt.shouldLog {
				t.Errorf("expected shouldLog=%v, got hasLog=%v, output=%s",
					tt.shouldLog, hasLog, output)
			}
		})
	}
}

func TestSlogLogger_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	config := Config{
		Level:  LevelInfo,
		Format: FormatJSON,
		Outputs: []OutputConfig{
			{Type: OutputStdout, Writer: buf},
		},
	}

	logger, err := NewSlogLogger(config)
	if err != nil {
		t.Fatalf("NewSlogLogger() error = %v", err)
	}
	defer logger.Shutdown()

	logger.Info("test", "key", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"test"`) {
		t.Errorf("JSON output missing msg field: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("JSON output missing key field: %s", output)
	}
}

func TestSlogLogger_With(t *testing.T) {
	buf := &bytes.Buffer{}
	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Outputs: []OutputConfig{
			{Type: OutputStdout, Writer: buf},
		},
	}

	logger, err := NewSlogLogger(config)
	if err != nil {
		t.Fatalf("NewSlogLogger() error = %v", err)
	}
	defer logger.Shutdown()

	childLogger := logger.With("component", "test")
	childLogger.Info("message")

	output := buf.String()
	if !strings.Contains(output, "component=test") {
		t.Errorf("child logger output missing context: %s", output)
	}
}

func TestSlogLogger_Sanitization(t *testing.T) {
	buf := &bytes.Buffer{}
	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Outputs: []OutputConfig{
			{Type: OutputStdout, Writer: buf},
		},
	}

	logger, err := NewSlogLogger(config)
	if err != nil {
		t.Fatalf("NewSlogLogger() error = %v", err)
	}
	defer logger.Shutdown()

	logger.Info("user logged in", "password", "secret123")

	output := buf.String()
	if strings.Contains(output, "secret123") {
		t.Errorf("log output contains unsanitized password: %s", output)
	}
}

func TestSlogLogger_FileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		File: FileConfig{
			Enabled:    true,
			Path:       logPath,
			MaxSizeMB:  1,
			MaxAgeDays: 7,
			MaxBackups: 3,
			Compress:   false,
		},
		Outputs: []OutputConfig{
			{Type: OutputFile},
		},
	}

	logger, err := NewSlogLogger(config)
	if err != nil {
		t.Fatalf("NewSlogLogger() error = %v", err)
	}

	logger.Info("test file logging")
	logger.Shutdown()

	// 驗證檔案已建立
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("log file not created: %s", logPath)
	}

	// 讀取並驗證內容
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test file logging") {
		t.Errorf("log file missing message: %s", string(content))
	}
}

func TestSlogLogger_MultipleOutputs(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	config := Config{
		Level:  LevelInfo,
		Format: FormatText,
		Outputs: []OutputConfig{
			{Type: OutputStdout, Writer: buf1},
			{Type: OutputStderr, Writer: buf2},
		},
	}

	logger, err := NewSlogLogger(config)
	if err != nil {
		t.Fatalf("NewSlogLogger() error = %v", err)
	}
	defer logger.Shutdown()

	logger.Info("test multi-output")

	// 兩個 buffer 都應該有內容
	if !strings.Contains(buf1.String(), "test multi-output") {
		t.Errorf("buffer1 missing message")
	}
	if !strings.Contains(buf2.String(), "test multi-output") {
		t.Errorf("buffer2 missing message")
	}
}
