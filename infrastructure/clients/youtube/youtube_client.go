package youtube

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/dto"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Client represents YouTube API client
type Client struct {
	service     *youtube.Service
	channelID   string
	accessToken string
	oauthConfig *oauth2.Config
	token       *oauth2.Token
	ctx         context.Context
	// in-memory reaction states (userID:commentID)
	commentLikes  map[string]map[string]bool
	commentHearts map[string]map[string]bool
}

// FetchAndUpdateFromYouTube is not supported in the API client; use repository for DB/cache update
func (c *Client) FetchAndUpdateFromYouTube(ctx context.Context, videoID string) (*model.YouTubeVideo, error) {
	return nil, errors.New("FetchAndUpdateFromYouTube is not implemented in the API client; use the repository layer")
}

// keys returns the map's keys as a slice of strings for logging
func keys(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Config represents YouTube API configuration
type Config struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ChannelID    string `json:"channel_id"`
	APIKey       string `json:"api_key"`
}

// NewYouTubeClient creates a new YouTube API client
func NewYouTubeClient(ctx context.Context, config *Config) (repository.IYouTube, error) {
	// If we don't have OAuth credentials but we do have an API key, use API key only mode (read-only)
	if (config.AccessToken == "" || config.RefreshToken == "") && config.APIKey != "" {
		service, err := youtube.NewService(ctx, option.WithAPIKey(config.APIKey))
		if err != nil {
			return nil, fmt.Errorf("failed to create YouTube service with API key: %w", err)
		}
		return &Client{
			service:       service,
			channelID:     config.ChannelID,
			accessToken:   "", // no bearer token
			oauthConfig:   nil,
			token:         nil,
			ctx:           ctx,
			commentLikes:  make(map[string]map[string]bool),
			commentHearts: make(map[string]map[string]bool),
		}, nil
	}

	// Full OAuth2 mode
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes: []string{
			youtube.YoutubeScope,
			youtube.YoutubeUploadScope,
			youtube.YoutubeForceSslScope,
		},
		Endpoint: google.Endpoint,
	}

	token := &oauth2.Token{
		AccessToken:  config.AccessToken,
		RefreshToken: config.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-1 * time.Minute), // Force refresh on first use
	}

	httpClient := oauth2Config.Client(ctx, token)
	service, err := youtube.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube service: %w", err)
	}

	client := &Client{
		service:     service,
		channelID:   config.ChannelID,
		accessToken: config.AccessToken,
		oauthConfig: oauth2Config,
		token:       token,
		ctx:         ctx,
		// ensure in-memory maps are initialized to avoid nil map assignment panics
		commentLikes:  make(map[string]map[string]bool),
		commentHearts: make(map[string]map[string]bool),
	}

	// If ChannelID is empty in OAuth mode, attempt to discover it via Channels.List(mine=true)
	if client.channelID == "" {
		if ch, err := client.GetMyChannel(ctx); err == nil && ch != nil {
			client.channelID = ch.ID
			logger.GetLogger().WithField("channelId", client.channelID).Info("Detected YouTube channel ID via OAuth")
		}
	}

	return client, nil
}

