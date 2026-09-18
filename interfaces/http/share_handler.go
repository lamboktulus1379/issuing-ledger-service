package http

import (
	"net/http"
	"strconv"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
	"github.com/lamboktulus1379/issuing-ledger-service/usecase"

	"github.com/gin-gonic/gin"
)

type IShareHandler interface {
	ShareVideo(ctx *gin.Context)
	GetShareStatus(ctx *gin.Context)
	GetPlatforms(ctx *gin.Context)
	ProcessJobs(ctx *gin.Context)
}

type ShareHandler struct {
	shareUsecase usecase.IShareUsecase
	platforms    []string
}

func NewShareHandler(uc usecase.IShareUsecase, platforms []string) IShareHandler {
	return &ShareHandler{shareUsecase: uc, platforms: platforms}
}

type shareRequest struct {
	Platforms []string `json:"platforms"`
	Mode      string   `json:"mode"`
	Force     bool     `json:"force"`
}

func (h *ShareHandler) ShareVideo(ctx *gin.Context) {
	videoID := ctx.Param("videoId")
	userID := ctx.GetString("user_id")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized: missing user_id"})
		return
	}
	var req shareRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	mode := usecase.ShareMode(req.Mode)
	if mode == "" {
		mode = usecase.ShareModeTrackOnly
	}
	results, err := h.shareUsecase.Share(ctx.Request.Context(), videoID, userID, req.Platforms, mode, req.Force)
	if err != nil {
		logger.GetLogger().WithField("video_id", videoID).WithField("user_id", userID).WithField("error", err.Error()).Warn("share request failed")
		// Distinguish validation vs server errors simplistically
		status := http.StatusBadRequest
		if err.Error() == "videoID and userID required" {
			status = http.StatusBadRequest
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"video_id": videoID, "results": results})
}

func (h *ShareHandler) GetShareStatus(ctx *gin.Context) {
	videoID := ctx.Param("videoId")
	userID := ctx.GetString("user_id")
	list, err := h.shareUsecase.GetStatus(ctx.Request.Context(), videoID, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []*model.VideoShareRecord{}
	}
	ctx.JSON(http.StatusOK, gin.H{"video_id": videoID, "records": list})
}

func (h *ShareHandler) GetPlatforms(ctx *gin.Context) {
	caps := make([]gin.H, 0, len(h.platforms))
	for _, p := range h.platforms {
		caps = append(caps, gin.H{"platform": p, "server_post_supported": p == "twitter" || p == "facebook", "implemented": false})
	}
	ctx.JSON(http.StatusOK, gin.H{"platforms": caps})
}

// ProcessJobs allows manual triggering of pending share job processing (admin/dev utility)
func (h *ShareHandler) ProcessJobs(ctx *gin.Context) {
	batchSize := 10
	if v := ctx.Query("batch"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			batchSize = n
		}
	}
	if err := h.shareUsecase.ProcessPending(ctx.Request.Context(), batchSize); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"processed": false, "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"processed": true, "batch": batchSize})
}
