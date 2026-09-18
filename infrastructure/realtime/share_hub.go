package realtime

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"

	"github.com/gin-gonic/gin"
)

// ShareStatusEvent represents an SSE payload for share status updates.
type ShareStatusEvent struct {
	Type         string  `json:"type"`
	VideoID      string  `json:"video_id"`
	Platform     string  `json:"platform"`
	Status       string  `json:"status"`
	ExternalRef  *string `json:"external_ref,omitempty"`
	Error        *string `json:"error,omitempty"`
	AttemptCount int     `json:"attempt_count,omitempty"`
}

// Hub maintains per-user subscribers listening for share status events.
type Hub struct {
	mu    sync.RWMutex
	users map[string]map[chan ShareStatusEvent]struct{}
}

func NewShareHub() *Hub {
	return &Hub{users: make(map[string]map[chan ShareStatusEvent]struct{})}
}

// Serve registers an SSE stream for the authenticated user (user_id set by middleware).
func (h *Hub) Serve(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.Status(http.StatusUnauthorized)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable nginx buffering

	ch := make(chan ShareStatusEvent, 8)
	h.addSubscriber(userID, ch)
	defer h.removeSubscriber(userID, ch)

	// Initial comment to keep connection open
	c.Writer.Write([]byte(":ok\n\n"))
	c.Writer.Flush()

	notify := c.Writer.CloseNotify()
	for {
		select {
		case <-notify:
			return
		case evt := <-ch:
			data, _ := json.Marshal(evt)
			_, _ = c.Writer.Write([]byte("event: share_status\n"))
			_, _ = c.Writer.Write([]byte("data: "))
			_, _ = c.Writer.Write(data)
			_, _ = c.Writer.Write([]byte("\n\n"))
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

func (h *Hub) addSubscriber(userID string, ch chan ShareStatusEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.users[userID] == nil {
		h.users[userID] = make(map[chan ShareStatusEvent]struct{})
	}
	h.users[userID][ch] = struct{}{}
}

func (h *Hub) removeSubscriber(userID string, ch chan ShareStatusEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs := h.users[userID]; subs != nil {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(h.users, userID)
		}
	}
}

// BroadcastShareStatus broadcasts to all subscribers of the user who owns the record.
func (h *Hub) BroadcastShareStatus(rec *model.VideoShareRecord) {
	if rec == nil {
		return
	}
	evt := ShareStatusEvent{
		Type:         "share_status",
		VideoID:      rec.VideoID,
		Platform:     rec.Platform,
		Status:       rec.Status,
		ExternalRef:  rec.ExternalRef,
		Error:        rec.ErrorMessage,
		AttemptCount: rec.AttemptCount,
	}
	h.mu.RLock()
	subs := h.users[rec.UserID]
	for ch := range subs {
		select { // non-blocking
		case ch <- evt:
		default:
		}
	}
	h.mu.RUnlock()
}
