package repository

import (
	"context"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/dto"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
)

// IYouTube defines the interface for YouTube repository operations
type IYouTube interface {
	// Video operations
	GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error)
	GetVideoDetails(ctx context.Context, videoID string) (*model.YouTubeVideo, error)
	UploadVideo(ctx context.Context, req *dto.YouTubeVideoUploadRequest) (*model.YouTubeVideo, error)
	UpdateVideo(ctx context.Context, videoID string, updates map[string]interface{}) (*model.YouTubeVideo, error)
	DeleteVideo(ctx context.Context, videoID string) error

	// Fetches latest video from YouTube API, updates DB, and returns it
	FetchAndUpdateFromYouTube(ctx context.Context, videoID string) (*model.YouTubeVideo, error)

	// Search operations
	SearchVideos(ctx context.Context, req *dto.YouTubeSearchRequest) (*dto.YouTubeVideoResponse, error)

	// Comment operations
	GetVideoComments(ctx context.Context, req *dto.YouTubeCommentListRequest) (*dto.YouTubeCommentResponse, error)
	AddComment(ctx context.Context, req *dto.YouTubeCommentRequest) (*model.YouTubeComment, error)
	UpdateComment(ctx context.Context, req *dto.YouTubeCommentUpdateRequest) (*model.YouTubeComment, error)
	DeleteComment(ctx context.Context, commentID string) error

	// Like operations
	LikeVideo(ctx context.Context, videoID string) error
	DislikeVideo(ctx context.Context, videoID string) error
	RemoveVideoRating(ctx context.Context, videoID string) error
	// (Removed unsupported direct comment like/dislike operations – YouTube API lacks endpoints)

	// Local reaction toggles (since YouTube API lacks comment like/heart endpoints)
	ToggleUserCommentLike(ctx context.Context, userID, commentID string) (bool, error)
	ToggleUserCommentHeart(ctx context.Context, userID, commentID string) (bool, error)

	// Channel operations
	GetMyChannel(ctx context.Context) (*model.YouTubeChannel, error)
	GetChannelDetails(ctx context.Context, channelID string) (*model.YouTubeChannel, error)
	GetChannelVideos(ctx context.Context, channelID string, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error)

	// Playlist operations
	GetMyPlaylists(ctx context.Context) ([]model.YouTubePlaylist, error)
	GetPlaylistVideos(ctx context.Context, req *dto.YouTubePlaylistRequest) (*dto.YouTubeVideoResponse, error)
	CreatePlaylist(ctx context.Context, title, description, privacy string) (*model.YouTubePlaylist, error)
	AddVideoToPlaylist(ctx context.Context, playlistID, videoID string) error
	RemoveVideoFromPlaylist(ctx context.Context, playlistID, videoID string) error

	// Analytics (if you want to add analytics data)
	GetVideoAnalytics(ctx context.Context, videoID string, startDate, endDate string) (interface{}, error)
	GetChannelAnalytics(ctx context.Context, startDate, endDate string) (interface{}, error)
}

// IShare defines operations for tracking & processing social shares
type IShare interface {
	UpsertTrackShares(ctx context.Context, videoID, userID string, platforms []string, initialStatus string) ([]*model.VideoShareRecord, error)
	GetShareStatus(ctx context.Context, videoID, userID string) ([]*model.VideoShareRecord, error)
	CreateAudit(ctx context.Context, audits []*model.VideoShareAudit) error
	EnqueueJobs(ctx context.Context, records []*model.VideoShareRecord) error
	FetchPendingJobs(ctx context.Context, limit int) ([]*model.ShareJob, error)
	MarkJobRunning(ctx context.Context, jobID int64) error
	MarkJobResult(ctx context.Context, jobID int64, success bool, errMsg *string) error
	UpdateRecordStatus(ctx context.Context, recordID int64, status string, errMsg *string) error
	GetRecordByID(ctx context.Context, id int64) (*model.VideoShareRecord, error)
	UpdateRecordExternalRef(ctx context.Context, recordID int64, ref string) error
}

// IOAuthToken manages storage of OAuth tokens per platform/user
type IOAuthToken interface {
	UpsertToken(ctx context.Context, token *model.OAuthToken) error
	GetToken(ctx context.Context, userID, platform string) (*model.OAuthToken, error)
}
