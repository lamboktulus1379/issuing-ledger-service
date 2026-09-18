--liquibase formatted sql

--changeset lamboktulus1379:1 labels:my_project-label context:my_project-context
--preconditions onFail:WARN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'user';
--comment: my_project comment
create table public.user (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name varchar(50) not null,
    user_name varchar(50),
    password varchar(50),
    created_by varchar(50),
    updated_by varchar(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
--rollback DROP TABLE public.user;

--changeset lamboktulus1379:2 labels:initialize context:development
--preconditions onFail:WARN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'user';
--comment: creating project table
CREATE TABLE public.project
(
    id serial PRIMARY KEY,
    name character varying NOT NULL,
    description character varying,
    created_at time with time zone DEFAULT NOW(),
    updated_at time with time zone DEFAULT CURRENT_TIMESTAMP
);
--rollback DROP TABLE public.project;

--changeset lamboktulus1379:3 labels:share-feature context:share
--comment: create share tracking table (video_share_records) for social platforms
CREATE TABLE IF NOT EXISTS public.video_share_records (
    id BIGSERIAL PRIMARY KEY,
    video_id VARCHAR(128) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending | success | failed
    error_message TEXT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_video_platform_user UNIQUE (video_id, platform, user_id)
);
--rollback DROP TABLE IF EXISTS public.video_share_records;

--changeset lamboktulus1379:4 labels:share-feature context:share
--comment: append-only audit log of share attempts
CREATE TABLE IF NOT EXISTS public.video_share_audit (
    id BIGSERIAL PRIMARY KEY,
    record_id BIGINT NOT NULL REFERENCES public.video_share_records(id) ON DELETE CASCADE,
    video_id VARCHAR(128) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL,
    error_message TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
--rollback DROP TABLE IF EXISTS public.video_share_audit;

--changeset lamboktulus1379:5 labels:share-feature context:share
--comment: job queue for server_post processing
CREATE TABLE IF NOT EXISTS public.share_jobs (
    id BIGSERIAL PRIMARY KEY,
    record_id BIGINT NOT NULL REFERENCES public.video_share_records(id) ON DELETE CASCADE,
    platform VARCHAR(32) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending | running | success | failed
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
--rollback DROP TABLE IF EXISTS public.share_jobs;

--changeset lamboktulus1379:6 labels:share-feature context:share
--comment: oauth tokens per user/platform for future automatic posting
CREATE TABLE IF NOT EXISTS public.oauth_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(128) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NULL,
    expires_at TIMESTAMPTZ NULL DEFAULT NULL,
    scopes TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_platform UNIQUE (user_id, platform)
);
--rollback DROP TABLE IF EXISTS public.oauth_tokens;

--changeset lamboktulus1379:7 labels:share-feature context:share
--comment: add page and token type fields for facebook page posting
ALTER TABLE public.oauth_tokens
    ADD COLUMN IF NOT EXISTS page_id VARCHAR(128) NULL,
    ADD COLUMN IF NOT EXISTS page_name VARCHAR(255) NULL,
    ADD COLUMN IF NOT EXISTS token_type VARCHAR(32) NULL; -- user | page
--rollback ALTER TABLE public.oauth_tokens DROP COLUMN IF EXISTS token_type; ALTER TABLE public.oauth_tokens DROP COLUMN IF EXISTS page_name; ALTER TABLE public.oauth_tokens DROP COLUMN IF EXISTS page_id;

--changeset lamboktulus1379:8 labels:share-feature context:share
--comment: add external_ref columns for tracking platform post IDs
ALTER TABLE IF EXISTS public.video_share_records
    ADD COLUMN IF NOT EXISTS external_ref TEXT NULL;
ALTER TABLE IF EXISTS public.share_jobs
    ADD COLUMN IF NOT EXISTS external_ref TEXT NULL;
--rollback ALTER TABLE IF EXISTS public.share_jobs DROP COLUMN IF EXISTS external_ref; ALTER TABLE IF EXISTS public.video_share_records DROP COLUMN IF EXISTS external_ref;

--changeset lamboktulus1379:9 labels:share-feature context:share
--comment: add next_attempt_at for retry/backoff scheduling on jobs
ALTER TABLE IF EXISTS public.share_jobs
    ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ NULL DEFAULT NULL;
--rollback ALTER TABLE IF EXISTS public.share_jobs DROP COLUMN IF EXISTS next_attempt_at;

--changeset lamboktulus1379:10 labels:share-feature context:share
--comment: add error_code columns to classify failures
ALTER TABLE IF EXISTS public.share_jobs
    ADD COLUMN IF NOT EXISTS error_code VARCHAR(64) NULL;
ALTER TABLE IF EXISTS public.video_share_records
    ADD COLUMN IF NOT EXISTS error_code VARCHAR(64) NULL;
--rollback ALTER TABLE IF EXISTS public.video_share_records DROP COLUMN IF EXISTS error_code; ALTER TABLE IF EXISTS public.share_jobs DROP COLUMN IF EXISTS error_code;

--changeset lamboktulus1379:11 labels:share-feature context:share
--comment: soft delete support for video_share_records
ALTER TABLE IF EXISTS public.video_share_records
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL DEFAULT NULL;
--rollback ALTER TABLE IF EXISTS public.video_share_records DROP COLUMN IF EXISTS deleted_at;

--changeset lamboktulus1379:12 labels:share-feature context:share
--comment: drop legacy unique constraint if present (simplified without DO block) and add partial unique index for active rows
ALTER TABLE public.video_share_records DROP CONSTRAINT IF EXISTS uq_video_platform_user;
CREATE UNIQUE INDEX IF NOT EXISTS uq_video_platform_user_active
    ON public.video_share_records (video_id, platform, user_id)
    WHERE deleted_at IS NULL;
--rollback DROP INDEX IF EXISTS uq_video_platform_user_active;
