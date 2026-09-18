--liquibase formatted sql

--changeset lamboktulus1379:mssql-1 labels:core context:production
--preconditions onFail:MARK_RAN onError:MARK_RAN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM sys.tables t JOIN sys.schemas s ON s.schema_id = t.schema_id WHERE s.name = 'dbo' AND t.name = 'users';
--comment: Create dbo.users table for SQL Server (MSSQL)
CREATE TABLE dbo.[users] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] VARCHAR(50) NOT NULL,
    [user_name] VARCHAR(50) NOT NULL,
    [password] VARCHAR(255) NULL,
    [created_by] VARCHAR(50) NULL,
    [updated_by] VARCHAR(50) NULL,
    [created_at] DATETIME2(7) NOT NULL CONSTRAINT DF_users_created_at DEFAULT SYSUTCDATETIME(),
    [updated_at] DATETIME2(7) NOT NULL CONSTRAINT DF_users_updated_at DEFAULT SYSUTCDATETIME(),
    CONSTRAINT UQ_users_user_name UNIQUE ([user_name])
);
--rollback DROP TABLE dbo.[users];

--changeset lamboktulus1379:mssql-2 labels:initialize context:development
--preconditions onFail:MARK_RAN onError:MARK_RAN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM sys.tables t JOIN sys.schemas s ON s.schema_id = t.schema_id WHERE s.name = 'dbo' AND t.name = 'project';
--comment: Create dbo.project table
CREATE TABLE dbo.[project] (
    [id] INT IDENTITY(1,1) PRIMARY KEY,
    [name] VARCHAR(255) NOT NULL,
    [description] VARCHAR(1024) NULL,
    [created_at] DATETIME2(7) NOT NULL CONSTRAINT DF_project_created_at DEFAULT SYSUTCDATETIME(),
    [updated_at] DATETIME2(7) NOT NULL CONSTRAINT DF_project_updated_at DEFAULT SYSUTCDATETIME()
);
--rollback DROP TABLE dbo.[project];

--changeset lamboktulus1379:mssql-3 labels:share-feature context:share
--comment: create share tracking table (video_share_records) for social platforms
--preconditions onFail:MARK_RAN onError:MARK_RAN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM sys.tables t JOIN sys.schemas s ON s.schema_id = t.schema_id WHERE s.name = 'dbo' AND t.name = 'video_share_records';
CREATE TABLE dbo.[video_share_records] (
    [id] BIGINT IDENTITY(1,1) PRIMARY KEY,
    [video_id] VARCHAR(128) NOT NULL,
    [platform] VARCHAR(32) NOT NULL,
    [user_id] VARCHAR(128) NOT NULL,
    [status] VARCHAR(16) NOT NULL CONSTRAINT DF_vsr_status DEFAULT 'pending',
    [error_message] VARCHAR(MAX) NULL,
    [attempt_count] INT NOT NULL CONSTRAINT DF_vsr_attempt_count DEFAULT 1,
    [created_at] DATETIME2(7) NOT NULL CONSTRAINT DF_vsr_created_at DEFAULT SYSUTCDATETIME(),
    [updated_at] DATETIME2(7) NOT NULL CONSTRAINT DF_vsr_updated_at DEFAULT SYSUTCDATETIME(),
    CONSTRAINT UQ_video_platform_user UNIQUE ([video_id],[platform],[user_id])
);
--rollback DROP TABLE IF EXISTS dbo.[video_share_records];

--changeset lamboktulus1379:mssql-4 labels:share-feature context:share
--comment: append-only audit log of share attempts
--preconditions onFail:MARK_RAN onError:MARK_RAN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM sys.tables t JOIN sys.schemas s ON s.schema_id = t.schema_id WHERE s.name = 'dbo' AND t.name = 'video_share_audit';
CREATE TABLE dbo.[video_share_audit] (
    [id] BIGINT IDENTITY(1,1) PRIMARY KEY,
    [record_id] BIGINT NOT NULL,
    [video_id] VARCHAR(128) NOT NULL,
    [platform] VARCHAR(32) NOT NULL,
    [user_id] VARCHAR(128) NOT NULL,
    [status] VARCHAR(16) NOT NULL,
    [error_message] VARCHAR(MAX) NULL,
    [created_at] DATETIME2(7) NOT NULL CONSTRAINT DF_vsa_created_at DEFAULT SYSUTCDATETIME(),
    CONSTRAINT FK_vsa_record FOREIGN KEY ([record_id]) REFERENCES dbo.[video_share_records]([id]) ON DELETE CASCADE
);
--rollback DROP TABLE IF EXISTS dbo.[video_share_audit];