// GetMyVideos retrieves videos from the authenticated user's channel
func (c *Client) GetMyVideos(ctx context.Context, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error) {
	// If no OAuth (API key mode) skip refresh
	if c.oauthConfig != nil && c.token != nil {
		if err := c.refreshTokenIfNeeded(); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	// Fallback mock data when channel ID not set
	channelID := c.channelID
	if req.ChannelID != "" {
		channelID = req.ChannelID
	}
	if channelID == "" {
		mock := &dto.YouTubeVideoResponse{
			YouTubeResponse: dto.YouTubeResponse{
				Kind:     "youtube#searchListResponse",
				PageInfo: dto.PageInfo{TotalResults: 2, ResultsPerPage: 2},
			},
			Items: []interface{}{
				map[string]interface{}{"id": "mock-video-1", "title": "Configure YOUTUBE_CHANNEL_ID", "description": "Set YOUTUBE_CHANNEL_ID env or add to config.json", "view_count": 0, "like_count": 0},
				map[string]interface{}{"id": "mock-video-2", "title": "Using API Key mode", "description": "Provide access & refresh token for authenticated channel data", "view_count": 0, "like_count": 0},
			},
		}
		return mock, nil
	}

	call := c.service.Search.List([]string{"id", "snippet"}).
		ChannelId(channelID).
		Type("video").
		Order("date")

	if req.MaxResults > 0 {
		call = call.MaxResults(req.MaxResults)
	} else {
		call = call.MaxResults(25)
	}
	if req.PageToken != "" {
		call = call.PageToken(req.PageToken)
	}
	if req.Order != "" {
		call = call.Order(req.Order)
	}
	if req.Q != "" {
		call = call.Q(req.Q)
	}
	if req.PublishedAfter != "" {
		if publishedAfter, err := time.Parse(time.RFC3339, req.PublishedAfter); err == nil {
			call = call.PublishedAfter(publishedAfter.Format(time.RFC3339))
		}
	}
	if req.PublishedBefore != "" {
		if publishedBefore, err := time.Parse(time.RFC3339, req.PublishedBefore); err == nil {
			call = call.PublishedBefore(publishedBefore.Format(time.RFC3339))
		}
	}

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	var videoIDs []string
	for _, item := range response.Items {
		videoIDs = append(videoIDs, item.Id.VideoId)
	}

	videos := make([]interface{}, 0)
	if len(videoIDs) > 0 {
		videoDetails, err := c.service.Videos.List([]string{"snippet", "statistics", "contentDetails", "status"}).Id(strings.Join(videoIDs, ",")).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get video details: %w", err)
		}
		for _, video := range videoDetails.Items {
			ytVideo := c.convertToYouTubeVideo(video)
			videos = append(videos, ytVideo)
		}
	}

	return &dto.YouTubeVideoResponse{
		YouTubeResponse: dto.YouTubeResponse{
			Kind:          response.Kind,
			ETag:          response.Etag,
			NextPageToken: response.NextPageToken,
			PrevPageToken: response.PrevPageToken,
			PageInfo:      dto.PageInfo{TotalResults: response.PageInfo.TotalResults, ResultsPerPage: response.PageInfo.ResultsPerPage},
		},
		Items: videos,
	}, nil
}

// GetVideoDetails retrieves details for a specific video
func (c *Client) GetVideoDetails(ctx context.Context, videoID string) (*model.YouTubeVideo, error) {
	call := c.service.Videos.List([]string{"snippet", "statistics", "contentDetails", "status"}).
		Id(videoID)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get video details: %w", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}

	video := c.convertToYouTubeVideo(response.Items[0])
	return &video, nil
}

