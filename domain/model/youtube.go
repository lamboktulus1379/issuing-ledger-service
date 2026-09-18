package model

import "time"

// YouTubeVideo represents a YouTube video
type YouTubeVideo struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	PublishedAt  time.Time `json:"published_at"`
	ChannelID    string    `json:"channel_id"`
	ChannelName  string    `json:"channel_name"`
	ViewCount    int64     `json:"view_count"`
	LikeCount    int64     `json:"like_count"`
	CommentCount int64     `json:"comment_count"`
	Duration     string    `json:"duration"`
	Thumbnails   struct {
		Default struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"default"`
		Medium struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"medium"`
		High struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"high"`
	} `json:"thumbnails"`
	Tags     []string `json:"tags"`
	Status   string   `json:"status"`
	Category string   `json:"category"`
}

// YouTubeComment represents a YouTube comment
type YouTubeComment struct {
	ID                string           `json:"id"`
	VideoID           string           `json:"video_id"`
	AuthorDisplayName string           `json:"author_display_name"`
	AuthorChannelID   string           `json:"author_channel_id"`
	Text              string           `json:"text"`
	LikeCount         int64            `json:"like_count"`
	PublishedAt       time.Time        `json:"published_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	ParentID          string           `json:"parent_id,omitempty"` // For replies
	ReplyCount        int64            `json:"reply_count"`
	Replies           []YouTubeComment `json:"replies,omitempty"` // Nested replies (direct children)
	Liked             bool             `json:"liked,omitempty"`   // Whether current user liked (not available in API; app-maintained)
	Loved             bool             `json:"loved,omitempty"`   // Whether current user hearted (owner heart) (app-maintained)
}

// YouTubeChannel represents a YouTube channel
type YouTubeChannel struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	CustomURL       string    `json:"custom_url"`
	PublishedAt     time.Time `json:"published_at"`
	SubscriberCount int64     `json:"subscriber_count"`
	VideoCount      int64     `json:"video_count"`
	ViewCount       int64     `json:"view_count"`
	Thumbnails      struct {
		Default struct {
			URL string `json:"url"`
		} `json:"default"`
		Medium struct {
			URL string `json:"url"`
		} `json:"medium"`
		High struct {
			URL string `json:"url"`
		} `json:"high"`
	} `json:"thumbnails"`
}

// YouTubePlaylist represents a YouTube playlist
type YouTubePlaylist struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	PublishedAt time.Time `json:"published_at"`
	ChannelID   string    `json:"channel_id"`
	ItemCount   int64     `json:"item_count"`
	Privacy     string    `json:"privacy"`
	Thumbnails  struct {
		Default struct {
			URL string `json:"url"`
		} `json:"default"`
		Medium struct {
			URL string `json:"url"`
		} `json:"medium"`
		High struct {
			URL string `json:"url"`
		} `json:"high"`
	} `json:"thumbnails"`
}