--changeset lamboktulus1379:mssql-5 labels:share-feature context:share
--comment: job queue for server_post processing
--preconditions onFail:MARK_RAN onError:MARK_RAN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM sys.tables t JOIN sys.schemas s ON s.schema_id = t.schema_id WHERE s.name = 'dbo' AND t.name = 'share_jobs';
CREATE TABLE dbo.[share_jobs] (
    [id] BIGINT IDENTITY(1,1) PRIMARY KEY,
    [record_id] BIGINT NOT NULL,
    [platform] VARCHAR(32) NOT NULL,
    [status] VARCHAR(16) NOT NULL CONSTRAINT DF_sj_status DEFAULT 'pending',
    [attempts] INT NOT NULL CONSTRAINT DF_sj_attempts DEFAULT 0,
    [last_error] VARCHAR(MAX) NULL,
    [created_at] DATETIME2(7) NOT NULL CONSTRAINT DF_sj_created_at DEFAULT SYSUTCDATETIME(),
    [updated_at] DATETIME2(7) NOT NULL CONSTRAINT DF_sj_updated_at DEFAULT SYSUTCDATETIME(),
    CONSTRAINT FK_sj_record FOREIGN KEY ([record_id]) REFERENCES dbo.[video_share_records]([id]) ON DELETE CASCADE
);
--rollback DROP TABLE IF EXISTS dbo.[share_jobs];

--changeset lamboktulus1379:mssql-6 labels:share-feature context:share
--comment: oauth tokens per user/platform for future automatic posting
--preconditions onFail:MARK_RAN onError:MARK_RAN
--precondition-sql-check expectedResult:0 SELECT COUNT(*) FROM sys.tables t JOIN sys.schemas s ON s.schema_id = t.schema_id WHERE s.name = 'dbo' AND t.name = 'oauth_tokens';
CREATE TABLE dbo.[oauth_tokens] (
    [id] BIGINT IDENTITY(1,1) PRIMARY KEY,
    [user_id] VARCHAR(128) NOT NULL,
    [platform] VARCHAR(32) NOT NULL,
    [access_token] VARCHAR(MAX) NOT NULL,
    [refresh_token] VARCHAR(MAX) NULL,
    [expires_at] DATETIME2(7) NULL,
    [scopes] VARCHAR(MAX) NOT NULL,
    [created_at] DATETIME2(7) NOT NULL CONSTRAINT DF_ot_created_at DEFAULT SYSUTCDATETIME(),
    [updated_at] DATETIME2(7) NOT NULL CONSTRAINT DF_ot_updated_at DEFAULT SYSUTCDATETIME(),
    CONSTRAINT UQ_oauth_user_platform UNIQUE ([user_id],[platform])
);
--rollback DROP TABLE IF EXISTS dbo.[oauth_tokens];

--changeset lamboktulus1379:mssql-7 labels:share-feature context:share
--comment: add page and token type fields for facebook page posting
--preconditions onFail:CONTINUE onError:CONTINUE
IF COL_LENGTH('dbo.oauth_tokens','page_id') IS NULL
BEGIN
    ALTER TABLE dbo.[oauth_tokens] ADD [page_id] VARCHAR(128) NULL;
END
IF COL_LENGTH('dbo.oauth_tokens','page_name') IS NULL
BEGIN
    ALTER TABLE dbo.[oauth_tokens] ADD [page_name] VARCHAR(255) NULL;
END
IF COL_LENGTH('dbo.oauth_tokens','token_type') IS NULL
BEGIN
    ALTER TABLE dbo.[oauth_tokens] ADD [token_type] VARCHAR(32) NULL; -- user | page
END
--rollback
IF COL_LENGTH('dbo.oauth_tokens','token_type') IS NOT NULL BEGIN ALTER TABLE dbo.[oauth_tokens] DROP COLUMN [token_type]; END
IF COL_LENGTH('dbo.oauth_tokens','page_name') IS NOT NULL BEGIN ALTER TABLE dbo.[oauth_tokens] DROP COLUMN [page_name]; END
IF COL_LENGTH('dbo.oauth_tokens','page_id') IS NOT NULL BEGIN ALTER TABLE dbo.[oauth_tokens] DROP COLUMN [page_id]; END