// convertToYouTubeVideo converts YouTube API video to our model
func (c *Client) convertToYouTubeVideo(video *youtube.Video) model.YouTubeVideo {
	var publishedAt time.Time
	var title, description, channelID, channelTitle, categoryID, duration, status string
	var tags []string

	// Snippet safe extraction
	if video.Snippet != nil {
		if video.Snippet.PublishedAt != "" {
			if ts, err := time.Parse(time.RFC3339, video.Snippet.PublishedAt); err == nil {
				publishedAt = ts
			}
		}
		title = video.Snippet.Title
		description = video.Snippet.Description
		channelID = video.Snippet.ChannelId
		channelTitle = video.Snippet.ChannelTitle
		categoryID = video.Snippet.CategoryId
		tags = video.Snippet.Tags
	}

	if video.ContentDetails != nil {
		duration = video.ContentDetails.Duration
	}

	if video.Status != nil {
		status = video.Status.PrivacyStatus
	}

	var viewCount, likeCount, commentCount int64
	if video.Statistics != nil {
		viewCount = int64(video.Statistics.ViewCount)
		likeCount = int64(video.Statistics.LikeCount)
		commentCount = int64(video.Statistics.CommentCount)
	}

	ytVideo := model.YouTubeVideo{
		ID:           video.Id,
		Title:        title,
		Description:  description,
		PublishedAt:  publishedAt,
		ChannelID:    channelID,
		ChannelName:  channelTitle,
		ViewCount:    viewCount,
		LikeCount:    likeCount,
		CommentCount: commentCount,
		Duration:     duration,
		Tags:         tags,
		Status:       status,
		Category:     categoryID,
	}

	// Thumbnails (nil-safe)
	if video.Snippet != nil && video.Snippet.Thumbnails != nil {
		thumbs := video.Snippet.Thumbnails
		if thumbs.Default != nil {
			ytVideo.Thumbnails.Default.URL = thumbs.Default.Url
			ytVideo.Thumbnails.Default.Width = int(thumbs.Default.Width)
			ytVideo.Thumbnails.Default.Height = int(thumbs.Default.Height)
		}
		if thumbs.Medium != nil {
			ytVideo.Thumbnails.Medium.URL = thumbs.Medium.Url
			ytVideo.Thumbnails.Medium.Width = int(thumbs.Medium.Width)
			ytVideo.Thumbnails.Medium.Height = int(thumbs.Medium.Height)
		}
		if thumbs.High != nil {
			ytVideo.Thumbnails.High.URL = thumbs.High.Url
			ytVideo.Thumbnails.High.Width = int(thumbs.High.Width)
			ytVideo.Thumbnails.High.Height = int(thumbs.High.Height)
		}
	}

	return ytVideo
}

// UploadVideo uploads a video to YouTube
func (c *Client) UploadVideo(ctx context.Context, req *dto.YouTubeVideoUploadRequest) (*model.YouTubeVideo, error) {
	// Open the uploaded file
	file, err := req.File.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Prepare video snippet
	video := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       req.Title,
			Description: req.Description,
			Tags:        req.Tags,
			CategoryId:  req.CategoryID,
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus: req.Privacy,
		},
	}

	// Create upload call
	call := c.service.Videos.Insert([]string{"snippet", "status"}, video)

	// Set the media content
	call = call.Media(file)

	// Execute upload
	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to upload video: %w", err)
	}

	// Convert response to our model
	ytVideo := c.convertToYouTubeVideo(response)
	return &ytVideo, nil
}

// SearchVideos searches for videos on YouTube
func (c *Client) SearchVideos(ctx context.Context, req *dto.YouTubeSearchRequest) (*dto.YouTubeVideoResponse, error) {
	call := c.service.Search.List([]string{"id", "snippet"}).
		Q(req.Q).
		Type("video")

	if req.MaxResults > 0 {
		call = call.MaxResults(req.MaxResults)
	} else {
		call = call.MaxResults(25)
	}

	if req.PageToken != "" {
		call = call.PageToken(req.PageToken)
	}

	if req.Order != "" {
		call = call.Order(req.Order)
	}

	if req.ChannelID != "" {
		call = call.ChannelId(req.ChannelID)
	}

	if req.PublishedAfter != "" {
		if publishedAfter, err := time.Parse(time.RFC3339, req.PublishedAfter); err == nil {
			call = call.PublishedAfter(publishedAfter.Format(time.RFC3339))
		}
	}

	if req.PublishedBefore != "" {
		if publishedBefore, err := time.Parse(time.RFC3339, req.PublishedBefore); err == nil {
			call = call.PublishedBefore(publishedBefore.Format(time.RFC3339))
		}
	}

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to search videos: %w", err)
	}

	// Get video IDs for additional details
	var videoIDs []string
	for _, item := range response.Items {
		if item.Id.VideoId != "" {
			videoIDs = append(videoIDs, item.Id.VideoId)
		}
	}

	// Get video statistics and content details
	videos := make([]interface{}, 0)
	if len(videoIDs) > 0 {
		videoDetails, err := c.service.Videos.List([]string{"snippet", "statistics", "contentDetails", "status"}).
			Id(strings.Join(videoIDs, ",")).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to get video details: %w", err)
		}

		for _, video := range videoDetails.Items {
			ytVideo := c.convertToYouTubeVideo(video)
			videos = append(videos, ytVideo)
		}
	}

	return &dto.YouTubeVideoResponse{
		YouTubeResponse: dto.YouTubeResponse{
			Kind:          response.Kind,
			ETag:          response.Etag,
			NextPageToken: response.NextPageToken,
			PrevPageToken: response.PrevPageToken,
			PageInfo: dto.PageInfo{
				TotalResults:   response.PageInfo.TotalResults,
				ResultsPerPage: response.PageInfo.ResultsPerPage,
			},
		},
		Items: videos,
	}, nil
}

