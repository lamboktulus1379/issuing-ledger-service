package repository

import (
	"context"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"time"
)

// IYouTubeCache defines a cache repository for YouTube metadata
type IYouTubeCache interface {
	// GetVideo returns a cached video if present. It also returns the expiration time.
	GetVideo(ctx context.Context, videoID string) (*model.YouTubeVideo, *time.Time, error)
	// UpsertVideo stores/updates the cached video with a TTL from now.
	UpsertVideo(ctx context.Context, videoID string, video *model.YouTubeVideo, etag *string, ttl time.Duration) error
	// ListVideos returns a page of cached videos ordered by published_at desc.
	// offset is the zero-based item offset; returns items and total count for pagination.
	ListVideos(ctx context.Context, limit, offset int) ([]model.YouTubeVideo, int64, error)
	// UpsertVideos stores or updates multiple videos efficiently.
	UpsertVideos(ctx context.Context, videos []model.YouTubeVideo, etag *string, ttl time.Duration) error
}