--changeset lamboktulus1379:mssql-8 labels:share-feature context:share
--comment: add external_ref columns for tracking platform post IDs
--preconditions onFail:CONTINUE onError:CONTINUE
IF COL_LENGTH('dbo.video_share_records','external_ref') IS NULL
BEGIN
    ALTER TABLE dbo.[video_share_records] ADD [external_ref] VARCHAR(MAX) NULL;
END
IF COL_LENGTH('dbo.share_jobs','external_ref') IS NULL
BEGIN
    ALTER TABLE dbo.[share_jobs] ADD [external_ref] VARCHAR(MAX) NULL;
END
--rollback
IF COL_LENGTH('dbo.share_jobs','external_ref') IS NOT NULL BEGIN ALTER TABLE dbo.[share_jobs] DROP COLUMN [external_ref]; END
IF COL_LENGTH('dbo.video_share_records','external_ref') IS NOT NULL BEGIN ALTER TABLE dbo.[video_share_records] DROP COLUMN [external_ref]; END

--changeset lamboktulus1379:mssql-9 labels:share-feature context:share
--comment: add next_attempt_at for retry/backoff scheduling on jobs
--preconditions onFail:CONTINUE onError:CONTINUE
IF COL_LENGTH('dbo.share_jobs','next_attempt_at') IS NULL
BEGIN
    ALTER TABLE dbo.[share_jobs] ADD [next_attempt_at] DATETIME2(7) NULL;
END
--rollback IF COL_LENGTH('dbo.share_jobs','next_attempt_at') IS NOT NULL BEGIN ALTER TABLE dbo.[share_jobs] DROP COLUMN [next_attempt_at]; END

--changeset lamboktulus1379:mssql-10 labels:share-feature context:share
--comment: add error_code columns to classify failures
--preconditions onFail:CONTINUE onError:CONTINUE
IF COL_LENGTH('dbo.share_jobs','error_code') IS NULL
BEGIN
    ALTER TABLE dbo.[share_jobs] ADD [error_code] VARCHAR(64) NULL;
END
IF COL_LENGTH('dbo.video_share_records','error_code') IS NULL
BEGIN
    ALTER TABLE dbo.[video_share_records] ADD [error_code] VARCHAR(64) NULL;
END
--rollback
IF COL_LENGTH('dbo.video_share_records','error_code') IS NOT NULL BEGIN ALTER TABLE dbo.[video_share_records] DROP COLUMN [error_code]; END
IF COL_LENGTH('dbo.share_jobs','error_code') IS NOT NULL BEGIN ALTER TABLE dbo.[share_jobs] DROP COLUMN [error_code]; END

--changeset lamboktulus1379:mssql-11 labels:share-feature context:share
--comment: soft delete support for video_share_records
--preconditions onFail:CONTINUE onError:CONTINUE
IF COL_LENGTH('dbo.video_share_records','deleted_at') IS NULL
BEGIN
    ALTER TABLE dbo.[video_share_records] ADD [deleted_at] DATETIME2(7) NULL;
END
--rollback IF COL_LENGTH('dbo.video_share_records','deleted_at') IS NOT NULL BEGIN ALTER TABLE dbo.[video_share_records] DROP COLUMN [deleted_at]; END

--changeset lamboktulus1379:mssql-12 labels:share-feature context:share
--comment: drop legacy unique and add filtered unique index for active rows
--preconditions onFail:CONTINUE onError:CONTINUE
IF EXISTS (
    SELECT 1 FROM sys.key_constraints kc
    WHERE kc.[name] = 'UQ_video_platform_user'
      AND kc.[parent_object_id] = OBJECT_ID('dbo.video_share_records')
)
BEGIN
    ALTER TABLE dbo.[video_share_records] DROP CONSTRAINT UQ_video_platform_user;
END
IF NOT EXISTS (
    SELECT 1 FROM sys.indexes i
    WHERE i.[name] = 'IX_uq_video_platform_user_active'
      AND i.[object_id] = OBJECT_ID('dbo.video_share_records')
)
BEGIN
    CREATE UNIQUE INDEX IX_uq_video_platform_user_active
        ON dbo.[video_share_records] ([video_id], [platform], [user_id])
        WHERE [deleted_at] IS NULL;
END
--rollback
IF EXISTS (
    SELECT 1 FROM sys.indexes i WHERE i.[name] = 'IX_uq_video_platform_user_active' AND i.[object_id] = OBJECT_ID('dbo.video_share_records')
)
BEGIN
    DROP INDEX IX_uq_video_platform_user_active ON dbo.[video_share_records];
END
