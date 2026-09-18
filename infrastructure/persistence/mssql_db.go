package persistence

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"

	_ "github.com/microsoft/go-mssqldb"
)

// NewMSSQLDB creates a sql.DB for Azure SQL / SQL Server using native database/sql.
func NewMSSQLDB() (*sql.DB, error) {
	cfg := configuration.C.Database.Mssql

	// Build sqlserver:// user:pass@host:port?database=DB&encrypt=true
	q := url.Values{}
	if cfg.Name != "" {
		q.Set("database", cfg.Name)
	}
	// Azure SQL requires encrypt=true; trust server cert left false by default
	q.Set("encrypt", "true")
	// For local/dev containers, allow trusting the self-signed server certificate unless explicitly disabled via env
	// If host looks like localhost or 127.0.0.1, set TrustServerCertificate=true
	host := cfg.Host
	if host == "localhost" || host == "127.0.0.1" {
		q.Set("TrustServerCertificate", "true")
	}
	// Recommended: disable retry to rely on driver defaults

	u := &url.URL{Scheme: "sqlserver", Host: fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)}
	if cfg.User != "" {
		if cfg.Password != "" {
			u.User = url.UserPassword(cfg.User, cfg.Password)
		} else {
			u.User = url.User(cfg.User)
		}
	}
	u.RawQuery = q.Encode()

	dsn := u.String()
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	// Verify connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
