package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/dto"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
)

// GetVideoDetailsWithSync fetches from YouTube API if forceSync is true, else uses cache-aside
func (u *YouTubeUseCase) GetVideoDetailsWithSync(ctx context.Context, videoID string, forceSync bool) (*model.YouTubeVideo, error) {
	if videoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}

	if forceSync {
		// Always fetch from YouTube API, update DB/cache, and return
		video, err := u.youtubeRepo.FetchAndUpdateFromYouTube(ctx, videoID)
		if err != nil {
			return nil, fmt.Errorf("failed to sync video from YouTube: %w", err)
		}
		if u.cache != nil && video != nil {
			_ = u.cache.UpsertVideo(ctx, videoID, video, nil, 10*time.Minute)
		}
		return video, nil
	}

	// Default: cache-aside logic
	return u.GetVideoDetails(ctx, videoID)
}

// IYouTubeUseCase defines the interface for YouTube use case operations
type IYouTubeUseCase interface {
	// Video operations
	GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error)
	// ListVideosFromDB returns videos from local DB cache (no YouTube calls)
	ListVideosFromDB(ctx context.Context, page, pageSize int) (*dto.YouTubeVideoResponse, error)
	GetVideoDetails(ctx context.Context, videoID string) (*model.YouTubeVideo, error)
	UploadVideo(ctx context.Context, req *dto.YouTubeVideoUploadRequest) (*model.YouTubeVideo, error)
	UpdateVideo(ctx context.Context, videoID string, req *dto.YouTubeVideoUpdateRequest) (*model.YouTubeVideo, error)
	SearchVideos(ctx context.Context, req *dto.YouTubeSearchRequest) (*dto.YouTubeVideoResponse, error)
	// SyncMyVideos pulls from YouTube and persists into DB cache, returns count
	SyncMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (int, error)

	// Comment operations
	GetVideoComments(ctx context.Context, req *dto.YouTubeCommentListRequest) (*dto.YouTubeCommentResponse, error)
	AddComment(ctx context.Context, req *dto.YouTubeCommentRequest) (*model.YouTubeComment, error)
	UpdateComment(ctx context.Context, req *dto.YouTubeCommentUpdateRequest) (*model.YouTubeComment, error)
	DeleteComment(ctx context.Context, commentID string) error

	// Like operations
	LikeVideo(ctx context.Context, videoID string) error
	DislikeVideo(ctx context.Context, videoID string) error
	RemoveVideoRating(ctx context.Context, videoID string) error
	ToggleCommentLike(ctx context.Context, userID, commentID string) (bool, error)
	ToggleCommentHeart(ctx context.Context, userID, commentID string) (bool, error)

	// Channel operations
	GetMyChannel(ctx context.Context) (*model.YouTubeChannel, error)
	GetChannelDetails(ctx context.Context, channelID string) (*model.YouTubeChannel, error)

	// Playlist operations
	GetMyPlaylists(ctx context.Context) ([]model.YouTubePlaylist, error)
	CreatePlaylist(ctx context.Context, title, description, privacy string) (*model.YouTubePlaylist, error)

	GetVideoDetailsWithSync(ctx context.Context, videoID string, forceSync bool) (*model.YouTubeVideo, error)

	// DashboardSummary aggregates metrics for admin dashboard from cache (fast, no API calls)
	DashboardSummary(ctx context.Context) (*dto.YouTubeDashboardSummary, error)
}

// YouTubeUseCase implements the YouTube use case operations
type YouTubeUseCase struct {
	youtubeRepo repository.IYouTube
	cache       repository.IYouTubeCache // optional
}

// NewYouTubeUseCase creates a new YouTube use case instance
func NewYouTubeUseCase(youtubeRepo repository.IYouTube) IYouTubeUseCase {
	return &YouTubeUseCase{youtubeRepo: youtubeRepo}
}

// NewYouTubeUseCaseWithCache creates a new YouTube use case with cache configured
func NewYouTubeUseCaseWithCache(youtubeRepo repository.IYouTube, cache repository.IYouTubeCache) IYouTubeUseCase {
	return (&YouTubeUseCase{youtubeRepo: youtubeRepo}).WithCache(cache)
}

