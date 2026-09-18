package persistence

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
)

// ShareRepositoryMSSQL implements sharing persistence for SQL Server/Azure SQL using database/sql.
type ShareRepositoryMSSQL struct{ db *sql.DB }

func NewShareRepositoryMSSQL(db *sql.DB) *ShareRepositoryMSSQL { return &ShareRepositoryMSSQL{db: db} }

// DB exposes the underlying *sql.DB
func (r *ShareRepositoryMSSQL) DB() *sql.DB { return r.db }

func (r *ShareRepositoryMSSQL) UpsertTrackShares(ctx context.Context, videoID, userID string, platforms []string, initialStatus string) ([]*model.VideoShareRecord, error) {
	out := make([]*model.VideoShareRecord, 0, len(platforms))
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	now := time.Now().UTC()
	for _, p := range platforms {
		p = strings.ToLower(p)
		// Use MERGE for upsert with conditional updates
		q := `MERGE dbo.[video_share_records] AS target
USING (VALUES (@p1, @p2, @p3)) AS src(video_id, platform, user_id)
ON target.video_id = src.video_id AND target.platform = src.platform AND target.user_id = src.user_id AND target.deleted_at IS NULL
WHEN MATCHED THEN UPDATE SET
  attempt_count = target.attempt_count + CASE WHEN target.status <> @p4 THEN 1 ELSE 0 END,
  status = CASE WHEN target.status = 'success' THEN target.status ELSE @p4 END,
  updated_at = @p5
WHEN NOT MATCHED THEN
  INSERT (video_id, platform, user_id, status, attempt_count, created_at, updated_at)
  VALUES (src.video_id, src.platform, src.user_id, @p4, 1, @p5, @p5);`
		if _, err = tx.ExecContext(ctx, q, videoID, p, userID, initialStatus, now); err != nil {
			return nil, err
		}
		// Fetch latest row
		row := tx.QueryRowContext(ctx, `SELECT TOP (1) id, video_id, platform, user_id, status, error_message, external_ref, attempt_count, created_at, updated_at
FROM dbo.[video_share_records]
WHERE video_id=@p1 AND platform=@p2 AND user_id=@p3 AND deleted_at IS NULL
ORDER BY updated_at DESC`, videoID, p, userID)
		rec := &model.VideoShareRecord{}
		var errMsg, extRef sql.NullString
		if err = row.Scan(&rec.ID, &rec.VideoID, &rec.Platform, &rec.UserID, &rec.Status, &errMsg, &extRef, &rec.AttemptCount, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		if errMsg.Valid {
			rec.ErrorMessage = &errMsg.String
		}
		if extRef.Valid {
			rec.ExternalRef = &extRef.String
		}
		out = append(out, rec)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *ShareRepositoryMSSQL) GetShareStatus(ctx context.Context, videoID, userID string) ([]*model.VideoShareRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, video_id, platform, user_id, status, error_message, external_ref, attempt_count, created_at, updated_at
FROM dbo.[video_share_records]
WHERE video_id=@p1 AND user_id=@p2 AND deleted_at IS NULL`, videoID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.VideoShareRecord
	for rows.Next() {
		rec := &model.VideoShareRecord{}
		var errMsg, extRef sql.NullString
		if err := rows.Scan(&rec.ID, &rec.VideoID, &rec.Platform, &rec.UserID, &rec.Status, &errMsg, &extRef, &rec.AttemptCount, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		if errMsg.Valid {
			rec.ErrorMessage = &errMsg.String
		}
		if extRef.Valid {
			rec.ExternalRef = &extRef.String
		}
		list = append(list, rec)
	}
	return list, nil
}

func (r *ShareRepositoryMSSQL) CreateAudit(ctx context.Context, audits []*model.VideoShareAudit) error {
	if len(audits) == 0 {
		return nil
	}
	q := `INSERT INTO dbo.[video_share_audit] (record_id, video_id, platform, user_id, status, error_message, created_at) VALUES (@p1,@p2,@p3,@p4,@p5,@p6,@p7)`
	now := time.Now().UTC()
	for _, a := range audits {
		if a.CreatedAt.IsZero() {
			a.CreatedAt = now
		}
		if _, err := r.db.ExecContext(ctx, q, a.RecordID, a.VideoID, a.Platform, a.UserID, a.Status, a.ErrorMessage, a.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *ShareRepositoryMSSQL) EnqueueJobs(ctx context.Context, records []*model.VideoShareRecord) error {
	if len(records) == 0 {
		return nil
	}
	q := `INSERT INTO dbo.[share_jobs] (record_id, platform, status, attempts, created_at, updated_at) VALUES (@p1,@p2,@p3,@p4,@p5,@p5)`
	now := time.Now().UTC()
	for _, rec := range records {
		if rec.Status != "pending" {
			continue
		}
		if _, err := r.db.ExecContext(ctx, q, rec.ID, rec.Platform, "pending", 0, now); err != nil {
			return err
		}
	}
	return nil
}

func (r *ShareRepositoryMSSQL) FetchPendingJobs(ctx context.Context, limit int) ([]*model.ShareJob, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p1) j.id, j.record_id, j.platform, j.status, j.attempts, j.last_error, j.external_ref, j.created_at, j.updated_at
FROM dbo.[share_jobs] j
JOIN dbo.[video_share_records] r ON r.id = j.record_id AND r.deleted_at IS NULL
WHERE j.status='pending'
ORDER BY j.created_at ASC`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []*model.ShareJob
	for rows.Next() {
		j := &model.ShareJob{}
		var lastErr, extRef sql.NullString
		if err := rows.Scan(&j.ID, &j.RecordID, &j.Platform, &j.Status, &j.Attempts, &lastErr, &extRef, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		if lastErr.Valid {
			j.LastError = &lastErr.String
		}
		if extRef.Valid {
			j.ExternalRef = &extRef.String
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *ShareRepositoryMSSQL) MarkJobRunning(ctx context.Context, jobID int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE dbo.[share_jobs] SET status='running', updated_at=@p1 WHERE id=@p2 AND status='pending'`, time.Now().UTC(), jobID)
	return err
}

func (r *ShareRepositoryMSSQL) MarkJobResult(ctx context.Context, jobID int64, success bool, errMsg *string) error {
	status := "failed"
	if success {
		status = "success"
	}
	_, err := r.db.ExecContext(ctx, `UPDATE dbo.[share_jobs] SET status=@p1, attempts=attempts+1, last_error=@p2, updated_at=@p3 WHERE id=@p4`, status, errMsg, time.Now().UTC(), jobID)
	return err
}

func (r *ShareRepositoryMSSQL) UpdateRecordStatus(ctx context.Context, recordID int64, status string, errMsg *string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE dbo.[video_share_records] SET status=@p1, error_message=@p2, updated_at=@p3 WHERE id=@p4 AND deleted_at IS NULL`, status, errMsg, time.Now().UTC(), recordID)
	return err
}

func (r *ShareRepositoryMSSQL) GetRecordByID(ctx context.Context, id int64) (*model.VideoShareRecord, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, video_id, platform, user_id, status, error_message, external_ref, attempt_count, created_at, updated_at FROM dbo.[video_share_records] WHERE id=@p1 AND deleted_at IS NULL`, id)
	rec := &model.VideoShareRecord{}
	var errMsg, extRef sql.NullString
	if err := row.Scan(&rec.ID, &rec.VideoID, &rec.Platform, &rec.UserID, &rec.Status, &errMsg, &extRef, &rec.AttemptCount, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, err
	}
	if errMsg.Valid {
		rec.ErrorMessage = &errMsg.String
	}
	if extRef.Valid {
		rec.ExternalRef = &extRef.String
	}
	return rec, nil
}

func (r *ShareRepositoryMSSQL) UpdateRecordExternalRef(ctx context.Context, recordID int64, ref string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE dbo.[video_share_records] SET external_ref=@p1, updated_at=@p2 WHERE id=@p3`, ref, time.Now().UTC(), recordID)
	return err
}

// Ensure interface compliance
var _ interface {
	UpsertTrackShares(context.Context, string, string, []string, string) ([]*model.VideoShareRecord, error)
} = (*ShareRepositoryMSSQL)(nil)
