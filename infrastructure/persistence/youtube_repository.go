package persistence

import (
	"context"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/dto"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"time"
)

// YouTubeRepository implements repository.IYouTube and uses YouTubeCacheRepository for cache operations
type YouTubeRepository struct {
	CacheRepo        repository.IYouTubeCache
	YouTubeAPIClient repository.IYouTube // or your actual YouTube API client interface
}

// FetchAndUpdateFromYouTube fetches the latest video from YouTube API, updates the cache, and returns it
func (r *YouTubeRepository) FetchAndUpdateFromYouTube(ctx context.Context, videoID string) (*model.YouTubeVideo, error) {
	// Fetch from YouTube API client
	video, err := r.YouTubeAPIClient.GetVideoDetails(ctx, videoID)
	if err != nil {
		return nil, err
	}
	// Update cache with a default TTL (e.g., 10 minutes)
	if r.CacheRepo != nil && video != nil {
		ttl := 10 * time.Minute
		_ = r.CacheRepo.UpsertVideo(ctx, videoID, video, nil, ttl)
	}
	return video, nil
}

// Forward all IYouTube methods to the API client

func (r *YouTubeRepository) GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error) {
	return r.YouTubeAPIClient.GetMyVideos(ctx, req)
}

func (r *YouTubeRepository) GetVideoDetails(ctx context.Context, videoID string) (*model.YouTubeVideo, error) {
	return r.YouTubeAPIClient.GetVideoDetails(ctx, videoID)
}

func (r *YouTubeRepository) UploadVideo(ctx context.Context, req *dto.YouTubeVideoUploadRequest) (*model.YouTubeVideo, error) {
	return r.YouTubeAPIClient.UploadVideo(ctx, req)
}

func (r *YouTubeRepository) UpdateVideo(ctx context.Context, videoID string, updates map[string]interface{}) (*model.YouTubeVideo, error) {
	return r.YouTubeAPIClient.UpdateVideo(ctx, videoID, updates)
}

func (r *YouTubeRepository) DeleteVideo(ctx context.Context, videoID string) error {
	return r.YouTubeAPIClient.DeleteVideo(ctx, videoID)
}

func (r *YouTubeRepository) SearchVideos(ctx context.Context, req *dto.YouTubeSearchRequest) (*dto.YouTubeVideoResponse, error) {
	return r.YouTubeAPIClient.SearchVideos(ctx, req)
}

func (r *YouTubeRepository) GetVideoComments(ctx context.Context, req *dto.YouTubeCommentListRequest) (*dto.YouTubeCommentResponse, error) {
	return r.YouTubeAPIClient.GetVideoComments(ctx, req)
}

func (r *YouTubeRepository) AddComment(ctx context.Context, req *dto.YouTubeCommentRequest) (*model.YouTubeComment, error) {
	return r.YouTubeAPIClient.AddComment(ctx, req)
}

func (r *YouTubeRepository) UpdateComment(ctx context.Context, req *dto.YouTubeCommentUpdateRequest) (*model.YouTubeComment, error) {
	return r.YouTubeAPIClient.UpdateComment(ctx, req)
}

func (r *YouTubeRepository) DeleteComment(ctx context.Context, commentID string) error {
	return r.YouTubeAPIClient.DeleteComment(ctx, commentID)
}

func (r *YouTubeRepository) LikeVideo(ctx context.Context, videoID string) error {
	return r.YouTubeAPIClient.LikeVideo(ctx, videoID)
}

func (r *YouTubeRepository) DislikeVideo(ctx context.Context, videoID string) error {
	return r.YouTubeAPIClient.DislikeVideo(ctx, videoID)
}

func (r *YouTubeRepository) RemoveVideoRating(ctx context.Context, videoID string) error {
	return r.YouTubeAPIClient.RemoveVideoRating(ctx, videoID)
}

func (r *YouTubeRepository) ToggleUserCommentLike(ctx context.Context, userID, commentID string) (bool, error) {
	return r.YouTubeAPIClient.ToggleUserCommentLike(ctx, userID, commentID)
}

func (r *YouTubeRepository) ToggleUserCommentHeart(ctx context.Context, userID, commentID string) (bool, error) {
	return r.YouTubeAPIClient.ToggleUserCommentHeart(ctx, userID, commentID)
}

func (r *YouTubeRepository) GetMyChannel(ctx context.Context) (*model.YouTubeChannel, error) {
	return r.YouTubeAPIClient.GetMyChannel(ctx)
}

func (r *YouTubeRepository) GetChannelDetails(ctx context.Context, channelID string) (*model.YouTubeChannel, error) {
	return r.YouTubeAPIClient.GetChannelDetails(ctx, channelID)
}

func (r *YouTubeRepository) GetChannelVideos(ctx context.Context, channelID string, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error) {
	return r.YouTubeAPIClient.GetChannelVideos(ctx, channelID, req)
}

func (r *YouTubeRepository) GetMyPlaylists(ctx context.Context) ([]model.YouTubePlaylist, error) {
	return r.YouTubeAPIClient.GetMyPlaylists(ctx)
}

func (r *YouTubeRepository) GetPlaylistVideos(ctx context.Context, req *dto.YouTubePlaylistRequest) (*dto.YouTubeVideoResponse, error) {
	return r.YouTubeAPIClient.GetPlaylistVideos(ctx, req)
}

func (r *YouTubeRepository) CreatePlaylist(ctx context.Context, title, description, privacy string) (*model.YouTubePlaylist, error) {
	return r.YouTubeAPIClient.CreatePlaylist(ctx, title, description, privacy)
}

func (r *YouTubeRepository) AddVideoToPlaylist(ctx context.Context, playlistID, videoID string) error {
	return r.YouTubeAPIClient.AddVideoToPlaylist(ctx, playlistID, videoID)
}

func (r *YouTubeRepository) RemoveVideoFromPlaylist(ctx context.Context, playlistID, videoID string) error {
	return r.YouTubeAPIClient.RemoveVideoFromPlaylist(ctx, playlistID, videoID)
}

func (r *YouTubeRepository) GetVideoAnalytics(ctx context.Context, videoID string, startDate, endDate string) (interface{}, error) {
	return r.YouTubeAPIClient.GetVideoAnalytics(ctx, videoID, startDate, endDate)
}

func (r *YouTubeRepository) GetChannelAnalytics(ctx context.Context, startDate, endDate string) (interface{}, error) {
	return r.YouTubeAPIClient.GetChannelAnalytics(ctx, startDate, endDate)
}