// WithCache enables cache on the use case (fluent)
func (u *YouTubeUseCase) WithCache(cache repository.IYouTubeCache) *YouTubeUseCase {
	u.cache = cache
	return u
}

// GetMyVideos retrieves videos from the authenticated user's channel
func (u *YouTubeUseCase) GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error) {
	if req == nil {
		req = &dto.YouTubeVideoListRequest{MaxResults: 25, Order: "date"}
	}
	if req.MaxResults == 0 {
		req.MaxResults = 25
	}
	if req.Order == "" {
		req.Order = "date"
	}
	if req.MaxResults > 50 {
		req.MaxResults = 50
	}

	response, err := u.youtubeRepo.GetMyVideos(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get my videos: %w", err)
	}
	return response, nil
}

// ListVideosFromDB returns videos from DB cache only
func (u *YouTubeUseCase) ListVideosFromDB(ctx context.Context, page, pageSize int) (*dto.YouTubeVideoResponse, error) {
	if u.cache == nil {
		return nil, fmt.Errorf("cache repository not configured")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	items, total, err := u.cache.ListVideos(ctx, pageSize, offset)
	if err != nil {
		return nil, err
	}
	// Convert []YouTubeVideo to []interface{}
	arr := make([]interface{}, len(items))
	for i := range items {
		arr[i] = items[i]
	}
	return &dto.YouTubeVideoResponse{
		YouTubeResponse: dto.YouTubeResponse{
			Kind:     "local#youtubeVideoList",
			PageInfo: dto.PageInfo{TotalResults: total, ResultsPerPage: int64(pageSize)},
		},
		Items: arr,
	}, nil
}

// GetVideoDetails retrieves details for a specific video (cache-aside)
func (u *YouTubeUseCase) GetVideoDetails(ctx context.Context, videoID string) (*model.YouTubeVideo, error) {
	if videoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}

	if u.cache != nil {
		if v, _, err := u.cache.GetVideo(ctx, videoID); err == nil && v != nil {
			return v, nil
		}
	}

	video, err := u.youtubeRepo.GetVideoDetails(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video details: %w", err)
	}

	if u.cache != nil && video != nil {
		_ = u.cache.UpsertVideo(ctx, videoID, video, nil, 10*time.Minute)
	}
	return video, nil
}

// SyncMyVideos fetches from YouTube and persists to DB cache (cache-aside warmup)
func (u *YouTubeUseCase) SyncMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (int, error) {
	if u.youtubeRepo == nil || u.cache == nil {
		return 0, fmt.Errorf("sync requires both youtube client and cache configured")
	}
	if req == nil {
		req = &dto.YouTubeVideoListRequest{MaxResults: 50, Order: "date"}
	}
	if req.MaxResults == 0 {
		req.MaxResults = 50
	}
	if req.MaxResults > 50 {
		req.MaxResults = 50
	}
	// Helper to adapt interface slice to concrete []model.YouTubeVideo
	adapt := func(items []interface{}) []model.YouTubeVideo {
		vids := make([]model.YouTubeVideo, 0, len(items))
		for _, it := range items {
			switch v := it.(type) {
			case model.YouTubeVideo:
				vids = append(vids, v)
			case *model.YouTubeVideo:
				vids = append(vids, *v)
			case map[string]interface{}:
				// skip mock rows
			default:
				// unknown type; skip
			}
		}
		return vids
	}

	// If full sync requested, iterate all pages
	if req.All {
		total := 0
		pageToken := req.PageToken
		// hard stop to avoid infinite loops
		for page := 0; page < 1000; page++ { // ~50k items max at 50/page
			local := *req
			local.PageToken = pageToken
			resp, err := u.youtubeRepo.GetMyVideos(ctx, &local)
			if err != nil {
				return total, err
			}
			// Nothing returned -> done
			if resp == nil || len(resp.Items) == 0 {
				break
			}
			videos := adapt(resp.Items)
			if len(videos) > 0 {
				if err := u.cache.UpsertVideos(ctx, videos, nil, 24*time.Hour); err != nil {
					return total, err
				}
				total += len(videos)
			}
			if resp.NextPageToken == "" || resp.NextPageToken == pageToken {
				break
			}
			pageToken = resp.NextPageToken
		}
		return total, nil
	}

	// Single page sync (default)
	resp, err := u.youtubeRepo.GetMyVideos(ctx, req)
	if err != nil {
		return 0, err
	}
	videos := adapt(resp.Items)
	if len(videos) == 0 {
		return 0, nil
	}
	if err := u.cache.UpsertVideos(ctx, videos, nil, 24*time.Hour); err != nil {
		return 0, err
	}
	return len(videos), nil
}

