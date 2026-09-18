package dto

// YouTubeVideoUpdateRequest represents fields that can be updated for a YouTube video.
// Pointer fields are used so we can distinguish between an omitted field (nil) and an
// explicit empty value (e.g. clearing description or tags).
type YouTubeVideoUpdateRequest struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Tags        *[]string `json:"tags"`
	Privacy     *string   `json:"privacy"`  // private | public | unlisted
	Category    *string   `json:"category"` // Category ID
}
