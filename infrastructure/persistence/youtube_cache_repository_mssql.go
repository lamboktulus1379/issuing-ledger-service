package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
)

// EnsureYouTubeCacheSchemaMSSQL creates the cache table on MSSQL if not exists
func EnsureYouTubeCacheSchemaMSSQL(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	// Create table if not exists
	ddl := `IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'dbo.youtube_video_cache') AND type in (N'U'))
BEGIN
    CREATE TABLE dbo.youtube_video_cache (
        video_id NVARCHAR(64) NOT NULL PRIMARY KEY,
        etag NVARCHAR(256) NULL,
        data NVARCHAR(MAX) NOT NULL,
        expires_at DATETIMEOFFSET NOT NULL,
        last_synced_at DATETIMEOFFSET NOT NULL,
        updated_at DATETIMEOFFSET NOT NULL
    );
END`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("create youtube_video_cache table (mssql): %w", err)
	}
	// Index on expires_at
	if _, err := db.Exec(`IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'idx_youtube_video_cache_expires_at' AND object_id = OBJECT_ID('dbo.youtube_video_cache'))
CREATE INDEX idx_youtube_video_cache_expires_at ON dbo.youtube_video_cache(expires_at)`); err != nil {
		// Non-fatal
	}
	return nil
}

// YouTubeCacheRepositoryMSSQL implements IYouTubeCache on MSSQL
type YouTubeCacheRepositoryMSSQL struct {
	db *sql.DB
}

func NewYouTubeCacheRepositoryMSSQL(db *sql.DB) *YouTubeCacheRepositoryMSSQL {
	return &YouTubeCacheRepositoryMSSQL{db: db}
}

// GetVideo returns cached video if not expired
func (r *YouTubeCacheRepositoryMSSQL) GetVideo(ctx context.Context, videoID string) (*model.YouTubeVideo, *time.Time, error) {
	if r.db == nil {
		return nil, nil, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT data, expires_at FROM dbo.youtube_video_cache WHERE video_id=@p1`, videoID)
	var raw string
	var expiresAt time.Time
	if err := row.Scan(&raw, &expiresAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	if time.Now().After(expiresAt) {
		return nil, &expiresAt, nil
	}
	var v model.YouTubeVideo
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, nil, err
	}
	return &v, &expiresAt, nil
}

// UpsertVideo stores/updates one row
func (r *YouTubeCacheRepositoryMSSQL) UpsertVideo(ctx context.Context, videoID string, video *model.YouTubeVideo, etag *string, ttl time.Duration) error {
	if r.db == nil {
		return nil
	}
	raw, err := json.Marshal(video)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	exp := now.Add(ttl)
	var etagVal interface{}
	if etag != nil {
		etagVal = *etag
	} else {
		etagVal = nil
	}
	q := `MERGE dbo.youtube_video_cache AS target
USING (SELECT @p1 AS video_id) AS src
ON (target.video_id = src.video_id)
WHEN MATCHED THEN UPDATE SET etag=@p2, data=@p3, expires_at=@p4, last_synced_at=@p5, updated_at=@p6
WHEN NOT MATCHED THEN INSERT (video_id, etag, data, expires_at, last_synced_at, updated_at)
VALUES (@p1, @p2, @p3, @p4, @p5, @p6);`
	_, err = r.db.ExecContext(ctx, q, videoID, etagVal, string(raw), exp, now, now)
	return err
}

// ListVideos returns a page ordered by published_at desc (from JSON)
func (r *YouTubeCacheRepositoryMSSQL) ListVideos(ctx context.Context, limit, offset int) ([]model.YouTubeVideo, int64, error) {
	if r.db == nil {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	// Count non-expired
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM dbo.youtube_video_cache WHERE expires_at > SYSUTCDATETIME()`).Scan(&total); err != nil {
		return nil, 0, err
	}
	// Order by published_at inside JSON if present
	rows, err := r.db.QueryContext(ctx, `SELECT data FROM dbo.youtube_video_cache WHERE expires_at > SYSUTCDATETIME()
ORDER BY TRY_CONVERT(datetimeoffset(7), JSON_VALUE(data,'$.published_at'), 127) DESC
OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY`, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]model.YouTubeVideo, 0, limit)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, 0, err
		}
		var v model.YouTubeVideo
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, nil
}

// UpsertVideos bulk upserts
func (r *YouTubeCacheRepositoryMSSQL) UpsertVideos(ctx context.Context, videos []model.YouTubeVideo, etag *string, ttl time.Duration) error {
	if r.db == nil || len(videos) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	now := time.Now().UTC()
	exp := now.Add(ttl)
	q := `MERGE dbo.youtube_video_cache AS target
USING (SELECT @p1 AS video_id) AS src
ON (target.video_id = src.video_id)
WHEN MATCHED THEN UPDATE SET etag=@p2, data=@p3, expires_at=@p4, last_synced_at=@p5, updated_at=@p6
WHEN NOT MATCHED THEN INSERT (video_id, etag, data, expires_at, last_synced_at, updated_at)
VALUES (@p1, @p2, @p3, @p4, @p5, @p6);`
	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()
	var etagVal interface{}
	if etag != nil {
		etagVal = *etag
	}
	for i := range videos {
		raw, mErr := json.Marshal(&videos[i])
		if mErr != nil {
			return mErr
		}
		if _, e := stmt.ExecContext(ctx, videos[i].ID, etagVal, string(raw), exp, now, now); e != nil {
			return e
		}
	}
	return tx.Commit()
}
