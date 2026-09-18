# Sharing Feature Checklist (Backend + Frontend)

This document tracks completed work, near‑term next steps, and longer‑term backlog items for the social sharing subsystem (Facebook + future platforms) spanning backend (Go) and frontend (Angular).

## ✅ Completed (Baseline)
- [x] Facebook OAuth (user token + page token upgrade; manual page linking by ID or URL)
- [x] Share tracking tables + server_post job queue (video_share_records, share_jobs, audits)
- [x] Manual job trigger endpoint `/api/share/process-jobs` + background ticker
- [x] Force re-share flag (`force=true`) enabling re-queuing even after success
- [x] External reference capture (Facebook post ID) stored as `external_ref`
- [x] Auto schema ensure for `external_ref` columns if missing at startup
- [x] Enriched Facebook post content: title, description (truncated), hashtags (extracted or fallback), YouTube link
- [x] Hashtag extraction + default fallback `#alkitab #ayatalkitab #grateful`
- [x] Frontend: disable already-shared platforms unless user toggles Force; structured share UI
- [x] JWT issuer claim fix & 24h expiry extension

## 🚀 Near-Term (Promoted Next Tasks)
- [ ] Configurable Hashtags
   - [ ] Add config (env / config.json) `share.default_hashtags`
   - [ ] Add optional `share.max_auto_hashtags` and enforce cap
   - [ ] De-dupe extracted + default hashtags case-insensitively
- [ ] Share Status Preload in UI
   - [ ] Implement `GET /api/youtube/videos/:id/share-status` (backend)
   - [ ] Pre-populate `sharedPlatforms` on video details load (frontend)
   - [ ] Display last shared timestamp & `external_ref` tooltip (frontend)
- [ ] Delete Facebook Post Endpoint
   - [ ] Implement `DELETE /api/share/facebook/:recordId`
   - [ ] Use stored `external_ref` (post ID) to call Graph API delete
   - [ ] Update audit trail and mark record status (e.g., `deleted`) or add soft-delete
- [ ] Retry / Backoff Logic
   - [ ] Add `next_attempt_at` column (share_jobs)
   - [ ] Schedule exponential backoff (1m, 5m, 15m, 1h; cap at 5 attempts)
   - [ ] Classify transient vs terminal errors (e.g., HTTP 5xx vs permission errors)
- [ ] Template Customization
   - [ ] Configurable message template per platform: `{{title}}\n\n{{description}}\n\n{{hashtags}}\n\n{{url}}`
   - [ ] Toggles to enable/disable description and/or hashtags
- [ ] YouTube OAuth Status Endpoint
   - [ ] Implement `GET /api/youtube/oauth/status`
   - [ ] Return JSON: `{ "authenticated": true|false }`

## 📊 Medium-Term Backlog
- [ ] Analytics Fetch Endpoint
   - [ ] Given `external_ref`, fetch engagement stats (likes, shares, comments)
   - [ ] Store snapshots in `share_metrics`
- [ ] Post Health Checker
   - [ ] Scheduled job verifies `external_ref` posts still exist
   - [ ] Update record status to `removed` if missing
- [ ] Multi-Platform Expansion
   - [ ] Stub adapters for Twitter/X (API v2)
   - [ ] WhatsApp share link (client-only)
   - [ ] Instagram (consider API limitations)
   - [ ] LinkedIn adapter
- [ ] Role-Based Force Policy
   - [ ] Only admins can Force re-share
   - [ ] Re-share cooldown for regular users
- [ ] Share Attempt Classification
   - [ ] Add `error_code` field for structured error analytics
- [ ] Observability / Metrics
   - [ ] Expose Prometheus metrics (jobs processed, failures, avg latency, retries, deletions)
- [ ] Bulk Share Endpoint
   - [ ] Share multiple videos in one request with per-item results

## 🧪 Testing & Quality Enhancements
- [ ] Integration tests for share job:
   - [ ] Happy path (success)
   - [ ] Missing token
   - [ ] Missing page token
   - [ ] Retry scenario
