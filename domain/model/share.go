package model

import "time"

// VideoShareRecord represents the latest state of a share attempt per (video, platform, user)
type VideoShareRecord struct {
	ID           int64     `json:"id"`
	VideoID      string    `json:"video_id"`
	Platform     string    `json:"platform"`
	UserID       string    `json:"user_id"`
	Status       string    `json:"status"` // pending | success | failed
	ErrorMessage *string   `json:"error_message,omitempty"`
	ExternalRef  *string   `json:"external_ref,omitempty"`
	AttemptCount int       `json:"attempt_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// VideoShareAudit is an append-only log of share attempts
type VideoShareAudit struct {
	ID           int64     `json:"id"`
	RecordID     int64     `json:"record_id"`
	VideoID      string    `json:"video_id"`
	Platform     string    `json:"platform"`
	UserID       string    `json:"user_id"`
	Status       string    `json:"status"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// OAuthToken stores platform OAuth credentials per user
type OAuthToken struct {
	ID           int64      `json:"id"`
	UserID       string     `json:"user_id"`
	Platform     string     `json:"platform"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Scopes       string     `json:"scopes"`
	PageID       *string    `json:"page_id,omitempty"`
	PageName     *string    `json:"page_name,omitempty"`
	TokenType    *string    `json:"token_type,omitempty"` // user | page
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ShareJob represents a queued backend share action (server_post)
type ShareJob struct {
	ID          int64     `json:"id"`
	RecordID    int64     `json:"record_id"`
	Platform    string    `json:"platform"`
	Status      string    `json:"status"` // pending | running | success | failed
	Attempts    int       `json:"attempts"`
	LastError   *string   `json:"last_error,omitempty"`
	ExternalRef *string   `json:"external_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
