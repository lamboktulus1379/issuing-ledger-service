package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
)

type OAuthTokenRepositoryMSSQL struct{ db *sql.DB }

func NewOAuthTokenRepositoryMSSQL(db *sql.DB) *OAuthTokenRepositoryMSSQL {
	return &OAuthTokenRepositoryMSSQL{db: db}
}

// EnsureOAuthTokenSchemaMSSQL creates the oauth_tokens table for SQL Server if it does not exist.
func EnsureOAuthTokenSchemaMSSQL(db *sql.DB) error {
	// Create table if not exists using SQL Server pattern
	ddl := `IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'dbo.oauth_tokens') AND type in (N'U'))
BEGIN
    CREATE TABLE dbo.[oauth_tokens] (
        id BIGINT IDENTITY(1,1) PRIMARY KEY,
        user_id NVARCHAR(128) NOT NULL,
        platform NVARCHAR(64) NOT NULL,
        access_token NVARCHAR(MAX) NOT NULL,
        refresh_token NVARCHAR(MAX) NULL,
        expires_at DATETIME2 NULL,
        scopes NVARCHAR(MAX) NOT NULL,
        page_id NVARCHAR(128) NULL,
        page_name NVARCHAR(255) NULL,
        token_type NVARCHAR(32) NULL,
        created_at DATETIME2 NOT NULL,
        updated_at DATETIME2 NOT NULL
    );
    CREATE UNIQUE INDEX UX_oauth_tokens_user_platform ON dbo.[oauth_tokens](user_id, platform);
END`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("create oauth_tokens (mssql): %w", err)
	}
	return nil
}

func (r *OAuthTokenRepositoryMSSQL) UpsertToken(ctx context.Context, t *model.OAuthToken) error {
	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	// Normalize nullable values for MSSQL driver
	var exp sql.NullTime
	if t.ExpiresAt != nil {
		exp.Valid = true
		exp.Time = *t.ExpiresAt
	}
	var pageID sql.NullString
	if t.PageID != nil {
		pageID.Valid = true
		pageID.String = *t.PageID
	}
	var pageName sql.NullString
	if t.PageName != nil {
		pageName.Valid = true
		pageName.String = *t.PageName
	}
	var tokenType sql.NullString
	if t.TokenType != nil {
		tokenType.Valid = true
		tokenType.String = *t.TokenType
	}
	// MERGE upsert by (user_id, platform)
	q := `MERGE dbo.[oauth_tokens] AS target
USING (VALUES (@p1, @p2)) AS src(user_id, platform)
ON target.user_id = src.user_id AND target.platform = src.platform
WHEN MATCHED THEN UPDATE SET
    access_token=@p3,
    refresh_token=@p4,
    expires_at=@p5,
    scopes=@p6,
    page_id=@p7,
    page_name=@p8,
    token_type=@p9,
    updated_at=@p11
WHEN NOT MATCHED THEN
    INSERT (user_id, platform, access_token, refresh_token, expires_at, scopes, page_id, page_name, token_type, created_at, updated_at)
    VALUES (@p1,@p2,@p3,@p4,@p5,@p6,@p7,@p8,@p9,@p10,@p11);`
	_, err := r.db.ExecContext(ctx, q,
		t.UserID, t.Platform,
		t.AccessToken,
		// Allow empty string for refresh_token; keep it nullable in schema but empty is fine
		t.RefreshToken,
		exp,
		t.Scopes,
		pageID,
		pageName,
		tokenType,
		t.CreatedAt,
		t.UpdatedAt,
	)
	return err
}

func (r *OAuthTokenRepositoryMSSQL) GetToken(ctx context.Context, userID, platform string) (*model.OAuthToken, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, user_id, platform, access_token, refresh_token, expires_at, scopes, page_id, page_name, token_type, created_at, updated_at FROM dbo.[oauth_tokens] WHERE user_id=@p1 AND platform=@p2`, userID, platform)
	tok := &model.OAuthToken{}
	var exp sql.NullTime
	var pageID, pageName, tokenType sql.NullString
	if err := row.Scan(&tok.ID, &tok.UserID, &tok.Platform, &tok.AccessToken, &tok.RefreshToken, &exp, &tok.Scopes, &pageID, &pageName, &tokenType, &tok.CreatedAt, &tok.UpdatedAt); err != nil {
		return nil, err
	}
	if exp.Valid {
		tok.ExpiresAt = &exp.Time
	}
	if pageID.Valid {
		v := pageID.String
		tok.PageID = &v
	}
	if pageName.Valid {
		v := pageName.String
		tok.PageName = &v
	}
	if tokenType.Valid {
		v := tokenType.String
		tok.TokenType = &v
	}
	return tok, nil
}