// Placeholder implementations for other methods
// You'll need to implement these based on your specific requirements

func (c *Client) UpdateVideo(ctx context.Context, videoID string, updates map[string]interface{}) (*model.YouTubeVideo, error) {
	if videoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("no updates provided")
	}

	// Need OAuth for update; API key mode insufficient
	if c.oauthConfig == nil || c.token == nil {
		return nil, fmt.Errorf("video update requires OAuth credentials (access + refresh token)")
	}
	if err := c.refreshTokenIfNeeded(); err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	// Fetch existing video to preserve unchanged fields
	logger.GetLogger().WithFields(map[string]interface{}{
		"video_id":    videoID,
		"update_keys": fmt.Sprintf("%v", keys(updates)),
	}).Info("YouTube UpdateVideo: fetching existing video")

	existingResp, err := c.service.Videos.List([]string{"snippet", "status"}).Id(videoID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing video: %w", err)
	}
	if len(existingResp.Items) == 0 {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}
	existing := existingResp.Items[0]

	// Ownership check (if channel IDs known)
	if existing.Snippet != nil && existing.Snippet.ChannelId != "" && c.channelID != "" && existing.Snippet.ChannelId != c.channelID {
		logger.GetLogger().WithFields(map[string]interface{}{
			"video_channel_id":      existing.Snippet.ChannelId,
			"authenticated_channel": c.channelID,
			"video_id":              videoID,
		}).Warn("YouTube UpdateVideo denied: channel mismatch")
		return nil, fmt.Errorf("cannot update video: authenticated channel (%s) does not own video (%s)", c.channelID, existing.Snippet.ChannelId)
	}

	// Apply updates
	if title, ok := updates["title"].(string); ok {
		existing.Snippet.Title = title
	}
	if desc, ok := updates["description"].(string); ok {
		existing.Snippet.Description = desc
	}
	if tags, ok := updates["tags"].([]string); ok {
		existing.Snippet.Tags = tags
	}
	if cat, ok := updates["category"].(string); ok {
		existing.Snippet.CategoryId = cat
	}
	if privacy, ok := updates["privacy"].(string); ok {
		if existing.Status == nil {
			existing.Status = &youtube.VideoStatus{}
		}
		existing.Status.PrivacyStatus = privacy
	}

	// Perform update call (videos.update requires both snippet & status parts when modifying those fields)
	logger.GetLogger().WithFields(map[string]interface{}{
		"video_id":        videoID,
		"applied_updates": updates,
	}).Info("YouTube UpdateVideo: performing update")
	updatedResp, err := c.service.Videos.Update([]string{"snippet", "status"}, existing).Do()
	if err != nil {
		// Unwrap googleapi error for better diagnostics
		var gErr *googleapi.Error
		if errors.As(err, &gErr) {
			// Build guidance
			guidance := ""
			reasons := []string{}
			for _, e := range gErr.Errors {
				if e.Reason != "" {
					reasons = append(reasons, e.Reason)
				}
			}
			switch gErr.Code {
			case 401:
				guidance = "Token unauthorized or expired. Re-run /auth/youtube to refresh OAuth credentials."
			case 403:
				guidance = "Forbidden: Ensure the OAuth consent granted includes youtube.upload scope and that the authenticated account owns the video. Re-run /auth/youtube and accept requested scopes."
			default:
				guidance = "Check OAuth scopes and video ownership."
			}
			logger.GetLogger().WithFields(map[string]interface{}{
				"video_id": videoID,
				"code":     gErr.Code,
				"message":  gErr.Message,
				"reasons":  reasons,
			}).Error("YouTube update failed")
			return nil, fmt.Errorf("failed to update video (code %d): %s | reasons: %v | guidance: %s", gErr.Code, gErr.Message, reasons, guidance)
		}
		return nil, fmt.Errorf("failed to update video: %w", err)
	}

	ytVideo := c.convertToYouTubeVideo(updatedResp)
	return &ytVideo, nil
}