// UploadVideo uploads a video to YouTube
func (u *YouTubeUseCase) UploadVideo(ctx context.Context, req *dto.YouTubeVideoUploadRequest) (*model.YouTubeVideo, error) {
	if req == nil {
		return nil, fmt.Errorf("upload request is required")
	}
	if req.Title == "" {
		return nil, fmt.Errorf("video title is required")
	}
	if req.File == nil {
		return nil, fmt.Errorf("video file is required")
	}
	if req.Privacy == "" {
		req.Privacy = "private"
	}
	validPrivacy := map[string]bool{"private": true, "public": true, "unlisted": true}
	if !validPrivacy[req.Privacy] {
		return nil, fmt.Errorf("invalid privacy setting: %s", req.Privacy)
	}

	video, err := u.youtubeRepo.UploadVideo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}
	return video, nil
}

// UpdateVideo updates metadata for an existing video
func (u *YouTubeUseCase) UpdateVideo(ctx context.Context, videoID string, req *dto.YouTubeVideoUpdateRequest) (*model.YouTubeVideo, error) {
	if videoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}
	if req == nil {
		return nil, fmt.Errorf("update request is required")
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.Privacy != nil {
		p := *req.Privacy
		validPrivacy := map[string]bool{"private": true, "public": true, "unlisted": true}
		if !validPrivacy[p] {
			return nil, fmt.Errorf("invalid privacy setting: %s", p)
		}
		updates["privacy"] = p
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no fields provided to update")
	}

	// Update video on YouTube
	_, err := u.youtubeRepo.UpdateVideo(ctx, videoID, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update video: %w", err)
	}

	// Immediately fetch latest data from YouTube and sync to DB/cache
	fresh, err := u.youtubeRepo.FetchAndUpdateFromYouTube(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("video updated but failed to sync latest data: %w", err)
	}
	return fresh, nil
}

// SearchVideos searches for videos on YouTube
func (u *YouTubeUseCase) SearchVideos(ctx context.Context, req *dto.YouTubeSearchRequest) (*dto.YouTubeVideoResponse, error) {
	if req == nil || req.Q == "" {
		return nil, fmt.Errorf("search query is required")
	}
	if req.MaxResults == 0 {
		req.MaxResults = 25
	}
	if req.Order == "" {
		req.Order = "relevance"
	}
	if req.Type == "" {
		req.Type = "video"
	}
	if req.MaxResults > 50 {
		req.MaxResults = 50
	}

	response, err := u.youtubeRepo.SearchVideos(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search videos: %w", err)
	}
	return response, nil
}

// GetVideoComments retrieves comments for a specific video
func (u *YouTubeUseCase) GetVideoComments(ctx context.Context, req *dto.YouTubeCommentListRequest) (*dto.YouTubeCommentResponse, error) {
	if req == nil || req.VideoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}
	if req.MaxResults == 0 {
		req.MaxResults = 20
	}
	if req.Order == "" {
		req.Order = "time"
	}
	if req.MaxResults > 100 {
		req.MaxResults = 100
	}

	response, err := u.youtubeRepo.GetVideoComments(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get video comments: %w", err)
	}
	return response, nil
}

