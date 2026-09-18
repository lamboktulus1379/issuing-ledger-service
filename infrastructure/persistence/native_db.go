package persistence

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"
)

func NewNativeDb() (*sql.DB, error) {
	cfg := configuration.C.Database.MySql

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 5)

	return db, nil
}