func (c *Client) DeleteVideo(ctx context.Context, videoID string) error {
	// Implementation for deleting video
	return fmt.Errorf("not implemented yet")
}

func (c *Client) GetVideoComments(ctx context.Context, req *dto.YouTubeCommentListRequest) (*dto.YouTubeCommentResponse, error) {
	// Read-only mode with API key only cannot access comments.list for mine? It can list public comments though.
	// Attempt anyway; if in pure API key mode (no oauthConfig/token) the YouTube Data API will still return public comments.
	if req == nil || req.VideoID == "" {
		return nil, fmt.Errorf("video ID is required")
	}

	if c.oauthConfig != nil && c.token != nil {
		if err := c.refreshTokenIfNeeded(); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	// Default max results
	max := req.MaxResults
	if max <= 0 || max > 100 {
		max = 20
	}

	// Order: time | relevance
	order := req.Order
	if order == "" {
		order = "time"
	}

	call := c.service.CommentThreads.List([]string{"snippet", "replies"}).VideoId(req.VideoID).MaxResults(max).Order(order)
	if req.PageToken != "" {
		call = call.PageToken(req.PageToken)
	}

	resp, err := call.Do()
	if err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) {
			return nil, fmt.Errorf("failed to get comments (code %d): %s", gErr.Code, gErr.Message)
		}
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	comments := make([]interface{}, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item == nil || item.Snippet == nil || item.Snippet.TopLevelComment == nil || item.Snippet.TopLevelComment.Snippet == nil {
			continue
		}
		cmt := c.convertThreadToCommentModel(item)
		// Build nested replies slice (instead of flattening) so FE can render threads
		if item.Replies != nil && len(item.Replies.Comments) > 0 {
			for _, r := range item.Replies.Comments {
				if r == nil || r.Snippet == nil {
					continue
				}
				replyModel := model.YouTubeComment{
					ID:                r.Id,
					VideoID:           req.VideoID,
					AuthorDisplayName: r.Snippet.AuthorDisplayName,
					AuthorChannelID:   r.Snippet.AuthorChannelId.Value,
					Text:              r.Snippet.TextDisplay,
					LikeCount:         int64(r.Snippet.LikeCount),
					PublishedAt:       parseTime(r.Snippet.PublishedAt),
					UpdatedAt:         parseTime(r.Snippet.UpdatedAt),
					ParentID:          item.Snippet.TopLevelComment.Id,
					ReplyCount:        0,
				}
				cmt.Replies = append(cmt.Replies, replyModel)
			}
		}
		comments = append(comments, cmt)
	}

	// Enrich with reaction state if user id context key exists
	if ctx != nil {
		if uidVal := ctx.Value("user_id"); uidVal != nil {
			if uid, ok := uidVal.(string); ok {
				c.EnrichReactions(uid, comments)
			}
		}
	}

	out := &dto.YouTubeCommentResponse{
		YouTubeResponse: dto.YouTubeResponse{
			Kind:          resp.Kind,
			ETag:          resp.Etag,
			NextPageToken: resp.NextPageToken,
			PrevPageToken: "", // Not provided by commentThreads.list
			PageInfo: dto.PageInfo{
				TotalResults:   int64(resp.PageInfo.TotalResults),
				ResultsPerPage: int64(resp.PageInfo.ResultsPerPage),
			},
		},
		Items: comments,
	}
	return out, nil
}

