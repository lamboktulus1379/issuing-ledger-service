package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// EnsureShareSchemaMSSQL ensures columns used by the sharing feature exist in MSSQL tables.
func EnsureShareSchemaMSSQL(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Helper to add a column if missing via COL_LENGTH check
	addIfMissing := func(table, column, ddl string) error {
		q := fmt.Sprintf(`IF COL_LENGTH('%s', '%s') IS NULL BEGIN %s END`, table, column, ddl)
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("ensure column %s.%s: %w", table, column, err)
		}
		return nil
	}
	if err := addIfMissing("dbo.video_share_records", "external_ref", "ALTER TABLE dbo.[video_share_records] ADD external_ref NVARCHAR(255) NULL"); err != nil {
		return err
	}
	if err := addIfMissing("dbo.share_jobs", "external_ref", "ALTER TABLE dbo.[share_jobs] ADD external_ref NVARCHAR(255) NULL"); err != nil {
		return err
	}
	return nil
}
