package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"
	httpHandler "github.com/lamboktulus1379/issuing-ledger-service/interfaces/http"
	"github.com/lamboktulus1379/issuing-ledger-service/interfaces/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func InitiateRouter(
	userHandler httpHandler.IUserHandler,
	testHandler httpHandler.ITestHandler,
	youtubeHandler httpHandler.IYouTubeHandler,
	youtubeAuthHandler httpHandler.IYouTubeAuthHandler,
	userRepository repository.IUser,
	shareHandler httpHandler.IShareHandler,
	facebookOAuthHandler httpHandler.IFacebookOAuthHandler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	// Determine allowed origins from environment (comma-separated), with sensible defaults
	// Env keys supported: ALLOWED_ORIGINS or CORS_ALLOWED_ORIGINS
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	if allowedOriginsEnv == "" {
		allowedOriginsEnv = os.Getenv("CORS_ALLOWED_ORIGINS")
	}
	defaultAllowed := []string{
		"https://tulus.tech",
		"https://admin.tulus.tech",
		"https://tulus.space",
		"https://admin.tulus.space",
		"https://user.tulus.space",
		"https://typing.tulus.space",
		"https://score.tulus.space",
		"https://gra.tulus.space",
		"https://gra.tulus.tech",
		"https://simamora.tech",
		"https://admin.simamora.tech",
		"http://localhost:4201",
		"http://localhost:4200",
		"https://localhost:4201",
		"https://localhost:4200",
	}
	// Parse env into a cleaned list; support wildcard patterns like https://*.tulus.tech
	parseList := func(s string) []string {
		parts := strings.Split(s, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			out = append(out, p)
		}
		return out
	}
	contains := func(list []string, target string) bool {
		for _, v := range list {
			if v == target {
				return true
			}
		}
		return false
	}
	// Wildcard match for entries like https://*.tulus.tech
	matchesWildcard := func(pattern, origin string) bool {
		// only support prefix "http://*." or "https://*." patterns
		if strings.HasPrefix(pattern, "http://*.") {
			suf := strings.TrimPrefix(pattern, "http://*.")
			return strings.HasPrefix(origin, "http://") && strings.HasSuffix(origin, "."+suf)
		}
		if strings.HasPrefix(pattern, "https://*.") {
			suf := strings.TrimPrefix(pattern, "https://*.")
			return strings.HasPrefix(origin, "https://") && strings.HasSuffix(origin, "."+suf)
		}
		return false
	}
	allowedList := defaultAllowed
	if allowedOriginsEnv != "" {
		allowedList = parseList(allowedOriginsEnv)
	}
	// Use both AllowOrigins (for simple cases) and AllowOriginFunc (for wildcard patterns)
	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedList,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			// exact match allowed
			if contains(allowedList, origin) {
				return true
			}
			// wildcard patterns
			for _, p := range allowedList {
				if matchesWildcard(p, origin) {
					return true
				}
			}
			return false
		},
		MaxAge: 12 * time.Hour,
	}))

	// Ensure all preflight requests get a 204 with middleware-applied CORS headers
	router.OPTIONS("/*corsPreflight", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	// Optional auth helper: sets user_id when a valid JWT is provided via Authorization header
	// or auth_token query param; otherwise it leaves context untouched.
	optionalAuth := func(c *gin.Context) {
		authorization := c.Request.Header.Get("Authorization")
		if authorization == "" {
			if qt := c.Query("auth_token"); qt != "" {
				authorization = "Bearer " + qt
			}
		}
		if authorization == "" || !strings.HasPrefix(authorization, "Bearer ") {
			return
		}
		secretKey := configuration.C.App.SecretKey
		tokenStr := strings.TrimPrefix(authorization, "Bearer ")
		var userClaims model.UserClaims
		token, err := jwt.ParseWithClaims(
			tokenStr,
			&userClaims,
			func(token *jwt.Token) (interface{}, error) { return []byte(secretKey), nil },
		)
		if err == nil && token != nil && token.Valid {
			if _, err := userRepository.GetByUserName(c.Request.Context(), userClaims.UserName); err == nil {
				c.Set("user_id", userClaims.Issuer)
			}
		}
	}

	api := router.Group("api")
	api.Use(middleware.Auth(userRepository))

	// Lightweight status endpoint to quickly diagnose YouTube configuration and mode
	api.GET("/youtube/status", func(ctx *gin.Context) {
		cfg, _ := configuration.GetYouTubeConfig()
		modeEnv := os.Getenv("YOUTUBE_MODE")
		enabledEnv := os.Getenv("YOUTUBE_ENABLED")

		hasAccess := cfg != nil && cfg.AccessToken != "" && cfg.AccessToken != "your_access_token_here"
		hasRefresh := cfg != nil && cfg.RefreshToken != "" && cfg.RefreshToken != "your_refresh_token_here"
		hasTokens := hasAccess && hasRefresh
		hasAPIKey := cfg != nil && cfg.APIKey != "" && cfg.APIKey != "YOUR_YOUTUBE_API_KEY"
		channelID := ""
		if cfg != nil {
			channelID = cfg.ChannelID
		}

		clientMode := "none"
		if hasTokens {
			clientMode = "oauth"
		} else if hasAPIKey {
			clientMode = "apiKey"
		}

		// Determine effective mode seen by router
		effectiveMode := "mock"
		if modeEnv == "disabled" || enabledEnv == "false" {
			effectiveMode = "disabled"
		} else if youtubeHandler != nil && (hasTokens || hasAPIKey) {
			effectiveMode = "live"
		}

		ctx.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"mode":            effectiveMode,
				"handlerActive":   youtubeHandler != nil,
				"hasApiKey":       hasAPIKey,
				"hasTokens":       hasTokens,
				"channelIdSet":    channelID != "",
				"channelId":       channelID,
				"clientMode":      clientMode,
				"YOUTUBE_MODE":    modeEnv,
				"YOUTUBE_ENABLED": enabledEnv,
			},
		})
	})

	// Public status (no auth) for quick diagnostics
	router.GET("/youtube/status", func(ctx *gin.Context) {
		cfg, _ := configuration.GetYouTubeConfig()
		modeEnv := os.Getenv("YOUTUBE_MODE")
		enabledEnv := os.Getenv("YOUTUBE_ENABLED")

		hasAccess := cfg != nil && cfg.AccessToken != "" && cfg.AccessToken != "your_access_token_here"
		hasRefresh := cfg != nil && cfg.RefreshToken != "" && cfg.RefreshToken != "your_refresh_token_here"
		hasTokens := hasAccess && hasRefresh
		hasAPIKey := cfg != nil && cfg.APIKey != "" && cfg.APIKey != "YOUR_YOUTUBE_API_KEY"
		channelID := ""
		if cfg != nil {
			channelID = cfg.ChannelID
		}

		clientMode := "none"
		if hasTokens {
			clientMode = "oauth"
		} else if hasAPIKey {
			clientMode = "apiKey"
		}

		effectiveMode := "mock"
		if modeEnv == "disabled" || enabledEnv == "false" {
			effectiveMode = "disabled"
		} else if youtubeHandler != nil && (hasTokens || hasAPIKey) {
			effectiveMode = "live"
		}

		ctx.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"mode":            effectiveMode,
				"handlerActive":   youtubeHandler != nil,
				"hasApiKey":       hasAPIKey,
				"hasTokens":       hasTokens,
				"channelIdSet":    channelID != "",
				"channelId":       channelID,
				"clientMode":      clientMode,
				"YOUTUBE_MODE":    modeEnv,
				"YOUTUBE_ENABLED": enabledEnv,
			},
		})
	})

	// Root endpoint - return a simple message
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "Gra")
	})

	router.POST("/login", userHandler.Login)
	router.POST("/register", userHandler.Register)

	// OAuth authentication routes
	if youtubeAuthHandler != nil {
		router.GET("/auth/youtube", youtubeAuthHandler.GetAuthURL)
		router.GET("/auth/youtube/callback", youtubeAuthHandler.HandleCallback)
		api.GET("/youtube/oauth/status", youtubeAuthHandler.Status)
	}
	if facebookOAuthHandler != nil {
		router.GET("/auth/facebook", facebookOAuthHandler.GetAuthURL)
		router.GET("/auth/facebook/callback", facebookOAuthHandler.Callback)
		// Public read-only status route with optional auth (so FE can check connection without JWT)
		router.GET("/facebook/status", func(c *gin.Context) {
			optionalAuth(c)
			facebookOAuthHandler.Status(c)
		})
		api.GET("/facebook/status", facebookOAuthHandler.Status)
		// Admin seeding endpoints are disabled after initial bootstrap; keep the code available behind API if needed later
		// Admin seeding endpoint removed for security. Use database migrations or controlled scripts if needed.
		api.POST("/facebook/refresh-pages", facebookOAuthHandler.RefreshPages)
		api.POST("/facebook/link-page", facebookOAuthHandler.LinkPage)
		api.POST("/facebook/link-page-url", facebookOAuthHandler.LinkPageURL)
	}

	// Temporary test route for YouTube API (bypasses authentication)
	if youtubeHandler != nil {
		router.GET("/test/youtube/videos", youtubeHandler.GetMyVideos)
		router.GET("/test/youtube/search", youtubeHandler.SearchVideos)
		router.GET("/test/youtube/channel", youtubeHandler.GetMyChannel)
	} else {
		// Fallback mock endpoint if YouTube handler is not available
		router.GET("/test/youtube/videos", func(ctx *gin.Context) {
			ctx.Header("Access-Control-Allow-Origin", "*")
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			ctx.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")

			ctx.JSON(http.StatusOK, gin.H{
				"error":   false,
				"message": "YouTube API not configured - returning mock data",
				"data": []gin.H{
					{
						"id":          "mock-video-1",
						"title":       "Sample Video 1",
						"description": "This is a mock video for testing",
						"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Video+1",
						"viewCount":   "1234",
						"likeCount":   "56",
						"publishedAt": "2025-08-06T00:00:00Z",
					},
					{
						"id":          "mock-video-2",
						"title":       "Sample Video 2",
						"description": "Another mock video for testing",
						"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Video+2",
						"viewCount":   "5678",
						"likeCount":   "89",
						"publishedAt": "2025-08-05T00:00:00Z",
					},
				},
			})
		})
	}

	// Health endpoint
	router.GET("/healthz", testHandler.Healthz)
	router.POST("/healthz", testHandler.Test)

	api.POST("/", func(ctx *gin.Context) {
		res := ctx.Request.Body
		ctx.JSON(http.StatusOK, res)
	})

	// Share platform capability endpoint (not tied to YouTube availability)
	if shareHandler != nil {
		api.GET("/share/platforms", shareHandler.GetPlatforms)
		api.POST("/share/process-jobs", shareHandler.ProcessJobs)
	}

	// Public share-status endpoint with optional auth (so callers without token get an empty list,
	// while authenticated callers receive their records). This exists regardless of YouTube handler presence.
	router.GET("/api/youtube/videos/:videoId/share-status", func(c *gin.Context) {
		if shareHandler != nil {
			optionalAuth(c) // set user_id if token present; otherwise proceed without it
			shareHandler.GetShareStatus(c)
			return
		}
		// Graceful fallback when share handler is not configured
		videoID := c.Param("videoId")
		c.JSON(http.StatusOK, gin.H{"video_id": videoID, "records": []interface{}{}})
	})

	// YouTube API routes (only if handler is available)
	if youtubeHandler != nil {
		youtube := api.Group("/youtube")
		{
			// Video operations
			youtube.GET("/videos", youtubeHandler.GetMyVideos)
			youtube.POST("/sync", youtubeHandler.SyncMyVideos)
			youtube.GET("/videos/:videoId", youtubeHandler.GetVideoDetails)
			youtube.POST("/videos/upload", youtubeHandler.UploadVideo)
			youtube.PATCH("/videos/:videoId", youtubeHandler.UpdateVideo)
			youtube.GET("/search", youtubeHandler.SearchVideos)
			youtube.GET("/summary", youtubeHandler.GetDashboardSummary)

			// Video rating operations
			youtube.POST("/videos/:videoId/like", youtubeHandler.LikeVideo)
			youtube.POST("/videos/:videoId/dislike", youtubeHandler.DislikeVideo)
			youtube.DELETE("/videos/:videoId/rating", youtubeHandler.RemoveVideoRating)

			// Comment operations
			youtube.GET("/videos/:videoId/comments", youtubeHandler.GetVideoComments)
			youtube.POST("/comments", youtubeHandler.AddComment)
			youtube.PUT("/comments/:commentId", youtubeHandler.UpdateComment)
			youtube.DELETE("/comments/:commentId", youtubeHandler.DeleteComment)
			youtube.POST("/comments/:commentId/like", youtubeHandler.ToggleCommentLike)
			youtube.POST("/comments/:commentId/heart", youtubeHandler.ToggleCommentHeart)

			// Channel operations
			youtube.GET("/channel", youtubeHandler.GetMyChannel)
			youtube.GET("/channels/:channelId", youtubeHandler.GetChannelDetails)

			// Playlist operations
			youtube.GET("/playlists", youtubeHandler.GetMyPlaylists)
			youtube.POST("/playlists", youtubeHandler.CreatePlaylist)
			// Share endpoints (video social sharing)
			youtube.POST("/videos/:videoId/share", func(c *gin.Context) {
				if shareHandler != nil {
					shareHandler.ShareVideo(c)
					return
				}
				c.JSON(http.StatusNotImplemented, gin.H{"error": "share handler not configured"})
			})
			// share-status route now exposed publicly above with optional auth; keep only POST here
		}
	} else {
		// Add fallback endpoints when YouTube is not configured
		youtube := api.Group("/youtube")
		{
			youtube.GET("/summary", func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"total_videos":   2,
						"total_views":    6912,
						"avg_likes":      72.5,
						"recent_uploads": 1,
						"monthly_uploads": []gin.H{
							{"month": "2025-04", "count": 0},
							{"month": "2025-05", "count": 1},
							{"month": "2025-06", "count": 0},
							{"month": "2025-07", "count": 1},
							{"month": "2025-08", "count": 0},
							{"month": "2025-09", "count": 0},
						},
						"top_videos": []gin.H{
							{"id": "mock-video-2", "title": "Sample Video 2", "views": 5678, "thumbnail": "https://placehold.co/320x180/png?text=Mock+Video+2", "published_at": "2025-08-05T00:00:00Z"},
							{"id": "mock-video-1", "title": "Sample Video 1", "views": 1234, "thumbnail": "https://placehold.co/320x180/png?text=Mock+Video+1", "published_at": "2025-08-06T00:00:00Z"},
						},
					},
				})
			})
			youtube.GET("/videos", func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{
					"error":   false,
					"message": "YouTube API not configured - returning mock data",
					"data": []gin.H{
						{
							"id":          "mock-video-1",
							"title":       "Sample Video 1",
							"description": "This is a mock video for testing",
							"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Video",
							"viewCount":   "1234",
							"likeCount":   "56",
							"publishedAt": "2025-08-06T00:00:00Z",
						},
						{
							"id":          "mock-video-2",
							"title":       "Sample Video 2",
							"description": "Another mock video for testing",
							"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Video+2",
							"viewCount":   "5678",
							"likeCount":   "89",
							"publishedAt": "2025-08-05T00:00:00Z",
						},
					},
				})
			})

			youtube.GET("/videos/:videoId", func(ctx *gin.Context) {
				videoId := ctx.Param("videoId")
				ctx.JSON(http.StatusOK, gin.H{
					"error":   false,
					"message": "YouTube API not configured - returning mock data",
					"data": gin.H{
						"id":          videoId,
						"title":       "Mock Video Details",
						"description": "This is a mock video detail for testing",
						"thumbnail":   "https://placehold.co/320x180/png?text=Mock+Thumb",
						"viewCount":   "1234",
						"likeCount":   "56",
						"publishedAt": "2025-08-06T00:00:00Z",
						"url":         "https://www.youtube.com/watch?v=" + videoId,
					},
				})
			})

			// Fallback for update route to avoid 404 and provide clear guidance
			youtube.PATCH("/videos/:videoId", func(ctx *gin.Context) {
				ctx.JSON(http.StatusNotImplemented, gin.H{
					"error":         true,
					"message":       "Video update not available - YouTube API not fully configured (OAuth credentials required)",
					"hint":          "Provide access & refresh tokens or complete OAuth flow at /auth/youtube to enable editing",
					"video_id":      ctx.Param("videoId"),
					"documentation": "/docs/YOUTUBE_API_SETUP.md",
				})
			})

			youtube.GET("/channel", func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{
					"error":   false,
					"message": "YouTube API not configured - returning mock data",
					"data": gin.H{
						"id":              "mock-channel-id",
						"title":           "Mock Channel",
						"description":     "This is a mock channel for testing",
						"subscriberCount": "1000",
						"viewCount":       "50000",
						"videoCount":      "25",
					},
				})
			})

			youtube.GET("/search", func(ctx *gin.Context) {
				query := ctx.Query("q")
				ctx.JSON(http.StatusOK, gin.H{
					"error":   false,
					"message": "YouTube API not configured - returning mock search results",
					"query":   query,
					"data": []gin.H{
						{
							"id":          "search-result-1",
							"title":       "Search Result 1 for: " + query,
							"description": "Mock search result",
							"thumbnail":   "https://via.placeholder.com/320x180",
							"viewCount":   "999",
						},
					},
				})
			})

			// For other endpoints, return a "not configured" message
			youtube.GET("/info", func(ctx *gin.Context) {
				ctx.JSON(http.StatusServiceUnavailable, gin.H{
					"error":       "YouTube API not configured",
					"message":     "Please configure YouTube API credentials to enable YouTube features",
					"setup_guide": "/docs/YOUTUBE_API_SETUP.md",
				})
			})
		}
	}

	return router
}
