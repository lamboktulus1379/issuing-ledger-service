package persistence

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
)

func NewNativeDb() (*sql.DB, error) {
	cfg := configuration.C.Database.MySql

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := otelsql.Open("mysql", dsn,
		otelsql.WithDBSystem("mysql"),
		otelsql.WithDBName(cfg.Name),
	)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(100)
	db.SetMaxIdleConns(100)
	db.SetConnMaxLifetime(time.Hour)
	otelsql.ReportDBStatsMetrics(db,
		otelsql.WithDBSystem("mysql"),
		otelsql.WithDBName(cfg.Name),
	)

	// Paksa Go membuka koneksi TCP untuk memverifikasi kredensial
	if err := db.Ping(); err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}

	return db, nil
}
