package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// EnsureShareSchema adds newer columns used by sharing feature if they are missing.
// Safe to call at startup; performs metadata lookups and conditional ALTER TABLE.
func EnsureShareSchema(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	checks := []struct {
		table  string
		column string
		ddl    string
	}{
		{"video_share_records", "external_ref", "ALTER TABLE video_share_records ADD COLUMN external_ref TEXT"},
		{"share_jobs", "external_ref", "ALTER TABLE share_jobs ADD COLUMN external_ref TEXT"},
	}

	for _, c := range checks {
		exists, err := columnExists(ctx, db, c.table, c.column)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := db.ExecContext(ctx, c.ddl); err != nil {
				return fmt.Errorf("adding column %s.%s failed: %w", c.table, c.column, err)
			}
		}
	}
	return nil
}

func columnExists(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	row := db.QueryRowContext(ctx, `SELECT 1 FROM information_schema.columns WHERE table_name=$1 AND column_name=$2`, table, column)
	var one int
	if err := row.Scan(&one); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
