package server

import (
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	httpHandler "github.com/lamboktulus1379/issuing-ledger-service/interfaces/http"
	"github.com/lamboktulus1379/issuing-ledger-service/interfaces/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func InitiateRouter(
	userHandler httpHandler.IUserHandler,
	testHandler httpHandler.ITestHandler,
	userRepository repository.IUser,
	issuingHandler *httpHandler.IssuingHandler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Metrics())
	registerPprof(router)
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
	router.Use(otelgin.Middleware("ledger-service"))

	// Ensure all preflight requests get a 204 with middleware-applied CORS headers
	router.OPTIONS("/*corsPreflight", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	api := router.Group("api")
	api.Use(middleware.Auth(userRepository))

	// Root endpoint - return a simple message
	router.GET("/", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "Gra")
	})

	router.POST("/login", userHandler.Login)
	router.POST("/register", userHandler.Register)

	// Health endpoint
	router.GET("/healthz", testHandler.Healthz)
	router.POST("/healthz", testHandler.Test)

	api.POST("/", func(ctx *gin.Context) {
		res := ctx.Request.Body
		ctx.JSON(http.StatusOK, res)
	})

	if issuingHandler != nil {
		api.POST("/authorize", gin.WrapF(issuingHandler.HTTPHandler().ServeHTTP))
	}

	return router
}

func registerPprof(router *gin.Engine) {
	if strings.ToLower(strings.TrimSpace(os.Getenv("PPROF_ENABLED"))) != "true" {
		return
	}

	token := strings.TrimSpace(os.Getenv("PPROF_TOKEN"))
	if token == "" {
		return
	}

	router.Use(func(ctx *gin.Context) {
		if strings.HasPrefix(ctx.Request.URL.Path, "/debug/pprof") && ctx.GetHeader("X-PPROF-TOKEN") != token {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		ctx.Next()
	})
	runtime.SetBlockProfileRate(1)
	pprof.Register(router, "/debug/pprof")
}
