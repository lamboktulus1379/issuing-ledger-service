package dto

// YouTubeDashboardTopVideo represents a condensed top video record
type YouTubeDashboardTopVideo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Views       int64  `json:"views"`
	Thumbnail   string `json:"thumbnail,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

// YouTubeMonthlyUpload represents uploads count per month (YYYY-MM)
type YouTubeMonthlyUpload struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

// YouTubeDashboardSummary aggregates metrics for the admin dashboard
type YouTubeDashboardSummary struct {
	TotalVideos   int64                      `json:"total_videos"`
	TotalViews    int64                      `json:"total_views"`
	AvgLikes      float64                    `json:"avg_likes"`
	RecentUploads int                        `json:"recent_uploads"`
	Monthly       []YouTubeMonthlyUpload     `json:"monthly_uploads"`
	TopVideos     []YouTubeDashboardTopVideo `json:"top_videos"`
}
