package postgres

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

func NewDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Retry connecting — postgres may still be starting in Docker
	var pingErr error
	for attempt := 1; attempt <= 5; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			break
		}
		slog.Warn("database not ready, retrying", "attempt", attempt, "error", pingErr)
		time.Sleep(2 * time.Second)
	}
	if pingErr != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database after 5 attempts: %w", pingErr)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	return db, nil
}