func (c *Client) AddComment(ctx context.Context, req *dto.YouTubeCommentRequest) (*model.YouTubeComment, error) {
	if req == nil || req.VideoID == "" || req.Text == "" {
		return nil, fmt.Errorf("video ID and text are required")
	}
	// Must have OAuth to write comments
	if c.oauthConfig == nil || c.token == nil {
		return nil, fmt.Errorf("comment creation requires OAuth credentials (access & refresh token)")
	}
	if err := c.refreshTokenIfNeeded(); err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if req.ParentID != "" {
		// This is a reply -> use comments.insert with parentId
		comment := &youtube.Comment{
			Snippet: &youtube.CommentSnippet{
				TextOriginal: req.Text,
				ParentId:     req.ParentID,
			},
		}
		insertCall := c.service.Comments.Insert([]string{"snippet"}, comment)
		created, err := insertCall.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to add reply: %w", err)
		}
		modelComment := &model.YouTubeComment{
			ID:                created.Id,
			VideoID:           req.VideoID,
			AuthorDisplayName: created.Snippet.AuthorDisplayName,
			AuthorChannelID:   created.Snippet.AuthorChannelId.Value,
			Text:              created.Snippet.TextDisplay,
			LikeCount:         int64(created.Snippet.LikeCount),
			PublishedAt:       parseTime(created.Snippet.PublishedAt),
			UpdatedAt:         parseTime(created.Snippet.UpdatedAt),
			ParentID:          req.ParentID,
			ReplyCount:        0,
		}
		return modelComment, nil
	}

	thread := &youtube.CommentThread{
		Snippet: &youtube.CommentThreadSnippet{
			VideoId: req.VideoID,
			TopLevelComment: &youtube.Comment{
				Snippet: &youtube.CommentSnippet{
					TextOriginal: req.Text,
				},
			},
		},
	}
	call := c.service.CommentThreads.Insert([]string{"snippet"}, thread)
	created, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	modelComment := c.convertThreadToCommentModel(created)
	return &modelComment, nil
}

func (c *Client) UpdateComment(ctx context.Context, req *dto.YouTubeCommentUpdateRequest) (*model.YouTubeComment, error) {
	if req == nil || req.CommentID == "" || req.Text == "" {
		return nil, fmt.Errorf("comment ID and text are required")
	}
	if c.oauthConfig == nil || c.token == nil {
		return nil, fmt.Errorf("comment update requires OAuth credentials")
	}
	if err := c.refreshTokenIfNeeded(); err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// Need to retrieve existing comment first because update requires full resource
	getCall := c.service.Comments.List([]string{"snippet"}).Id(req.CommentID)
	listResp, err := getCall.Do()
	if err != nil || len(listResp.Items) == 0 {
		return nil, fmt.Errorf("failed to fetch existing comment for update: %w", err)
	}
	orig := listResp.Items[0]
	orig.Snippet.TextOriginal = req.Text

	updateCall := c.service.Comments.Update([]string{"snippet"}, orig)
	updated, err := updateCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}
	modelComment := &model.YouTubeComment{
		ID:                updated.Id,
		VideoID:           updated.Snippet.VideoId,
		AuthorDisplayName: updated.Snippet.AuthorDisplayName,
		AuthorChannelID:   updated.Snippet.AuthorChannelId.Value,
		Text:              updated.Snippet.TextDisplay,
		LikeCount:         int64(updated.Snippet.LikeCount),
		PublishedAt:       parseTime(updated.Snippet.PublishedAt),
		UpdatedAt:         parseTime(updated.Snippet.UpdatedAt),
		ParentID:          updated.Snippet.ParentId,
		ReplyCount:        0,
	}
	return modelComment, nil
}

func (c *Client) DeleteComment(ctx context.Context, commentID string) error {
	if commentID == "" {
		return fmt.Errorf("comment ID is required")
	}
	if c.oauthConfig == nil || c.token == nil {
		return fmt.Errorf("comment deletion requires OAuth credentials")
	}
	if err := c.refreshTokenIfNeeded(); err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	call := c.service.Comments.Delete(commentID)
	if err := call.Do(); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}
	return nil
}

