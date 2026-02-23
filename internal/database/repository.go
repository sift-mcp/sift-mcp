package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"sift/internal/core"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Store(ctx context.Context, report *core.TestReport) error {
	rawJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	now := time.Now()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO test_reports (id, source, framework, total_tests, passed, failed, skipped, errored, duration_ms, raw_json, timestamp, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID, report.Source, report.Framework,
		report.TotalTests, report.Passed, report.Failed, report.Skipped, report.Errored,
		report.Duration.Milliseconds(), string(rawJSON),
		report.Timestamp, now,
	)
	if err != nil {
		return fmt.Errorf("failed to store report: %w", err)
	}

	for _, suite := range report.Suites {
		for _, tc := range suite.Cases {
			if tc.Status != core.TestStatusFailed && tc.Status != core.TestStatusErrored {
				continue
			}

			failureMessage := ""
			stackTrace := ""
			severity := string(core.FailureSeverityMedium)

			if tc.Failure != nil {
				failureMessage = tc.Failure.Message
				stackTrace = tc.Failure.StackTrace
			}

			if tc.Status == core.TestStatusErrored {
				severity = string(core.FailureSeverityCritical)
			} else {
				severity = string(core.FailureSeverityHigh)
			}

			_, err = tx.ExecContext(ctx,
				`INSERT INTO test_failures (id, report_id, test_name, classname, severity, failure_message, stack_trace, error_fingerprint, duration_ms, timestamp, created_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				uuid.New().String(), report.ID, tc.Name, tc.Classname,
				severity, failureMessage, stackTrace, "",
				tc.Duration.Milliseconds(), report.Timestamp, now,
			)
			if err != nil {
				return fmt.Errorf("failed to store failure record: %w", err)
			}
		}
	}

	return tx.Commit()
}

func (r *ReportRepository) GetByID(ctx context.Context, id string) (*core.TestReport, error) {
	var rawJSON string
	err := r.db.QueryRowContext(ctx,
		`SELECT raw_json FROM test_reports WHERE id = ?`, id,
	).Scan(&rawJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	var report core.TestReport
	if err := json.Unmarshal([]byte(rawJSON), &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal report: %w", err)
	}

	return &report, nil
}

func (r *ReportRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*core.TestReport, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT raw_json FROM test_reports WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp DESC`,
		start, end,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query reports: %w", err)
	}
	defer rows.Close()

	var reports []*core.TestReport
	for rows.Next() {
		var rawJSON string
		if err := rows.Scan(&rawJSON); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		var report core.TestReport
		if err := json.Unmarshal([]byte(rawJSON), &report); err != nil {
			continue
		}
		reports = append(reports, &report)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return reports, nil
}

func (r *ReportRepository) GetFailureHistory(ctx context.Context, testName string, limit int) ([]*core.TestReport, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT report_id FROM test_failures WHERE test_name = ? ORDER BY timestamp DESC LIMIT ?`,
		testName, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query failure history: %w", err)
	}
	defer rows.Close()

	var reportIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan failure row: %w", err)
		}
		reportIDs = append(reportIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	if len(reportIDs) == 0 {
		return []*core.TestReport{}, nil
	}

	placeholders := strings.Repeat("?,", len(reportIDs))
	placeholders = placeholders[:len(placeholders)-1]

	args := make([]interface{}, len(reportIDs))
	for i, id := range reportIDs {
		args[i] = id
	}

	reportRows, err := r.db.QueryContext(ctx,
		`SELECT raw_json FROM test_reports WHERE id IN (`+placeholders+`) ORDER BY timestamp DESC`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query reports: %w", err)
	}
	defer reportRows.Close()

	var reports []*core.TestReport
	for reportRows.Next() {
		var rawJSON string
		if err := reportRows.Scan(&rawJSON); err != nil {
			return nil, fmt.Errorf("failed to scan report row: %w", err)
		}
		var report core.TestReport
		if err := json.Unmarshal([]byte(rawJSON), &report); err != nil {
			continue
		}
		reports = append(reports, &report)
	}

	if err := reportRows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return reports, nil
}

func (r *ReportRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM test_reports`,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count reports: %w", err)
	}
	return count, nil
}

func (r *ReportRepository) Delete(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		`DELETE FROM test_failures WHERE report_id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete failure records: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`DELETE FROM test_reports WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	return tx.Commit()
}

func (r *ReportRepository) GetTestFailureCounts(ctx context.Context, testName string, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM test_failures WHERE test_name = ? AND timestamp >= ?`,
		testName, since,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count test failures: %w", err)
	}
	return count, nil
}

func (r *ReportRepository) GetTestLastSuccess(ctx context.Context, testName string) (time.Time, error) {
	var lastSuccess sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT MAX(tr.timestamp) FROM test_reports tr
		 WHERE NOT EXISTS (
			SELECT 1 FROM test_failures tf
			WHERE tf.report_id = tr.id AND tf.test_name = ?
		 )`,
		testName,
	).Scan(&lastSuccess)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get last success: %w", err)
	}
	if !lastSuccess.Valid {
		return time.Time{}, nil
	}
	return lastSuccess.Time, nil
}

func (r *ReportRepository) UpsertErrorFingerprint(ctx context.Context, fingerprint string, normalizedTrace string, timestamp time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO error_fingerprints (fingerprint, normalized_trace, first_seen, last_seen, occurrence_count)
		 VALUES (?, ?, ?, ?, 1)
		 ON CONFLICT(fingerprint) DO UPDATE SET
			last_seen = ?,
			occurrence_count = occurrence_count + 1`,
		fingerprint, normalizedTrace, timestamp, timestamp, timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert error fingerprint: %w", err)
	}
	return nil
}

func (r *ReportRepository) GetErrorFingerprint(ctx context.Context, fingerprint string) (int, time.Time, time.Time, error) {
	var occurrenceCount int
	var firstSeen, lastSeen time.Time
	err := r.db.QueryRowContext(ctx,
		`SELECT occurrence_count, first_seen, last_seen FROM error_fingerprints WHERE fingerprint = ?`,
		fingerprint,
	).Scan(&occurrenceCount, &firstSeen, &lastSeen)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, time.Time{}, time.Time{}, nil
		}
		return 0, time.Time{}, time.Time{}, fmt.Errorf("failed to get error fingerprint: %w", err)
	}
	return occurrenceCount, firstSeen, lastSeen, nil
}
