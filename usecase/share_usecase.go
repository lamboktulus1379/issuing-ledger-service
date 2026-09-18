package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
	"github.com/lamboktulus1379/issuing-ledger-service/domain/repository"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence"
)

type ShareMode string

const (
	ShareModeTrackOnly  ShareMode = "track_only"
	ShareModeServerPost ShareMode = "server_post"
)

type ShareResult struct {
	Platform      string `json:"platform"`
	Status        string `json:"status"`
	AlreadyShared bool   `json:"alreadyShared"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

type IShareUsecase interface {
	Share(ctx context.Context, videoID, userID string, platforms []string, mode ShareMode, force bool) ([]ShareResult, error)
	GetStatus(ctx context.Context, videoID, userID string) ([]*model.VideoShareRecord, error)
	ProcessPending(ctx context.Context, batchSize int) error
	WithBroadcaster(func(*model.VideoShareRecord)) IShareUsecase
}

type shareUsecase struct {
	shareRepo   repository.IShare
	tokenRepo   repository.IOAuthToken
	ytRepo      repository.IYouTube // optional for enrichment
	allowed     map[string]struct{}
	broadcaster func(*model.VideoShareRecord) // optional SSE broadcaster
}

func NewShareUsecase(shareRepo repository.IShare, tokenRepo repository.IOAuthToken, allowed []string, ytRepo ...repository.IYouTube) IShareUsecase {
	m := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		m[strings.ToLower(a)] = struct{}{}
	}
	su := &shareUsecase{shareRepo: shareRepo, tokenRepo: tokenRepo, allowed: m}
	if len(ytRepo) > 0 {
		su.ytRepo = ytRepo[0]
	}
	return su
}

// WithBroadcaster allows injecting a broadcaster (e.g. SSE hub) after construction.
func (u *shareUsecase) WithBroadcaster(b func(*model.VideoShareRecord)) IShareUsecase {
	u.broadcaster = b
	return u
}

func (u *shareUsecase) Share(ctx context.Context, videoID, userID string, platforms []string, mode ShareMode, force bool) ([]ShareResult, error) {
	if videoID == "" || userID == "" {
		return nil, errors.New("videoID and userID required")
	}
	if len(platforms) == 0 {
		return nil, errors.New("platforms required")
	}
	norm := make([]string, 0, len(platforms))
	for _, p := range platforms {
		p = strings.ToLower(p)
		if _, ok := u.allowed[p]; !ok {
			return nil, errors.New("unsupported platform: " + p)
		}
		norm = append(norm, p)
	}
	initialStatus := "success"
	if mode == ShareModeServerPost {
		initialStatus = "pending"
	}
	records, err := u.shareRepo.UpsertTrackShares(ctx, videoID, userID, norm, initialStatus)
	if err != nil {
		return nil, err
	}
	if mode == ShareModeServerPost {
		errEnq := u.shareRepo.EnqueueJobs(ctx, records)
		if errEnq != nil {
			logger.GetLogger().WithError(errEnq).Error("enqueue share jobs failed")
		}
	}
	results := make([]ShareResult, 0, len(records))
	for _, r := range records {
		// Consider a share already done if status is success (attempt count may not increment on re-calls)
		already := r.Status == "success" && !force
		if already && force {
			// force re-share: reset status to pending and enqueue job if server_post
			if mode == ShareModeServerPost {
				// Direct DB update to set pending again
				errMsg := (*string)(nil)
				_ = u.shareRepo.UpdateRecordStatus(ctx, r.ID, "pending", errMsg)
				_ = u.shareRepo.EnqueueJobs(ctx, []*model.VideoShareRecord{r})
				already = false
				r.Status = "pending"
			}
		}
		msg := ""
		if r.ErrorMessage != nil {
			msg = *r.ErrorMessage
		}
		results = append(results, ShareResult{Platform: r.Platform, Status: r.Status, AlreadyShared: already, ErrorMessage: msg})
		// Immediate broadcast of current state (track_only success or server_post pending)
		if u.broadcaster != nil {
			// copy to avoid race
			recCopy := *r
			go u.broadcaster(&recCopy)
		}
	}
	// If server_post, optionally trigger immediate async processing (fire-and-forget)
	if mode == ShareModeServerPost {
		go func() {
			logger.GetLogger().WithFields(map[string]interface{}{"video_id": videoID, "user_id": userID, "platforms": norm}).Info("processing share jobs immediately")
			// small context timeout to avoid hanging
			c2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if errProc := ProcessShareJobs(c2, u.shareRepo, u.tokenRepo, u.ytRepo, 5, func(rec *model.VideoShareRecord) {
				if u.broadcaster != nil {
					u.broadcaster(rec)
				}
			}); errProc != nil {
				logger.GetLogger().WithError(errProc).Error("immediate job processing failed")
			}
		}()
	}
	return results, nil
}

func (u *shareUsecase) GetStatus(ctx context.Context, videoID, userID string) ([]*model.VideoShareRecord, error) {
	return u.shareRepo.GetShareStatus(ctx, videoID, userID)
}

func (u *shareUsecase) ProcessPending(ctx context.Context, batchSize int) error {
	return ProcessShareJobs(ctx, u.shareRepo, u.tokenRepo, u.ytRepo, batchSize)
}

// ProcessShareJobs processes pending jobs (placeholder platform logic)
func ProcessShareJobs(ctx context.Context, shareRepo repository.IShare, tokenRepo repository.IOAuthToken, ytRepo repository.IYouTube, batchSize int, callbacks ...func(*model.VideoShareRecord)) error {
	lg := logger.GetLogger()
	jobs, err := shareRepo.FetchPendingJobs(ctx, batchSize)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		_ = shareRepo.MarkJobRunning(ctx, job.ID)
		success := false
		var errMsg *string
		platform := strings.ToLower(job.Platform)
		retryable := false
		switch platform {
		case "facebook":
			// Look up record to get user & video context
			rec, rErr := shareRepo.GetRecordByID(ctx, job.RecordID)
			if rErr != nil || rec == nil {
				m := "record_lookup_failed"
				errMsg = &m
				lg.WithField("job_id", job.ID).WithError(rErr).Warn("facebook share: record lookup failed")
				break
			}
			tok, tErr := tokenRepo.GetToken(ctx, rec.UserID, "facebook")
			if tErr != nil || tok == nil || tok.AccessToken == "" {
				m := "missing_token"
				errMsg = &m
				lg.WithField("job_id", job.ID).WithError(tErr).Warn("facebook share: missing token")
				break
			}
			if tok.PageID == nil {
				m := "no_page_token"
				errMsg = &m
				lg.WithField("job_id", job.ID).Warn("facebook share: no page id linked")
				break
			}
			// Build a richer post using the public YouTube watch link so Facebook generates a preview card.
			watchURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", rec.VideoID)
			title := rec.VideoID
			desc := ""
			if ytRepo != nil {
				ctxVid, cancel := context.WithTimeout(ctx, 2*time.Second)
				if v, vErr := ytRepo.GetVideoDetails(ctxVid, rec.VideoID); vErr == nil && v != nil {
					if v.Title != "" {
						title = v.Title
					}
					if v.Description != "" {
						desc = v.Description
					}
				}
				cancel()
			}
			// Compose simplified message: title, optional description, URL
			if desc != "" {
				if len(desc) > 500 {
					desc = desc[:497] + "..."
				}
			}
			// Extract existing hashtags from title/description
			hashtagSet := make(map[string]struct{})
			extractTags := func(text string) {
				word := strings.Builder{}
				runes := []rune(text)
				for i := 0; i < len(runes); i++ {
					if runes[i] == '#' {
						word.Reset()
						word.WriteRune('#')
						j := i + 1
						for j < len(runes) {
							r := runes[j]
							if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
								word.WriteRune(r)
								j++
								continue
							}
							break
						}
						tag := word.String()
						if len(tag) > 1 {
							hashtagSet[strings.ToLower(tag)] = struct{}{}
						}
						i = j - 1
					}
				}
			}
			extractTags(title)
			extractTags(desc)
			// If none, use defaults
			if len(hashtagSet) == 0 {
				hashtagSet["#alkitab"] = struct{}{}
				hashtagSet["#ayatalkitab"] = struct{}{}
				hashtagSet["#grateful"] = struct{}{}
			}
			// Build stable slice (preserve default order preference)
			ordered := []string{}
			preferredOrder := []string{"#alkitab", "#ayatalkitab", "#grateful"}
			for _, d := range preferredOrder {
				if _, ok := hashtagSet[d]; ok {
					ordered = append(ordered, d)
				}
			}
			// Add any other discovered hashtags (excluding already added)
			for k := range hashtagSet {
				found := false
				for _, o := range ordered {
					if o == k {
						found = true
						break
					}
				}
				if !found && k != "#alkitab" && k != "#ayatalkitab" && k != "#grateful" {
					ordered = append(ordered, k)
				}
			}
			parts := []string{title}
			if desc != "" {
				parts = append(parts, desc)
			}
			if len(ordered) > 0 {
				parts = append(parts, strings.Join(ordered, " "))
			}
			parts = append(parts, watchURL)
			rawMessage := strings.Join(parts, "\n\n")
			form := url.Values{}
			form.Set("message", rawMessage)
			form.Set("link", watchURL)
			form.Set("access_token", tok.AccessToken)
			postURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/feed", url.PathEscape(*tok.PageID))
			// Per-request timeout (10s) to avoid using possibly soon-to-expire batch ctx
			reqCtx, cancelReq := context.WithTimeout(ctx, 10*time.Second)
			req, _ := http.NewRequestWithContext(reqCtx, http.MethodPost, postURL, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, pErr := http.DefaultClient.Do(req)
			cancelReq()
			if pErr != nil {
				m := "post_request_failed"
				errMsg = &m
				lg.WithField("job_id", job.ID).WithError(pErr).Warn("facebook share: request error")
				retryable = true
				break
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != 200 {
				m := fmt.Sprintf("facebook_post_failed:%s", string(body))
				errMsg = &m
				lg.WithField("job_id", job.ID).WithField("status", resp.StatusCode).WithField("body", string(body)).Warn("facebook share: non-200 response")
				if resp.StatusCode >= 500 || resp.StatusCode == 429 {
					retryable = true
				}
				break
			}
			// Parse post id if present
			var fbResp struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(body, &fbResp) == nil && fbResp.ID != "" {
				_ = shareRepo.UpdateRecordExternalRef(ctx, rec.ID, fbResp.ID)
			}
			success = true
		case "twitter":
			m := "not_implemented"
			errMsg = &m
		default:
			m := "unsupported_platform"
			errMsg = &m
		}
		_ = shareRepo.MarkJobResult(ctx, job.ID, success, errMsg)
		status := "failed"
		if success {
			status = "success"
		}
		_ = shareRepo.UpdateRecordStatus(ctx, job.RecordID, status, errMsg)
		// Simple single retry logic
		if !success && retryable && job.Attempts+1 < 2 {
			lg.WithField("job_id", job.ID).Info("requeueing facebook job for retry")
			if sr, ok := shareRepo.(*persistence.ShareRepository); ok {
				// direct SQL to reset status
				if _, err := sr.DB().ExecContext(ctx, `UPDATE share_jobs SET status='pending', updated_at=$1 WHERE id=$2`, time.Now().UTC(), job.ID); err == nil {
					continue
				}
			} else if sr2, ok := shareRepo.(*persistence.ShareRepositoryMSSQL); ok {
				if _, err := sr2.DB().ExecContext(ctx, `UPDATE dbo.[share_jobs] SET status='pending', updated_at=@p1 WHERE id=@p2`, time.Now().UTC(), job.ID); err == nil {
					continue
				}
			}
		}
		// Fetch record for accurate audit + broadcast
		if rec, rErr := shareRepo.GetRecordByID(ctx, job.RecordID); rErr == nil && rec != nil {
			_ = shareRepo.CreateAudit(ctx, []*model.VideoShareAudit{{RecordID: job.RecordID, VideoID: rec.VideoID, Platform: platform, UserID: rec.UserID, Status: status, ErrorMessage: errMsg, CreatedAt: time.Now().UTC()}})
			for _, cb := range callbacks {
				cb(rec)
			}
		} else {
			_ = shareRepo.CreateAudit(ctx, []*model.VideoShareAudit{{RecordID: job.RecordID, VideoID: "", Platform: platform, UserID: "", Status: status, ErrorMessage: errMsg, CreatedAt: time.Now().UTC()}})
		}
		if errMsg != nil {
			lg.WithField("job_id", job.ID).WithField("platform", platform).WithField("error", *errMsg).Warn("share job failed")
		}
	}
	return nil
}