// AddComment adds a comment to a video
func (u *YouTubeUseCase) AddComment(ctx context.Context, req *dto.YouTubeCommentRequest) (*model.YouTubeComment, error) {
	if req == nil {
		return nil, fmt.Errorf("comment request is required")
	}
	if req.VideoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}
	if req.Text == "" {
		return nil, fmt.Errorf("comment text is required")
	}
	if len(req.Text) > 10000 {
		return nil, fmt.Errorf("comment text too long (max 10000 characters)")
	}

	comment, err := u.youtubeRepo.AddComment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}
	return comment, nil
}

// UpdateComment updates an existing comment
func (u *YouTubeUseCase) UpdateComment(ctx context.Context, req *dto.YouTubeCommentUpdateRequest) (*model.YouTubeComment, error) {
	if req == nil {
		return nil, fmt.Errorf("comment update request is required")
	}
	if req.CommentID == "" {
		return nil, fmt.Errorf("comment ID is required")
	}
	if req.Text == "" {
		return nil, fmt.Errorf("comment text is required")
	}
	if len(req.Text) > 10000 {
		return nil, fmt.Errorf("comment text too long (max 10000 characters)")
	}

	comment, err := u.youtubeRepo.UpdateComment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}
	return comment, nil
}

// DeleteComment deletes a comment
func (u *YouTubeUseCase) DeleteComment(ctx context.Context, commentID string) error {
	if commentID == "" {
		return fmt.Errorf("comment ID is required")
	}
	if err := u.youtubeRepo.DeleteComment(ctx, commentID); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}
	return nil
}

// LikeVideo likes a video
func (u *YouTubeUseCase) LikeVideo(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("video ID is required")
	}
	if err := u.youtubeRepo.LikeVideo(ctx, videoID); err != nil {
		return fmt.Errorf("failed to like video: %w", err)
	}
	return nil
}

// DislikeVideo dislikes a video
func (u *YouTubeUseCase) DislikeVideo(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("video ID is required")
	}
	if err := u.youtubeRepo.DislikeVideo(ctx, videoID); err != nil {
		return fmt.Errorf("failed to dislike video: %w", err)
	}
	return nil
}

// RemoveVideoRating removes rating from a video
func (u *YouTubeUseCase) RemoveVideoRating(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("video ID is required")
	}
	if err := u.youtubeRepo.RemoveVideoRating(ctx, videoID); err != nil {
		return fmt.Errorf("failed to remove video rating: %w", err)
	}
	return nil
}

// ToggleCommentLike toggles like state for a user's comment (local)
func (u *YouTubeUseCase) ToggleCommentLike(ctx context.Context, userID, commentID string) (bool, error) {
	if userID == "" || commentID == "" {
		return false, fmt.Errorf("userID and commentID are required")
	}
	liked, err := u.youtubeRepo.ToggleUserCommentLike(ctx, userID, commentID)
	if err != nil {
		return false, fmt.Errorf("failed to toggle comment like: %w", err)
	}
	return liked, nil
}

// ToggleCommentHeart toggles heart state for a user's comment (local)
func (u *YouTubeUseCase) ToggleCommentHeart(ctx context.Context, userID, commentID string) (bool, error) {
	if userID == "" || commentID == "" {
		return false, fmt.Errorf("userID and commentID are required")
	}
	loved, err := u.youtubeRepo.ToggleUserCommentHeart(ctx, userID, commentID)
	if err != nil {
		return false, fmt.Errorf("failed to toggle comment heart: %w", err)
	}
	return loved, nil
}

// GetMyChannel retrieves the authenticated user's channel information
func (u *YouTubeUseCase) GetMyChannel(ctx context.Context) (*model.YouTubeChannel, error) {
	channel, err := u.youtubeRepo.GetMyChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get my channel: %w", err)
	}
	return channel, nil
}

// GetChannelDetails retrieves details for a specific channel
func (u *YouTubeUseCase) GetChannelDetails(ctx context.Context, channelID string) (*model.YouTubeChannel, error) {
	if channelID == "" {
		return nil, fmt.Errorf("channel ID is required")
	}
	channel, err := u.youtubeRepo.GetChannelDetails(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel details: %w", err)
	}
	return channel, nil
}

