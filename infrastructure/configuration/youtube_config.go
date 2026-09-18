package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// YouTubeConfig represents YouTube API configuration
type YouTubeConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
	AccessToken  string `mapstructure:"access_token"`
	RefreshToken string `mapstructure:"refresh_token"`
	ChannelID    string `mapstructure:"channel_id"`
	APIKey       string `mapstructure:"api_key"`
}

// GetYouTubeConfig returns YouTube configuration from JSON config with environment variable fallback
func GetYouTubeConfig() (*YouTubeConfig, error) {
	// Prefer https redirect locally if TLS is enabled, else http fallback,
	// and honor the configured application port.
	scheme := "http"
	if C.App.TLSEnabled {
		scheme = "https"
	}
	port := C.App.Port
	if port == 0 {
		port = 10001
	}
	defaultRedirect := fmt.Sprintf("%s://localhost:%d/auth/youtube/callback", scheme, port)
	config := &YouTubeConfig{
		ClientID:     getConfigValue(C.YouTube.ClientID, "YOUTUBE_CLIENT_ID", ""),
		ClientSecret: getConfigValue(C.YouTube.ClientSecret, "YOUTUBE_CLIENT_SECRET", ""),
		RedirectURL:  getConfigValue(C.YouTube.RedirectURI, "YOUTUBE_REDIRECT_URL", defaultRedirect),
		AccessToken:  getEnv("YOUTUBE_ACCESS_TOKEN", ""),
		RefreshToken: getEnv("YOUTUBE_REFRESH_TOKEN", ""),
		ChannelID:    getConfigValue(C.YouTube.ChannelID, "YOUTUBE_CHANNEL_ID", ""),
		APIKey:       getConfigValue(C.YouTube.APIKey, "YOUTUBE_API_KEY", ""),
	}

	// Fallback: if access/refresh tokens are empty, attempt to read token.json produced by OAuth callback
	if config.AccessToken == "" || config.RefreshToken == "" {
		if data, err := os.ReadFile("token.json"); err == nil {
			var tokenFile struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			}
			if jsonErr := json.Unmarshal(data, &tokenFile); jsonErr == nil {
				if config.AccessToken == "" && tokenFile.AccessToken != "" {
					config.AccessToken = tokenFile.AccessToken
				}
				if config.RefreshToken == "" && tokenFile.RefreshToken != "" {
					config.RefreshToken = tokenFile.RefreshToken
				}
			}
		}
	}

	// Do not hard-fail when API key or tokens are missing; allow OAuth-only flows to proceed.
	// Client initialization will decide between API-key mode (read-only) and OAuth mode.
	// Note: You can force mock-only behavior by setting YOUTUBE_MODE=mock (or YOUTUBE_ENABLED=false).
	return config, nil
}

// getConfigValue gets value from config first, then environment variable, then default
func getConfigValue(configValue, envKey, defaultValue string) string {
	// Environment variable takes precedence when provided
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	// Otherwise use config value if set and not a placeholder
	if configValue != "" && !strings.HasPrefix(configValue, "YOUR_") {
		return configValue
	}
	// Fallback default
	return defaultValue
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