func (c *Client) LikeVideo(ctx context.Context, videoID string) error {
	// Implementation for liking video
	return fmt.Errorf("not implemented yet")
}

func (c *Client) DislikeVideo(ctx context.Context, videoID string) error {
	// Implementation for disliking video
	return fmt.Errorf("not implemented yet")
}

func (c *Client) RemoveVideoRating(ctx context.Context, videoID string) error {
	// Implementation for removing video rating
	return fmt.Errorf("not implemented yet")
}

// (Removed unsupported comment like/dislike/rating methods)

// ToggleUserCommentLike implements repository method using in-memory maps
func (c *Client) ToggleUserCommentLike(ctx context.Context, userID, commentID string) (bool, error) {
	return c.ToggleCommentLike(userID, commentID), nil
}

// ToggleUserCommentHeart implements repository method using in-memory maps
func (c *Client) ToggleUserCommentHeart(ctx context.Context, userID, commentID string) (bool, error) {
	return c.ToggleCommentHeart(userID, commentID), nil
}

// convertThreadToCommentModel converts a CommentThread to our model (top-level comment only)
func (c *Client) convertThreadToCommentModel(th *youtube.CommentThread) model.YouTubeComment {
	var out model.YouTubeComment
	if th == nil || th.Snippet == nil || th.Snippet.TopLevelComment == nil || th.Snippet.TopLevelComment.Snippet == nil {
		return out
	}
	sn := th.Snippet.TopLevelComment.Snippet
	out = model.YouTubeComment{
		ID:                th.Snippet.TopLevelComment.Id,
		VideoID:           th.Snippet.VideoId,
		AuthorDisplayName: sn.AuthorDisplayName,
		AuthorChannelID:   sn.AuthorChannelId.Value,
		Text:              sn.TextDisplay,
		LikeCount:         int64(sn.LikeCount),
		PublishedAt:       parseTime(sn.PublishedAt),
		UpdatedAt:         parseTime(sn.UpdatedAt),
		ReplyCount:        int64(th.Snippet.TotalReplyCount),
	}
	return out
}

// ToggleCommentLike toggles like state in memory for a user
func (c *Client) ToggleCommentLike(userID, commentID string) (liked bool) {
	if userID == "" || commentID == "" {
		return false
	}
	if c.commentLikes[userID] == nil {
		c.commentLikes[userID] = make(map[string]bool)
	}
	cur := c.commentLikes[userID][commentID]
	c.commentLikes[userID][commentID] = !cur
	return !cur
}

// ToggleCommentHeart toggles heart (love) state in memory for a user (or channel owner)
func (c *Client) ToggleCommentHeart(userID, commentID string) (hearted bool) {
	if userID == "" || commentID == "" {
		return false
	}
	if c.commentHearts[userID] == nil {
		c.commentHearts[userID] = make(map[string]bool)
	}
	cur := c.commentHearts[userID][commentID]
	c.commentHearts[userID][commentID] = !cur
	return !cur
}

// EnrichReactions annotates comments with liked/loved flags for a user
func (c *Client) EnrichReactions(userID string, comments []interface{}) {
	if userID == "" {
		return
	}
	likes := c.commentLikes[userID]
	hearts := c.commentHearts[userID]
	for i, v := range comments {
		if cm, ok := v.(model.YouTubeComment); ok {
			if likes != nil && likes[cm.ID] {
				cm.Liked = true
			}
			if hearts != nil && hearts[cm.ID] {
				cm.Loved = true
			}
			// update slice
			comments[i] = cm
		}
	}
}

// parseTime safely parses RFC3339 time returning zero time on failure
func parseTime(v string) time.Time {
	t, _ := time.Parse(time.RFC3339, v)
	return t
}

