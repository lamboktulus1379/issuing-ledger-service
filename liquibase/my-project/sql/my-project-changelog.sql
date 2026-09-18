--liquibase formatted sql

--changeset lamboktulus1379:1 labels:my-project-label context:my-project-context
--comment: my-project comment
create table user (
    id int primary key auto_increment not null,
    name varchar(50) not null,
    user_name varchar(50),
    password varchar(50),
    created_by varchar(50),
    updated_by varchar(50),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
)
--rollback DROP TABLE user;

--changeset lamboktulus1379:2 labels:my-project-label context:my-project-context
--comment: my-project comment
ALTER TABLE `user`
    MODIFY `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
--rollback ALTER TABLE `user` MODIFY `updated_at` TIMESTAMP;

--changeset lamboktulus1379:3 labels:share-feature context:share
--comment: create tables for social share tracking (video_share_records, video_share_audit, share_jobs, oauth_tokens)
CREATE TABLE IF NOT EXISTS video_share_records (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    video_id VARCHAR(128) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending | success | failed
    error_message TEXT NULL,
    attempt_count INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_video_platform_user (video_id, platform, user_id),
    INDEX idx_video_user (video_id, user_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
--rollback DROP TABLE IF EXISTS video_share_records;

--changeset lamboktulus1379:4 labels:share-feature context:share
--comment: append-only audit log of share attempts
CREATE TABLE IF NOT EXISTS video_share_audit (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    record_id BIGINT NOT NULL,
    video_id VARCHAR(128) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    user_id VARCHAR(128) NOT NULL,
    status VARCHAR(16) NOT NULL,
    error_message TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_record (record_id),
    INDEX idx_video_platform (video_id, platform),
    CONSTRAINT fk_audit_record FOREIGN KEY (record_id) REFERENCES video_share_records(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
--rollback DROP TABLE IF EXISTS video_share_audit;

--changeset lamboktulus1379:5 labels:share-feature context:share
--comment: job queue for server_post processing
CREATE TABLE IF NOT EXISTS share_jobs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    record_id BIGINT NOT NULL,
    platform VARCHAR(32) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending', -- pending | running | success | failed
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_status_created (status, created_at),
    CONSTRAINT fk_job_record FOREIGN KEY (record_id) REFERENCES video_share_records(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
--rollback DROP TABLE IF EXISTS share_jobs;

--changeset lamboktulus1379:6 labels:share-feature context:share
--comment: oauth tokens per user/platform for future automatic posting
CREATE TABLE IF NOT EXISTS oauth_tokens (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id VARCHAR(128) NOT NULL,
    platform VARCHAR(32) NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NULL,
    expires_at TIMESTAMP NULL DEFAULT NULL,
    scopes TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_user_platform (user_id, platform),
    INDEX idx_platform_user (platform, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
--rollback DROP TABLE IF EXISTS oauth_tokens;

--changeset lamboktulus1379:7 labels:share-feature context:share
--comment: add external_ref columns for tracking platform post IDs
ALTER TABLE video_share_records
    ADD COLUMN IF NOT EXISTS external_ref TEXT NULL;
ALTER TABLE share_jobs
    ADD COLUMN IF NOT EXISTS external_ref TEXT NULL;
--rollback ALTER TABLE share_jobs DROP COLUMN IF EXISTS external_ref; ALTER TABLE video_share_records DROP COLUMN IF EXISTS external_ref;

--changeset lamboktulus1379:8 labels:share-feature context:share
--comment: add next_attempt_at for retry/backoff scheduling
ALTER TABLE share_jobs
    ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMP NULL DEFAULT NULL;
--rollback ALTER TABLE share_jobs DROP COLUMN IF EXISTS next_attempt_at;

--changeset lamboktulus1379:9 labels:share-feature context:share
--comment: add error_code to classify failures (jobs and records)
ALTER TABLE share_jobs
    ADD COLUMN IF NOT EXISTS error_code VARCHAR(64) NULL;
ALTER TABLE video_share_records
    ADD COLUMN IF NOT EXISTS error_code VARCHAR(64) NULL;
--rollback ALTER TABLE video_share_records DROP COLUMN IF EXISTS error_code; ALTER TABLE share_jobs DROP COLUMN IF EXISTS error_code;
