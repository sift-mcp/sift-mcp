package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Provider interface {
	GetDB() *sql.DB
	Close() error
}

type SQLiteProvider struct {
	db *sql.DB
}

func NewSQLiteProvider(dbPath string) (*SQLiteProvider, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &SQLiteProvider{db: db}, nil
}

func (p *SQLiteProvider) GetDB() *sql.DB {
	return p.db
}

func (p *SQLiteProvider) Close() error {
	return p.db.Close()
}