- [ ] Mock Facebook Graph API client interface for deterministic tests
- [ ] Frontend e2e: share flow including Force toggle and status preload

## 🔐 Security & Compliance
- [ ] Encrypt stored OAuth access tokens at rest
- [ ] Add per-user rate limits for share requests
- [ ] Expand audit logs to include IP/user-agent for share & delete actions

## 🗄️ Database / Schema Roadmap
- [x] external_ref columns — Added via EnsureShareSchema
- [ ] next_attempt_at column (share_jobs) — Needed for backoff (Planned)
- [ ] error_code column (share_jobs, records) — For structured retry logic (Planned)
- [ ] share_metrics table — Analytics snapshots (Backlog)
- [ ] share_templates table (optional) — Overrides per platform/user (Backlog)

### 📦 Migrations Consolidation (Manual -> Managed)
- [ ] Audit codebase for any runtime/boot-time schema changes and replace with migrations
   - [x] Move `external_ref` additions into Liquibase (MySQL changeset :7, Postgres changeset :8)
   - [x] Add `next_attempt_at` to `share_jobs` in Liquibase (MySQL :8, Postgres :9)
   - [x] Add `error_code` to `share_jobs` and `video_share_records` (MySQL :9, Postgres :10)
   - [x] Add `deleted_at` (nullable) to `video_share_records` for soft delete (Postgres :11)
   - [x] Replace unique constraint with partial unique index ignoring soft-deleted rows (Postgres :12)
   - [ ] Add `share_metrics` table schema (analytics snapshots)
- [ ] Remove/disable any auto schema ensure at startup (e.g., EnsureShareSchema)
- [ ] Provide idempotent Liquibase rollback for new changesets
- [ ] Document migration run steps for dev/stage/prod in README

## 🛠 Implementation Notes (Guidance)
- Prefer a thin platform adapter layer: `PlatformPoster` interface with methods `Post`, `Delete`, `FetchMetrics`
- Message templating: use `text/template` with a safe variable whitelist
- Retry scheduler: run in same ticker loop; select pending jobs where `status='pending' AND (next_attempt_at IS NULL OR next_attempt_at <= now())`
- Deletion: keep original record; add `deleted_at` nullable timestamp or new status `deleted`

## 🧾 Acceptance Criteria for Promoted Tasks
- [ ] Configurable Hashtags
   - [ ] Given custom env list, fallback hashtags are replaced
   - [ ] Extraction still de-dupes case-insensitively
- [ ] Share Status Preload
   - [ ] Visiting video page reflects shared state immediately before any manual sharing
- [ ] Delete Endpoint
   - [ ] Successful deletion removes Facebook post via Graph API
   - [ ] Deletion is audited with `external_ref`
   - [ ] Returns 200 JSON `{ "deleted": true }`
- [ ] Retry/Backoff
   - [ ] Transient network failure triggers reschedule
   - [ ] Permission error marks job as terminal failure
- [ ] Template Customization
   - [ ] Changing config template and restarting updates subsequent share messages (not retroactive)
- [ ] YouTube OAuth Status Endpoint
   - [ ] Authenticated users return `{ "authenticated": true }`
   - [ ] Unauthenticated return `{ "authenticated": false }`

## 🧭 Suggested Order of Execution
1. Configurable hashtags & template foundation (unblocks message customization)
2. Share-status preload (small FE win)
3. Delete endpoint (unlocks lifecycle completeness)
4. Retry/backoff (resilience)
5. Metrics & analytics (observability & insights)
6. YouTube OAuth status endpoint (completes OAuth flow visibility)

## 📝 Quick Commands (Reference)
(Only if manual migration needed; runtime auto-migration already covers external_ref.)
```sql
ALTER TABLE video_share_records ADD COLUMN IF NOT EXISTS external_ref TEXT;
ALTER TABLE share_jobs ADD COLUMN IF NOT EXISTS external_ref TEXT;
```

---
Keep this file updated after each feature merge (append Completed section & move items from Near-Term to Completed).
