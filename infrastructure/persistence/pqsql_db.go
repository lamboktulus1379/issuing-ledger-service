package persistence

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/configuration"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

func NewPostgreSQLDB() (*sql.DB, error) {
	cfg := configuration.C.Database.Psql

	port, err := strconv.Atoi(cfg.Port)
	if err != nil {
		logger.GetLogger().
			WithField("error", err).
			WithField("port", cfg.Port).
			Error("Error while converting postgres port to int")
		return nil, err
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&search_path=public",
		cfg.User,
		cfg.Password,
		cfg.Host,
		port,
		cfg.Name,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while connection to postgres")
		return nil, err
	}
	db.SetConnMaxIdleTime(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 5)

	_, err = db.Exec("SET SEARCH_PATH TO public")
	if err != nil {
		fmt.Println(err)
	}

	return db, nil
}
