package database

import "fmt"

func Migrate(provider Provider) error {
	db := provider.GetDB()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS test_reports (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			framework TEXT NOT NULL,
			total_tests INTEGER DEFAULT 0,
			passed INTEGER DEFAULT 0,
			failed INTEGER DEFAULT 0,
			skipped INTEGER DEFAULT 0,
			errored INTEGER DEFAULT 0,
			duration_ms INTEGER DEFAULT 0,
			raw_json TEXT,
			timestamp DATETIME NOT NULL,
			created_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_test_reports_source ON test_reports(source)`,
		`CREATE INDEX IF NOT EXISTS idx_test_reports_framework ON test_reports(framework)`,
		`CREATE INDEX IF NOT EXISTS idx_test_reports_timestamp ON test_reports(timestamp)`,

		`CREATE TABLE IF NOT EXISTS test_failures (
			id TEXT PRIMARY KEY,
			report_id TEXT NOT NULL REFERENCES test_reports(id),
			test_name TEXT NOT NULL,
			classname TEXT,
			severity TEXT NOT NULL,
			failure_message TEXT,
			stack_trace TEXT,
			error_fingerprint TEXT NOT NULL DEFAULT '',
			duration_ms INTEGER DEFAULT 0,
			timestamp DATETIME NOT NULL,
			created_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_test_failures_report_id ON test_failures(report_id)`,
		`CREATE INDEX IF NOT EXISTS idx_test_failures_test_name ON test_failures(test_name)`,
		`CREATE INDEX IF NOT EXISTS idx_test_failures_classname ON test_failures(classname)`,
		`CREATE INDEX IF NOT EXISTS idx_test_failures_severity ON test_failures(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_test_failures_timestamp ON test_failures(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_test_failures_fingerprint ON test_failures(error_fingerprint)`,

		`CREATE TABLE IF NOT EXISTS error_fingerprints (
			fingerprint TEXT PRIMARY KEY,
			normalized_trace TEXT NOT NULL,
			first_seen DATETIME NOT NULL,
			last_seen DATETIME NOT NULL,
			occurrence_count INTEGER DEFAULT 1
		)`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}
