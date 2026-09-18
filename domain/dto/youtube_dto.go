package dto

import "mime/multipart"

// YouTubeVideoRequest represents request for video operations
type YouTubeVideoRequest struct {
	VideoID string `json:"video_id" binding:"required"`
}

// YouTubeVideoListRequest represents request for listing videos
type YouTubeVideoListRequest struct {
	ChannelID       string `json:"channel_id,omitempty"`
	MaxResults      int64  `json:"max_results,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
	Order           string `json:"order,omitempty"` // date, rating, relevance, title, videoCount, viewCount
	PublishedAfter  string `json:"published_after,omitempty"`
	PublishedBefore string `json:"published_before,omitempty"`
	Q               string `json:"q,omitempty"`   // search query
	All             bool   `json:"all,omitempty"` // when true, iterate all pages during sync
}

// YouTubeVideoUploadRequest represents request for video upload
type YouTubeVideoUploadRequest struct {
	Title       string                `form:"title" binding:"required"`
	Description string                `form:"description"`
	Tags        []string              `form:"tags"`
	CategoryID  string                `form:"category_id"`
	Privacy     string                `form:"privacy"` // private, public, unlisted
	File        *multipart.FileHeader `form:"file" binding:"required"`
}

// YouTubeCommentRequest represents request for comment operations
type YouTubeCommentRequest struct {
	VideoID  string `json:"video_id" binding:"required"`
	Text     string `json:"text" binding:"required"`
	ParentID string `json:"parent_id,omitempty"` // For replies
}

// YouTubeCommentListRequest represents request for listing comments
type YouTubeCommentListRequest struct {
	VideoID    string `json:"video_id" binding:"required"`
	MaxResults int64  `json:"max_results,omitempty"`
	PageToken  string `json:"page_token,omitempty"`
	Order      string `json:"order,omitempty"` // time, relevance
}

// YouTubeCommentUpdateRequest represents request for updating comments
type YouTubeCommentUpdateRequest struct {
	CommentID string `json:"comment_id" binding:"required"`
	Text      string `json:"text" binding:"required"`
}

// YouTubeLikeRequest represents request for like/unlike operations
type YouTubeLikeRequest struct {
	VideoID   string `json:"video_id,omitempty"`
	CommentID string `json:"comment_id,omitempty"`
	Action    string `json:"action" binding:"required"` // like, dislike, none
}

// YouTubeSearchRequest represents request for searching videos
type YouTubeSearchRequest struct {
	Q               string `json:"q" binding:"required"`
	MaxResults      int64  `json:"max_results,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
	Order           string `json:"order,omitempty"` // date, rating, relevance, title, viewCount
	Type            string `json:"type,omitempty"`  // video, channel, playlist
	ChannelID       string `json:"channel_id,omitempty"`
	PublishedAfter  string `json:"published_after,omitempty"`
	PublishedBefore string `json:"published_before,omitempty"`
	Duration        string `json:"duration,omitempty"`   // short, medium, long
	Definition      string `json:"definition,omitempty"` // high, standard
}

// YouTubeChannelRequest represents request for channel operations
type YouTubeChannelRequest struct {
	ChannelID string `json:"channel_id"`
	Username  string `json:"username"`
}

// YouTubePlaylistRequest represents request for playlist operations
type YouTubePlaylistRequest struct {
	PlaylistID string `json:"playlist_id" binding:"required"`
	MaxResults int64  `json:"max_results,omitempty"`
	PageToken  string `json:"page_token,omitempty"`
}

// YouTubeResponse represents generic YouTube API response
type YouTubeResponse struct {
	Kind          string      `json:"kind"`
	ETag          string      `json:"etag"`
	NextPageToken string      `json:"next_page_token,omitempty"`
	PrevPageToken string      `json:"prev_page_token,omitempty"`
	PageInfo      PageInfo    `json:"page_info"`
	Items         interface{} `json:"items"`
}

// PageInfo represents pagination information
type PageInfo struct {
	TotalResults   int64 `json:"total_results"`
	ResultsPerPage int64 `json:"results_per_page"`
}

// YouTubeVideoResponse represents video response
type YouTubeVideoResponse struct {
	YouTubeResponse
	Items []interface{} `json:"items"` // Will contain YouTubeVideo objects
}

// YouTubeCommentResponse represents comment response
type YouTubeCommentResponse struct {
	YouTubeResponse
	Items []interface{} `json:"items"` // Will contain YouTubeComment objects
}