// GetMyPlaylists retrieves the authenticated user's playlists
func (u *YouTubeUseCase) GetMyPlaylists(ctx context.Context) ([]model.YouTubePlaylist, error) {
	playlists, err := u.youtubeRepo.GetMyPlaylists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get my playlists: %w", err)
	}
	return playlists, nil
}

// CreatePlaylist creates a new playlist
func (u *YouTubeUseCase) CreatePlaylist(ctx context.Context, title, description, privacy string) (*model.YouTubePlaylist, error) {
	if title == "" {
		return nil, fmt.Errorf("playlist title is required")
	}
	if privacy == "" {
		privacy = "private"
	}
	validPrivacy := map[string]bool{"private": true, "public": true, "unlisted": true}
	if !validPrivacy[privacy] {
		return nil, fmt.Errorf("invalid privacy setting: %s", privacy)
	}
	playlist, err := u.youtubeRepo.CreatePlaylist(ctx, title, description, privacy)
	if err != nil {
		return nil, fmt.Errorf("failed to create playlist: %w", err)
	}
	return playlist, nil
}

// DashboardSummary computes aggregated stats from cached videos
func (u *YouTubeUseCase) DashboardSummary(ctx context.Context) (*dto.YouTubeDashboardSummary, error) {
	if u.cache == nil {
		return nil, fmt.Errorf("cache repository not configured")
	}
	// Pull first 500 cached videos (5 pages of 100) for aggregation
	var all []model.YouTubeVideo
	var total int64
	pageSize := 100
	for page := 0; page < 5; page++ {
		items, tot, err := u.cache.ListVideos(ctx, pageSize, page*pageSize)
		if err != nil {
			return nil, err
		}
		if page == 0 {
			total = tot
		}
		if len(items) == 0 {
			break
		}
		all = append(all, items...)
		if len(all) >= 500 || int64(len(all)) >= total {
			break
		}
	}
	// Compute aggregates
	var views int64
	var likes int64
	recent := 0
	now := time.Now()
	thirtyDays := 30 * 24 * time.Hour
	// month -> count map for last 6 months
	months := make([]string, 0, 6)
	counts := make(map[string]int)
	for i := 5; i >= 0; i-- {
		d := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		key := fmt.Sprintf("%04d-%02d", d.Year(), int(d.Month()))
		months = append(months, key)
		counts[key] = 0
	}

	tops := make([]dto.YouTubeDashboardTopVideo, 0, len(all))
	for i := range all {
		v := all[i]
		views += v.ViewCount
		likes += v.LikeCount
		if now.Sub(v.PublishedAt) <= thirtyDays {
			recent++
		}
		key := fmt.Sprintf("%04d-%02d", v.PublishedAt.Year(), int(v.PublishedAt.Month()))
		if _, ok := counts[key]; ok {
			counts[key]++
		}
		thumb := v.Thumbnails.Medium.URL
		if thumb == "" {
			thumb = v.Thumbnails.Default.URL
		}
		tops = append(tops, dto.YouTubeDashboardTopVideo{
			ID:          v.ID,
			Title:       v.Title,
			Views:       v.ViewCount,
			Thumbnail:   thumb,
			PublishedAt: v.PublishedAt.Format(time.RFC3339),
		})
	}
	// Top 5 by views
	sort.Slice(tops, func(i, j int) bool { return tops[i].Views > tops[j].Views })
	if len(tops) > 5 {
		tops = tops[:5]
	}
	monthly := make([]dto.YouTubeMonthlyUpload, 0, len(months))
	for _, m := range months {
		monthly = append(monthly, dto.YouTubeMonthlyUpload{Month: m, Count: counts[m]})
	}
	avgLikes := 0.0
	if n := len(all); n > 0 {
		avgLikes = float64(likes) / float64(n)
	}
	return &dto.YouTubeDashboardSummary{
		TotalVideos:   total,
		TotalViews:    views,
		AvgLikes:      avgLikes,
		RecentUploads: recent,
		Monthly:       monthly,
		TopVideos:     tops,
	}, nil
}
