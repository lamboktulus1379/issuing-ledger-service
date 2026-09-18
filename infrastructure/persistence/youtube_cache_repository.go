package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

// EnsureYouTubeCacheSchema creates the table for caching YouTube videos if not exists
func EnsureYouTubeCacheSchema(db *sql.DB) error {
	ddl := `CREATE TABLE IF NOT EXISTS youtube_video_cache (
        video_id TEXT PRIMARY KEY,
        etag TEXT,
        data JSONB NOT NULL,
        expires_at TIMESTAMPTZ NOT NULL,
        last_synced_at TIMESTAMPTZ NOT NULL,
        updated_at TIMESTAMPTZ NOT NULL
    )`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("create youtube_video_cache table: %w", err)
	}

	// Helpful index to purge or check expiry
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_youtube_video_cache_expires_at ON youtube_video_cache(expires_at)`); err != nil {
		logger.GetLogger().WithField("error", err).Warn("failed creating idx_youtube_video_cache_expires_at")
	}

	// Optional functional index on published_at extracted from JSON for ordering
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_youtube_video_cache_published_at ON youtube_video_cache (( (data->>'published_at')::timestamptz ))`); err != nil {
		logger.GetLogger().WithField("error", err).Warn("failed creating idx_youtube_video_cache_published_at")
	}

	return nil
}

// YouTubeCacheRepository implements repository for caching video metadata
// Stored as JSONB for flexibility without strict relational mapping

// YouTubeCacheRepository provides caching for YouTube video metadata.
type YouTubeCacheRepository struct {
	db *sql.DB // Database connection for cache table
}

func NewYouTubeCacheRepository(db *sql.DB) *YouTubeCacheRepository {
	return &YouTubeCacheRepository{db: db}
}

// GetVideo returns a cached video and its expiry time if present and not expired
func (r *YouTubeCacheRepository) GetVideo(ctx context.Context, videoID string) (*model.YouTubeVideo, *time.Time, error) {
	if r.db == nil {
		return nil, nil, nil
	}
	row := r.db.QueryRowContext(ctx, `SELECT data, expires_at FROM youtube_video_cache WHERE video_id=$1`, videoID)
	var raw []byte
	var expiresAt time.Time
	if err := row.Scan(&raw, &expiresAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	// If expired, treat as miss but return expiry for caller if needed
	if time.Now().After(expiresAt) {
		return nil, &expiresAt, nil
	}
	var v model.YouTubeVideo
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, nil, err
	}
	return &v, &expiresAt, nil
}

// UpsertVideo stores or updates the cache row with TTL from now
func (r *YouTubeCacheRepository) UpsertVideo(ctx context.Context, videoID string, video *model.YouTubeVideo, etag *string, ttl time.Duration) error {
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
	q := `INSERT INTO youtube_video_cache(video_id, etag, data, expires_at, last_synced_at, updated_at)
          VALUES ($1,$2,$3,$4,$5,$6)
          ON CONFLICT (video_id) DO UPDATE SET etag=EXCLUDED.etag, data=EXCLUDED.data, expires_at=EXCLUDED.expires_at, last_synced_at=EXCLUDED.last_synced_at, updated_at=EXCLUDED.updated_at`
	_, err = r.db.ExecContext(ctx, q, videoID, etagVal, raw, exp, now, now)
	return err
}

// ListVideos returns cached videos ordered by published_at desc with pagination
func (r *YouTubeCacheRepository) ListVideos(ctx context.Context, limit, offset int) ([]model.YouTubeVideo, int64, error) {
	if r.db == nil {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	// Exclude expired
	countRow := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM youtube_video_cache WHERE expires_at > NOW()`)
	var total int64
	if err := countRow.Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT data FROM youtube_video_cache WHERE expires_at > NOW() ORDER BY (data->>'published_at')::timestamptz DESC NULLS LAST LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]model.YouTubeVideo, 0, limit)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, 0, err
		}
		var v model.YouTubeVideo
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, nil
}

// UpsertVideos bulk upserts videos
func (r *YouTubeCacheRepository) UpsertVideos(ctx context.Context, videos []model.YouTubeVideo, etag *string, ttl time.Duration) error {
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
	q := `INSERT INTO youtube_video_cache(video_id, etag, data, expires_at, last_synced_at, updated_at)
		  VALUES ($1,$2,$3,$4,$5,$6)
		  ON CONFLICT (video_id) DO UPDATE SET etag=EXCLUDED.etag, data=EXCLUDED.data, expires_at=EXCLUDED.expires_at, last_synced_at=EXCLUDED.last_synced_at, updated_at=EXCLUDED.updated_at`
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
		if _, e := stmt.ExecContext(ctx, videos[i].ID, etagVal, raw, exp, now, now); e != nil {
			return e
		}
	}
	return tx.Commit()
}