func (c *Client) GetMyChannel(ctx context.Context) (*model.YouTubeChannel, error) {
	call := c.service.Channels.List([]string{"snippet", "statistics", "contentDetails"}).
		Mine(true)

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get my channel: %w", err)
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("no channel found for authenticated user")
	}

	channel := response.Items[0]
	publishedAt, _ := time.Parse(time.RFC3339, channel.Snippet.PublishedAt)

	ytChannel := &model.YouTubeChannel{
		ID:          channel.Id,
		Title:       channel.Snippet.Title,
		Description: channel.Snippet.Description,
		CustomURL:   channel.Snippet.CustomUrl,
		PublishedAt: publishedAt,
	}

	// Set thumbnails
	if channel.Snippet.Thumbnails != nil {
		if channel.Snippet.Thumbnails.Default != nil {
			ytChannel.Thumbnails.Default.URL = channel.Snippet.Thumbnails.Default.Url
		}
		if channel.Snippet.Thumbnails.Medium != nil {
			ytChannel.Thumbnails.Medium.URL = channel.Snippet.Thumbnails.Medium.Url
		}
		if channel.Snippet.Thumbnails.High != nil {
			ytChannel.Thumbnails.High.URL = channel.Snippet.Thumbnails.High.Url
		}
	}

	// Set statistics
	if channel.Statistics != nil {
		ytChannel.ViewCount = int64(channel.Statistics.ViewCount)
		ytChannel.SubscriberCount = int64(channel.Statistics.SubscriberCount)
		ytChannel.VideoCount = int64(channel.Statistics.VideoCount)
	}

	return ytChannel, nil
}

func (c *Client) GetChannelDetails(ctx context.Context, channelID string) (*model.YouTubeChannel, error) {
	// Implementation for getting channel details
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) GetChannelVideos(ctx context.Context, channelID string, req *dto.YouTubeVideoListRequest) (*dto.YouTubeVideoResponse, error) {
	// Implementation for getting channel videos
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) GetMyPlaylists(ctx context.Context) ([]model.YouTubePlaylist, error) {
	// Implementation for getting my playlists
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) GetPlaylistVideos(ctx context.Context, req *dto.YouTubePlaylistRequest) (*dto.YouTubeVideoResponse, error) {
	// Implementation for getting playlist videos
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) CreatePlaylist(ctx context.Context, title, description, privacy string) (*model.YouTubePlaylist, error) {
	// Implementation for creating playlist
	return nil, fmt.Errorf("not implemented yet")
}

func (c *Client) AddVideoToPlaylist(ctx context.Context, playlistID, videoID string) error {
	// Implementation for adding video to playlist
	return fmt.Errorf("not implemented yet")
}

func (c *Client) RemoveVideoFromPlaylist(ctx context.Context, playlistID, videoID string) error {
	// Implementation for removing video from playlist
	return fmt.Errorf("not implemented yet")
}

func (c *Client) GetVideoAnalytics(ctx context.Context, videoID string, startDate, endDate string) (interface{}, error) {
	// Implementation for getting video analytics
	return nil, fmt.Errorf("not implemented yet")
}

// refreshTokenIfNeeded checks if the token is expired and refreshes it automatically
func (c *Client) refreshTokenIfNeeded() error {
	// In API key mode (no oauthConfig/token) nothing to do
	if c.oauthConfig == nil || c.token == nil {
		return nil
	}
	if c.token.Expiry.IsZero() || time.Until(c.token.Expiry) < 5*time.Minute {
		newToken, err := c.oauthConfig.TokenSource(c.ctx, c.token).Token()
		if err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}
		c.token = newToken
		c.accessToken = newToken.AccessToken
		httpClient := c.oauthConfig.Client(c.ctx, newToken)
		service, err := youtube.NewService(c.ctx, option.WithHTTPClient(httpClient))
		if err != nil {
			return fmt.Errorf("failed to recreate YouTube service with refreshed token: %w", err)
		}
		c.service = service
		fmt.Printf("Token refreshed successfully. New expiry: %v\n", newToken.Expiry)
	}
	return nil
}

func (c *Client) GetChannelAnalytics(ctx context.Context, startDate, endDate string) (interface{}, error) {
	// Implementation for getting channel analytics
	return nil, fmt.Errorf("not implemented yet")
}
