package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Manager handles state persistence and execution history
type Manager struct {
	db *sql.DB
}

// ExecutionRecord represents a single sync execution
type ExecutionRecord struct {
	ID          int64
	RuleName    string
	StartTime   time.Time
	EndTime     time.Time
	Status      string // "success", "failed", "partial"
	FilesSynced int
	BytesSynced int64
	Error       string
}

// NewManager creates a new state manager
func NewManager(dataDir string) (*Manager, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("data directory cannot be empty")
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "syncrules.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Limit connection pool to prevent "database is locked" errors
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Enable WAL mode for better concurrency and set busy timeout
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode and busy timeout: %w", err)
	}

	manager := &Manager{db: db}

	// Initialize schema
	if err := manager.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return manager, nil
}

// initSchema creates the database schema
func (m *Manager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		rule_name TEXT NOT NULL,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP NOT NULL,
		status TEXT NOT NULL,
		files_synced INTEGER DEFAULT 0,
		bytes_synced INTEGER DEFAULT 0,
		error TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_executions_rule_time ON executions(rule_name, start_time DESC);
	CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
	`

	_, err := m.db.Exec(schema)
	return err
}

// SaveExecution records a sync execution
func (m *Manager) SaveExecution(record ExecutionRecord) error {
	// Validate status
	if record.Status != "success" && record.Status != "failed" && record.Status != "partial" {
		return fmt.Errorf("invalid status: %s (must be 'success', 'failed', or 'partial')", record.Status)
	}

	query := `
		INSERT INTO executions (rule_name, start_time, end_time, status, files_synced, bytes_synced, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := m.db.Exec(query,
		record.RuleName,
		record.StartTime,
		record.EndTime,
		record.Status,
		record.FilesSynced,
		record.BytesSynced,
		record.Error,
	)

	if err != nil {
		return fmt.Errorf("failed to save execution record: %w", err)
	}

	return nil
}

// GetHistory retrieves execution history for a rule
func (m *Manager) GetHistory(ruleName string, limit int) ([]ExecutionRecord, error) {
	// Validate limit
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	query := `
		SELECT id, rule_name, start_time, end_time, status, files_synced, bytes_synced, error
		FROM executions
		WHERE rule_name = ?
		ORDER BY start_time DESC
		LIMIT ?
	`

	rows, err := m.db.Query(query, ruleName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var records []ExecutionRecord
	for rows.Next() {
		var record ExecutionRecord
		err := rows.Scan(
			&record.ID,
			&record.RuleName,
			&record.StartTime,
			&record.EndTime,
			&record.Status,
			&record.FilesSynced,
			&record.BytesSynced,
			&record.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating records: %w", err)
	}

	return records, nil
}

// GetLastSuccess retrieves the last successful execution for a rule
func (m *Manager) GetLastSuccess(ruleName string) (*ExecutionRecord, error) {
	query := `
		SELECT id, rule_name, start_time, end_time, status, files_synced, bytes_synced, error
		FROM executions
		WHERE rule_name = ? AND status = 'success'
		ORDER BY start_time DESC
		LIMIT 1
	`

	var record ExecutionRecord
	err := m.db.QueryRow(query, ruleName).Scan(
		&record.ID,
		&record.RuleName,
		&record.StartTime,
		&record.EndTime,
		&record.Status,
		&record.FilesSynced,
		&record.BytesSynced,
		&record.Error,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No successful execution found
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query last success: %w", err)
	}

	return &record, nil
}

// GetAllHistory retrieves all execution history (for all rules)
func (m *Manager) GetAllHistory(limit int) ([]ExecutionRecord, error) {
	// Validate limit
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be positive, got %d", limit)
	}

	query := `
		SELECT id, rule_name, start_time, end_time, status, files_synced, bytes_synced, error
		FROM executions
		ORDER BY start_time DESC
		LIMIT ?
	`

	rows, err := m.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query all history: %w", err)
	}
	defer rows.Close()

	var records []ExecutionRecord
	for rows.Next() {
		var record ExecutionRecord
		err := rows.Scan(
			&record.ID,
			&record.RuleName,
			&record.StartTime,
			&record.EndTime,
			&record.Status,
			&record.FilesSynced,
			&record.BytesSynced,
			&record.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating records: %w", err)
	}

	return records, nil
}

// Close closes the database connection
func (m *Manager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
