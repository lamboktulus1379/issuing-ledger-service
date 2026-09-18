# Social Sharing & Tracking Feature (Backend Design)

## 1. Goals
Track and (where technically feasible) trigger social sharing actions for a YouTube video from the application, and expose the share status per platform to the frontend.

## 2. Scope (Current Iteration)
- Persist which social platforms a video has been (declared) shared to.
- Provide API to:
  - Query share status for a video.
  - Submit a share request for one or more platforms (idempotent per video + platform + user).
- (Optional FE assist) FE may still open native share / platform intent URLs (WhatsApp / Twitter) because true server-side posting is not always feasible.
- Record an audit trail (who, when, platform, outcome, error if any).

## 3. Important Reality Check
| Platform  | True Server-side Post Today? | Notes |
|-----------|------------------------------|-------|
| WhatsApp  | No                           | No public API for arbitrary server-side message send (only Business APIs → require phone number + approved templates + user opt-in). We'll only track & let FE open a share link. |
| Twitter/X | Possible with OAuth tokens   | Requires user auth (3-legged OAuth) & elevated API access; not implemented yet. |
| Others (FB, LinkedIn) | Possible with proper OAuth + scopes | Out of current iteration. |

Therefore: This iteration treats BE as a tracking & orchestration layer; real share UI (opening intents) remains FE responsibility. Later, for Twitter we can add token store + background publisher.

## 4. Terminology
- Share Platform: canonical string (e.g. `twitter`, `whatsapp`, `facebook`, `linkedin`).
- Share Record: A single attempt for a video-platform pair by a user.
- Share Status Aggregate: Computed per (video, platform) whether a successful record exists.

## 5. Data Model
Two tables (MySQL example):

```sql
CREATE TABLE video_share_records (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  video_id VARCHAR(64) NOT NULL,
  platform VARCHAR(32) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  status ENUM('pending','success','failed') NOT NULL DEFAULT 'pending',
  error_message TEXT NULL,
  attempt_count INT NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_video_platform_user (video_id, platform, user_id)
);

CREATE TABLE video_share_audit (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  record_id BIGINT NOT NULL,
  video_id VARCHAR(64) NOT NULL,
  platform VARCHAR(32) NOT NULL,
  user_id VARCHAR(64) NOT NULL,
  status ENUM('pending','success','failed') NOT NULL,
  error_message TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_video_platform (video_id, platform),
  CONSTRAINT fk_share_audit_record FOREIGN KEY (record_id) REFERENCES video_share_records(id)
);
```

### Rationale
- `video_share_records` stores the latest state per (video, platform, user). Idempotent updates.
- `video_share_audit` is append-only for historical trace (optional; can defer if minimalism is desired).

## 6. API Design

### 6.1 POST /videos/:videoId/share
Request:
```json
{
  "platforms": ["twitter", "whatsapp"],
  "mode": "track_only" // future: "server_post" to request BE to actually post when implemented
}
```
Behavior:
1. Validate platforms against an allowlist.
2. Upsert/create `video_share_records` rows (status = success immediately for track_only, or pending if server_post needs async job).
3. Insert audit entries.
4. Return aggregate statuses.

Response (example):
```json
{
  "video_id": "abc123",
  "results": [
    { "platform": "twitter", "status": "success", "alreadyShared": false },
    { "platform": "whatsapp", "status": "success", "alreadyShared": true }
  ]
}
```

Idempotency: If a record exists with status success, mark `alreadyShared: true` and DO NOT change state (unless forced re-share is a future flag).

### 6.2 GET /videos/:videoId/share-status
Response:
```json
{
  "video_id": "abc123",
  "platforms": [
    { "platform": "twitter", "shared": true, "lastSharedAt": "2025-08-09T12:34:56Z" },
    { "platform": "whatsapp", "shared": false }
  ]
}
```

### 6.3 (Future) POST /videos/:videoId/share/:platform/retry
Triggers an async retry if last failed.

## 7. Backend Flow (track_only mode)
1. FE sends POST share request (selected or default all platforms).
2. BE validates & writes success records (representing user declared share) instantly.
3. FE concurrently opens native share URLs (handled by FE, not BE) to let user actually share.
4. FE updates UI using POST response OR calls GET share-status.

## 8. Future Server-side Posting Flow (server_post mode)
Additional prerequisites:
- OAuth token store table per platform (encrypted).
- Background worker / queue (e.g., use existing worker pattern) consumes pending share jobs.
- For each platform implement a client (e.g., Twitter API) & error handling; update status & audit.

## 9. Allowlist & Validation
Config (e.g., in `config.json`):
```json
{
  "share_platforms": ["twitter", "whatsapp", "facebook", "linkedin"]
}
```
Reject request if unknown platform present.

## 10. Security & Auth
- Require authenticated user (already have user_id in context for reactions).
- Authorization: Ensure user has permission to share (maybe all authenticated users allowed initially).
- Rate limiting (optional) to prevent abuse (e.g., re-share spamming).

## 11. Error Handling
- Platform unsupported → 400 with list of unsupported entries.
- DB failure → 500 (log and return generic error).
- Partial success (future async) → return mixed statuses; caller can poll.

## 12. Open Questions / Assumptions
| Topic | Assumption | Action |
|-------|------------|--------|
| User OAuth tokens | Not yet implemented | Add table & flows when enabling real posting |
| WhatsApp real server post | Not possible for generic flow | FE intent only |
| Re-share override | Not required now | Add `force` flag later |
| Bulk share all default | Yes (default platforms list) | FE loads allowlist then selects all |
| DB engine | MySQL (assumed existing) | Confirm & add Liquibase changeset |

## 13. Implementation Steps (Backend)
1. Add Liquibase changeset (or migration) for the two tables.
2. Domain model structs (VideoShareRecord, VideoShareAuditEntry).
3. Repository methods:
   - UpsertShareRecords(videoID, userID, platforms[])
   - GetShareStatus(videoID)
4. Usecase service wrapping repository.
5. HTTP handlers:
   - POST /videos/:videoId/share
   - GET /videos/:videoId/share-status
6. Integrate router.
7. Add unit tests for repository & handlers (idempotency, unknown platform, partial exist).
8. Update README & API reference.

## 14. Frontend Interaction (Summary)
(See frontend doc for details.)

## 15. Metrics & Observability
- Count shares per platform (prometheus counter if metrics system exists).
- Error counter for failed share attempts.

## 16. Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| Misinterpretation that BE truly posts to all platforms | Explicitly label current mode as tracking only. |
| Race conditions on rapid double clicks | Rely on unique constraint + idempotent logic. |
| Future OAuth complexity | Isolate platform clients behind interface for testability. |

## 17. Example Handler Pseudocode (POST)
```go
func (h *VideoShareHandler) ShareVideo(c *gin.Context) {
  vid := c.Param("videoId")
  userID := c.GetString("user_id")
  var req ShareRequest
  if err := c.BindJSON(&req); err != nil { /* 400 */ }
  platforms := normalize(req.Platforms)
  if err := validate(platforms, h.cfg.AllowedPlatforms); err != nil { /* 400 */ }
  results, err := h.usecase.TrackShares(vid, userID, platforms)
  if err != nil { /* 500 */ }
  c.JSON(http.StatusOK, ShareResponse{VideoID: vid, Results: results})
}
```

## 18. Next Iterations
1. Add real Twitter posting (OAuth + background worker).
2. Add UI for viewing audit history.
3. Add re-share force and retry endpoints.
4. Extend to other platforms.

---
Status: DESIGN ONLY (no code yet). Adjust assumptions before implementation.
